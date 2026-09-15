# Implementation Waves — Post-Harness, Pre-TUI

> Authority: repository-canonical. Supersedes ad-hoc gap lists.
> Source: deep research analysis (2026-09-14), Documentation Control Plane Deep Audit (2026-09-14),
> gap-register, roadmap-status, Living Book (pages 13, 14, 18, 21, 23, 25–38).

These waves sit between the harness headless completion (HA0–HA11, split gate
READY conditional) and the H10 TUI spike. Each wave has explicit entry/exit
criteria. Waves are sequential within priority tiers; P0 waves block P1 waves.

---

## W0 — Authority & Version Cleanup

**Priority**: P0 (blocks everything)  
**Goal**: Eliminate version drift, authority routing confusion, and prevent unmanaged parallel truths.

| Task | Detail |
|------|--------|
| W0.1 | Update `AGENTS.md` routing table: v0.4 → current version references |
| W0.2 | Update `docs/PRUMO.md` to consistent version |
| W0.3 | Update `README.md` to match docs version |
| W0.4 | Update `docs/product/scope-v0.4.md` — add v0.5 addendum and scope clarification |
| W0.5 | Audit all `docs/` files for stale v0.4 references; update or annotate |
| W0.6 | Formalize authority order in a single canonical location |
| W0.7 | Create a machine-readable authority map or derive one from existing contracts/bindings |
| W0.8 | Add a drift rule for active docs that reference obsolete project versions without historical annotation |
| W0.9 | Mark every major top-level doc as canonical/projection/historical where inference is ambiguous |
| W0.10 | Validate `AGENTS.md` pointers and routing targets |
| W0.11 | Establish policy: agent/provider adapters are projections and may not introduce independent project facts |
| W0.12 | Add a permanent authority/drift CI gate so W0 does not become one-time cleanup |

**Entry**: PR #64 merged (harness branch landed).  
**Exit**: `grep -r 'v0\.4' docs/` returns only historical/migration references; authority routing is unambiguous; no active canonical or agent-routing document contains an unexplained version/authority contradiction.

---

## W1 — Schema / Runtime Conformance Gate

**Priority**: P0  
**Goal**: Automatically verify enums, required fields, and round-trip between JSON Schema and Go types. Prevent contract/runtime drift.

| Task | Detail |
|------|--------|
| W1.1 | Create `conformance/schema-runtime/` test suite |
| W1.2 | Verify all schema enums match Go const blocks (e.g., `pressure` in schema vs Go) |
| W1.3 | Round-trip tests: JSON → Go struct → JSON for all harness schemas |
| W1.4 | Add documentation schemas ↔ Go type roundtrip (contracts, profiles, translations, agent surfaces) |
| W1.5 | Add enum migration and unknown-field policy tests |
| W1.6 | Add forward/backward migration tests when schema v2 appears |
| W1.7 | Add to CI as required check |
| W1.8 | Fix any discovered drift (ADR 007 `compact` enum, etc.) |

**Entry**: W0 complete.  
**Exit**: `go test ./conformance/schema-runtime/...` green in CI; zero enum/field mismatches; schema freshness guaranteed.

---

## W2 — Canonical Format Cleanup

**Priority**: P0  
**Goal**: Eliminate YAML from new canonical designs; enforce Markdown + JSON policy.

| Task | Detail |
|------|--------|
| W2.1 | Audit Notion-promoted designs for `session.yaml` or YAML structures |
| W2.2 | Convert any candidate YAML specs to JSON/Markdown |
| W2.3 | Distinguish source vs generated adapter directories |
| W2.4 | Prohibit generated vendor-specific configs from becoming canonical sources |
| W2.5 | Add lint check: no new `.yaml`/`.yml` in canonical paths |
| W2.6 | Preserve YAML read compatibility only where legacy migration behavior requires it |

**Entry**: W0 complete.  
**Exit**: No YAML in canonical specs; lint enforced in CI.

---

## W3 — KnowledgeUnit & Stable Identity Foundation

**Priority**: P0  
**Goal**: Establish the atom of retrievable knowledge with stable IDs independent of paths/titles.

