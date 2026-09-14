# Human Documentation Runtime (HD0–HD4)

Package `internal/harness/humandocs`. Human docs are a projection of
capabilities/contracts/audiences — never hallucinated prose.

## Stages

- **HD0 Spec + profile composition.** `Spec` binds audience, profiles,
  per-intent policies (`GENERATED/ASSISTED/CURATED`) and source pointers.
  `ComposeProfile` merges base + overlays deterministically (union minus
  excluded, sorted). README/reference default GENERATED; tutorial/how-to/
  concepts/manual default ASSISTED; CURATED is never defaulted.
- **HD1 Units + Page Graph + Planner.** `Planner` derives one unit per
  profile contract × intent from the `docengine` registry. Contracts without
  sources become `missing` gaps — reported, never written. `Graph.Order`
  topologically sorts navigation; cycles are errors.
- **HD2 README + Reference + tree.** `GenerateTree` rebuilds `README.md`,
  `docs/reference.md` (contract tables, automation-first) and `docs/INDEX.md`
  (unit statuses) through `doccompile` CAS/no-op atomic writes. Missing units
  get index rows, never files.
- **HD3 Assisted briefs.** `Brief` outlines CURATED work: sections from
  required knowledge, blocking questions as `TODO(CURATED)`, source pointers,
  evidence requirements. It contains no claims beyond its inputs.
- **HD4 Coverage + Readiness.** `Coverage` splits ready/missing;
  `Readiness` blocks on missing REQUIRED units and structural gaps
  (unknown profiles/contracts). Optional-unit gaps are warnings.

## Out of scope (HD5+)

Example/screenshot/diagram lifecycle, Starlight renderer, i18n/versioning,
changelog/release automation, devlog projection, static publishing — they
consume this runtime's units and never block the Harness.
