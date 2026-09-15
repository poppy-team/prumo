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
| W0.1 | Update `AGENTS.md` routing table to the current release line (done: v0.6) |
| W0.2 | Update `docs/PRUMO.md` to consistent version |
| W0.3 | Update `README.md` to match docs version |
| W0.4 | Add a current-line addendum and scope clarification to the scope doc (done: `docs/product/scope-v0.4.md`) |
| W0.5 | Audit all `docs/` files for stale release references; update or annotate (done: `prumo docs authority`) |
| W0.6 | Formalize authority order in a single canonical location |
| W0.7 | Create a machine-readable authority map or derive one from existing contracts/bindings |
| W0.8 | Add a drift rule for active docs that reference obsolete project versions without historical annotation |
| W0.9 | Mark every major top-level doc as canonical/projection/historical where inference is ambiguous |
| W0.10 | Validate `AGENTS.md` pointers and routing targets |
| W0.11 | Establish policy: agent/provider adapters are projections and may not introduce independent project facts |
| W0.12 | Add a permanent authority/drift CI gate so W0 does not become one-time cleanup |

**Entry**: PR #64 merged (harness branch landed).  
**Exit**: `grep -r 'v0\.4' docs/` returns only historical/migration references; authority routing is unambiguous; no active canonical or agent-routing document contains an unexplained version/authority contradiction.

**Status**: ✅ complete 2026-09-14 — `docs/governance/authority.md` (authority order + projection policy), `docs/AUTHORITY_MAP.json` (127 docs: 97 canonical / 2 projection / 28 historical), `prumo docs authority` + `TestAuthorityGateClean` as the permanent `docs-authority` CI gate, routing links repaired. Drift fixes across ~30 canonical docs; explicit `drift_exempt` reasons recorded for the migration-blueprint and phases documents.

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

**Status**: ✅ complete 2026-09-14 — `conformance/schema-runtime/`: schema identity/draft/enum hygiene, enum↔Go vocabulary equality, contract/profile referential integrity, required-schema-set gate, JSON→Go→JSON round-trip. Fixed 3 schemas missing `$id`. Runs in the existing CI `go test ./...` job.

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

**Status**: ✅ complete 2026-09-14 — audit found only allowlisted legacy fixtures (`testdata/brownfield`, `lang-yaml` skill examples); `TestCanonicalPathsHaveNoYAML` enforces the policy over `docs/`, `schemas/`, `.prumo/` and the canonical catalog.

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

**Status**: ✅ complete 2026-09-15 — `knowledge-unit`/`knowledge-claim`/`knowledge-manifest` schemas and `internal/knowledge` (stable IDs, 12 kinds, 13 typed relationships, rename-safe alias table, fail-closed visibility) landed, together with W3.6 (derived manifest builder + `prumo knowledge manifest`) and W3.8 (the harness Knowledge baseline now seeds and reloads by stable identity, so a rename no longer creates a new unit).

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

**Status**: ✅ complete 2026-09-15 — `context-policy` schema (L0–L4, dedup policy, explainability, instruction-token accounting) plus `level`/`excluded`/`instruction_tokens` on `context-manifest` landed. W4.4 disclosure semantics, W4.5/W4.6 canonical-preference dedup, W4.7 explanation and W4.8 CLI surfacing all landed: `prumo context compile` and `prumo context explain <id>` report what was selected, at which level, and why anything was excluded.

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
| W5.8 | Expand `ui-state-matrix` applicability set: partial, reconnecting, permission-requested, permission-denied, read-only, destructive-confirmation, narrow viewport, alternate theme, high contrast, alternate locale (applicability explicit; not every component needs every state) |
| W5.9 | Add `ui.personalization` contract: user-customizable surface, defaults, persistence location, import/export, fallback, invalid-value behavior, platform limitations |
| W5.10 | Require component/state → implementation symbol traceability edges |
| W5.11 | TUI additions: mouse-optionality policy, explicit focus/selection semantics, terminal/screen-reader accessibility strategy |
| W5.12 | Add user-flow inventory and command/interaction map coverage to `ui.product-ux` / `ui.interaction` |

**Entry**: W3 complete (contracts need stable IDs).  
**Exit**: TUI profile (`tui`) exists and is composable; component/state/token schemas validate.

**Status**: ✅ complete 2026-09-14, with the state matrix delivered in H10 — 14 specialized `ui.*` contracts (incl. `ui.personalization`) plus `tui.interaction`/`tui.accessibility`; `ui.documentation` deprecated; `tui` profile composable; `ui-component-contract` and `ui-state-matrix` schemas with the full 22-state applicability set and component→implementation traceability.

> **Correction (2026-09-15):** this wave was reported as also delivering the state
> matrix artifact. It did not: only the schema existed, and nothing noticed because
> the `ui.*` contracts had no bindings and the `tui` profile was never composed, so
> they were never evaluated. `docs/ui-ux/state-matrix.json` landed in H10 and is now
> enforced by `TestStateMatrixIsCompleteAndExplicit`. Recorded in
> `docs/harness/gap-register.md` (DOC-GAP-015).

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
| W6.8 | Define locale-key identity independent of file path (stable segment IDs for source/translation alignment) |
| W6.9 | Define fallback policy: never render an empty locale surface; explicit fallback chain |
| W6.10 | Define date/number/plural formatting policy per locale |
| W6.11 | Add translation evidence/reviewer fields (source digest, terminology revision, reviewer, quality findings, fallback status) |
| W6.12 | Add a translation coverage report per release |

