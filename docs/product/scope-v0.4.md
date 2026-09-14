# Prumo v0.4 Scope

## In Scope

### Core Runtime Migration
- Go Core & CLI replacing Python runtime
- Single binary distribution (Linux amd64/arm64, macOS amd64/arm64, Windows amd64/arm64)
- Clean Code pragmático como regra permanente
- Zero CGO por padrão
- `gofmt`, `go test -race`, `go vet`, `staticcheck` como quality gates obrigatórios

### Protocol Parity (v0.3 → Go)
- CLI surface: `init`, `resolve`, `validate`, `goal`, `context plan`, `report`, `migrate`, `compile`, `snapshot`, `doctor`, `explain`, `framework-check`
- JSON Schema validation (Draft 2020-12)
- Goal v2 lifecycle (locks, amendments, state transitions)
- Plans/Tasks/Events/Evidence/Gates
- Workforce resolver (agents, skills, recipes, model policy)
- Compiler targets: Codex, Claude Code, Generic (+ ancillary)
- Machine JSON envelope: `protocol_version`, `ok`, `data`, `diagnostics`, `warnings`

### Documentation System v2
- **Documentation Contracts**: obrigatoriedade definida por knowledge requirements, não existência de arquivos
- **Documentation Profiles**: capability packs proporcionais ao projeto
- **UI Documentation Pack**: wireframes semânticos, layout, spacing, tokens, cores, tipografia, components, states, accessibility, localization, visual tests
- **Docs Delta & Contradiction Framework**: rastreamento de mudanças e detecção de conflitos

### Living Plan & Interview Engine
- Conversational incremental planning
- Decision extraction with authority model
- Open questions tracking
- Confidence scoring
- Readiness gates before implementation

### Adoption Engine
- Repository scanner & semantic doc mapping
- Capability detection
- Confidence ledger
- Non-destructive migration proposals

### Control Plane (Foundation)
- **Run Engine**: state machine, checkpoints, retry taxonomy, resumption, cancellation, livelock detection
- **Budget Manager**: hierarchical scopes (user→workspace→project→Goal→run→agent→step→tool), hard/soft limits, cost metadata, rate limiting
- **Context Compiler**: authority/freshness/relevance pipeline, token budgets, compaction, cache awareness
- **Model Registry/Router**: metadata-driven, eval-based routing, drift detection, fallback, provider health
- **Tool Gateway**: descriptor, lazy discovery, side-effect journal, MCP governance, output control
- **Execution Environments**: sandbox contract, isolation requirements
- **Automation Engine**: event-driven rules, DLQ, idempotency, concurrency keys
- **Observability**: structured timeline, explainability commands, incident bundles, local-first storage
- **Package/Runtime Manager**: `prumo.lock`, provider isolation, supply chain verification

### Integration Layer
- **Connector Contract**: capability declaration, enforcement levels, protocol version negotiation, cleanup manifests
- **OpenCode Native Harness**: native compiler, Prumo primary agent, subagents/skills/commands, TypeScript plugin, tool guards, session hooks
- **Generic Fallback**: preserved for harnesses without native primitives

### Security & Trust
- Trust layers (Core binary → canonical config → generated adapters → 3rd party → user content → model output)
- Safe mode (`prumo --safe`)
- Data classification & egress governance
- SecretProvider contract
- Generated artifact ownership markers
- SBOM, govulncheck, license verification in CI

### Testing & Quality
- Test pyramid: Unit → Integration → Conformance → Connector Contract → E2E → LLM Evals (probabilistic only)
- Test Provider Contract (Playwright, ZAP, CodeQL, fuzzers, sanitizers, UIA/AT-SPI/AX, KUnit, syzkaller, etc.)
- Quality Orchestrator: deterministic plan from change impact + risk + capabilities
- Evidence normalization schema
- Failure taxonomy (product/test/env/flaky/infra/security/inconclusive)
- Flakiness policy (distinct `flaky` result, debt tracking)
- Independent verification for high/critical risk
- Conformance 100% on critical contracts before Python deprecation

### Skill System v3
- Package versionado: manifest + `SKILL.md` + knowledge/workflows/templates/checks/scripts/schemas/tests/evals/examples
- Activation modes: `auto`, `contextual`, `manual`, `required`, `disabled`
- Permissions declaration (never self-elevate)
- Progressive disclosure (manifest → SKILL.md → workflows on demand)
- Lifecycle: `draft → experimental → recommended → verified → deprecated → retired`
- Config layering: built-in → user → workspace → project → recipe → session
- Ambient Skill Resolver (metadata-first, minimal composition)
- Skill Curator (overlap, conflict, staleness, weak evals)
- Experience → Skill Proposal pipeline
- KPIs: coverage, activation precision, evals, redundancy, freshness, incidents avoided, context cost, stability

## Compatibility Constraints

v0.4 must preserve v0.3 canonical schemas, project readability, Goal lock semantics, deterministic conformance, and provider-neutral project state while Python remains the compatibility oracle.

## Out of Scope (v0.4)

- Distributed team runtime / shared server
- Mandatory embeddings or vector DB
- Web dashboard UI
- Public Go SDK
- Fork of any harness (OpenCode, Codex, etc.)
- ORM or external graph DB
- Game Development Suite (built on Provider Contract + Skill v3 + Evidence Schema)
- Advanced Verification Profiles (compiler, kernel, GUI, network adversarial)
- Knowledge Publishing / Engineering Blog
- Team/Advanced Runtime (shared runtime, leases, concurrency coordination)

## Non-goals

The v0.4 non-goals are the deferred features listed above: distributed team runtime, mandatory embeddings, dashboard UI, public SDK, harness forks, ORM, external graph database, and broad specialized domain packs before their dependencies mature.

## Migration Boundary

Python v0.3 remains as **executable oracle** during migration:
- Zero new v0.4 features in Python
- All Go development validated against Python conformance
- Python deprecated only after 100% critical contract parity

## v0.5 Addendum — Pre-TUI Foundation Waves

The deep research analysis (2026-09-14) identified structural gaps between
the harness headless completion and the TUI spike. These are tracked as
implementation waves W0–W14 in `docs/development/waves.md`.

Key additions to scope:

- **KnowledgeUnit & stable identity** — atom of retrievable knowledge with IDs independent of paths.
- **ContextManifest** — every compiled context is explainable and replayable.
- **UI contract decomposition** — monolithic `ui.documentation` replaced by 13 specialized contracts.
- **Locale foundation** — English canonical, TranslationRecord lifecycle.
- **Gauntlet schema** — quality iteration policy (mode: off default).
- **Schema/runtime conformance gate** — automated drift detection.
- **Semantic design tokens** — three-tier system (primitive → semantic → component).
- **Accessibility contracts** — surface-specific (TUI, desktop, web, CLI).
- **Behavioral eval corpus** — evaluation infrastructure before automated loops.

These items are in scope for the current development cycle and gate the H10
TUI spike start.