| Task | Detail |
|------|--------|
| W3.1 | Define `schemas/knowledge-unit.schema.json` with revision, authority, visibility, provenance, relationships |
| W3.2 | Define `schemas/knowledge-manifest.schema.json` |
| W3.3 | Define `schemas/knowledge-claim.schema.json` |
| W3.4 | Implement KnowledgeUnit Go types in `internal/knowledge/` |
| W3.5 | Define explicit documentation relationships: `documents`, `explains`, `references`, `implements`, `verifies`, `supersedes`, `contradicts`, `projects_to`, `translated_as`, `illustrated_by`, `affects` |
| W3.6 | Implement KnowledgeManifest builder (derived index, not new source of truth) |
| W3.7 | Add stable ID generation and validation across rename/path move |
| W3.8 | Migrate existing Knowledge Runtime baseline (GAP-018/019) to use stable IDs |

**Entry**: W1 complete (schema conformance infra exists).  
**Exit**: `prumo knowledge manifest` produces valid manifest; IDs survive file renames and moves.

---

## W4 — ContextManifest & Progressive Disclosure

**Priority**: P0  
**Goal**: Every compiled context is explainable and replayable; documentation units are first-class candidates.

| Task | Detail |
|------|--------|
| W4.1 | Define `schemas/context-manifest.schema.json` |
| W4.2 | Define `schemas/context-policy.schema.json` |
| W4.3 | Implement ContextManifest generation in Context Compiler |
| W4.4 | Implement progressive disclosure levels L0–L4 (Pointer → Full) |
| W4.5 | Treat documentation units as first-class retrieval candidates with selection reasoning |
| W4.6 | Deduplicate equivalent canonical and projection content; prefer canonical claim pointers |
| W4.7 | `prumo context compile --goal G... --budget N` emits manifest |
| W4.8 | `prumo context explain CTX-...` command |
| W4.9 | Context sufficiency evaluation (required claims covered, dependencies included) |
| W4.10 | Measure instruction-token overhead separately |

**Entry**: W3 complete (KnowledgeUnits exist to reference).  
**Exit**: Every context compilation produces auditable manifest; L0–L4 levels functional; context efficiency measurable.

---

## W5 — UI Contract Decomposition

**Priority**: P0  
**Goal**: Replace monolithic `ui.documentation` with 13 specialized contracts and composable profiles.

| Task | Detail |
|------|--------|
| W5.1 | Define contract schemas: `ui.product-ux`, `ui.information-architecture`, `ui.screen-inventory`, `ui.layout`, `ui.interaction`, `ui.state-model`, `ui.design-tokens`, `ui.component-contracts`, `ui.accessibility`, `ui.localization`, `ui.theming`, `ui.visual-validation`, `ui.platform-conventions` |
| W5.2 | Define profile compositions: `desktop-gui`, `web-application`, `tui`, `cli` |
| W5.3 | Create `schemas/ui-component-contract.schema.json` (anatomy, states, interactions, tokens) |
| W5.4 | Create `schemas/ui-state-matrix.schema.json` (default, hover, focus, disabled, loading, empty, error, offline, recovery) |
| W5.5 | Create `schemas/design-token-set.schema.json` (aligned with W3C Design Tokens format) |
| W5.6 | Formalize TUI-specific contracts: size classes, color capability, Unicode fallback, keyboard-only completeness, AT-aware focus |
| W5.7 | Deprecate monolithic `ui.documentation` in favor of composed contracts |

**Entry**: W3 complete (contracts need stable IDs).  
**Exit**: TUI profile (`tui`) exists and is composable; component/state/token schemas validate.

---

## W6 — Locale & Translation Foundation

**Priority**: P0  
**Goal**: Formalize English as canonical locale; establish translation lifecycle and terminology control.

