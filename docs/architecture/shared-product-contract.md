# Shared Product Contract

> Authority: repository-canonical. Promoted from Living Book page 23 (W8).
> Machine-readable: `docs/architecture/shared-product-contract.json`.

The Shared Product Contract is the interface every Prumo client (CLI, TUI,
desktop, daemon, SDKs, connectors) consumes. Clients **render** contract
entities; they never reimplement domain rules or own canonical state.

## Capabilities

Capability IDs are stable and usable by documentation coverage, UI contracts,
release notes, agent context and Project Intelligence:

`cap:cli`, `cap:projects`, `cap:goals`, `cap:agent-runtime`, `cap:context`,
`cap:knowledge`, `cap:documentation`, `cap:evidence-gates`, `cap:workforce`,
`cap:daemon`, `cap:connectors`, `cap:observability`, `cap:security`,
`cap:tui`, `cap:desktop`.

## Entities and state ownership

| Entity | State owner | Mutability |
|--------|-------------|------------|
| Goal | canonical repository (`.ai/goals`) | Goal lifecycle only |
| Plan | canonical repository | via Goal |
| Task | canonical repository | via Plan |
| Run | runtime store (derived) | runtime only |
| Checkpoint | runtime store (derived) | runtime only |
| Evidence | canonical repository | append-only |
| KnowledgeUnit | canonical knowledge | Delta-first |
| DocumentationUnit | canonical sources + projections | compiler + review |
| Approval | runtime journal | append-only |
| Budget | runtime | runtime only |
| Handoff | canonical artifact | explicit |
| Event | runtime stream (derived) | append-only |

## Invariants

1. Clients consume the contract; they never reimplement domain rules.
2. Canonical state is repository-owned; runtime state is derived and GC-able.
3. Every client projection resolves to the same entity IDs.
4. A capability may not be declared twice under different owners.

## Consumers

- **Documentation** (W15/W17): capabilities drive coverage and impact.
- **UI contracts** (W5/W11): surfaces reference capability IDs.
- **Release notes** (W20): capability changes produce release-note units.
- **Agent context** (W4/W16): capability knowledge is retrievable by ID.
- **Project Intelligence** (W21): metrics aggregate per capability.
