# Changelog

## Unreleased

- **opencode is a model provider** (`--provider opencode`): the turn is delegated to the opencode CLI, which runs it with its own tools and its own authentication — including the models it serves for free — so a real agent runs with no api-key of ours. The delegation is declared (`ToolCalls: false`), never papered over. The wire format was read from the installed binary, not guessed (GAP-093).
- **First-run offer**: the client checks for opencode once and, only when nothing is configured, offers to install it (automatic, via npm), configure an api-key (variables printed, no secret written), or wait (GAP-094).
- **Sandbox fix found by installing podman**: the availability probe asked every runtime a Docker question, so a working podman was reported unreachable and the sandbox disappeared from a host that could still run containers. Each runtime is asked in its own language and detection walks the ladder until one answers (GAP-092). Rootless podman is measured: `TestContainerLive` passes in 3.66 s.

- **Command renamed to `prumo-agent`** (ADR 015): the framework, the protocol, `.prumo/`, `prumo.json`, `PRUMO_*` and the module path stay **Prumo**; the binary a person types became `prumo-agent`, and the terminal client is reached as `prumo-agent tui` (executable `prumo-agent-tui`). 75 documents and 11 source files touched. Documents written before the rename that say `prumo <command>` mean the same binary.
- **Model capabilities reach the client**: the `models` operation reports what each model declares it can do (`model_info`), read from the workspace's `.prumo/models.json` (new schema `model-declarations.schema.json`). The TUI picker shows `[text reasoning vision tools audio]` and says "not declared" when nobody said — a declaration is never invented, and an absent feature is the absence of a claim, not a denial. This is the gate the image-attachment work needs.
- **ADR 007 (SQLite driver) and ADR 009 (performance budgets) accepted**; podman rootless measurement and Local Intel remain open (`docs/harness/open-work.md`).

## 0.6.0 — Headless Harness

- **Prumo Harness (headless, Go-native):** NativeAgent reentrant state machine, `ModelProvider`/`AgentProvider` contracts, FakeProvider determinístico + conformance suite, adapters OpenAI-compat e Anthropic, gateway com routing/fallback/retry/quota, Coding ACI completa (`fs/code/edit/git/test/process`), Permission Engine determinística, checkpoints crash-safe com journal idempotente, Handoff v2 tipado, Context Compiler v2 (gates, BM25, repo-map, LSP, Memory Atlas), Knowledge Runtime (records, Delta, contradiction/coverage/readiness, seeding, research ledger), Documentation Compiler + Human Docs Runtime HD0–HD4, workforce multi-agent com review, daemon local (Unix socket + TLS remoto) com scheduler, ACP v1 agent server, MCP stdio/HTTP, SDK Go público + IDL versionada + types TypeScript.
- **CLI:** `prumo agent <run|resume|handoff|events|protocol|serve|ps|logs|steer|stop|schedule|unschedule|jobs|promote|acp|gc|providers>`; flags de budget (`--budget-tokens/usd/tools`), `--strict`, `--gates`, sandbox (`--sandbox/--sandbox-image/--sandbox-runtime`), egress (`--egress-deny/--allow`), contexto (`--context-budget/--level`), compactação e MCP (`--mcp`).
- **Docs:** `docs/harness/` canônica (overview, runtime, providers, security, context-knowledge, workforce, daemon, human-docs, roadmap, promotion-report, gap-register) reconciliada com o caderno Prumo Code Agent (snapshot `2026-09-11-d41f18bb65f5`).
- **Evidência:** `go vet` + `go test ./...` verdes; interop live contra `opencode serve` 1.18.30; SIGKILL recovery; fuzz 1.5M execs sem falhas; relatório em `HARNESS_IMPLEMENTATION_REPORT.md`.
- **Limites honestos:** sends live que gastam quota, chaves de models, daemon Docker/runsc e modelos locais seguem pendentes de ambiente/aprovação (ver gap-register). `raillen/prumo-code` NÃO criado (split gate NOT READY).

## 0.5.0 — Prumo rebrand

