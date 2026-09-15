# Prumo Testing Strategy

## Principles

- Test code **and** protocol behavior and orchestration/documentation quality.
- Deterministic output must never depend on an LLM.
- Python v0.3 remains the compatibility oracle until Go reaches critical-contract parity.
- Quality evidence must expose infrastructure failures instead of masking them as passes.

## Test Levels (Pyramid)

### Unit

Pure domain rules, parsers, resolvers, risk classification, applicability, lifecycle state machines, budget calculations, context selection, schema contracts.

### Integration

Temporary filesystems, Git repositories, schema registry, SQLite derived index, connector generation, package/runtime installation. Tests use isolated `t.TempDir()` and do not touch the user's project.

### Conformance / Golden

Implementation-independent inputs and expected outputs. Compare Python v0.3 (oracle) and Go for:
- process exit code;
- stdout and relevant stderr;
- JSON envelope and error codes;
- filesystem effects;
- canonical artifacts.

Golden outputs are updated only when an intentional protocol change is approved.

### Connector Contract Tests

For each harness: setup, generated files, hooks, permissions, capability negotiation, cleanup, compatibility, and ownership behavior.

### End-to-End

Fixture projects execute `init → plan/adopt → compile → validate → docs → handoff` as capabilities become available.

### LLM Evals

Only for probabilistic behavior:
- interview quality;
- decision extraction;
- documentation completeness suggestions;
- retrieval;
- contradiction suggestions;
- routing/context heuristics.

## Quality Gates

Required in CI when the corresponding implementation exists:
```text
gofmt check
go test -race ./...
go vet ./...
staticcheck ./...       # recommended gate
conformance
schema validation
connector fixtures
govulncheck ./...        # when dependencies exist
```

Python baseline (retired under ADR 002):
The legacy Python runtime and pytest suite were retired once all conformance fixtures achieved parity. Go quality gates (`gofmt`, `go vet`, `go test -race ./...`) now serve as the primary enforcement mechanism.


## Conformance Fixtures

Include:
- valid and invalid schemas;
- locked Goal and amendment;
- valid and cyclic Plan DAG;
- context pack;
- evidence and gate waiver;
- event stream;
- FakeRuntime happy path, retry, fallback;
- project diagnostics;
- all public CLI commands (v0.3 oracle surface).

Documentation contracts additionally need incomplete fixtures distinguishing `missing`, `partial`, `ready`, and `not-applicable`.

Adoption fixtures need: good non-Prumo docs, duplicate docs, README/config contradiction, no docs, monorepo, mixed languages.

## Test Provider Contract

Every provider declares:
- `id/version`;
- host/platform requirements;
- installation strategy;
- capabilities;
- input/config schema;
- invocation and output parser;
- evidence artifacts;
- isolation and permission requirements;
- timeout/resource model;
- supported risk profiles;
- cleanup ownership.

Evidence expectations and normalized evidence records include test run, assertion/finding, artifact pointer, environment/toolchain fingerprint, seed/corpus, retries/flakiness, timing, failure category, and provenance.

## Failure Taxonomy

`product_failure`, `test_failure`, `environment_failure`, `flaky`, `infrastructure_failure`, `security_finding`, `inconclusive`.

Retries never silently convert failure to success. A `flaky` result remains distinct from `pass`; recurring flakiness creates debt/finding and can block according to profile.

## Independent Verification

High/critical risk may require a different agent/provider/model than the implementer and a clean evidence execution.

## Performance Budgets

Define and measure budgets for:
- startup;
- `doctor` on small/medium/large repositories;
- `adopt --audit-only`;
- trace lookup;
- connector compilation;
- context packing and compaction;
- resume success and diagnosis time.

Exact thresholds: **OPEN QUESTION** until repository size classes and baseline measurements exist.

## Runtime Eval Corpus

Scenarios include: small task overhead, 100k+ LOC repository, context pressure/compaction, long migration + resume, provider 429/timeout/5xx, ambiguous side effect, malicious README/prompt injection, conflicting policies, stale docs, flaky tests, hard budget exhaustion, connector crash, model alias drift, and concurrent agents in one scope.

Metrics include task completion, policy violations, context precision/recall, tokens/cost per task, unnecessary tool calls, activation precision/recall, wrong-tool rate, resume/recovery success, evidence completeness, false pass, time-to-diagnosis, and security containment.

Heuristics require baseline comparison, then shadow eval → canary subset → promotion or rollback.

## Dogfooding

- M3+: Prumo builds/tests itself.
- M5+: Prumo applies documentation contracts to its own repository.
- M6+: Prumo plans features through Living Plan.
- M7+: Prumo adopts its own repository.
- M8+: Prumo traces its own decisions.
- M9+: Prumo processes its structured experience.
- M10+: Prumo development uses OpenCode native harness.