| Task | Detail |
|------|--------|
| W6.1 | Add `source_locale: en` and `locales: ["en", "pt-BR"]` to project config schema |
| W6.2 | Define `schemas/translation-record.schema.json` |
| W6.3 | Define translation lifecycle states: `missing → machine-draft → needs-review → current → needs-update → stale` |
| W6.4 | Add terminology/glossary records and placeholder integrity verification |
| W6.5 | Implement digest-based staleness detection (source changes → translation `needs-update`) |
| W6.6 | Add pseudo-localization verifier and RTL-ready data model |
| W6.7 | Invariant: agent context remains sourced from canonical English unless task explicitly targets localized content |

**Entry**: W0 complete.  
**Exit**: Translation records track source digest; staleness auto-detected; terminology consistency enforced.

---

## W7 — Gauntlet Schema & Policy (mode: off default)

**Priority**: P0  
**Goal**: Define Gauntlet quality iteration contracts with bounded rounds and stopping conditions.

| Task | Detail |
|------|--------|
| W7.1 | Define `schemas/gauntlet-policy.schema.json` (mode: off/auto/force, max_rounds, critic_isolation, activation rules) |
| W7.2 | Define `schemas/gauntlet-run.schema.json` (run_id, rounds, scorecards, stop_reason, evidence) |
| W7.3 | Add documentation-specific Gauntlet profile (factual_grounding, semantic_coverage, freshness, cross_doc_consistency) |
| W7.4 | Invariant: deterministic checks run first; model critics cannot override hard deterministic failures |
| W7.5 | Implement GauntletPolicy Go types |
| W7.6 | Add Framework invariant: Gauntlet default = `off`; activation requires quality oracle |

**Entry**: W1 complete.  
**Exit**: Schemas validate; policy loads; mode defaults to `off`; no uncontrolled recursion.

---

## W8 — Shared Product Contract Promotion

**Priority**: P1  
**Goal**: Promote Living Book page 23 (Shared Product Contract) into repo contracts with stable IDs.

| Task | Detail |
|------|--------|
| W8.1 | Extract entities, state ownership, capabilities from page 23 |
| W8.2 | Assign stable capability IDs usable by docs, UI contracts, release notes, and agent context |
| W8.3 | Create `docs/architecture/shared-product-contract.md` |
| W8.4 | Create corresponding JSON schemas where applicable |
| W8.5 | Update SOURCE_MAP.json with provenance |

**Entry**: W0 complete.  
**Exit**: Contract is repository-authoritative; Notion page becomes historical reference.

---

## W9 — Semantic Design Tokens

**Priority**: P1  
**Goal**: Define the three-tier token system (primitive → semantic → component) as a versioned API.

| Task | Detail |
|------|--------|
| W9.1 | Define primitive tokens (colors, spacing, typography) |
| W9.2 | Define semantic tokens (surface, text, border, focus, accent, success, warning, error) |
| W9.3 | Define TUI-specific component tokens |
| W9.4 | Implement token change impact rules (components, themes, visual evidence, migration) |
| W9.5 | Create Visual Constitution for Prumo Code TUI |

**Entry**: W5 complete (design-token-set schema exists).  
**Exit**: Token set validates; TUI semantic tokens defined; Visual Constitution written.

---

## W10 — Documentation Compiler Contracts

**Priority**: P1  
**Goal**: Promote Documentation Planner/Compiler architecture from Notion to repo specifications.

| Task | Detail |
|------|--------|
| W10.1 | Define DocumentationSpec, DocumentationPlan, DocumentationUnit, DocumentationBrief, DocumentationSurface, DocumentationManifest schemas |
| W10.2 | Define generation tiers: GENERATED (deterministic) / ASSISTED (model + review) / CURATED (protected human) |
| W10.3 | Define expanded Documentation Delta chain: KnowledgeDelta → semantic impact → DocumentationPlan → DocumentationDelta → projections → TranslationDelta → MediaDelta → verification manifest |
| W10.4 | Add compiler invariants: deterministic output, no-op writes on unchanged fingerprint, CURATED protection, cycle prohibition |
| W10.5 | Create `docs/architecture/documentation-compiler.md` |

**Entry**: W3 + W6 complete.  
**Exit**: Schemas validate; compiler pipeline documented; delta chain specified.

---

## W11 — Accessibility & TUI Contracts

**Priority**: P1  
**Goal**: Surface-specific accessibility contracts; TUI interaction contract.

