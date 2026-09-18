# Prumo Implementation Phases

> **Rule**: Only fully detail current phase + next phase. Future phases remain architectural until dependencies mature.
>
> **Version scope**: M0–M4 record the historical v0.3→v0.4 migration; Pre-TUI waves
> (W0–W21) and the H10 TUI spike are current. Classification:
> `docs/AUTHORITY_MAP.json` (`drift_exempt`).

---

## M0 — Specification Freeze

**Objective**: Establish immutable baseline for migration; prevent scope creep.

**Dependencies**: None (starting state)

**Scope**:
- Record Architecture Decision: Go Core (ADR)
- Inventory all v0.3 observable contracts:
  - CLI commands, subcommands, flags, exit codes
  - JSON output schemas (envelope, data shapes)
  - File system effects (created/modified/deleted paths)
  - Schema validation behavior
  - Golden fixture outputs
- Catalog all Python modules, scripts, tests, resources
- Define Go module path and initial package boundaries
- Create `conformance/` matrix: Python fixture → expected Go output
- Freeze v0.3: no new features in Python except critical bug fixes

**Non-Goals**:
- Writing Go implementation code
- Designing v0.4 features beyond migration needs

**Deliverables**:
- `docs/migration/protocol-inventory.md`
- `docs/migration/conformance-strategy.md`
- `conformance/V03_BASELINE.json`
- `conformance/golden/` populated from Python v0.3
- Go module initialized (`go.mod`)

**Test/Eval Requirements**:
- All 127 Python tests passing (baseline)
- Conformance fixtures defined for: `init`, `resolve`, `validate`, `goal`, `context plan`, `report`, `migrate`, `compile`, `snapshot`, `doctor`, `explain`, `framework-check`

**Acceptance Criteria**:
- [ ] Architecture Freeze ADR committed
- [ ] 100% of v0.3 CLI surface cataloged with golden outputs
- [ ] Conformance matrix covers all critical contracts
- [ ] Go module builds empty `cmd/prumo` + `internal/*` skeleton

**Exit Gate**: M0 complete when `docs/migration/protocol-inventory.md` and `conformance/V03_BASELINE.json` are reviewed and approved.

---

## M1 — Go Foundation

**Objective**: Build minimal Go skeleton that compiles, tests, and runs; establish quality gates.

**Dependencies**: M0 complete

**Scope**:
- Go module + `go.sum`
- `cmd/prumo/main.go` with machine envelope (`protocol_version`, `ok`, `data`, `diagnostics`, `warnings`)
- `internal/protocol`: envelope types, version constants
- `internal/app`: `Service` with `Version()`, `ProjectRoot()` stubs
- `internal/project`: `FindRoot()` via `prumo.json` discovery
- `internal/validation`: JSON Schema validation (Draft 2020-12) — initially delegate to Python or use `gojsonschema`
- `internal/resources`: access to `schemas/` + `src/prumo/resources/` via `io/fs.FS` abstraction (prepare for `go:embed`)
- Error model: wrapped errors with `%w`, sentinel errors, no panic for user errors
- Test infrastructure: `go test -race ./...`, table-driven tests
- CI: `gofmt`, `go vet`, `go test -race`, `staticcheck` (recommended)
- Logging/output discipline: stdout for `--json`, stderr for diagnostics, no ANSI in JSON

**Non-Goals**:
- Protocol domain logic (Goals, Plans, etc.)
- Real schema validation (can delegate to Python subprocess initially)
- Full CLI command surface

**Deliverables**:
- `cmd/prumo/main.go` with `version` command (`--json` support)
- `internal/protocol/protocol.go` + tests
- `internal/app/service.go` + tests
- `internal/project/project.go` + tests
- `internal/validation/validation.go` + tests
- `internal/resources/resources.go` + tests
- CI workflow with Go job
- `.gitignore` for Go artifacts

**Test/Eval Requirements**:
- `go test -race ./...` passes
- `go vet ./...` passes
- `gofmt -l .` empty
- `staticcheck ./...` passes (if configured)
- Python tests still 100% passing (no regression)

**Acceptance Criteria**:
- [ ] `go run ./cmd/prumo-agent version` → `0.4.0-dev`
- [ ] `go run ./cmd/prumo-agent --json version` → valid envelope
- [ ] `go test -race ./...` all green
- [ ] CI runs Python + Go quality gates
- [ ] No circular dependencies in `internal/`

