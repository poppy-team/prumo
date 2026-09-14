# Prumo — Documentation Router

PRUMO is an intent router. Read only the document needed for the current task; do not load the full repository documentation tree.

## Current implementation line

- Go v0.5: active Core and CLI.
- Python v0.3: retired compatibility oracle (canonical resources preserved under `src/prumo/`).
- Canonical formats: Markdown, JSON, JSON Schema, Git.
- Derived state: caches, indexes, runtime context, generated adapters.

## User manuals

- [Installation](manual/installation.md)
- [Uninstallation](manual/uninstallation.md)
- [Usage and command reference](manual/usage.md)
- [First project walkthrough](getting-started/first-project.md)
- [Core concepts](getting-started/concepts.md)
- [Troubleshooting](user-guide/troubleshooting.md)

## Product and architecture

- [Product vision](product/vision.md)
- [v0.4 scope](product/scope-v0.4.md)
- [Architecture overview](architecture/overview.md)
- [Dependency rules](architecture/dependency-rules.md)
- [Repository layout](development/repository-layout.md)
- [Runtime Control Plane](runtime/control-plane.md)
- [Trust model](security/trust-model.md)

## Harness (headless, Prumo-native)

- [Harness overview](../harness/overview.md)
- [Agent runtime](../harness/agent-runtime.md)
- [Providers + Gateway](../harness/providers.md)
- [Security](../harness/security.md)
- [Context + Knowledge](../harness/context-knowledge.md)
- [Workforce + Handoff](../harness/workforce-handoff.md)
- [Daemon](../harness/daemon.md)
- [Human docs](../harness/human-docs.md)
- [Roadmap status](../harness/roadmap-status.md)
- [Promotion report](../harness/promotion-report.md)

## Development

- [Implementation blueprint](development/implementation-blueprint.md)
- [Implementation phases](development/phases.md)
- [Coding standards](development/coding-standards.md)
- [Testing strategy](development/testing-strategy.md)
- [Contributing](development/contributing.md)
- [Migration protocol inventory](migration/protocol-inventory.md)
- [Conformance strategy](migration/conformance-strategy.md)
- [Migration status](migration/v0.3-to-v0.4-go.md)

## Integrations and authoring

- [Codex](integration/codex.md)
- [Claude Code](integration/claude-code.md)
- [OpenCode](integration/opencode.md)
- [Gemini](integration/gemini.md)
- [Generic harness](integration/generic.md)
- [Writing Skills](authoring/writing-skills.md)
- [Writing Agents](authoring/writing-agents.md)
- [Writing Recipes](authoring/writing-recipes.md)
- [Platform adapters](authoring/adapters.md)

## Documentation System v2

- [Documentation Contracts](contracts/builtin.json)
- [Documentation Profiles](profiles/builtin.json)
- `prumo docs contracts`
- `prumo docs profiles`
- `prumo docs audit`
- `prumo docs readiness`

M5 documentation analysis is deterministic and read-only. Missing knowledge is reported; Prumo does not create empty documents or silently promote model prose to canonical state.

## Protocol reference

- [CLI and machine interface](manual/usage.md#command-reference)
- [Schemas](reference/schemas.md)
- [Events](reference/events.md)
- [Project layout](reference/project-layout.md)
- [Workforce resolution](reference/workforce-resolution.md)
- [Installation contract](reference/installation-contract.md)
- [Goal system](GOAL_SYSTEM.md)
- [Event protocol](EVENT_PROTOCOL.md)
- [Quality gates](QUALITY.md)

## Canonical machine contracts

- Schemas: `schemas/`
- Workforce registry: `src/prumo/resources/catalog/catalog.json`
- Workforce packages: `src/prumo/resources/workforce/`
- Conformance fixtures: `conformance/`
- Source provenance: `SOURCE_MAP.json`

## Agent routing

1. Read `ENTRYPOINT.md`.
2. Read `AGENTS.md`.
3. Identify the active Goal and acceptance criteria.
4. Load the smallest sufficient documentation section.
5. Expand only when evidence is insufficient.
