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

### Viewer Extension Contract

- validate manifests against `viewer-extension-manifest.schema.json` and the SDK validator;
- load valid, duplicate, malformed and unsafe-path manifests;
- prove declarative commands cannot invoke unsupported host actions;
- bound JSON-RPC messages and reject malformed frames;
- run process extensions with crash, timeout, oversized-output and permission-expansion fixtures;
- test install, enable, disable, update, rollback and cleanup before public distribution;
- round-trip `.prumoext` archives, reject traversal/symlink/oversized payloads, verify Ed25519 signatures and revocation, and enforce lockfile digests;
- exercise bounded LSP/DAP framing, handshake, timeout and capability failures, including completion, active-file code-action edits and formatting with UTF-16 positions and undo/redo;
- opcionalmente, com `PRUMO_RUN_RUST_ANALYZER_SMOKE=1`, executar o smoke real em `extension-sdk/tests/rust_analyzer_smoke.rs`; sem `rust-src`, o teste aceita o aviso de sysroot do servidor, mas com sysroot completo exige `textDocument/publishDiagnostics`;
- para validar o viewer completo, usar `PRUMO_VIEWER_EXTENSIONS=prumo-viewer/extensions` e um grant temporário para `com.prumo.rust-analyzer`; `rustc` e `rust-analyzer` devem pertencer ao mesmo toolchain;
- o fork vendorizado `prumo-viewer/vendor/freya-code-editor` não é membro do workspace, então seus testes internos rodam sobre uma cópia fora do repositório com `cargo test --manifest-path <copia>/Cargo.toml --offline --lib`; ele cobre a métrica incremental da linha mais longa.

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