| Task | Detail |
|------|--------|
| W11.1 | Decompose `accessibility` into sub-contracts: keyboard, focus, contrast, screen-reader, motion, zoom-reflow, target-size, drag-alternatives, accessible-auth, cognitive-clarity |
| W11.2 | Define evidence classes: automated, manual, visual, interaction, platform-specific |
| W11.3 | Create `tui.accessibility` contract with TUI-specific invariants |
| W11.4 | Create `tui.interaction` contract |
| W11.5 | Reference WCAG 2.2 + WCAG2ICT for non-web guidance |
| W11.6 | Invariant: automated checks alone never constitute full accessibility verification |

**Entry**: W5 complete.  
**Exit**: TUI accessibility contract specifiable; evidence types defined.

---

## W12 — Reconnect / Replay Client Verification

**Priority**: P1 (TUI precondition from H10 spike scope)  
**Goal**: Verify reconnect/replay as a client-visible contract.

| Task | Detail |
|------|--------|
| W12.1 | Add client-facing conformance test for JSONL event replay |
| W12.2 | Add client-facing conformance test for daemon reconnect after kill |
| W12.3 | Verify no UI state corruption on reconnect (test harness) |
| W12.4 | Document reconnect/replay client contract, troubleshooting, and TUI state recovery |

**Entry**: PR #64 merged.  
**Exit**: Conformance tests pass; reconnect/replay is a tested public contract.

---

## W13 — Behavioral Eval Corpus (baseline)

**Priority**: P1  
**Goal**: Establish evaluation infrastructure before Gauntlet `auto`.

| Task | Detail |
|------|--------|
| W13.1 | Create `evals/context/` — context quality evaluations |
| W13.2 | Create `evals/documentation/` — documentation sufficiency evaluations (reproduce false-green keyword match, stale paths, contradiction detection) |
| W13.3 | Create `evals/ui-specification/` — UI spec completeness evaluations |
| W13.4 | Define reconstruction eval: spec → independent agent → conformance comparison |

**Entry**: W4 + W5 complete.  
**Exit**: At least one eval per category runs; baselines recorded.

---

## W14 — Framework Invariants Update

**Priority**: P1  
**Goal**: Add deep-research constitutional directives to FRAMEWORK.md.

| Task | Detail |
|------|--------|
| W14.1 | Add "Knowledge is canonical; prose is a projection" |
| W14.2 | Add "Missing knowledge produces a gap, never invented documentation" |
| W14.3 | Add "Every durable semantic entity has stable identity independent of path" |
| W14.4 | Add "Every generated context is explainable and replayable" |
| W14.5 | Add "Documentation architecture is planned before prose is generated" |
| W14.6 | Add "Design systems are versioned APIs, not styling suggestions" |
| W14.7 | Add "Readiness is semantic evidence coverage, not keyword presence" |
| W14.8 | Add "Agent instruction files are scoped projections, not parallel truth" |
| W14.9 | Add "A Goal is not complete while required documentation projections are stale" |
| W14.10 | Add "Deterministic quality failures cannot be overridden by model self-evaluation" |

**Entry**: W3 + W4 complete.  
**Exit**: FRAMEWORK.md updated; no contradiction with existing invariants.

---

## W15 — Semantic Documentation Control Plane & Readiness v2

**Priority**: P0  
**Dependencies**: W1, W3, W10  
**Goal**: Replace lexical/fragmented documentation readiness with a typed semantic control plane.

| Task | Detail |
|---|---|
| W15.1 | Define architecture boundary for Documentation Control Plane coordinator |
| W15.2 | Reconcile `internal/documentation` with harness Knowledge/HumanDocs/DocCompile interfaces |
| W15.3 | Define requirement → claim → evidence coverage model |
| W15.4 | Implement semantic readiness v2 |
| W15.5 | Remove/retire ANY-word required-knowledge satisfaction from authoritative readiness |
| W15.6 | Bind evidence requirements to explicit requirement/claim IDs and evidence types |
| W15.7 | Add authority/freshness/revision checks |
| W15.8 | Integrate Knowledge contradiction engine into documentation readiness |
| W15.9 | Add canonical/projection lineage validation |
| W15.10 | Migrate existing contracts/bindings without breaking CLI compatibility |
| W15.11 | Add semantic dogfood report for Prumo itself |
| W15.12 | Add conformance/eval fixtures for false-green cases |

