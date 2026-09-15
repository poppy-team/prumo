# Documentation Compiler

> Authority: repository-canonical (W10.5). Architecture decision: ADR 010.
> Control plane layers: `docs/DOCUMENTATION_SYSTEM.md`.

The compiler turns typed canonical knowledge into governed documentation
projections. It never invents facts: missing knowledge produces a gap.

## Contracts

| Contract | Schema | Role |
|----------|--------|------|
| DocumentationSpec | `schemas/documentation-spec.schema.json` | audiences, profiles, surfaces |
| DocumentationPlan | `schemas/documentation-plan.schema.json` | pre-change preflight |
| DocumentationUnit | `schemas/documentation-unit.schema.json` | addressable doc unit + lifecycle |
| DocumentationBrief | `schemas/documentation-brief.schema.json` | assisted outline, no invented claims |
| DocumentationSurface | `schemas/documentation-surface.schema.json` | projection target + visibility |
| DocumentationProjection | `schemas/documentation-projection.schema.json` | rendered artifact + fingerprint |
| DocumentationQualityPolicy | `schemas/documentation-quality-policy.schema.json` | dimensions + blocking rules |
| DocumentationEvidenceRequirement | `schemas/documentation-evidence-requirement.schema.json` | evidence bound to a requirement ID |
| DocumentationManifest | `schemas/documentation-manifest.schema.json` | derived index |
| MediaRecord | `schemas/media-record.schema.json` | visual evidence + freshness |
| AgentInstructionSurface | `schemas/agent-instruction-surface.schema.json` | compiled agent surface |

Also relevant: `documentation-delta`, `documentation-coverage`,
`documentation-readiness`, `documentation-contract`, `documentation-profile`.

## Generation tiers

- **GENERATED** — deterministic full rebuild; must be reproducible.
- **ASSISTED** — Prumo drafts; human/agent review is required before `current`.
- **CURATED** — protected human content the compiler may never overwrite.

A generated artifact never gains canonical authority merely by existing.

## Delta chain

```text
KnowledgeDelta
→ semantic impact graph
→ DocumentationPlan
→ DocumentationDelta
→ human projections
→ implementation projections
→ agent-surface projections
→ API/reference projections
→ TranslationDelta
→ MediaDelta
→ site / AI-retrieval projections
→ verification manifest
```

## Compiler invariants

1. Deterministic output for deterministic inputs.
2. No-op write when the content fingerprint is unchanged (CAS).
3. Generated-region ownership is explicit; CURATED regions are protected.
4. Projection provenance is always recoverable.
5. A canonical source may not depend on a projection that depends on it
   (projection cycles are errors).
6. Missing knowledge creates a gap, never invented documentation.

## Lifecycle

```text
proposed → planned → draft → review → current
                                  ↓ dependency changed
                              needs-update → stale → superseded → archived
```

Verification and gating live in W19 (`docs/development/waves.md`).