**Entry**: W0 complete.  
**Exit**: Translation records track source digest; staleness auto-detected; terminology consistency enforced.

**Status**: ✅ complete 2026-09-15 — `source_locale`/`locales` in `prumo.schema.json`; `translation-record` schema (lifecycle, locale-key identity, placeholder integrity, reviewer evidence, fallback, RTL); `docs/development/localization.md`. W6 had left two items open and both are now closed in W20: digest-based staleness (`doclifecycle.TranslationStatuses`) and the pseudo-localization verifier (`PseudoLocalize` / `VerifyPseudoLoc`).

---

## W7 — Gauntlet Schema & Policy (mode: off default)

**Priority**: P0  
**Goal**: Define Gauntlet quality iteration contracts with bounded rounds and stopping conditions.

| Task | Detail |
|------|--------|
| W7.1 | Define `schemas/gauntlet-policy.schema.json` (mode: off/auto/force, max_rounds, critic_isolation, activation rules) |
| W7.2 | Define `schemas/gauntlet-run.schema.json` (run_id, rounds, scorecards, stop_reason, evidence) |
| W7.3 | Add documentation-specific Gauntlet profile dimensions: factual_grounding, semantic_coverage, cross_document_consistency, freshness, reference_integrity, example_verification, information_architecture, task_findability, agent_retrieval_quality, accessibility, localization, token_efficiency, duplication |
| W7.4 | Invariant: deterministic checks run first; model critics cannot override hard deterministic failures |
| W7.5 | Implement GauntletPolicy Go types |
| W7.6 | Add Framework invariant: Gauntlet default = `off`; activation requires quality oracle |
| W7.7 | Define round structure: deterministic verify → collect blocking findings → bounded repair → optional isolated critic → re-verify affected checks → compare scorecard + finding closure |
| W7.8 | Enforce: each round emits typed findings/evidence (not prose inflation); score cannot rise without resolved findings |
| W7.9 | Define stop conditions: all mandatory gates pass, no critical/high findings, max rounds, budget exhausted, no measurable improvement |

**Entry**: W1 complete.  
**Exit**: Schemas validate; policy loads; mode defaults to `off`; no uncontrolled recursion.

**Status**: ✅ complete 2026-09-14 — `gauntlet-policy`/`gauntlet-run` schemas and `internal/gauntlet` (13 dimensions, bounded rounds 1..10, mandatory critic isolation, score-inflation guard, deterministic stopping conditions, default `off`). The runtime loop lands in W19.

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

**Status**: ✅ complete 2026-09-14 — `docs/architecture/shared-product-contract.md` + `shared-product-contract.json` (15 stable capability IDs, 12 entities with explicit state owners).

---

## W9 — Semantic Design Tokens

**Priority**: P1  
**Goal**: Define the three-tier token system (primitive → semantic → component) as a versioned API.

| Task | Detail |
|------|--------|
| W9.1 | Define primitive tokens (colors, spacing, typography) |
| W9.2 | Define semantic tokens (surface, text, border, focus, accent, success, warning, error) |
| W9.3 | Define TUI-specific component tokens |
| W9.4 | Implement token change impact rules (components, themes, visual evidence, accessibility, user-customization docs, translations where text/layout changes, migration when token IDs are renamed/removed) |
| W9.5 | Create Visual Constitution for Prumo Code TUI |

**Entry**: W5 complete (design-token-set schema exists).  
**Exit**: Token set validates; TUI semantic tokens defined; Visual Constitution written.

**Status**: ✅ complete 2026-09-14 — `design-token-set` schema; `docs/ui-ux/design-tokens.json` (primitive/semantic/component, 4 themes incl. no-color / high-contrast / reduced-motion, measured contrast evidence, TUI capability fallbacks); `docs/architecture/visual-constitution.md`.

---

## W10 — Documentation Compiler Contracts

**Priority**: P1  
**Goal**: Promote Documentation Planner/Compiler architecture from Notion to repo specifications.

| Task | Detail |
|------|--------|
| W10.1 | Define DocumentationSpec, DocumentationPlan, DocumentationUnit, DocumentationBrief, DocumentationSurface, DocumentationProjection, DocumentationQualityPolicy, DocumentationEvidenceRequirement, MediaRecord, AgentInstructionSurface, DocumentationManifest schemas |
| W10.2 | Define generation tiers: GENERATED (deterministic) / ASSISTED (model + review) / CURATED (protected human) |
| W10.3 | Define expanded Documentation Delta chain: KnowledgeDelta → semantic impact → DocumentationPlan → DocumentationDelta → projections → TranslationDelta → MediaDelta → verification manifest |
| W10.4 | Add compiler invariants: deterministic output, no-op writes on unchanged fingerprint, CURATED protection, cycle prohibition, projection provenance always recoverable |
| W10.6 | Invariant: a canonical source may not depend on a projection that depends on it (projection cycles are errors) |
| W10.5 | Create `docs/architecture/documentation-compiler.md` |

