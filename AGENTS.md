# AGENTS.md

This repository develops Prumo itself.

<!-- prumo:begin agents-core -->
<!-- prumo:generated adapter=agents-md scope=root fingerprint=4659acf1db2af284 sources=ENTRYPOINT.md,FRAMEWORK.md,docs/AI_WORKFORCE.md,docs/DOCUMENTATION_SYSTEM.md,docs/LEAN_PROGRESSIVE_CONTEXT.md,docs/PRUMO.md,docs/architecture/dependency-rules.md,docs/architecture/documentation-compiler.md,docs/architecture/documentation-control-plane.md,docs/development/testing-strategy.md,schemas -->
- Start with `ENTRYPOINT.md`, `FRAMEWORK.md`, and `docs/PRUMO.md`.
- Preserve provider-neutral core policy.
- Follow Lean Progressive Context: smallest sufficient context, progressive expansion, bounded output.
- Treat `src/prumo/resources/catalog/catalog.json` as the canonical workforce registry.
- Platform-specific behavior belongs in adapters, not core policy.
- New human-maintained project artifacts use Markdown or JSON only.
- YAML support is legacy read compatibility; do not generate new YAML artifacts.
- Derived indexes, summaries and Working Context Capsules belong in runtime/cache and must not become canonical files.
- Add/adjust JSON Schema whenever machine contracts change.
- Add tests for resolver, Goals, compiler, validation, scaffolding and migration behavior.
- Keep `ENTRYPOINT.md` and generated adapters short.
- Do not encode one project's preferred model roster as a framework default.
- Do not make any orchestrator/model provider mandatory for the core framework.
- New context mechanisms must report token/cost impact and have a stopping condition.
- Deep recursive execution remains experimental and disabled by default.
<!-- prumo:end agents-core -->

# Prumo Development Authority

## Knowledge authority

1. Canonical repository specs, schemas and ADRs
2. Accepted local engineering documentation
3. Prumo Living Book in Notion
4. Agent inference
5. External sources

The Prumo Living Book is available through the Notion MCP.

Living Book:
https://app.notion.com/p/raillen/Project-Prumo-Framework-Livro-Vivo-v0-4-3d59bb7d023f8168a69ac57326eddc85

Use Notion on demand when:
- a local specification is missing;
- an architectural decision needs clarification;
- planning a new implementation phase;
- checking approved design (ADR / Living Book);
- resolving ambiguity.

Do not load the entire Living Book into context.

Retrieve only pages relevant to the current Goal.

Once a design decision is promoted into a canonical repository specification,
the repository version has authority over the Notion version for implementation.

# Prumo Development Context

## Canonical Local Documentation

All authoritative engineering specifications live under `docs/` in this repository:

- **Product**: `docs/product/vision.md`, `docs/product/scope-v0.4.md`
- **Architecture**: `docs/architecture/overview.md`, `docs/architecture/dependency-rules.md`
- **Development**: `docs/development/implementation-blueprint.md`, `docs/development/phases.md`, `docs/development/repository-layout.md`, `docs/development/coding-standards.md`, `docs/development/testing-strategy.md`
- **Migration**: `docs/migration/protocol-inventory.md`, `docs/migration/conformance-strategy.md`
- **Runtime**: `docs/runtime/control-plane.md`, `docs/runtime/living-plan.md`
- **Security**: `docs/security/trust-model.md`
- **Source Map**: `docs/SOURCE_MAP.json` — maps each local doc to its Notion Living Book page(s)

## Notion Living Book (On-Demand Retrieval)

The Living Book is available via Notion MCP:
https://app.notion.com/p/raillen/Project-Prumo-Framework-Livro-Vivo-v0-4-3d59bb7d023f8168a69ac57326eddc85

Agents must:
1. **Retrieve only pages relevant to the current Goal** — never load the entire book.
2. **Consult on demand** when: local spec is missing; architectural decision needs clarification; planning a new phase; checking approved design; resolving ambiguity.
3. **Respect authority order**: canonical repository → approved Notion design → agent inference.

## Implementation Discipline

- Implementation begins only when the Goal has sufficient local documentation and acceptance criteria.
- Clean Code pragmático applies permanently: responsabilidades explícitas, baixo acoplamento, alta coesão, nomes de domínio, funções pequenas, composição, dependências apontando para dentro, testes determinísticos, erros explícitos, sem abstração sem necessidade concreta.
- No silent changes to locked/canonical decisions (ADRs, schemas, documented policies). If a change is needed, update the canonical document and ADR first.

## Documentation authority

Authority order, roles (canonical/projection/historical) and drift rules are
canonical in `docs/governance/authority.md`. The machine-readable classification
lives in `docs/AUTHORITY_MAP.json` and is verified by `prumo-agent docs authority`
(CI gate `docs-authority`). Agent/provider adapters are projections and may not
introduce independent project facts.

## Repository governance

Repository governance is canonical in `docs/governance/repository-governance.md`.

Before remote Git/SCM mutations:
- read the effective repository policy in `.prumo/repository/policy.json`;
- do not push directly to `main`;
- do not force push protected branches;
- use Pull Requests;
- respect risk, review, and merge rules.

## Routing Quick Reference

| Need | Start Here |
|------|------------|
| What is Prumo building now? | `docs/product/vision.md` |
| What is the implementation plan? | `docs/development/waves.md` |
| What is still missing? | `docs/harness/gap-register.md` |
| Who owns a doc; is it canonical? | `docs/governance/authority.md`, `docs/AUTHORITY_MAP.json` |
| What is in/out of scope? | `docs/product/scope-v0.4.md` |
| How do packages depend? | `docs/architecture/dependency-rules.md` |
| What are the macro phases? | `docs/development/phases.md` |
| How is the repo structured? | `docs/development/repository-layout.md` |
| What coding style? | `docs/development/coding-standards.md` |
| How are we testing? | `docs/development/testing-strategy.md` |
| How does Python migrate to Go? | `docs/migration/protocol-inventory.md`, `docs/migration/conformance-strategy.md` |
| What is the Control Plane? | `docs/runtime/control-plane.md` |
| What is the trust model? | `docs/security/trust-model.md` |
| Where did a doc come from? | `docs/SOURCE_MAP.json` |