**Exit Gate**: M1 complete when all Go quality gates pass in CI and `prumo-agent version --json` produces valid envelope.

---

## M2 — Protocol Parity

**Objective**: Port all v0.3 protocol domain to Go with 100% conformance on critical contracts.

**Dependencies**: M1 complete

**Scope**:
### Protocol Domain (`internal/protocol/`)
- `goals/`: Goal v2 struct, lock/verify, amendment, state transitions, hashing
- `plans/`: Plan, Task DAG, cycle detection
- `tasks/`: Task struct, execution state
- `events/`: Event protocol, structured logging
- `evidence/`: Evidence struct, validation
- `gates/`: Gate, waiver, release readiness

### Project Service (`internal/project/`)
- `prumo.json` load/parse with schema validation
- Profile loading, capabilities, model policy
- Project state snapshot

### Resolution Service (`internal/resolver/`)
- Workforce resolution (agents, skills, recipes)
- Risk assessment
- Model policy evaluation
- Context selection

### Validator (`internal/validator/`)
- Full JSON Schema validation (Draft 2020-12)
- Schema registry with cross-ref resolution
- Project validation against schemas

### CLI Commands (Parity)
- `prumo-agent init`, `resolve`, `validate`
- `prumo-agent goal new|state|amend|list`
- `prumo-agent context plan`
- `prumo-agent report add|summary`
- `prumo-agent migrate`
- `prumo-agent doctor`, `explain`, `framework-check`

### Compiler (`internal/compiler/`)
- Target adapter interface
- Codex, Claude Code, Generic generators

**Non-Goals**:
- Documentation System v2 (contracts, profiles, UI pack)
- Living Plan / Interview Engine
- Adoption Engine
- Control Plane (Run, Budget, Context Compiler, etc.)
- Experience Layer
- OpenCode native harness

**Deliverables**:
- Full protocol domain packages with tests
- All 12 CLI commands ported with `--json` support
- Compiler targets: Codex, Claude Code, Generic
- Conformance suite executing Go vs Python on identical fixtures

**Test/Eval Requirements**:
- **Conformance 100% on critical contracts**: identical exit codes, JSON envelopes, filesystem effects
- Unit tests for all domain logic
- Integration tests with temp repos
- Golden fixtures: `conformance/golden/<cmd>.golden.json` matched exactly
- `go test -race ./...`, `go vet`, `gofmt`, `staticcheck` all pass

**Acceptance Criteria**:
- [ ] `prumo-agent init --non-interactive --profile X` creates valid project
- [ ] `prumo-agent resolve` matches Python workforce/skills/recipes exactly
- [ ] `prumo-agent validate` catches same schema violations
- [ ] `prumo-agent goal` lifecycle identical (lock, amendment, transitions)
- [ ] `prumo-agent context plan` produces same budget/strategy
- [ ] `prumo-agent doctor` finds same issues
- [ ] `prumo-agent compile --target codex|claude-code|generic` generates byte-identical or semantically equivalent outputs
- [ ] `prumo-agent explain` outputs match for workforce, agent, skill, recipe, context, model, execution
- [ ] Conformance diff: zero failures on critical contracts

**Exit Gate**: M2 complete when `go test -run Conformance` (or equivalent) shows 100% parity on all cataloged critical contracts.

---

## M3 — Compiler Parity

**Objective**: Full compiler target parity including ancillary targets; conformance gate.

**Dependencies**: M2 complete

**Scope**:
- All v0.3 compiler targets ported and tested
- Ancillary targets (ChatGPT, Kimi, Traycer, etc.) if still supported
- Template system with `go:embed` for adapter templates
- Generated artifact ownership markers
- Conformance validation on compiler outputs

**Non-Goals**: New compiler targets, OpenCode native (M10)

**Deliverables**: All compiler targets with golden fixture tests

**Test/Eval Requirements**: 100% conformance on all compiler outputs

**Acceptance Criteria**: [ ] All v0.3 targets pass conformance; no regressions

**Exit Gate**: M3 complete when compiler conformance = 100% and M2 gate still holds.

---

## M4 — Distribution v1

**Objective**: Usable install/uninstall experience; single binary releases.