**Entry**: W3 + W6 complete.  
**Exit**: Schemas validate; compiler pipeline documented; delta chain specified.

**Status**: ✅ complete 2026-09-14 — 11 new schemas (spec, plan, unit, brief, surface, projection, quality-policy, evidence-requirement, manifest, media-record, agent-instruction-surface) plus `docs/architecture/documentation-compiler.md` (generation tiers, expanded delta chain, compiler invariants, unit lifecycle).

---

## W11 — Accessibility & TUI Contracts

**Priority**: P1  
**Goal**: Surface-specific accessibility contracts; TUI interaction contract.

| Task | Detail |
|------|--------|
| W11.1 | Decompose `accessibility` into sub-contracts: keyboard, focus, contrast, screen-reader, motion, zoom-reflow, target-size, drag-alternatives, accessible-auth, cognitive-clarity |
| W11.2 | Define evidence classes: automated, manual, visual, interaction, platform-specific, exception-with-rationale |
| W11.3 | Create `tui.accessibility` contract with TUI-specific invariants |
| W11.4 | Create `tui.interaction` contract |
| W11.5 | Reference WCAG 2.2 + WCAG2ICT for non-web guidance |
| W11.6 | Invariant: automated checks alone never constitute full accessibility verification |

**Entry**: W5 complete.  
**Exit**: TUI accessibility contract specifiable; evidence types defined.

**Status**: ✅ complete 2026-09-14 — 10 `accessibility.*` sub-contracts (keyboard, focus, contrast, screen-reader, motion, zoom-reflow, target-size, drag-alternatives, accessible-auth, cognitive-clarity) with per-contract `evidence_requirements`; 6 evidence classes incl. `exception-with-rationale`; composed into the `tui`, `web-application` and `desktop-gui` profiles; `docs/architecture/accessibility.md` (WCAG 2.2 + WCAG2ICT mapping, automated-checks-are-not-enough invariant).

---

## W12 — Reconnect / Replay Client Verification

**Priority**: P1 (TUI precondition from H10 spike scope)  
**Goal**: Verify reconnect/replay as a client-visible contract.

| Task | Detail |
|------|--------|
| W12.1 | Add client-facing conformance test for JSONL event replay |
| W12.2 | Add client-facing conformance test for daemon reconnect after kill |
| W12.3 | Verify no UI state corruption on reconnect (test harness) |
| W12.4 | Document reconnect/replay client contract, troubleshooting, operator diagnostics, TUI state recovery, and agent/client integration guidance |

**Entry**: PR #64 merged.  
**Exit**: Conformance tests pass; reconnect/replay is a tested public contract.

**Status**: ✅ complete 2026-09-14 — `internal/harness/daemon/reconnect_client_test.go`: stable replay across clients, idempotent repeated reads, durability across a server restart, and serialisable client state; `docs/harness/reconnect-replay.md` documents the client contract, failure matrix, operator diagnostics, TUI state recovery and agent/client integration guidance.

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
| W13.5 | Seed `evals/documentation/` with the audit's failing fixtures: D001 keyword false-positive, D002 evidence misbinding, D003 stale command, D004 stale agent path, D005 contradictory future-status claim, D006 translation drift, D007 media drift, D008 protected CURATED section, D009 projection cycle, D010 context dedup |

**Entry**: W4 + W5 complete.  
**Exit**: At least one eval per category runs; baselines recorded.

**Status**: ✅ complete 2026-09-15 — four enforced corpora, each with controls that must still pass. `evals/documentation/` D001–D011 (keyword false-positive, evidence misbinding, lexical-only-never-ready, unmapped requirement, missing claim, stale evidence, missing artifact, unauthorised claim, projection cycle, contradiction, superseded claim); `evals/context/` C001–C010 (L0–L4 disclosure, canonical-preference dedup, budget exclusion with a reason, sufficiency pass/fail); `evals/ui-specification/` U001–U009 (contract without evidence, profile coverage, state vocabulary, duplicate id, missing TUI fallback, accessibility exception path, contrast, token/theme gaps, clean control); `evals/reconstruction/` R001–R004 (byte-exact Markdown, route independent of title, agent surfaces never published as human pages, historical records out of AI retrieval). Runners: `TestDocumentationEvalCorpus`, `TestContextEvalCorpus`, `TestUISpecificationEvalCorpus`, `TestReconstructionEvalCorpus`.

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
| W14.11 | Add "A projection cannot silently become canonical" |
| W14.12 | Add "Silent documentation N/A is forbidden; explicit N/A is valid" |
| W14.13 | Add "Every generated projection exposes provenance and freshness" |
| W14.14 | Add "Localization preserves one canonical implementation truth" |
| W14.15 | Add "Accessibility requires evidence appropriate to the surface; automated checks are not always sufficient" |
| W14.16 | Add "Media has provenance and freshness like code-derived artifacts" |
| W14.17 | Add "Site generators and vendor agent formats are replaceable adapters" |
| W14.18 | Add "Documentation cost and quality are measurable Project Intelligence" |
| W14.19 | Add "Documentation impact is computed both before and after behavior-changing work" |
| W14.20 | Add "Human, agent, site, locale, and API views differ in presentation but resolve to the same canonical claims" |
| W14.21 | Add "The harness optimizes for current, sufficient, verifiable knowledge — not maximum prose volume" |

