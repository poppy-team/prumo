# Implementation Waves — Post-Harness, Pre-TUI

> Authority: repository-canonical. Supersedes ad-hoc gap lists.
> Source: deep research analysis (2026-09-14), gap-register, roadmap-status,
> Living Book (pages 13, 14, 18, 21, 23, 25–38).

These waves sit between the harness headless completion (HA0–HA11, split gate
READY conditional) and the H10 TUI spike. Each wave has explicit entry/exit
criteria. Waves are sequential within priority tiers; P0 waves block P1 waves.

---

## W0 — Authority & Version Cleanup

**Priority**: P0 (blocks everything)
**Goal**: Eliminate version drift and authority routing confusion.

| Task | Detail |
|------|--------|
| W0.1 | Update `AGENTS.md` routing table: v0.4 → current version references |
| W0.2 | Update `docs/PRUMO.md` to consistent version |
| W0.3 | Update `README.md` to match docs version |
| W0.4 | Update `docs/product/scope-v0.4.md` — either rename to scope-v0.5 or add v0.5 addendum |
| W0.5 | Audit all `docs/` files for stale v0.4 references; update or annotate |
| W0.6 | Formalize authority order in a single canonical location |

**Entry**: PR #64 merged (harness branch landed).
**Exit**: `grep -r 'v0\.4' docs/` returns only historical/migration references; authority routing is unambiguous.

---

## W1 — Schema / Runtime Conformance Gate

**Priority**: P0
**Goal**: Automatically verify enums, required fields, and round-trip between
JSON Schema and Go types. Prevent contract/runtime drift.

| Task | Detail |
|------|--------|
| W1.1 | Create `conformance/schema-runtime/` test suite |
| W1.2 | Verify all schema enums match Go const blocks (e.g., `pressure` in schema vs Go) |
| W1.3 | Round-trip tests: JSON → Go struct → JSON for all harness schemas |
| W1.4 | Add to CI as required check |
| W1.5 | Fix any discovered drift (ADR 007 `compact` enum, etc.) |

**Entry**: W0 complete.
**Exit**: `go test ./conformance/schema-runtime/...` green in CI; zero enum/field mismatches.

---

## W2 — Canonical Format Cleanup

**Priority**: P0
**Goal**: Eliminate YAML from new canonical designs; enforce Markdown + JSON policy.

| Task | Detail |
|------|--------|
| W2.1 | Audit Notion-promoted designs for `session.yaml` or YAML structures |
| W2.2 | Convert any candidate YAML specs to JSON/Markdown |
| W2.3 | Add lint check: no new `.yaml`/`.yml` in canonical paths |

**Entry**: W0 complete.
**Exit**: No YAML in canonical specs; lint enforced.

---

## W3 — KnowledgeUnit & Stable Identity Foundation

**Priority**: P0
**Goal**: Establish the atom of retrievable knowledge with stable IDs
independent of paths/titles.

| Task | Detail |
|------|--------|
| W3.1 | Define `schemas/knowledge-unit.schema.json` |
| W3.2 | Define `schemas/knowledge-manifest.schema.json` |
| W3.3 | Define `schemas/knowledge-claim.schema.json` |
| W3.4 | Implement KnowledgeUnit Go types in `internal/knowledge/` |
| W3.5 | Implement KnowledgeManifest builder (derived index, not new source of truth) |
| W3.6 | Add stable ID generation and validation |
| W3.7 | Migrate existing Knowledge Runtime baseline (GAP-018/019) to use stable IDs |

**Entry**: W1 complete (schema conformance infra exists).
**Exit**: `prumo knowledge manifest` produces valid manifest; IDs survive rename.

---

## W4 — ContextManifest & Progressive Disclosure

**Priority**: P0
**Goal**: Every compiled context is explainable and replayable.

| Task | Detail |
|------|--------|
| W4.1 | Define `schemas/context-manifest.schema.json` |
| W4.2 | Define `schemas/context-policy.schema.json` |
| W4.3 | Implement ContextManifest generation in Context Compiler |
| W4.4 | Implement progressive disclosure levels L0–L4 (Pointer → Full) |
| W4.5 | `prumo context compile --goal G... --budget N` emits manifest |
| W4.6 | `prumo context explain CTX-...` command |
| W4.7 | Context sufficiency evaluation (required claims covered, dependencies included) |