**Entry**: W3 stable IDs and W10 core schemas exist.  
**Exit**: `prumo docs readiness` cannot pass merely because matching words or enough evidence items exist; every obligation is semantically bound.

---

## W16 — Agent Surface Compiler & Context-Rot Guard

**Priority**: P1  
**Dependencies**: W0, W3, W4, W15  
**Goal**: Compile minimal, scoped, verifiable agent instruction surfaces from canonical knowledge.

| Task | Detail |
|---|---|
| W16.1 | Define `AgentInstructionIR` and schema |
| W16.2 | Define deterministic precedence/scoping model |
| W16.3 | Implement root `AGENTS.md` adapter |
| W16.4 | Implement nested/path-scoped AGENTS adapter |
| W16.5 | Implement GitHub Copilot adapter |
| W16.6 | Implement Cursor rules adapter |
| W16.7 | Implement Claude adapter/import strategy |
| W16.8 | Implement generic `SKILL.md` projection where applicable |
| W16.9 | Add source fingerprint/provenance comments or manifest sidecar |
| W16.10 | Validate code paths, symbols, commands, tests, and docs pointers referenced by agent surfaces |
| W16.11 | Enforce instruction token budgets |
| W16.12 | Detect duplicate/conflicting instructions across scopes |
| W16.13 | Invariant: core does not depend on vendor format; adapters depend on AgentInstructionIR |
| W16.14 | Add context-rot CI gate |
| W16.15 | Add agent-surface compilation to documentation impact |

**Entry**: W4 and W15 complete.  
**Exit**: Invariant change regenerates agent surfaces; stale code references cause a failing CI check.

---

## W17 — Semantic Impact Graph & Documentation Delta Orchestration

**Priority**: P1  
**Dependencies**: W3, W5, W6, W9, W15  
**Goal**: Make documentation impact a semantic preflight/postflight process connected to Goals.

| Task | Detail |
|---|---|
| W17.1 | Extend typed impact triggers with symbol/API/schema/CLI/event/permission/UI/token/locale/media/Goal/Wave/ADR/release/evidence matchers |
| W17.2 | Resolve symbol impact through repo map/LSP |
| W17.3 | Resolve schema/API impact through structural diff |
| W17.4 | Resolve UI impact through component/state/token graph |
| W17.5 | Add `prumo docs plan --goal` command |
| W17.6 | Populate GoalPlan `DocumentationGap`/DocumentationPlan in real path |
| W17.7 | Run preflight at Goal lock |
| W17.8 | Run postflight after implementation |
| W17.9 | Diff predicted vs actual impact for Project Intelligence |
| W17.10 | Generate TranslationDelta and MediaDelta |
| W17.11 | Require explicit N/A reasons |
| W17.12 | Add impact explanation command |

**Entry**: W15 complete.  
**Exit**: Every behavior-changing Goal has a replayable DocumentationPlan and post-change DocumentationDelta.

---

## W18 — Documentation Publishing & AI Retrieval Plane (HD5–HD7)

**Priority**: P1  
**Dependencies**: W6, W10, W15  
**Goal**: Project the same documentation graph into human site and AI-native retrieval surfaces.

| Task | Detail |
|---|---|
| W18.1 | Define renderer interface independent of site generator |
| W18.2 | Implement Starlight as first reference site adapter |
| W18.3 | Generate navigation/breadcrumbs/related-doc edges from page graph |
| W18.4 | Add search index generation |
| W18.5 | Add version-aware routes |
| W18.6 | Add locale-aware routes/fallback |
| W18.7 | Add public/internal visibility filtering |
| W18.8 | Generate `/llms.txt` |
| W18.9 | Generate `/llms-full.txt` optionally with size/budget policy |
| W18.10 | Generate/furnish raw Markdown page views |
| W18.11 | Expose read-only documentation MCP resources/search |
| W18.12 | Keep mutation MCP separate and permission-gated |
| W18.13 | Add API reference adapters from authoritative machine contracts |
| W18.14 | Add Project Intelligence documentation dashboard projection |
| W18.15 | Add replaceability/conformance test for renderer boundary |

