# ADR 020 — Model routing is quota-aware, role-aware, and decided before the call

Status: Accepted (2026-09-23)
Relates to: GAP-005, GAP-100, GAP-102, GAP-103, GAP-108, GAP-129, GAP-130, GAP-131, GAP-132, GAP-133, GAP-155, GAP-156

## Context

`docs/runtime/control-plane.md:90-96` and
`docs/runtime/model-portfolio-and-routing.md:137-184` require routing that
combines task, agent role, budget, quota, provider availability and model
capability. None of that reaches a model call.

The actual path is:

```
CLI/daemon provider flag → model.ForName() → Runner.Services.Models.Stream()
```

`internal/harness/gateway` implements selection, quota state, retry, fallback
and circuit breaking, and has **zero importers outside tests**. `internal/modelregistry`
implements a real scoring function, reached only by the standalone
`prumo model route` command, over a hard-coded three-model registry whose
providers (`local`, `trusted-cloud`, `external-cloud`) the provider factory
cannot construct (`cmd/prumo/d2_commands.go:109-141` vs
`internal/harness/model/model.go:120-166`).

Specific defects in the unwired gateway:

- `CooldownUntil` is written and never read, so an open circuit never closes
  (`gateway.go:198-220`, `77-85`);
- `quotaExhausted` reads the quota map without the mutex that `SetQuota` writes
  it under (`gateway.go:70-75` vs `165-169`);
- `LocalOnly` admits a target with an empty privacy value
  (`gateway.go:67-69`), violating the restricted-data rule in
  `docs/security/trust-model.md:76-86`;
- backoff is linear with deterministic jitter, not the exponential +
  `Retry-After` the control plane requires (`gateway.go:126-140`, `330-342`);
- `isRateLimit` does not recognise `RESOURCE_EXHAUSTED` (`gateway.go:316-327`).

Budget is post-hoc. The provider has already been called when usage arrives
(`runtime.go:333-355`); `Envelope.Consume` then compares a spent amount against
the limit (`budget/budget.go:32-41`), and `Reserve` records a reservation without
subtracting it from what is available (`budget/budget.go:23-30`).

Cost is absent for the two main providers. OpenAI-compatible and Anthropic both
populate token fields and neither sets `CostUSD` (`model/model.go:405-428`,
`anthropic.go:263-281`); the tracker trusts the field
(`runlayer/runlayer.go:40-52`). So `--budget-usd` does not constrain a run
against either provider. Cache-read and cache-write tokens are parsed and then
discarded by the tracker, and `agent.Usage` has no reasoning-token field
(`runlayer.go:40-47`, `agent/agent.go:194-205`).

The industry has settled patterns worth copying:

- **Retry inside a model group, fallback outside it.** liteLLM distinguishes
  retrying on another deployment of the same model from falling back to the
  next model in the chain; the two operate at different layers and Prumo has
  only one.
- **429 with `Retry-After` is the gateway's limit; without it, the provider's.**
  liteLLM also exposes `rate_limit_type` and `reset_at`, which is what decides
  "wait" versus "try someone else".
- **529 for overload shedding**, so a saturated instance sheds load instead of
  queueing until it degrades.
- **Order by stability, then price, then the rest as fallback.** OpenRouter
  deprioritises providers with an outage in the last 30 seconds, then selects
  among the stable ones weighted by the inverse square of price.
- **Provider fallback and model fallback are separate layers.** Provider routing
  keeps one model alive across providers; model fallback handles all providers
  for that model failing.
- **Score with explicit weights and an uptime penalty.** LLM Gateway uses
  uptime 50%, throughput 20%, price 20%, latency 10%, with an exponential
  penalty below a 95% uptime threshold.
- **Budget is checked before the provider is contacted; spend logging is
  asynchronous**, off the request path (redis cache first, database on miss).
- **Concurrency is a separate limit from rate.** Long agentic streams hold
  connections open in a way a per-minute budget cannot see.

## Decision

1. **The gateway is the execution path.** `Runner` selects through it; it stops
   being a library that only tests import. `model.Provider` becomes a
   deployment behind the gateway rather than the thing the runner calls.

2. **Routing is two layers.** *Provider routing* keeps one model alive across
   deployments of that model, retrying inside the group. *Model fallback* leaves
   the group to the next model in the policy. A 429 with `Retry-After` waits; a
   429 without it moves down the chain; a 529 sheds immediately; a 4xx does not
   retry at all.

3. **Circuit breakers recover.** `closed → open → half-open → closed`, driven by
   the existing `CooldownUntil` plus a success threshold in half-open.

4. **Quota is real state, not an inference.** A provider that exposes credit or
   rate headers is read; otherwise the gateway decrements from observed usage
   against a declared ceiling. `SetQuota` and `quotaExhausted` share one lock.

5. **`LocalOnly` is fail-closed.** An undeclared privacy value is not "local".

6. **Budget is enforced before the call.** A pre-flight estimate reserves against
   the envelope; an over-budget run degrades to a cheaper capable model or
   refuses. Cost is computed from tokens and a versioned pricing catalog when the
   provider does not report it, and is labelled estimated rather than observed.

7. **One pricing catalog.** `modelregistry` input/output rates, the gateway's
   `CostPer1k` and the table in `budget/hierarchy.go:29-45` collapse into a single
   versioned source with effective date, currency and source. Estimates and
   observations are distinguished, per the `FRAMEWORK.md` invariant.

8. **Cache and reasoning tokens count.** The tracker consumes cache read/write
   and reasoning tokens with their own prices. `agent.Usage` gains a
   reasoning field, which `extagent` already has (`extagent.go:301-307`).

9. **Role policy is resolved at runtime.** `model-policy.json` is read by the
   router, and the generated shape matches what the doctor validates
   (`cliops/ops.go:88-92` emits a map; `compile.go:597-609` expects a string).
   `DirectiveIR.ModelRoute` is honoured. Scorecards are an input, not a report.

10. **Capabilities gate selection.** `RouteTarget` carries vision, reasoning,
    tools, context and cache capability, and a task requiring vision cannot be
    routed to a text-only model. `Weight` becomes functional.

11. **A truncated stream is not a completion.** `scanner.Err()` is checked
    before emitting `completed` in every adapter, so a broken connection cannot
    be recorded as a successful turn.

12. **The catalog gains a model/provider/pricing section**, generated from code,
    so the routing CLI and the runtime cannot disagree about what exists.

## Consequences

- Model choice becomes a runtime decision instead of a flag, which is what
  makes "cost-aware" true rather than aspirational. A user who passes
  `--provider` explicitly still pins; the router governs everything else.
- Pinning a provider is a supported mode, matching OpenRouter's
  `allow_fallbacks: false` for compliance, BYOK or region constraints.
- Cost becomes trustworthy for OpenAI-compatible and Anthropic, which is a
  prerequisite for any real budget claim.
- The generated catalog gains a source of truth it currently lacks, and
  `prumo model route` stops being able to recommend a model the runtime cannot
  instantiate.
- Pre-flight estimation is an estimate. It is labelled as one, and the
  authoritative number remains observed usage after the call.