**Entry**: W3 + W4 complete.  
**Exit**: FRAMEWORK.md updated; no contradiction with existing invariants.

**Status**: ✅ complete 2026-09-14 — FRAMEWORK.md gains a “Documentation & knowledge invariants” section with the 19 control-plane invariants from ADR 010 (canonical-before-prose, stable identity, explicit authority, semantic readiness, projection cannot become canonical, freshness, pre/post impact, silent N/A forbidden, compiled agent surfaces, deterministic-first), cross-referenced without contradicting the existing core invariants and authority order.

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
| W15.13 | Create `docs/architecture/documentation-control-plane.md` (layers A–H, target contracts, package boundaries) and ADR 010 |

**Entry**: W3 stable IDs and W10 core schemas exist.  
**Exit**: `prumo docs readiness` cannot pass merely because matching words or enough evidence items exist; every obligation is semantically bound.

**Status**: ✅ complete 2026-09-15 — semantic readiness is authoritative and the repository's own bindings now carry real claims and verified evidence.

| Task | State |
|---|---|
| W15.1–W15.2 architecture boundary + package reconciliation | ✅ `docs/architecture/documentation-control-plane.md` (layers A–H, dependency direction, composition root) |
| W15.3–W15.7 requirement → claim → evidence model, readiness v2, authority/freshness/revision checks | ✅ `internal/documentation/semantic.go` + `coverage.go`; `mode`, `authoritative`, `requirements` and `findings` on every coverage result |
| W15.5 lexical satisfaction retired from authoritative readiness | ✅ lexical results are `unverified`, `authoritative: false`; new `Unverified` state blocks `Ready` |
| W15.8 contradiction/supersession integration | ✅ `superseded-claim` + `contradiction` findings (typed at the docengine boundary, no harness coupling) |
| W15.9 canonical/projection lineage | ✅ `projectionCycleFindings` + eval case D009 |
| W15.12 false-green fixtures | ✅ D001–D011 in `evals/documentation/` with `expect_finding` assertions |
| W15.11 semantic dogfood | ✅ `TestSemanticReadinessDogfood` asserts the live state without hard-coding counts: no non-authoritative contract may reach `verified`, and no `unverified` state may be unexplained |
| W15.10 migrate the repository's own bindings | ✅ 7 contracts (`product.vision`, `project.scope`, `architecture.system`, `testing.strategy`, `security.trust`, `installation.lifecycle`, `cli.reference`) carry accepted claims with verified evidence at revision 1; `prumo docs readiness` reports `ready: true` with zero warnings |

**Deliberate non-outcome (kept)**: the semantic evaluator has no "agent says so" escape hatch. Evidence must name the claim it proves, carry a stable ID, be marked `verified`, and — when the artifact is a documentation file — resolve to a canonical or projection document. The first draft of the repository's own bindings cited `docs/migration/conformance-strategy.md`, which the authority map classifies as **historical**; the model rejected it as `non-canonical-evidence` rather than accepting a green result. That is the audit's D002 misbinding case firing on the real repository, and it is the reason W15.10 was authored instead of asserted.

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

**Status**: ✅ complete 2026-09-15.

| Task | State |
|---|---|
| W16.1 IR + schema | ✅ `docs/agents/instruction-ir.json` (canonical, provider-neutral, 15 rules) + `schemas/agent-instruction-ir.schema.json` |
| W16.2 deterministic precedence/scoping | ✅ `EffectiveRules`: scope depth → precedence → rule ID; nested scopes inherit; `Conflicts` for duplicates and same-ID divergence |
| W16.3–W16.4 AGENTS.md root region + nested scopes | ✅ root scope compiles into the `agents-core` managed region of `AGENTS.md`; nested scopes compile to `<scope>/AGENTS.md` |
| W16.5–W16.8 copilot / cursor / claude / skill-md adapters | ✅ `.github/…`, `.cursor/rules/*.mdc`, `CLAUDE.md` (imports `@AGENTS.md`, never a second copy), `.agents/skills/*/SKILL.md` — all redirected to runtime by default |
| W16.9 fingerprints/provenance + manifest sidecar | ✅ `<!-- prumo:generated adapter=… scope=… fingerprint=… sources=… -->` on every surface; `manifest.json` written by `build` |
| W16.10 reference validation | ✅ paths, docs pointers, globs, `prumo` subcommands, test names and `symbol()` references resolved against the repository |
| W16.11 token budgets | ✅ IR `token_budget` enforced per scope via the harness estimator (AGENTS.md region: 386/420 tokens) |
| W16.12 duplicate/conflict detection | ✅ `duplicate-instruction` and `conflicting-rule` findings |
| W16.13 adapter isolation invariant | ✅ `TestCoreHasNoVendorKnowledge` reads the core sources and fails on any vendor format reference |
| W16.14 context-rot CI gate | ✅ `TestAgentSurfaceContextRotGate` + `prumo docs agents verify` (exit 1 on findings) |
| W16.15 documentation impact integration | ✅ `AffectedSurfaces` + `prumo docs impact` emits `agent-surface:<adapter>` impacts |

