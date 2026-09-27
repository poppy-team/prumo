# v0.3 → v0.4 Protocol Inventory

## Authority

- v0.3 Python runtime: `src/prumo/` — executable compatibility oracle
- v0.3 package metadata: `pyproject.toml`
- v0.3 canonical schemas: `schemas/*.schema.json`
- v0.3 bundled resources: `src/prumo/resources/`
- v0.3 fixtures: `conformance/`
- v0.3 examples: `examples/`
- v0.3 tests: `tests/`

No Python runtime, schema, resource, fixture, or example is removed or rewritten as part of the migration bootstrap.

## Current Python Components

| Module | Responsibility | Go Target |
|--------|----------------|-----------|
| `cli.py` | CLI parser, dispatch, exit codes, output | `cmd/prumo-agent` + `internal/app` |
| `compiler.py` | Target adapter generation | `internal/compiler` |
| `context.py` | Lean Progressive Context planning | `internal/control-plane/context` |
| `doctor.py` | Diagnostics, DAG checks, health checks | `internal/evidence-gates` + `internal/app` |
| `explain.py` | Structured explanations | `internal/app` / control-plane observability |
| `fake_runtime.py` | Deterministic execution simulation | conformance test support |
| `goals.py` | Goal lifecycle, lock, amendment, digest | `internal/protocol/goals` |
| `intelligence.py` | Task reports, derived project intelligence | `internal/knowledge` |
| `io.py` | JSON/YAML compatibility I/O | `internal/project` / storage edge |
| `migration.py` | v0.x project migration | `internal/install` / migration service |
| `model_policy.py` | Model policy resolution | `internal/resolver` / control-plane model |
| `profile.py` | Project profile loading | `internal/project` |
| `resolver.py` | Agents, skills, recipes resolution | `internal/resolver` |
| `scaffolder.py` | Project initialization | `internal/project` / `internal/app` |
| `snapshot.py` | Recovery snapshot | `internal/project` / storage |
| `validator.py` | JSON Schema and project validation | `internal/validator` |
| `workforce.py` | Workforce catalog and selection | `internal/resolver` |

## Public v0.3 CLI Surface

| Command | Python Behavior | Go Parity Target |
|---------|-----------------|------------------|
| `prumo-agent init [path]` | Initializes project and compiles profile | M2 |
| `prumo-agent resolve <profile>` | Resolves agents/skills/recipes | M2 |
| `prumo-agent validate [path]` | Validates project schemas | M2 |
| `prumo-agent goal new|state|amend|list` | Goal lifecycle | M2 |
| `prumo-agent context plan` | Context strategy/budget | M2 |
| `prumo-agent report add|summary` | Project intelligence | M2 |
| `prumo-agent migrate` | Legacy project migration | M2 |
| `prumo-agent compile --target` | Adapter generation | M3 |
| `prumo-agent snapshot` | Recovery archive | M2 |
| `prumo-agent doctor` | Health diagnostics | M2 |
| `prumo-agent explain` | Workforce/context/policy explanation | M2 |
| `prumo-agent framework-check` | Bundled catalog validation | M2 |

Compatibility aliases must remain where the v0.4 CLI introduces a more structured surface (notably `prumo-agent compile --target <id>`).

## Canonical Schemas

The repository has 28 root schemas in `schemas/`: agent, approval-policy, prumo, attempt, context-budget, context-item, context-pack, context-plan, context-request, event, evidence, execution-policy, gate, gate-waiver, goal, goal-amendment, model-policy, permission-policy, plan, project-manifest, project-profile, recipe, repository-policy, run, skill, task, task-report, trust-policy.

The v0.4 protocol must conform to these schemas unless an explicit versioned schema migration is approved. Cross-schema references and Draft 2020-12 behavior must be preserved.

## Canonical Resources

- Workforce registry: `src/prumo/resources/catalog/catalog.json`
- Catalog files: `agents.json`, `skills.json`, `recipes.json`, `bundles.json`, `risk_rules.json`
- Adapter templates: `src/prumo/resources/adapters/`
- Workforce packages: `src/prumo/resources/workforce/`
- Bundled schemas: `src/prumo/resources/schemas/`

## Root Scripts

- `fix_schemas.py`: **OPEN QUESTION** — classify as permanent maintainer tool, one-shot migration, or remove after review.
- `write_docs.py`: **OPEN QUESTION** — classify as permanent maintainer tool, one-shot migration, or remove after review.

Scripts are not runtime dependencies of the Go Core unless explicitly reclassified and ported.

## Baseline

- Python package version: `0.3.0`
- Python test suite: 127 tests passing at bootstrap
- Existing Go bootstrap: `go.mod`, `cmd/prumo-agent`, `internal/app`, `internal/project`, `internal/protocol`, `internal/resources`, `internal/validation`
- Baseline manifest: `conformance/V03_BASELINE.json`

## Critical Contract Inventory To Freeze

1. CLI arguments, defaults, parser errors, exit code policy
2. JSON field names, omission/null behavior, ordering where golden-tested
3. Human-readable output where compatibility requires it
4. Schema resolution and validation errors
5. Goal digest, lock, amendment, and transition behavior
6. Resolver ordering and reason/trace output
7. Generated file paths, contents, ownership, and overwrite rules
8. Snapshot contents and deterministic ordering
9. Migration dry-run, backup, and post-validation behavior
10. Doctor severity/categories and exit behavior

## OPEN QUESTIONS

- [ ] Which of the 27 schemas are “critical contracts” for the M2 gate?
- [ ] Exact JSON normalization rules for golden comparison
- [ ] Whether YAML read compatibility remains in Go M2 or is isolated to migration tooling
- [ ] Whether ancillary compiler targets remain supported after M3
- [ ] Precise Python version matrix for final oracle runs
- [ ] Ownership metadata format for generated artifacts before compiler parity