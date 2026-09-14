# Prumo Harness — Overview

Status: canonical. Target architecture ACCEPTED (snapshot `2026-09-11-d41f18bb65f5`,
pages 03–08, 18–19, 22–26, 31–35, 40–41); implementation status is tracked
explicitly per subsystem in [roadmap-status](roadmap-status.md).

## What the Harness is

A headless, Prumo-native agent engine in Go. Clients (CLI/TUI/Desktop/ACP)
consume a versioned protocol/event stream; they never own Run state.

```text
                PRUMO HARNESS

          Canonical Prumo State
                  │
       ┌──────────┴──────────┐
       │                     │
 Native AgentRuntime    External AgentProvider
       │                     │
 ModelProvider          ACP / SDK / RPC /
 Model Gateway          structured server/CLI
       │                     │
 LLMs                  Codex / OpenCode /
                       Goose / Claude Code / etc.
```

## Non-goals of this Goal

Full Prumo Code Desktop/TUI, remote distributed execution, microVM
infrastructure, complex A2A federation, public docs website, visual polish.
Priority: HEADLESS HARNESS FIRST.

## Map

- [agent-runtime](agent-runtime.md) — HA0 contracts + HA2 state machine.
- [providers](providers.md) — ModelProvider vs AgentProvider, Gateway.
- [security](security.md) — permissions, sandbox, egress.
- [context-knowledge](context-knowledge.md) — Context v2, Knowledge, doc compiler.
- [workforce-handoff](workforce-handoff.md) — Handoff v2, multi-agent.
- [daemon](daemon.md) — local Unix-socket server, reconnect baseline.
- [human-docs](human-docs.md) — HD0–HD4 generation runtime.
- [gap-register](gap-register.md) — registro canônico do que falta (atualizado por rodada).- [roadmap-status](roadmap-status.md) — HA0–HA11 + split gate + truth table.
- [promotion-report](promotion-report.md) — DocumentationMigrationReport.

## Invariants

- `ModelProvider != AgentProvider`. Codex/OpenCode/etc. are runtimes, not models.
- No fork of Goose/OpenCode/ADK/etc. as kernel; they are references/adapters.
- Core language is Go; Python v0.3 is a read-only oracle.
- No vendor types escape adapters into domain packages.
- `prumo-code` repo is NOT created until the split gate passes.
- Harness is headless: no desktop/Bubble Tea/Floem assumptions.