**Entry**: W10 and W15 complete.  
**Exit**: Repository Markdown, site pages, AI indexes, and MCP retrieval resolve to the same stable IDs.

---

## W19 — Continuous Documentation Verification & Docs Gauntlet Runtime

**Priority**: P1  
**Dependencies**: W7, W13, W15, W16, W17, W18  
**Goal**: Make documentation correctness a continuous completion gate rather than a best-effort review.

| Task | Detail |
|---|---|
| W19.1 | Define `DocumentationQualityPolicy` |
| W19.2 | Implement layered verifier pipeline: deterministic first, model-assisted second |
| W19.3 | Integrate docs profile into Gauntlet runtime |
| W19.4 | Add isolated critic roles |
| W19.5 | Add bounded repair loop with strict stop conditions |
| W19.6 | Persist findings/evidence per round |
| W19.7 | Invariant: score cannot increase without resolved findings/evidence |
| W19.8 | Add CI command `prumo docs verify` |
| W19.9 | Add `--strict` completion integration |
| W19.10 | Add regression corpus from real Prumo drift findings |

**Entry**: W7 and W15 complete.  
**Exit**: Stale or uncovered required documentation cannot pass; deterministic blockers cannot be overridden.

---

## W20 — Documentation Lifecycle: Localization, Media, Versioning, Release & Deprecation

**Priority**: P1/P2  
**Dependencies**: W6, W9, W17, W18  
**Goal**: Manage post-generation lifecycle across releases and presentation variants.

| Task | Detail |
|---|---|
| W20.1 | Complete TranslationRecord QA lifecycle |
| W20.2 | Add terminology/glossary management |
| W20.3 | Add pseudo-localization verifier |
| W20.4 | Define MediaRecord + stale propagation |
| W20.5 | Link visual evidence to UI state/theme/locale/platform |
| W20.6 | Add docs version policy |
| W20.7 | Add deprecation lifecycle |
| W20.8 | Generate release documentation plan |
| W20.9 | Generate compatibility/migration projection inputs |
| W20.10 | Add release readiness docs gate |
| W20.11 | Preserve historical docs without contaminating current context |

**Entry**: W17 and W18 complete.  
**Exit**: Release can explain status of all docs/translations/media; deprecations cleanly isolated.

---

## W21 — Documentation Intelligence, Feedback, Adoption & Scale

**Priority**: P2  
**Dependencies**: W15–W20  
**Goal**: Turn documentation behavior into measurable project intelligence and support brownfield adoption.

| Task | Detail |
|---|---|
| W21.1 | Add documentation metrics to Project Intelligence |
| W21.2 | Track retrieval/query success without storing sensitive user content |
| W21.3 | Track top missing/failed documentation intents |
| W21.4 | Track context/instruction token efficiency |
| W21.5 | Track freshness SLA and drift recurrence |
| W21.6 | Track translation/media/example health |
| W21.7 | Implement brownfield docs discovery |
| W21.8 | Propose contracts/bindings from existing docs |
| W21.9 | Detect duplicates/conflicts during adoption |
| W21.10 | Require human/reviewer promotion before inferred authority |
| W21.11 | Benchmark large-repository build/invalidation |
| W21.12 | Add documentation debt integration |

**Entry**: W15–W20 operational.  
**Exit**: Measurable documentation efficiency; brownfield adoption supported without truth duplication.

---

## Recommended Dependency Graph

