# M5 Documentation System v2 — Exit Gate

Status: **COMPLETE (M5 Exit Gate Passed)**

## Exit gate criteria

M5 exit gate (per `docs/development/phases.md`): documentation contracts enforceable; profiles selectable; delta tracked; contradictions detected.

### 1. Documentation contracts enforceable

- Contract registry: `docs/contracts/builtin.json` (8 contracts), schema `schemas/documentation-contract.schema.json`.
- Bindings: `docs/contracts/bindings.json` with `owned` sources, `answered_questions`, and `evidence`.
- Machine schema validation on load; every stor/delta validated against `schemas/documentation-delta.schema.json`.
- Verification: `prumo-agent docs audit` reports all applicable contracts `implementation-ready`.

### 2. Profiles selectable

- Profile registry: `docs/profiles/builtin.json` (7 profiles), schema `schemas/documentation-profile.schema.json`.
- Applicability resolution via project capabilities (`prumo.json` project type/features plus detected `cmd`, `schemas`, `go.mod`).
- Verification: `prumo-agent docs profiles` lists `api-service`, `cli`, `compiler`, `core-software`, `desktop-gui`, `library`, `web-application`.

### 3. Delta tracked

- Persistent Delta lifecycle: `internal/documentation/delta.go` (C-G06).
- Deterministic IDs (SHA-256 of goal, sorted contracts, sorted documents); stable across re-analysis.
- Lifecycle: `proposed → reviewed → accepted → applied` (applied requires evidence) or `proposed → rejected`; versions advance and all transitions validate.
- Local persistence under `.ai/docs/deltas/DD-<12hex>.json`, schema-validated on read, corruption-rejected.
- CLI: `prumo-agent docs delta propose|list|show|transition`.
- Conformance fixture: `conformance/documentation/valid_delta.json`.

### 4. Contradictions detected

- Deterministic contradiction and causal-staleness findings (`DetectContradictions`, `DetectStaleness`).
- Contradictions are findings only; they are never silently resolved by recency or model output.
- CLI: `prumo-agent docs contradictions`.

## Evidence (framework dogfood on this repository)

- `prumo-agent docs audit` — all 7 applicable contracts `implementation-ready`.
- `prumo-agent docs readiness --goal M5` — `ready: true`, no blocking contracts, no blocking questions.
- `prumo-agent docs contradictions` — deterministic findings (0 for current tree).
- `prumo-agent docs profiles` — all 7 built-in profiles resolvable.

## Test evidence

- `go test ./... -race` — pass across all packages.
- `go vet ./...` — clean.
- `pytest -q` — 127 pass (Python v0.3 oracle preserved).
- Delta lifecycle, storage round-trip, corruption rejection, schema validation, and CLI integration tests all pass.
- Delta determinism and conformance fixture regression tests pass.

## Governance

- Documentation Delta application remains governed by Repository Change Governance (`.prumo/repository/policy.json`): branch, commit, Pull Request, validation, review, and merge.
- `feat/c-g06-documentation-delta-tracking` merged to `main` via PR #21 (squash).
- Machine schemas under `schemas/documentation-*.schema.json` are the contract surface; changes require schema updates and conformance fixtures.

## Excluded from M5 exit gate

- Semantic prose contradiction assistance remains a provider-future finding; deterministic contract-driven findings already satisfy the exit gate.
- Living Plan interview (M6), Adoption (M7), full traceability (M8), and Experience (M9) are explicitly non-goals for M5.