**Dependencies**: M3 complete

**Scope**:
- GitHub Releases with signed binaries (Linux amd64/arm64, macOS amd64/arm64, Windows amd64)
- Install script (`curl | sh`) with checksum verification
- Homebrew tap
- Windows packaging baseline (Scoop/WinGet)
- `prumo-agent setup` wizard (detect harnesses, select integrations, validate PATH, run doctor)
- `prumo-agent install connector <id>`, `prumo-agent uninstall [--connectors|--purge-cache|--purge-global-config]`
- Installation manifest (versions, paths, connectors, managed config fragments)
- Cleanup manifests per connector
- Portable mode (`prumo-agent --home ./path`)
- Idempotent install/uninstall

**Non-Goals**: Auto-update from non-GitHub sources, complex multi-user server

**Deliverables**: Release artifacts, install script, Homebrew formula, `prumo-agent setup/install/uninstall`

**Test/Eval Requirements**: Install/uninstall on clean VMs for all platforms; idempotency verified

**Acceptance Criteria**:
- [ ] `curl -fsSL install.sh | sh` installs working `prumo`
- [ ] `prumo-agent uninstall` removes binary + connectors + cache (not project data)
- [ ] `prumo-agent setup` detects harnesses and configures project-local adapters
- [ ] Homebrew install works
- [ ] Windows package installs

**Exit Gate**: M4 complete when release pipeline produces signed binaries and install/uninstall verified on all target platforms.

---

## Pre-TUI Waves (W0–W21)

**Dependencies**: Harness headless split gate READY (HA0–HA11).

Canonical specification: `docs/development/waves.md`.

These waves address structural gaps identified by deep research analysis
and the Documentation Control Plane Deep Audit (2026-09-14) that must be
resolved before or alongside the H10 TUI spike:

- **W0–W2** (P0): Authority/version cleanup, schema/runtime conformance, format cleanup.
- **W3–W4** (P0): KnowledgeUnit foundation, ContextManifest, progressive disclosure L0–L4.
- **W5** (P0): UI contract decomposition (monolithic → 13 specialized contracts + profiles).
- **W6** (P0): Locale foundation (English canonical, TranslationRecord lifecycle).
- **W7** (P0): Gauntlet schema/policy (mode: off default).
- **W8–W9** (P1): Shared Product Contract promotion, semantic design tokens.
- **W10–W11** (P1): Documentation compiler contracts, accessibility/TUI contracts.
- **W12** (P1): Reconnect/replay client-facing verification.
- **W13–W14** (P1): Behavioral eval corpus, Framework invariants update.
- **W15** (P0): Semantic Documentation Control Plane & Readiness v2 (replaces lexical/word matching).
- **W16** (P1): Agent Surface Compiler & Context-Rot Guard (compiled AGENTS/Cursor/Copilot/Claude).
- **W17** (P1): Semantic Impact Graph & Documentation Delta Orchestration (Goal-driven).
- **W18** (P1): Documentation Publishing & AI Retrieval Plane (HD5–HD7, Starlight, llms.txt, MCP).
- **W19** (P1): Continuous Documentation Verification & Docs Gauntlet Runtime.
- **W20–W21** (P1/P2): Documentation Lifecycle (i18n/media/release) & Intelligence/Adoption.

**Exit Gate**: All P0 waves (W0–W7, W15) complete; P1 waves W8, W9, W11, W12 complete.
TUI spike (H10) may begin. Remaining P1 waves complete before H10 exit evaluation.

---

## H10 — TUI Spike

**Dependencies**: Pre-TUI waves exit gate passed.

Canonical specification: `docs/product/tui-spike-h10.md`.

**Scope**: Bubble Tea v2 terminal client proving the harness can drive a real
Zed-like terminal client. Command palette, agent run panel, event streaming,
permissions, file tree, embedded + remote daemon modes.

**Exit Gate**: Acceptance criteria 1–4 from `tui-spike-h10.md` pass once end-to-end.

**Progress**: Fase A complete (2026-09-15) — the UI gate is live before the first
line of TUI code.

- `prumo.json` declares `cap:tui`, so the 16 `ui.*`/`tui.*` contracts and the 10
  `accessibility.*` sub-contracts apply to this repository instead of being
  skipped as not applicable (7 → 33 applicable contracts).
