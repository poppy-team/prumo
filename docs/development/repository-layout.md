# Prumo Canonical Repository Layout

> Directories only exist when they have real content. No empty packages for diagrammatic completeness.

## Root Structure

```text
prumo/
├── cmd/
│   └── prumo/
│       ├── main.go
│       └── main_test.go
├── internal/
│   ├── app/
│   ├── protocol/
│   │   ├── goals/
│   │   ├── plans/
│   │   ├── tasks/
│   │   ├── events/
│   │   ├── evidence/
│   │   └── gates/
│   ├── project/
│   ├── resolver/
│   ├── compiler/
│   ├── validator/
│   ├── documentation/
│   │   ├── contracts/
│   │   ├── profiles/
│   │   ├── readiness/
│   │   └── delta/
│   ├── planning/
│   │   ├── interview/
│   │   └── decisions/
│   ├── adoption/
│   ├── knowledge/
│   ├── experience/
│   ├── evidence-gates/
│   ├── control-plane/
│   │   ├── run/
│   │   ├── budget/
│   │   ├── context/
│   │   ├── model/
│   │   ├── tool/
│   │   ├── environment/
│   │   ├── automation/
│   │   ├── observability/
│   │   └── events/
│   ├── install/
│   └── storage/
│       ├── git/
│       └── sqlite/
├── integrations/
│   ├── opencode/
│   ├── codex/
│   ├── claude-code/
│   └── generic/
├── schemas/
├── resources/
│   ├── documentation-contracts/
│   ├── documentation-profiles/
│   ├── agents/
│   ├── skills/
│   └── recipes/
├── conformance/
│   ├── fixtures/
│   ├── golden/
│   ├── V03_BASELINE.json
│   └── README.md
├── testdata/
├── docs/
│   ├── product/
│   ├── architecture/
│   ├── development/
│   ├── migration/
│   ├── runtime/
│   ├── security/
│   └── reference/
├── scripts/
├── go.mod
├── go.sum
├── README.md
├── CHANGELOG.md
├── AGENTS.md
├── ENTRYPOINT.md
├── FRAMEWORK.md
└── docs/PRUMO.md
```

## Directory Responsibilities

| Path | Purpose | Embedded? |
|------|---------|-----------|
| `cmd/prumo/` | CLI entrypoint — thin adapter to `internal/app` | No |
| `internal/app/` | Application services (use cases, orchestration) | No |
| `internal/protocol/` | Stable domain types & invariants (Goals, Plans, Tasks, Events, Evidence, Gates) | No |
| `internal/project/` | Project discovery, `prumo.json`, profiles, capabilities | No |
| `internal/resolver/` | Workforce, risk, recipe, model policy, context selection | No |
| `internal/compiler/` | Target adapter interface + generators | No |
| `internal/validator/` | JSON Schema validation (Draft 2020-12) | No |
| `internal/documentation/` | Contracts, profiles, readiness, delta, contradictions | No |
| `internal/planning/` | Interview engine, decision extraction, open questions | No |
| `internal/adoption/` | Scanner, semantic mapping, confidence ledger, migration | No |
| `internal/knowledge/` | Typed traceability graph, retrieval, contradictions | No |
| `internal/experience/` | Session events, summaries, handoffs, proposals | No |
| `internal/evidence-gates/` | Deterministic validation, risk-based evidence, release gates | No |
| `internal/control-plane/` | Run, Budget, Context Compiler, Model Router, Tool Gateway, Env, Automation, Observability | No |
| `internal/install/` | Install/uninstall/setup, manifests, cleanup | No |
| `internal/storage/git/` | Git filesystem operations (Repository port) | No |
| `internal/storage/sqlite/` | Derived index (DerivedIndex port) | No |
| `integrations/` | Harness adapters (thin, generated templates + host code) | No |
| `schemas/` | **Canonical** JSON Schemas (source of truth) | No (git) |
| `resources/` | **Canonical** agent/skill/recipe catalogs, doc contracts, profiles | Yes (`go:embed`) |
| `conformance/` | Fixtures, golden outputs, baseline metadata | No (git) |
| `testdata/` | Complex fixtures, synthetic repos, integration test inputs | No (git) |
| `docs/` | Canonical local documentation (this structure) | No (git) |
| `scripts/` | One-off maintenance scripts (not runtime deps) | No |

## `go:embed` Strategy

- `resources/` embedded at build time for single-binary distribution
- `schemas/` NOT embedded (must remain editable git files); accessed via `internal/resources` FS abstraction
- `internal/resources` provides `OpenSchema()`, `OpenResource()` using `io/fs.FS` — defaults to OS filesystem, swappable to embedded FS in release builds

## Package Rules

1. **Start consolidated** — split only when responsibility justifies
2. **No package per file/type** — `internal/protocol/goals/` exists because Goals have multiple files; `internal/project/` stays flat
3. **No circular dependencies** — enforced by `go mod graph` check in CI
4. **No `utils`, `common`, `helpers`** — every package has domain semantics
5. **`internal` protects API** — nothing in `internal` is public SDK
6. **Interfaces at consumer boundary** — defined where multiple implementations exist

## Canonical vs Derived

| Canonical (Git) | Derived (Reconstructible) |
|-----------------|---------------------------|
| `schemas/` | SQLite index (`internal/storage/sqlite/`) |
| `resources/` | Search index, embeddings |
| `docs/` | HTML site, PDF exports |
| `conformance/fixtures/` | Golden outputs (stored for regression) |
| `testdata/` | Test databases, temp repos |
| Source code | Compiled binary, `go.sum` |

## OPEN QUESTIONS

- [ ] Whether `internal/evidence-gates` merges into `internal/quality` or stays separate
- [ ] Exact split of `internal/control-plane` sub-packages (flat vs per-service)
- [ ] Whether `integrations/opencode` contains TypeScript plugin source or only Go-generated templates