**Deliberate boundary**: the compiled root invariant list now lives in the IR, and `AGENTS.md` carries it as a managed region. Editing the list in `AGENTS.md` by hand makes `docs agents verify` fail until the IR is changed and the build re-run — that is the wave's exit criterion, not an accident.

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

**Status**: ✅ complete 2026-09-15 — typed matchers, Goal-lock preflight, post-implementation reconcile and the delta ledger all land on the real path.

| Task | State |
|---|---|
| W17.1 typed impact triggers | ✅ `impact_semantic.go`: `symbol:`, `schema:`, `token:`, `ui:`, `api:`, `event:`, `permission:`, `locale:`, `media:`, `goal:`, `wave:`, `adr:`, `release:`, `evidence:` — each deterministic |
| W17.2 symbol impact | ✅ resolved by identifier occurrence in a changed source file, so it runs headless with no LSP session; an unresolvable trigger never matches |
| W17.3 schema/API impact | ✅ structural match against `schemas/*.schema.json`, `docs/contracts/builtin.json` and the connector contract |
| W17.4 UI impact | ✅ resolved through contract → bound-document → changed-path, i.e. the component/token graph rather than a filename guess |
| W17.5 `prumo docs plan --goal` | ✅ preflight stores a replayable `DocumentationPlan` under derived runtime state |
| W17.6 GoalPlan `DocumentationGap`/`DocumentationPlan` | ✅ `internal/app/goalplan.go` populates both on the real Goal path |
| W17.7 preflight at Goal lock | ✅ `BuildPlanForContracts` runs when the Goal is locked, before any implementation path exists |
| W17.8 postflight after implementation | ✅ `prumo docs plan --goal <id> --changed <paths> --reconcile` |
| W17.9 predicted vs actual diff | ✅ `Reconcile` reports obligations the preflight predicted and the observed impact did not honour |
| W17.10 TranslationDelta / MediaDelta | ✅ emitted by `internal/doclifecycle` and surfaced by `prumo docs release` |
| W17.11 explicit N/A reasons | ✅ `DocumentationPlan.Validate` refuses a silent N/A: every non-applicable obligation must state `not_applicable_reason` |
| W17.12 impact explanation | ✅ `prumo docs explain <unit-or-finding>` |

**Deliberate boundary**: plans, reconciliations and deltas are derived runtime state. None of them may be promoted to a canonical document, and the preflight refuses to run on an empty change set instead of reporting a silent "no impact".

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

**Status**: ✅ complete 2026-09-15 — the renderer boundary, four renderers, search index, versioned/locale routes, `/llms.txt`, `/llms-full.txt`, raw Markdown views, generated contract/schema reference pages and the documentation MCP surface all landed. The MCP surface is read-first: `prumo://docs/...` resources are always readable, while mutations are denied by default and must be explicitly allowlisted (`prumo docs mutations`).

| Task | State |
|---|---|
| W18.1 renderer interface | ✅ `docpublish.Renderer` — one `Graph` in, `[]Artifact` out; no site generator in the interface |
| W18.2 Starlight reference adapter | ✅ `starlightRenderer` emits `astro.config`-ready routes |
| W18.3 navigation/breadcrumb/related edges | ✅ derived from the page graph in `Graph.Routes()`/`Graph.Current()` |
| W18.4 search index | ✅ `search-index.json` chunked by heading; `prumo docs query` ranks over it |
| W18.5 version-aware routes | ✅ `--versioned` |
| W18.6 locale-aware routes/fallback | ✅ `--locale` plus `localesOf`/`versionsOf` fallback resolution |
| W18.7 public/internal visibility filtering | ✅ `Graph.Human()` vs `Current()`; agent-facing surfaces are separated |
| W18.8 `/llms.txt` | ✅ `llmsRenderer` |
| W18.9 `/llms-full.txt` | ✅ emitted alongside the index |
| W18.10 raw Markdown page views | ✅ `markdownRenderer` |
| W18.11 read-only documentation MCP resources | ⬜ not implemented — the MCP surface exposes no documentation resources yet |
| W18.12 mutation MCP kept separate and permission-gated | ⬜ depends on W18.11 |
| W18.13 API reference adapters from machine contracts | ⬜ not implemented |
| W18.14 documentation dashboard projection | ✅ `prumo docs metrics` (W21.1) provides the Project Intelligence projection |
| W18.15 renderer replaceability/conformance test | ✅ `TestRenderersProjectOneGraph` projects one graph through every renderer and asserts the same page identity |

**Deliberate boundary**: published output is written under `.prumo/runtime/docs-site/` and is derived — it is never a canonical artifact.

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