- The `tui` documentation profile is now reachable: `ResolveProfiles` did not map
  the `tui` capability, and did not union the capabilities a composed profile
  declares, so the profile selected 26 contracts and the applicability filter
  removed all 26 again.
- A binding may declare `applicability: not-applicable` with a required
  `not_applicable_reason`; a waiver without a reason fails closed to `missing`.
  19 contracts carry reviewable waivers naming why their surface does not exist
  yet; the 7 the vertical slice owes are bound to real artifacts.
- Specification written for the vertical slice: `docs/ui-ux/state-matrix.json`
  (all 22 states), `docs/ui-ux/interaction.md`, plus the terminal screen-reader
  strategy and the visual evidence/capture/regression policy in
  `docs/architecture/visual-constitution.md`.

**Progress**: Fase B/C complete (2026-09-15) — the vertical slice runs.

- `tui/` lives outside `internal/` and imports only the public SDK and the
  standard library. `TestBoundaryNoInternalImports` (criterion 1) fails the
  build if that stops being true, including for test files.
- The package is split so the parts worth testing need no terminal: `timeline.go`
  (bounded, severity-classified event rows), `palette.go` (fuzzy ranking),
  `styles.go` (token → style, no literal colours) and `session.go` (run over the
  protocol) are pure or SDK-only; `app.go` only routes keys and messages.
- `theme` is a generated projection of the canonical token set with a digest
  freshness check, and `tui/theme.Resolve` is the only place a token becomes a
  value.
