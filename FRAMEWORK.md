# Framework Contract

## Purpose

Prumo is a versioned, Git-native engineering protocol for durable, portable and cost-aware AI-assisted development. Prompts, skills, agents, platform adapters and orchestrator configurations are distributions of the protocol, not the protocol itself.

The framework serves four audiences at once: product users, developers/contributors, operators/maintainers and AI agents. It also treats documentation publishing, project intelligence and LLM context efficiency as first-class engineering concerns.

## Core invariants

- **Git is durable memory.** Stable project truth must survive loss of chat history, provider state and local caches.
- **Canonical before generated.** Human knowledge is maintained primarily in Markdown; machine configuration is maintained in JSON. Generated indexes, summaries, caches, dashboards and adapters never replace canonical sources.
- **Few persistent formats.** New Prumo projects maintain Markdown and JSON. SQLite is permitted only for derived/runtime state and is not canonical. YAML is legacy read compatibility only.
- **Project decisions outrank framework defaults.** An explicit project ADR/RFC is authoritative for that project.
- **Goals define completion.** Roadmaps describe direction; locked Goals define measurable done conditions.
- **Acceptance is protected.** Agents may propose amendments but cannot silently weaken locked criteria.
- **Evidence over assertion.** Builds, tests, benchmarks, screenshots, logs, reviews and recorded gates determine completion.
- **Roles are abstract.** Agents are roles; models are routed to roles by project policy.
- **Least workforce.** Load only the agents, skills and recipes needed for the task.
- **Least context.** Under Lean Progressive Context, begin with the smallest sufficient context and expand only when evidence requires it.
- **Pointer over payload.** Prefer stable document/symbol/test references to repeatedly injecting large bodies of text.
- **Context is ephemeral by default.** Working Context Capsules, generated summaries, retrieval traces and derived indexes are runtime state. Persist only durable knowledge and compact project history.
- **Input and output are budgeted.** Token economy covers prompt/context tokens and generated/intermediate output.
- **No unbounded recursion.** Deep recursive LLM execution is not a core requirement. Single-level isolated delegation is the default maximum unless a project explicitly enables and benchmarks an experiment.
- **Documentation follows the audience.** User, developer, operations and agent documentation are explicit surfaces, not afterthoughts.
- **Documentation site is derived.** A public/internal documentation site may be generated continuously from canonical project docs; the site is never a second source of truth.
- **Project Intelligence is durable and honest.** Costs, token usage, effort, quality and debt distinguish observed measurements from estimates and include confidence.
- **Orchestrators are replaceable.** No orchestration product becomes the source of truth.
- **Model preferences are per-project.** Bootstrap explicitly obtains the preferred model/provider roster.
- **Portability first.** A generic Markdown entrypoint remains sufficient for unknown platforms.

## Documentation & knowledge invariants

Added by W14 (architecture: `docs/adr/010-documentation-control-plane.md`):

- **Canonical knowledge precedes prose.** Documentation is a governed projection of typed canonical knowledge, never an independent source of truth.
- **Missing knowledge creates a gap.** Documentation is never invented to satisfy coverage.
- **Stable identity.** Every durable semantic entity keeps an ID independent of file path, title or URL.
- **Explicit authority.** Every artifact is canonical, projection or historical, and machine-resolvable (`docs/AUTHORITY_MAP.json`).
- **A projection cannot silently become canonical.** Generated surfaces are replaceable and never authoritative by existence.
- **Readiness is semantic evidence coverage, not keyword presence.** Requirement → claim → evidence, with revision and freshness.
- **Every generated projection is explainable and freshness-checkable.** Provenance is always recoverable.
- **Documentation impact is computed before and after behavior-changing work.** Silent N/A is forbidden; explicit N/A is valid.
- **A Goal is not complete while its required documentation projections are stale.**
- **Agent instruction files are small, scoped maps — not parallel truth.** They are compiled projections with budgets and freshness checks.
- **Localization preserves one canonical implementation truth.** Translations are projections of an English canonical source.
- **Accessibility requires evidence appropriate to the surface.** Automated checks are not always sufficient.
- **Media has provenance and freshness like code-derived artifacts.**
- **Design systems and public schemas are versioned APIs.** Token renames are breaking changes.
- **Site generators and vendor agent formats are replaceable adapters.**
- **Documentation cost and quality are measurable Project Intelligence.**
- **Deterministic quality failures cannot be overridden by model self-evaluation.** Deterministic checks run first.
- **Human, agent, site, locale and API views may differ in presentation but must resolve to the same canonical claims.**
- **The harness optimizes for current, sufficient, verifiable knowledge — not maximum prose volume.**

## Authority order

1. Explicit current user/project decision.
2. Accepted project ADR/RFC and locked Goal.
3. Canonical project documentation.
4. `prumo.json` and other project JSON contracts.
5. Prumo protocol.
6. Generated platform adapter.
7. Generic model/tool defaults.

## Persistence classes

| Class | Purpose | Persistence |
|---|---|---|
| Canonical Knowledge | Architecture, product, user/dev/ops docs, specs, ADRs | Git |
| Project History | Compact Goals, cost/task intelligence, durable evidence pointers | Git |
| Working Context | WCC, retrieval evidence, intermediate findings, generated summaries | Runtime/SQLite; garbage-collected |

See `docs/LEAN_PROGRESSIVE_CONTEXT.md`, `docs/DOCUMENTATION_SYSTEM.md` and `docs/PROJECT_INTELLIGENCE.md`.