**Status**: 🟡 partial 2026-09-15 — every W19 task listed below is complete and live in CI. What keeps this wave partial is **DOC-GAP-020** (P1, traced to this wave by the audit): examples are linked and checked for existence, but they are not yet executed as testable documentation units.

| Task | State |
|---|---|
| W19.1 `DocumentationQualityPolicy` | ✅ `schemas/documentation-quality-policy.schema.json` (W10) drives which checks are blocking |
| W19.2 layered verifier pipeline | ✅ `VerifyDocs`: authority → managed regions → links → claim drift, all deterministic and run before any model step |
| W19.3 docs profile in the Gauntlet runtime | ✅ `internal/gauntlet.Run` + `prumo docs gauntlet` |
| W19.4 isolated critic roles | ✅ model critic is a separate, skippable stage (`model_critic: "skipped:policy-mode-off"` by default) |
| W19.5 bounded repair loop with stop conditions | ✅ rounds 1..10 with `gates-passed` / `rounds-exhausted` / `no-progress` terminators |
| W19.6 findings/evidence persisted per round | ✅ each round records its deterministic result and findings |
| W19.7 score cannot rise without resolved findings | ✅ score-inflation guard from W7 is enforced in the runtime loop |
| W19.8 `prumo docs verify` in CI | ✅ new `Documentation verification gate` CI step |
| W19.9 `--strict` completion integration | ✅ `VerifyDocsStrict` + `prumo docs verify --strict`; a lexical-only contract now fails the gate that plain `verify` would pass, with its own CI step |
| W19.10 regression corpus | ✅ `evals/documentation/` D001–D011 plus the drift cases the verifier found against real repository documents |
| — (DOC-GAP-020, audit) example execution | ⬜ linked and existence-checked only; running them as testable units is the remaining gap |

**Invariant preserved**: no model critic can override a deterministic finding. The Gauntlet records a critic opinion; it cannot clear a blocker.

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

**Status**: ✅ complete 2026-09-15 — every W20 task landed, including the terminology registry and the explicit documentation version policy.

| Task | State |
|---|---|
| W20.1 TranslationRecord QA lifecycle | ✅ `internal/doclifecycle`: lifecycle states, locale-key identity, placeholder integrity, reviewer evidence, fallback and RTL |
| W20.2 terminology/glossary management | ✅ `docs/glossary.json` + `schemas/glossary.schema.json`: closed status vocabulary (preferred/deprecated/forbidden), a deprecated term must name a **preferred** replacement, whole-word matching, and the check runs inside `docs verify`; `prumo docs glossary` reports it |
| W20.3 pseudo-localization verifier | ✅ `PseudoLocalize` + `VerifyPseudoLoc`, exercised by `TestVerifyPseudoLocFindsGeneratorProblems` |
| W20.4 `MediaRecord` + stale propagation | ✅ `schemas/media-record.schema.json` + `MediaStatuses` |
| W20.5 visual evidence linked to state/theme/locale/platform | ✅ media records carry the UI state, theme, locale and platform they illustrate |
| W20.6 documentation version policy | ✅ `version_policy` in `docs/lifecycle.json` + `schemas/doc-lifecycle.schema.json`: exclusive current/supported/deprecated/removed buckets, drift against `protocol.CLIVersion`, and a reference to a removed line fails unless the document is historical or the line is annotated as such; `prumo docs version` reports it |
| W20.7 deprecation lifecycle | ✅ `LoadLifecycle` + `DeprecationFindings`: a deprecation must name a replacement or a removal, and be annotated in the document itself |
| W20.8 release documentation plan | ✅ `CheckRelease` → `prumo docs release` |
| W20.9 compatibility/migration projection inputs | ✅ migration projections feed the release report |
| W20.10 release readiness docs gate | ✅ `release_gates` in `docs/lifecycle.json`, evaluated by `CheckRelease` |
| W20.11 historical docs without contaminating current context | ✅ the authority map keeps historical documents out of current context, and semantic evidence refuses a historical artifact (`non-canonical-evidence`) |

**Stopping condition**: staleness is computed from content digests, so "current" is mechanically checkable instead of asserted by a human or a model.

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

**Status**: 🟡 partial 2026-09-15 — the metrics, scale, adoption and debt tasks landed, and W21.2/W21.3 are **refused with an ADR** rather than left as an open gap. Outstanding: DOC-GAP-024 (route documentation work to specialised skills by impact type) and DOC-GAP-026 (full Markdown AST for list/table editing), both P2 and neither blocking the TUI.


