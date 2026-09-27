# API Contract Testing Reference Guide

## 1. Core Concepts

### 1.1 Schema Validation vs Contract Testing
Schema validation proves one message matches one schema. Contract testing proves two independently
deployed systems agree over time: the consumer declares what it needs, the provider proves it still
delivers. OpenAPI plus examples gives message truth; Pact gives bilateral agreement; can-i-deploy gives
release-time proof. All three layers are needed: schemas rot silently, mocks lie politely, only broker
verification binds both sides to a deployable pair.

### 1.2 Consumer-Driven Contracts with Pact
The consumer writes the contract as executable interactions: given a provider state, upon a request,
the provider will respond with a body matching flexible matchers (like, eachLike, term). Matchers keep
contracts resilient: assert types and required fields, ignore volatile values like timestamps and UUIDs.
Provider states are setup hooks, not fixtures: `given order ord_9f31 exists` must create exactly the
record the interaction needs, isolated per run, torn down afterwards.

### 1.3 Breaking-Change Taxonomy
Breaking: removing a field or endpoint, narrowing a type (string to integer), adding a required request
field, changing auth scope requirements. Dangerous: adding an enum value consumers switch on exhaustively,
changing default semantics, loosening validation the consumer relied on. Safe: new optional fields, new
endpoints, new optional query parameters, additional error codes consumers already handle generically.
oasdiff automates detection; humans own classification, with consumer sign-off required for anything above
safe.

### 1.4 Semantic Versioning for APIs
MAJOR for breaking changes, MINOR for backward-compatible additions, PATCH for clarifications and examples.
The broker matrix enforces this socially: web-2.14.0 works with orders-api-1.4.x, and can-i-deploy refuses
orders-api-2.0.0 until consumer pacts verify against it. Sunset headers advertise removals 90 days ahead;
Deprecation: true plus a Sunset date is machine-readable courtesy.

### 1.5 Mock Servers and Fidelity
Prism turns an OpenAPI file into a validating mock: requests that violate the spec get 422 from the mock
itself, and responses are generated from examples or schemas. Fidelity rule: the mock and the provider
verification must consume the identical spec revision. A test green against the mock but red against the
provider is a spec bug, filed against the provider team with the failing interaction attached.

## 2. Patterns and Anti-Patterns

| Area | Pattern (do) | Anti-Pattern (avoid) |
|---|---|---|
| Matchers | Type and structure matchers, ignore UUIDs | Exact-equality on timestamps and generated IDs |
| States | Isolated provider-state setup per interaction | Shared staging database mutated by parallel runs |
| Versioning | Broker tags web-2.14.0 with git SHAs | Latest tag floating across incompatible mains |
| Mocks | Prism validating mode from release spec | Hand-written stub JSON drifting from the spec |
| Errors | Contract tests for 404, 409, 429 paths | Happy-path-only pacts missing failure modes |
| Secrets | Redacted pact logs in CI artifacts | Real API keys recorded in pact files |
| Gating | can-i-deploy blocks promotion on red | Manual Slack approval instead of broker proof |

## 3. Code Example: Pact Consumer Interaction and can-i-deploy Gate

```python
# consumer test (AcmeWeb) declaring what it needs from Orders API 1.4.x
from pact import Consumer, Provider

pact = Consumer("AcmeWeb").has_pact_with(Provider("OrdersAPI"))
(pact
 .given("order ord_9f31 exists")
 .upon_receiving("a request for order ord_9f31")
 .with_request("get", "/v1/orders/ord_9f31")
 .will_respond_with(200, body={
     "id": "ord_9f31",
     "total_cents": 8399,
     "currency": "BRL",
 }))
with pact:
    result = requests.get(pact.uri + "/v1/orders/ord_9f31").json()
    assert result["currency"] == "BRL"
```

```bash
# provider side: verify against staging, then gate the release pair
pact-provider-verifier --provider-base-url https://staging-api.acme.example \
  --pact-broker-base-url https://broker.acme.example --provider OrdersAPI \
  --provider-app-version 1.4.3 --publish-verification-results
pact-broker can-i-deploy --pacticipant AcmeWeb --version 2.14.0 \
  --pacticipant OrdersAPI --version 1.4.3 --to-environment production
```

The consumer test publishes a versioned pact, the verifier replays it against real staging code, and
can-i-deploy answers the only release question that matters: may this exact pair ship together.