**Entry**: W3 complete (KnowledgeUnits exist to reference).
**Exit**: Every context compilation produces auditable manifest; L0–L4 levels functional.

---

## W5 — UI Contract Decomposition

**Priority**: P0
**Goal**: Replace monolithic `ui.documentation` with specialized contracts.

| Task | Detail |
|------|--------|
| W5.1 | Define contract schemas: `ui.product-ux`, `ui.information-architecture`, `ui.screen-inventory`, `ui.layout`, `ui.interaction`, `ui.state-model`, `ui.design-tokens`, `ui.component-contracts`, `ui.accessibility`, `ui.localization`, `ui.theming`, `ui.visual-validation`, `ui.platform-conventions` |
| W5.2 | Define profile compositions: `desktop-gui`, `web-application`, `tui`, `cli` |
| W5.3 | Create `schemas/ui-component-contract.schema.json` |
| W5.4 | Create `schemas/ui-state-matrix.schema.json` |
| W5.5 | Create `schemas/design-token-set.schema.json` (aligned with W3C Design Tokens format) |
| W5.6 | Deprecate monolithic `ui.documentation` in favor of composed contracts |

**Entry**: W3 complete (contracts need stable IDs).
**Exit**: TUI profile (`tui`) exists and is composable; component/state/token schemas validate.

---

## W6 — Locale & Translation Foundation

**Priority**: P0
**Goal**: Formalize English as canonical locale; establish translation lifecycle.

| Task | Detail |
|------|--------|
| W6.1 | Add `source_locale: en` and `locales: ["en", "pt-BR"]` to project config schema |
| W6.2 | Define `schemas/translation-record.schema.json` |
| W6.3 | Define translation lifecycle states: `missing → machine-draft → needs-review → current → needs-update → stale` |
| W6.4 | Implement digest-based staleness detection (source changes → translation `needs-update`) |

**Entry**: W0 complete.
**Exit**: Translation records track source digest; staleness auto-detected.

---

## W7 — Gauntlet Schema & Policy (mode: off)

**Priority**: P0
**Goal**: Define Gauntlet loop contracts; keep `off` as default.

| Task | Detail |
|------|--------|
| W7.1 | Define `schemas/gauntlet-policy.schema.json` (mode: off/auto/force, max_rounds, critic_isolation, activation rules) |
| W7.2 | Define `schemas/gauntlet-run.schema.json` (run_id, rounds, scorecards, stop_reason, evidence) |
| W7.3 | Implement GauntletPolicy Go types |
| W7.4 | Add Framework invariant: Gauntlet default = `off`; activation requires quality oracle |

**Entry**: W1 complete.
**Exit**: Schemas validate; policy loads; mode defaults to `off`; no runtime behavior yet.

---

## W8 — Shared Product Contract Promotion

**Priority**: P1
**Goal**: Promote Living Book page 23 (Shared Product Contract) into repo contracts.

| Task | Detail |
|------|--------|
| W8.1 | Extract entities, state ownership, capabilities from page 23 |
| W8.2 | Create `docs/architecture/shared-product-contract.md` |
| W8.3 | Create corresponding JSON schemas where applicable |
| W8.4 | Update SOURCE_MAP.json with provenance |

**Entry**: W0 complete.
**Exit**: Contract is repository-authoritative; Notion page becomes historical.

---

## W9 — Semantic Design Tokens

**Priority**: P1
**Goal**: Define the three-tier token system (primitive → semantic → component)
for TUI and future surfaces.

| Task | Detail |
|------|--------|
| W9.1 | Define primitive tokens (colors, spacing, typography) |
| W9.2 | Define semantic tokens (surface, text, border, focus, accent, success, warning, error) |
| W9.3 | Define TUI-specific component tokens |
| W9.4 | Implement token schema validation |
| W9.5 | Create Visual Constitution for Prumo Code TUI |

**Entry**: W5 complete (design-token-set schema exists).
**Exit**: Token set validates; TUI semantic tokens defined; Visual Constitution written.

---

## W10 — Documentation Compiler Contracts

**Priority**: P1
**Goal**: Promote Documentation Planner/Compiler architecture from Notion to repo.