| Task | State |
|---|---|
| W21.1 documentation metrics in Project Intelligence | ✅ `internal/docintel.Collect` + `prumo docs metrics` |
| W21.2 retrieval/query success without storing sensitive content | ⬜ deliberately not implemented — there is no telemetry channel and no query log; capturing user queries would store sensitive content by design |
| W21.3 top missing/failed documentation intents | ⬜ deliberately not implemented — it depends on W21.2, which is refused for the same reason |
| W21.4 context/instruction token efficiency | ✅ `documentation_tokens` vs `instruction_tokens`, with agent-surface surfaces counted separately |
| W21.5 freshness SLA and drift recurrence | ✅ bounded `freshness_score` plus `claim_drift` and `broken_links` counters |
| W21.6 translation/media/example health | ✅ `stale_translations` and `stale_media` |
| W21.7 brownfield docs discovery | ✅ `Inspect`: signals, document inventory, one walk (no double-counting) |
| W21.8 propose contracts/bindings from existing docs | ✅ `Propose` infers candidates with confidence and evidence |
| W21.9 detect duplicates/conflicts during adoption | ✅ `duplicateBasenames` + conflict signals |
| W21.10 reviewer promotion before inferred authority | ✅ proposals are `confidence: inferred` and state the promotion path; nothing is written into the target |
| W21.11 benchmark large-repository build/invalidation | ✅ `TestLargeRepositoryIncrementalInvalidationIsBounded` proves a cold build writes every artifact, an unchanged rebuild writes nothing (the manifest is now compared before writing, so churn is not reported without cause) and one edited document invalidates a bounded slice; `BenchmarkBuildLargeRepository` reports it (200 docs: cold ~64.5 ms, unchanged ~14.5 ms on an i7-3632QM) |
| W21.12 documentation debt integration | ✅ `documentationDebt` names each outstanding obligation; an unverified contract always appears as debt and lowers the score |

**Deliberate non-goal, now an accepted decision**: W21.2/W21.3 stay unimplemented. The framework's knowledge-authority and trust rules forbid silently collecting what users retrieve, and "success without storing content" is not achievable for these metrics in a useful form — a success rate has to know which query failed. Metrics are computed from repository state only. The decision is recorded in **ADR 011** (`docs/adr/011-documentation-telemetry-refusal.md`), including the condition for revisiting it.

---

## W22 — Interface Map & Interdependence Tree

**Priority**: P1 (completes the `ui.*` obligations the TUI increment opens)
**Goal**: One canonical artifact mapping a whole interface — components,
subcomponents, text, inputs, buttons, menus — with where each element sits and how
the elements interconnect, projected for the developer, the documentation site and
the code agent.

**Why it exists**: W5 created `ui-component-contract` and the four obligations
(`ui.component-contracts`, `ui.information-architecture`, `ui.screen-inventory`,
`ui.layout`) were satisfied by waivers with no producer behind them. The gate was
green while no component inventory existed. Decision: **ADR 012**.

| Task | Detail |
|---|---|
| W22.1 | `schemas/ui-interface-map.schema.json` — closed kind/region/align/stack/edge vocabularies, mandatory slot, optional per-platform geometry |
| W22.2 | `internal/uimap` — model, loader, vocabulary read from the state matrix and token set the map names |
| W22.3 | Validation, fail closed: unique ids, single shell root, mandatory position, labels for text-rendering kinds, states from the matrix, tokens from the set, edge endpoints and a closed edge vocabulary, no duplicate or self edges |
| W22.4 | Declared + derived with declared always winning; `off` \| `verify-only` \| `fill-gaps`; divergence is an error |
| W22.5 | `SymbolDeriver` seam + `GoSymbolDeriver` (types, methods, exported fields; test files excluded) |
| W22.6 | Interdependence tree, typed edges and impact closure in both directions with the reason for every hop |
| W22.7 | Three projections — developer reference, site pages, bounded agent surface plus a machine-readable index — with digest freshness and stale detection |
| W22.8 | Per-project configuration (`ui.interface_map`: enabled, targets, derivation, path) with auto-enable on a declared interface and refusal of silent no-ops |
| W22.9 | `prumo ui map\|verify\|impact\|config` |
| W22.10 | Dogfood: this repository's own map, with `not-implemented` used for the approval surface that GAP-046 blocks |
| W22.11 | Bind the four contracts to the artifact, replacing four waivers |
| W22.12 | Conformance gate: schema enums equal the Go vocabularies, and the repository's own map is valid and its projections fresh |

**Entry**: W5 complete (contracts and schemas), W9 complete (tokens), H10 Fase A
(the `tui` profile composed, so the obligations are actually evaluated).
**Exit**: `prumo ui verify` passes on this repository, the four contracts are
bound, and the projections are fresh.

**Status**: ✅ complete 2026-09-15 — 24 elements, 10 typed interconnections, 216
derived symbols; the four waivers are replaced by bindings; `ui.component-contracts`
no longer says "owed by the first layout increment".