- **New identity:** Project Atlas Framework is now **Prumo** — a Git-native project protocol and CLI for software built by humans and AI agents. No Atlas compatibility layer is kept (pre-v1 clean break).
- **CLI:** `atlas` → `prumo` (`cmd/prumo`, binary `prumo`/`prumo.exe`).
- **Project format:** `atlas.json` → `prumo.json`, `.atlas/` → `.prumo/`, `docs/ATLAS.md` → `docs/PRUMO.md`, `ATLAS_HOME` → `PRUMO_HOME`.
- **Module:** `github.com/raillen/project-atlas-framework` → `github.com/raillen/prumo`.
- **Schemas:** `schemas/atlas*.schema.json` → `schemas/prumo*.schema.json` with `$id` under `https://raw.githubusercontent.com/raillen/prumo/main/schemas/`; envelope codes `ATLAS_*` → `PRUMO_*`; `framework.name` const is now `prumo`.
- **Workforce:** `atlas-navigation` skill → `prumo-navigation`; connector ownership markers are `.prumo-generated.json` with `prumo_version`.
- **Tooling:** install/uninstall/release scripts, CI and docs target `raillen/prumo` and the `prumo` binary.
- See ADR 004 for the full decision record.

## 0.3.0 — Execution-Ready Protocol

- **Strict Machine Contracts:** 24 JSON schemas governing Goals v2, Plans, Tasks, Evidence, Gates, Events, Policies (Permissions, Approvals, Trust, Models, Execution), and Context Packs.
- **Goal System v2:** Cryptographic SHA256 locking, mutation prevention, formal Goal amendments with audit trails.
- **Standardized Workforce Packages:** Modular canonical directory structure for 98 Skills (`manifest.json` + `SKILL.md`), 26 Agents (`manifest.json` + `AGENT.md`), and 14 Recipes (`recipe.json` + `RECIPE.md`).
- **Explainable Workforce Resolution:** Multi-pass deterministic workforce resolver emitting explainability traces and reasons.
- **Platform Adapter Compiler v2:** Skill package compiler injecting complete skill bundles into Codex (`AGENTS.md`, `.codex/skills/`) and Claude Code (`CLAUDE.md`, `.claude/skills/`).
- **Event Protocol & Trust Model:** Structured append-only event stream specification and 3-tier trust hierarchy (trusted policy > contextual data > untrusted data).
- **Diagnostics & Explainability CLI:** `prumo doctor` with comprehensive DAG cycle and lock checks, `prumo explain` covering workforce/agents/skills/recipes/context/models, and `prumo migrate` with automated backup snapshots.
- **Fake Runtime & Conformance Suite:** Deterministic provider-neutral execution simulator and golden test vectors in `conformance/`.
- **Expanded Documentation & Real Examples:** Comprehensive guides across getting-started, user-guide, authoring, integration, reference, and development; real working example projects for `rust-cli`, `react-saas`, `rust-desktop`, `game-engine`, `prumo-flow`, and `conformance-project`.

## 0.2.0 — Lean Progressive Context evolution

- Adopt Lean Progressive Context (LPC) / Progressive Context Architecture (PCA).
- Add explicit input/output token economy, rolling Working Context Capsules, pointer-over-payload and context garbage collection rules.
- Make user, developer, operations and agent documentation first-class framework surfaces.
- Define documentation-site publishing as a derived view of canonical Markdown.
- Add Project Intelligence contract for task/project costs, token usage, effort, quality and debt.
- Reduce maintained project formats to Markdown + JSON; SQLite is runtime/derived state only.
- Migrate framework catalogs and generated project contracts from YAML to JSON.
- Keep YAML as legacy read compatibility for v0.1 migration only.
- Replace persistent generic context packs with runtime/generated context artifacts.
- Add semantic/structural fragmentation guidance instead of physical microfile proliferation.
- Keep deep recursive LLM execution experimental and disabled by default.
- Add migration guidance from v0.1.

## 0.1.0 — Initial implementation

- Git-native framework contract and recovery entrypoint.
- Project profile and workforce capability resolver.
- Agent, Skill, Recipe and project-type Bundle registries.
- Project Orchestration Protocol with per-project LLM selection and fallbacks.
- Goal state machine, acceptance/gate/evidence schema and CLI operations.
- Adapters for generic chat, ChatGPT, Claude, Kimi, Codex, Claude Code and Traycer.
- Platform adapter compiler.
- Project bootstrap, validation, doctor and portable snapshots.
- Initial test suite and GitHub Actions CI.
