# API Contract Testing — Verification Checklist

## 1. Spec Validity & Examples
- [ ] Spectral lint passes with zero errors on operation IDs, path style, and problem-details error shapes.
- [ ] Every request and response example validates against its schema via openapi-examples-validator.
- [ ] All error codes in the catalog have a documented schema, example, and retryable flag.
- [ ] Prism mock server starts from the same spec file and serves only spec-valid responses.

## 2. Breaking-Change Detection
- [ ] oasdiff breaking (or buf breaking) runs against the last released spec in CI on every provider change.
- [ ] Findings are triaged as breaking, dangerous, or safe; zero breaking findings remain untriaged.
- [ ] Removed or renamed fields carry a Sunset header and a migration note with a 90-day window.
- [ ] New required request fields are treated as dangerous and need explicit consumer acknowledgment.

## 3. Pact Consumer-Driven Contracts
- [ ] Each consumer publishes versioned Pact files to the broker with tags such as web-2.14.0.
- [ ] Interactions declare provider states (given order ord_9f31 exists) that the provider setup honors exactly.
- [ ] Provider verification runs against a real staging deployment, not against the mock, and results publish to the broker.
- [ ] pact-broker can-i-deploy returns green for the exact consumer/provider version pair being released.

## 4. Mock Fidelity & Coverage
- [ ] Frontend and partner tests run against Prism with spec validation enabled; invalid responses fail tests.
- [ ] Every mock-only test scenario is either promoted to a Pact interaction or deleted within the same release.
- [ ] Pagination cursors, error codes, and rate-limit headers behave identically in mock and provider.

## 5. Release Evidence
- [ ] Broker matrix screenshot or JSON export is attached to the release with all cells green.
- [ ] Contract test suite (provider + consumer) passes at 100 percent with flaky tests quarantined, not ignored.
- [ ] scripts/verify.sh executes with exit code 0.