| Task | Detail |
|------|--------|
| W10.1 | Define DocumentationSpec, DocumentationPlan, DocumentationUnit, DocumentationBrief schemas |
| W10.2 | Define generation tiers: GENERATED (deterministic) / ASSISTED (model + review) / CURATED (protected human) |
| W10.3 | Define Documentation Delta chain: Knowledge Delta → affected DocUnits → Translation Delta → Media staleness |
| W10.4 | Create `docs/architecture/documentation-compiler.md` |

**Entry**: W3 + W6 complete.
**Exit**: Schemas validate; compiler pipeline documented; delta chain specified.

---

## W11 — Accessibility & TUI Contracts

**Priority**: P1
**Goal**: Surface-specific accessibility contracts; TUI interaction contract.

| Task | Detail |
|------|--------|
| W11.1 | Decompose `accessibility` into sub-contracts: keyboard, focus, contrast, screen-reader, motion, zoom-reflow, target-size, drag-alternatives, accessible-auth, cognitive-clarity |
| W11.2 | Create `tui.accessibility` contract with TUI-specific invariants |
| W11.3 | Create `tui.interaction` contract |
| W11.4 | Define evidence types per accessibility sub-contract |
| W11.5 | Reference WCAG 2.2 + WCAG2ICT for non-web guidance |

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

**Entry**: PR #64 merged.
**Exit**: Conformance tests pass; reconnect/replay is a tested public contract.

---

## W13 — Behavioral Eval Corpus (baseline)

**Priority**: P1
**Goal**: Establish evaluation infrastructure before Gauntlet `auto`.

| Task | Detail |
|------|--------|
| W13.1 | Create `evals/context/` — context quality evaluations |
| W13.2 | Create `evals/documentation/` — documentation sufficiency evaluations |
| W13.3 | Create `evals/ui-specification/` — UI spec completeness evaluations |
| W13.4 | Define reconstruction eval: spec → independent agent → conformance comparison |

**Entry**: W4 + W5 complete.
**Exit**: At least one eval per category runs; baselines recorded.

---

## W14 — Framework Invariants Update

**Priority**: P1
**Goal**: Add deep-research constitutional diretivas to FRAMEWORK.md.

| Task | Detail |
|------|--------|
| W14.1 | Add "Knowledge is canonical; prose is a projection" |
| W14.2 | Add "Missing knowledge produces a gap, never invented documentation" |
| W14.3 | Add "Every durable semantic entity has stable identity independent of path" |
| W14.4 | Add "Every generated context is explainable and replayable" |
| W14.5 | Add "Documentation architecture is planned before prose is generated" |
| W14.6 | Add "Design systems are versioned APIs, not styling suggestions" |
| W14.7 | Add "Agents do not silently weaken or satisfy their own quality gates" |
| W14.8 | Add "A generated artifact never becomes canonical merely because an agent produced it" |

**Entry**: W3 + W4 complete (invariants reference concepts that must exist first).
**Exit**: FRAMEWORK.md updated; no contradiction with existing invariants.

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

P1 waves W10, W13, W14 may proceed in parallel with early TUI work but must
complete before H10 exit criteria are evaluated.

---

## Traceability

| Wave | Research Report Section | Gap Register |
|------|------------------------|---------------|
| W0 | Drift table (L39–44) | — |
| W1 | Schema/runtime conformance (L43) | GAP-034, GAP-036 |
| W2 | Format cleanup (L42) | — |
| W3 | KnowledgeUnit (L161–220) | GAP-018, GAP-019, GAP-027 |
| W4 | ContextManifest (L222–278) | GAP-006, GAP-009 |
| W5 | UI contract decomposition (L719–799) | — |
| W6 | Locale decision (L403–458) | — |
| W7 | Gauntlet schema (L459–711) | — |
| W8 | Shared Product Contract | H10 precondition #3 |
| W9 | Design tokens (L876–926) | H10 precondition |
| W10 | Documentation Compiler (L356–401) | GAP-007, GAP-028 |
| W11 | Accessibility (L1015–1122) | — |
| W12 | Reconnect/replay | H10 precondition #2 |
| W13 | Behavioral evals (L1433–1487) | — |
| W14 | Framework invariants (L1526–1567) | — |
