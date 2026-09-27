# ADR 016 — Product Identity: `prumo` is the Framework CLI, `prumo agent` or `pa` Launches the Interactive Agent

Status: Accepted (2026-09-18)
Supersedes: the binary allocation in ADR 015.

## Context

ADR 015 attempted to rename the CLI to `prumo-agent`, which created friction with the product identity:
- The main command for the framework CLI should remain `prumo` — simple, memorable, canonical.
- To execute the interactive terminal agent (TUI), developers want either:
  1. `prumo agent` (part of the unified `prumo` command suite), or
  2. `pa` (short, instant alias directly in the terminal, like `gh` for GitHub CLI).
- A subordinate `prumo-agent-tui` name was awkward and confusing.

## Decision

1. **`prumo` is the Canonical CLI for the Framework and Harness.**
   - It is built from `cmd/prumo` (installed to `~/.local/bin/prumo`).
   - It exposes the entire suite of engineering framework, lifecycle, and harness commands:
     `prumo init`, `prumo validate`, `prumo doctor`, `prumo goal`, `prumo compile`, `prumo docs`, `prumo repo`, `prumo tool`, `prumo agent <subcommands>`.
   - Running `prumo agent` (without subcommands or with client flags) launches the interactive coding agent interface.
   - `prumo tui` remains supported as an alias for `prumo agent`.

2. **`pa` (and `prumo-agent`) is the Dedicated Interactive Agent Binary.**
   - It is built from `prumo-tui/cmd/prumo-tui` (installed to `~/.local/bin/prumo-agent` with symlink `~/.local/bin/pa`).
   - Running `pa` or `prumo-agent` directly launches the pair-programming TUI.
   - Supports `--prompt`, `--provider`, `--model`, `--theme`, interactive `/slash` commands, provider configuration modal, and collapsible sidebar.

3. **Symmetrical Discovery:**
   - When `prumo agent` or `prumo tui` is invoked, `prumo` finds `pa` or `prumo-agent` (sibling executable, on `PATH`, or via `PRUMO_AGENT_BIN` / `PRUMO_TUI_BIN`) and delegates execution to it.
   - When `pa` / `prumo-agent` is invoked directly, it connects to an existing daemon or starts `prumo agent serve` using `prumo` (sibling executable, on `PATH`, or via `PRUMO_BIN`).

## Ergonomics Summary

| Goal | Command |
|---|---|
| Launch Interactive Coding Agent | `prumo agent` or `pa` |
| Direct headless run | `pa --prompt "fix bug"` or `prumo agent --prompt "fix bug"` |
| Framework Lifecycle & Governance | `prumo <init\|doctor\|validate\|goal\|compile\|docs\|repo>` |
| Daemon Management | `prumo agent serve`, `prumo agent ps`, `prumo agent logs` |