- The flow `palette → goal → run → stream → evidence` is verified twice: by
  `go test` with a scripted endpoint (`TestVerticalSliceFlow`) and against a live
  supervised daemon (`TestLiveDaemonFlow`, criterion 2's shape, FakeProvider, no
  API key). Reconnect/replay reproduces the daemon's rows instead of merging two
  half-views (criterion 3's mechanism).
- `prumo-agent tui` supervises `prumo-agent agent serve` as a subprocess, which is what keeps
  the boundary real and makes the remote mode the same client with another
  address rather than a second code path.

**Remaining**: none for the spike itself. Criterion 2's approval step, criterion 4
(live remote TLS+token) and criterion 5 (redraw budget per ADR 009 methodology)
are all closed — the protocol gained the approval operations, and the remote flow
and the 200-row baseline are recorded. The spike's **client** is superseded:
**ADR 013** archives `tui/` to `archive/tui-poc/` and adopts the archived OpenCode
Go view layer as the foundation of a separate `prumo-agent tui` module. What the spike
produced outlives it — the approval operations, `awaiting_approval`, the redraw
baseline, and a boundary that ADR 013 makes structural rather than merely tested.

**Remaining (Fase A gate)**: none — `prumo-agent docs verify --strict` is green, so the
gate is met rather than deferred.

---

## M5 — Documentation System v2

**Dependencies**: M2, M4

**Scope**:
- Documentation Contracts (knowledge requirements → required docs)
- Documentation Profiles (capability → doc pack)
- Coverage/Readiness engine (missing/partial/ready/not-applicable)
- UI Documentation Pack (semantic wireframes, tokens, components, states, accessibility)
- Docs Delta (incremental updates on decisions)
- Contradiction Framework (detection, reporting, resolution proposals)

**Non-Goals**: Living Plan interview (M6), Adoption (M7)

**Exit Gate**: Documentation contracts enforceable; profiles selectable; delta tracked; contradictions detected.

---

## M6 — Living Plan

**Dependencies**: M5

Canonical specification: `docs/runtime/living-plan.md`.

**Scope**:
- Interview protocol (conversational, incremental, least ceremony)
- Decision extraction with authority model
- Open questions tracking (blocker vs non-blocker, traceable resolution)
- Confidence scoring (inference-only, evidence-backed)
- Decision preview with contradiction handling
- Readiness gates (block implementation until critical decisions confirmed)
- Incremental documentation updates from decisions (Documentation Delta)

**Non-Goals**: brownfield discovery (Fase F Adoption), autonomous architecture invention, raw transcript as canonical memory, model-specific conversation format, code generation as part of `prumo-agent plan`.

**Goal Decomposition**:
- E-G01: Question / OpenQuestion schemas + priority resolver.
- E-G02: answer classification + Decision proposal model.
- E-G03: authority/confidence resolver.
- E-G04: decision preview + contradiction handling.
- E-G05: Documentation Delta/readiness feedback loop.
- E-G06: Goal/Plan output integration.
- E-G07: resume/checkpoint + context compilation.
- E-G08: CLI/harness-neutral interaction protocol.
- E-G09: zero-to-ready Prumo sample/dogfood.

**Exit Gate (LIVING PLAN READY)**: New project can go from initial intent to implementation-ready via interview without a manual megaprompt; Goal-specific planning closes only relevant gaps; decisions/open questions carry authority and provenance; resume does not depend on transcript; docs/readiness/governance feedback loop works; question/decision evals reach approved baseline; no agent suggestion silently promoted.

---

## M7 — Adoption Engine

**Dependencies**: M5, M6

Canonical specification: `docs/runtime/adoption-engine.md`.

**Scope**:
- Repository scanner (file types, frameworks, configs, docs)
- Semantic documentation mapping (non-Prumo layouts → Prumo concepts)
- Capability detection
- Confidence ledger (scored mapping)
- Adoption report
- Migration proposals (non-destructive, reversible)

**Non-Goals**: rewriting the entire layout to "look Prumo", trusting README as authority, mandatory embeddings, auto-deleting legacy docs, inferring user intent without confirmation, installing every detected connector/tool.

**Goal Decomposition**:
- F-G01: scanner facts + revision-aware sources.
- F-G02: repository/project/workspace classification.
- F-G03: capability/profile candidates.
- F-G04: semantic doc mapping + candidate bindings.
- F-G05: Confidence Ledger.
- F-G06: Adoption Report.
- F-G07: Living Plan uncertainty resolution.
- F-G08: migration proposal + Review/Migration integration.
- F-G09: brownfield corpus/evals/dogfood.

**Exit Gate (ADOPTION READY)**: Arbitrary existing repo can be audited without mutation; observed facts and inferences are kept separate; confidence/evidence accompany inferences; M5 evaluates candidate bindings correctly; ambiguity enters the Living Plan/Open Questions; migration proposals are dry-run/review governed; scanner is incremental/branch-aware; malicious content/secrets tests pass.

---

## M8 — History + Traceability

**Dependencies**: M2, M6

**Scope**:
- Implementation Journal (synthesis, not chain-of-thought)
- Experiment/Rejection/Debt registers
- Typed traceability graph (req↔dec↔code↔test↔doc↔evidence)
- `prumo-agent trace <ref>` CLI

**Exit Gate**: Any code change traceable to decision/goal; journal queryable.

---

## M9 — Experience Layer

**Dependencies**: M5, M6, M7, M8

**Scope**:
- Session events (structured, not transcript)
- Summaries (per session, per Goal)
- Handoff Protocol (state transfer between sessions/agents)
- Experience Provider Contract
- Experience Proposals (validated before promotion)
- Retention policies

**Exit Gate**: Agent can hand off to another agent/session with full structured context.

---

## M10 — OpenCode Native Harness

**Dependencies**: M3, M4, M9

**Scope**:
- OpenCode native compiler (TypeScript plugin + agents + skills + commands + hooks)
- Prumo primary agent for OpenCode
- Subagents, skills, commands mapped to OpenCode primitives
- Tool guards (pre-tool validation)
- Session hooks (start/end/tool events)
- Connector Contract tests for OpenCode

**Exit Gate**: `prumo-agent connector install opencode` produces fully functional native integration.

---

## M11 — Connector SDK

**Dependencies**: M10

**Scope**:
- Capability negotiation protocol
- Cleanup manifests
- Test kit (fixtures, contract tests)
- Gemini CLI connector
- Claude Code / Codex harness elevation

**Exit Gate**: New connector can be built against SDK and passes contract tests.

---

## M12 — Team/Advanced Runtime

**Dependencies**: M11 (only after real usage validates need)

**Scope**: Shared runtime, leases, concurrency coordination, optional server — **deferred until proven necessary**.

---

## Phase Detail Rule

Only **M0** and **M1** are fully detailed above. M2–M4 have sufficient detail
for planning. Pre-TUI waves (W0–W21) are detailed in `docs/development/waves.md`.
M5–M12 remain architectural — they will be detailed when their dependencies
are mature and implementation begins.