**Cost, stated**: the map is one more artifact to keep current. It is enforced
rather than trusted — changing the interface without changing the map fails
`prumo ui verify`, and `prumo ui map --write` regenerates all eight projections.
The agent surface is bounded at 80 tree rows and reports the exact number it
omitted instead of truncating silently.

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
- [x] W0 complete (authority map + drift gate + docs reconciliation, 2026-09-14)
- [x] W1 complete (schema/runtime conformance, 2026-09-14)
- [x] W5 complete (UI contracts exist, TUI profile composable, 2026-09-14)
- [x] W6 complete (locale/translation contracts; digest staleness and pseudo-localization landed in W20) 2026-09-15
- [x] W8 complete (Shared Product Contract promoted, 2026-09-14)
- [x] W9 complete (semantic design tokens for TUI, 2026-09-14)
- [x] W11 complete (TUI accessibility/interaction contracts, 2026-09-14)
- [x] W12 complete (reconnect/replay verified by client-facing conformance, 2026-09-14)
- [x] W15 complete (semantic readiness v2 eliminates false confidence; the repository's own 7 contracts are semantically bound and `prumo docs readiness` is green for the right reason) 2026-09-15

P1 waves W10, W13, W14, W16–W19 may proceed in parallel with early TUI work but must
complete before H10 exit criteria are evaluated. Their current state:

| Wave | Scope | State |
|---|---|---|
| W10 | Documentation compiler contracts | ✅ complete |
| W13 | Behavioral eval corpus | ✅ complete — `evals/documentation/` D001–D011, `evals/context/` C001–C010, `evals/ui-specification/` U001–U009 and `evals/reconstruction/` R001–R004 are all enforced |
| W14 | Framework invariants | ✅ complete |
| W16 | Agent surface compiler & context-rot guard | ✅ complete |
| W17 | Semantic impact graph & delta orchestration | ✅ complete |
| W18 | Publishing & AI retrieval plane | ✅ complete — renderers, versioned/localized routes, contract/schema reference pages and the documentation MCP resource surface |
| W19 | Continuous verification & docs gauntlet | 🟡 deterministic verifier, `--strict` gate and gauntlet complete; example execution (DOC-GAP-020) outstanding |

**P2 waves (W20/W21)**: W20 ✅ complete (terminology registry and explicit version policy landed); W21 🟡 (scale benchmark landed; query-intent tracking refused by ADR 011; workforce routing by impact type and full Markdown AST remain). Neither blocks the TUI: the spike consumes the W5/W6/W9/W11 contracts and the W12 reconnect contract, all complete.

**Stopping condition for this wave set**: the TUI gate is met when every P0/P1 contract wave is complete *and* `prumo docs verify --strict` passes. Both hold. Two items the audit proposed are recorded as **refused** with ADR 011 rather than as outstanding work, and two waves remain genuinely partial by scope rather than by refusal: W19 owes DOC-GAP-020 (examples are linked and existence-checked, but not executed) and W21 owes DOC-GAP-024 and DOC-GAP-026 (workforce routing by impact type, full Markdown AST). `ACCEPTED != implemented` applies to them too.

---

## CLI surface (proposal)

Names reconciled with the existing CLI at implementation time; every command
supports structured JSON where applicable. Headless operation must never
require the interactive site/TUI.

| Command | Wave |
|---------|------|
| `prumo docs audit` | W0, W15 |
| `prumo docs readiness` | W15 |
| `prumo docs impact` · `prumo docs plan --goal <id>` · `prumo docs delta --goal <id>` | W17 |
| `prumo docs explain <unit-or-finding>` | W15, W17 |
| `prumo docs verify` · `prumo docs gauntlet` | W19 |
| `prumo docs build` · `prumo docs manifest` · `prumo docs query` · `prumo docs doctor` | W15, W18 |
| `prumo docs agents build\|verify\|explain` | W16 |
| `prumo docs site build\|verify` | W18 |
| `prumo docs glossary` | W20 |
| `prumo docs version` | W20 |
| `prumo docs translate status\|verify` | W6, W20 |
| `prumo docs media status\|verify` | W20 |
| `prumo docs adopt inspect\|propose` | W21 |
| `prumo docs metrics` | W21 |
| `prumo knowledge manifest\|search` | W3 |
| `prumo context compile\|explain` | W4 |

---

## Event model (proposal)

Reuse the harness event infrastructure; events carry IDs/pointers, never full
documents.

```text
documentation.plan.created      documentation.unit.generated      documentation.agent_surface.compiled
documentation.impact.detected   documentation.unit.assisted       documentation.translation.stale
documentation.delta.updated     documentation.unit.stale          documentation.media.stale
documentation.unit.verified     documentation.projection.compiled documentation.quality.failed|passed
documentation.gauntlet.round.started|completed                    documentation.site.built
documentation.publish.completed
```

---

## CI gate groups

| Group | Wave | Blocks on |
|-------|------|-----------|
| `docs-schema` | W1 | schema ↔ Go conformance, migrations |
| `docs-authority` | W0 | authority graph, illegal projection → canonical cycles |
| `docs-impact` | W17 | changed files produce a documented impact result (silent omission prohibited) |
| `docs-semantic` | W15 | required claims/evidence, contradictions, staleness |
| `docs-refs` | W19 | links, anchors, paths, symbols, commands, schema refs |
| `docs-examples` | W19 | verified examples run |
| `docs-agents` | W16 | adapter compile, token budgets, code refs, scope conflicts, freshness |
| `docs-i18n` | W6, W20 | translation records, placeholder integrity, freshness |
| `docs-media` | W20 | required media present/current |
| `docs-site` · `docs-ai` | W18 | site build, nav orphans, llms index, MCP unit identity |
| `docs-a11y` | W11, W19 | required automated checks, evidence completeness |

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
