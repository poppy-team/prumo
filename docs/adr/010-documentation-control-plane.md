# ADR 010: Documentation Control Plane

# Status

Accepted (2026-09-15) — implementation landed with waves W15–W21; the only tasks not implemented (W21.2/W21.3) are refused by ADR 011

# Context

- The Documentation Control Plane Deep Audit (2026-09-14) found Prumo already
  owns the right primitives — Knowledge Runtime, documentation contracts and
  bindings, HumanDocs HD0–HD4, `doccompile`, Context Compiler v2, typed impact,
  Goals/Waves — but no coordinator expressing the documentation lifecycle.
- Critical finding: readiness could pass on loose lexical matching
  (`internal/documentation/coverage.go`) and quantity-based evidence counting,
  producing **false-green** "documentation ready" results that disagree with
  implementation truth. A lexical stopgap (all-keyword + stopwords, distinct
  non-empty evidence) has since landed with regression tests, but the target is
  a typed requirement → claim → evidence model.
- Critical finding: agent instructions (`AGENTS.md`) are maintained directly and
  can rot into an unmanaged parallel truth as they reference stale symbols,
  commands and paths.
- The harness headless runtime (HA0–HA11) is complete; documentation is the
  next mechanism through which the harness controls and verifies development.

# Decision

1. Introduce a **Documentation Control Plane** as a lifecycle/coordination
   concept over existing modules — not a new monolith and never a second
   Knowledge Runtime. Layers: Canonical Knowledge, Documentation Contracts,
   Documentation Planning, Projection/Compiler, Agent Surfaces, Publication,
   Verification, Observability.
2. **Canonical knowledge and typed contracts precede prose.** Documentation,
   agent instructions, context packs, site pages, translations/media and
   AI-retrieval surfaces are governed projections with explicit authority,
   provenance, lifecycle and verification.
3. **Readiness is semantic requirement → claim → evidence coverage** — not
   keyword presence and not evidence count. The lexical matcher is a stopgap,
   not the target; semantic binding lands in W15.
4. **Agent instruction surfaces are compiled**: a provider-neutral
   `AgentInstructionIR` plus per-tool adapters (AGENTS.md, Copilot, Cursor,
   Claude, SKILL.md), fingerprinted and freshness-checked. Vendor formats are
   replaceable adapters and are never canonical.
5. **Deterministic verification precedes model-assisted critique**, and a model
   critic may not override a deterministic failure.
6. Existing packages (`internal/documentation`, `internal/harness/knowledge`,
   `humandocs`, `doccompile`, `contextv2`) are retained with compatibility
   wrappers; the control plane orchestrates them behind interfaces. Dependency
   direction points inward: adapters depend on the core, never the reverse.
7. Delivery follows `docs/development/waves.md` (W15–W21). No YAML canonical
   artifacts are introduced.

# Consequences

- "Docs ready" can no longer disagree with implementation truth; stale or
  contradictory required documentation blocks Goal/release completion (W19).
- Agent context stops being duplicated parallel truth: instruction files
  regenerate from canonical knowledge and fail CI on stale code references (W16).
- The false-green cases become reproducible eval fixtures and permanent guards
  (W13/W15).
- Cost: new schemas, a coordinator package, and adapter maintenance. Explicit
  non-goals: a general CMS, a Notion replacement, a WYSIWYG editor, a mandatory
  vector database/LLM/site generator, or a full translation SaaS.
- `FRAMEWORK.md` invariants land with W14, not with this ADR.

# References

- Input audit: `PRUMO_DOCUMENTATION_CONTROL_PLANE_DEEP_AUDIT_AND_WAVES.md`
  (gitignored input artifact; reconciled into `docs/development/waves.md`)
- `docs/development/waves.md` (W0–W21)
- `docs/harness/gap-register.md` §5 (DOC-GAP-001 – DOC-GAP-030)
- `docs/DOCUMENTATION_SYSTEM.md`, `docs/harness/human-docs.md`,
  `docs/harness/context-knowledge.md`