```text
W0 ─┬─ W1 ── W3 ── W4
    │        │
    │        ├─ W5 ── W9 ── W11
    │        │
    └─ W6 ──┘

W1 ── W7
W0 ── W8
W3 + W6 ── W10
W4 + W5 ── W13
W3 + W4 ── W14
Harness ── W12

W1 + W3 + W10
        │
        ▼
       W15 (Readiness v2)
        │
        ├──────── W16 (Agent Surfaces)
        │
W5+W6+W9 ─────── W17 (Impact Graph)
        │           │
W6+W10+W15 ─────── W18 (Publishing / AI Retrieval)
        │           │
W7+W13+W15+W16+W17+W18
        │
        ▼
       W19 (Verification & Gauntlet)

W6+W9+W17+W18 ── W20 (Lifecycle & Release)
W15–W20 ───────── W21 (Intelligence & Adoption)
```

---

## Wave → TUI Gate

The H10 TUI spike (defined in `docs/product/tui-spike-h10.md`) may begin when:

- [x] PR #64 merged (harness branch landed)
- [ ] W0 complete (authority/version clean)
- [ ] W1 complete (schema/runtime conformance)
- [ ] W5 complete (UI contracts exist, TUI profile composable)
- [ ] W6 complete (locale foundation — TUI has i18n implications)
- [ ] W8 complete (Shared Product Contract promoted)
- [ ] W9 complete (semantic design tokens for TUI)
- [ ] W11 complete (TUI accessibility/interaction contracts)
- [ ] W12 complete (reconnect/replay verified)
- [ ] W15 complete (semantic readiness v2 eliminates false confidence)

P1 waves W10, W13, W14, W16–W19 may proceed in parallel with early TUI work but must
complete before H10 exit criteria are evaluated.

---

## Traceability: Waves & Gaps

| Wave | Research Report / Audit Section | Gap Register |
|------|--------------------------------|---------------|
| W0 | Drift table (L39–44), Audit §9 | DOC-GAP-001 |
| W1 | Schema/runtime conformance (L43), Audit §9 | GAP-034, GAP-036, DOC-GAP-001 |
| W2 | Format cleanup (L42), Audit §9 | — |
| W3 | KnowledgeUnit (L161–220), Audit §9 | GAP-018, GAP-019, GAP-027, DOC-GAP-011, DOC-GAP-012 |
| W4 | ContextManifest (L222–278), Audit §6 | GAP-006, GAP-009, DOC-GAP-007 |
| W5 | UI contract decomposition (L719–799), Audit §24 | DOC-GAP-015 |
| W6 | Locale decision (L403–458), Audit §25 | DOC-GAP-017 |
| W7 | Gauntlet schema (L459–711), Audit §23 | — |
| W8 | Shared Product Contract, Audit §17 | H10 precondition #3 |
| W9 | Design tokens (L876–926), Audit §24 | H10 precondition, DOC-GAP-018 |
| W10 | Documentation Compiler (L356–401), Audit §12 | GAP-007, GAP-028, DOC-GAP-006, DOC-GAP-011 |
| W11 | Accessibility (L1015–1122), Audit §24 | — |
| W12 | Reconnect/replay, Audit §17 | H10 precondition #2 |
| W13 | Behavioral evals (L1433–1487), Audit §32 | DOC-GAP-006 |
| W14 | Framework invariants (L1526–1567), Audit §49 | — |
| W15 | Audit §4, §9, §18 | DOC-GAP-002, DOC-GAP-003, DOC-GAP-004, DOC-GAP-009, DOC-GAP-010 |
| W16 | Audit §6, §9, §18, §26, §27 | DOC-GAP-007 |
| W17 | Audit §9, §18 | DOC-GAP-005, DOC-GAP-008 |
| W18 | Audit §10, §18, §28, §29, §30 | DOC-GAP-013, DOC-GAP-014, DOC-GAP-019 |
| W19 | Audit §9, §10, §18, §22, §23 | DOC-GAP-004, DOC-GAP-006, DOC-GAP-020 |
| W20 | Audit §10, §18, §24, §25 | DOC-GAP-016, DOC-GAP-017, DOC-GAP-021 |
| W21 | Audit §11, §18, §42, §43 | DOC-GAP-022, DOC-GAP-023, DOC-GAP-024, DOC-GAP-025, DOC-GAP-026, DOC-GAP-027, DOC-GAP-028, DOC-GAP-029, DOC-GAP-030 |
