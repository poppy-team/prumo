# M6 Living Plan / Interview Engine — Exit Gate

Status: **COMPLETE (M6 Exit Gate Passed)**

## Exit gate criteria

M6 exit gate (per `docs/development/phases.md`): a new project can go from initial
intent to implementation-ready via interview without a manual megaprompt;
Goal-specific planning closes only relevant gaps; decisions/open questions carry
authority and provenance; resume does not depend on transcript;
docs/readiness/governance feedback loop works; question/decision evals reach the
approved baseline; no agent suggestion is silently promoted.

### 1. Initial intent → implementation-ready without a manual megaprompt

- Deterministic end-to-end dogfood test `TestDogfoodZeroToReady`
  (`cmd/prumo/plan_z2r_test.go`) drives the full loop through `run()`: blocked
  readiness → questions → answered decisions → proposed delta → governance patch
  → applied delta → passing readiness → blueprint proposal.
- Committed sample project `examples/living-plan-sample/` ships at the ready
  state with a replayable seven-step recipe (`README.md`).
- Verified on the sample with the real CLI: `docs readiness --goal G-SAMPLE`
  transitions from `{"ready": false, "blocking_contracts": ["project.scope"]}`
  to `{"ready": true, "coverage": [[product.vision, implementation-ready],
  [project.scope, implementation-ready]]}`.

### 2. Goal-specific planning closes only the relevant gaps

- Planning and readiness are goal-scoped: `prumo-agent plan --goal <id>`,
  `docs readiness --goal <id>`; the documentation engine ignores unrelated
  contracts when computing coverage for a Goal (M5 regression suite).
- The sample loop only touches `product.vision` and `project.scope`; unrelated
  contracts stay untouched.

### 3. Decisions / open questions carry authority and provenance

- Decision proposals expose class, actor/source, authority, confidence
  (inference-only), status, and affected contracts/docs
  (`internal/planning/decision.go`, resolver `internal/planning/resolver.go`,
  E-G02/E-G03).
- Open questions registry tracks blocker status, linked contract, owner,
  created/resolved/superseded state, and resolution evidence
  (`internal/planning/question.go`, E-G01).
- A resolved question is not deleted; it remains traceable to its decision.

### 4. Resume does not depend on transcript

- Session checkpoint persisted under `.ai/plan/sessions/<id>.json` as
  `session.Checkpoint()` (`internal/app/sessionstore.go`, E-G07).
- `prumo-agent plan resume` recompiles context from canonical decisions + open
  questions + checkpoint (`internal/app/resume_test.go`,
  `internal/planning/session_test.go`); no transcript is reintroduced.

### 5. Docs/readiness/governance feedback loop works

- Loop proven by `TestDogfoodZeroToReady` and the sample README steps:
  readiness → questions → answered decisions → delta propose → repository
  governance authoring → delta apply (records decision evidence) → readiness
  recompute → goal/plan proposal.
- `plan delta [--apply]` never writes canonical docs; governance authoring +
  README documents that flow (E-G05/E-G09).

### 6. Question/decision evals reach the approved baseline

- The approved design (Notion §70.18) defines the baseline as deterministic
  conversation evals; live-model evals belong to the future Harness Eval Suite
  and are explicitly **not** a unit-test requirement.
- Every listed deterministic scenario has a regression test: blocker sorted
  before nice-to-have; explicit decision over inference; agent suggestion not
  auto-accepted; locked-decision conflict produces a finding (E-G04);
  resolved question updates the registry; unrelated docs gaps ignored for Goal
  readiness; resume preserves accepted decisions without transcript.
- Run in CI via `go test -race ./...`.

### 7. No agent suggestion silently promoted

- `agent-suggestion` classifications never promote to decisions and keep the
  question open (answer authority enforcement in `internal/planning` +
  `cmd/prumo/plan_z2r_test.go`: `TestPlanAnswerNeverPromotesAgentSuggestion`,
  `TestPlanAnswerUnresolvedKeepsQuestionOpen`).

## Evidence (framework dogfood on this repository)

- `examples/living-plan-sample/` — committed sample, ready state, replay
  README, and the applied delta record
  `docs/governance-delta-applied.json`.
- Full CLI surface exercised against the sample: `plan questions`, `plan
  answer`, `plan decisions`, `plan delta [--apply]`, `plan blueprint [--plan]`,
  `docs readiness --goal` (E-G08 JSON envelope + text modes).

## Test evidence

- `go test -race ./...` — pass across all packages.
- `go vet ./...`, `gofmt -l .` — clean.
- `pytest -q` — 127 pass (Python v0.3 oracle preserved).
- M5/M6 regression suites: coverage/re readiness isolation, delta lifecycle,
  decision preview contradiction findings, authority resolution, checkpoint
  round-trip, resume context compilation, command-line JSON envelopes.

## Governance

- M6 merged to `main` via squash PRs: E-G01 (#24), E-G02 (#25), E-G03 (#26),
  E-G04 (#27), E-G05 (#28), E-G06 (#29), E-G07 (#30), E-G08 (#31),
  E-G09 (#32).
- Canonical spec: `docs/runtime/living-plan.md`; samples/dogfood under
  `examples/living-plan-sample/`.

## Excluded from M6 exit gate

- Live-model conversation/decision evals remain a Harness Eval Suite item
  (Notion pages 61 and 70 §18); deterministic regression evals already satisfy
  the gate.
- Adoption Engine (M7), traceability (M8), and Experience (M9) are explicit
  non-goals for M6.