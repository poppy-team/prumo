# CLI Reference

The active CLI is the Go binary built from `cmd/prumo`.

```bash
prumo <command> [options]
```

Use the [usage manual](../manual/usage.md) for workflows and examples. This page records the current command surface.

## Surfaces

| Surface | Command | Description |
|---|---|---|
| **Prumo** | `prumo <command>` | CLI harness, governance, compiler, validation and background daemon |
| **Prumo Code Agent** | `prumo code-agent` (or `prumo agent`, `prumo tui`) | Interactive terminal coding agent (TUI) with multi-tabs, sidebar telemetry and live theme preview |
| **Prumo IDE** | `prumo native` | High-performance graphical desktop environment (GUI) |

## Global options

| Option | Meaning |
|--------|---------|
| `--json` | Emit the machine-readable envelope on stdout |
| `--home <path>` | Use an isolated global Prumo home |

`prumo --help` provides top-level help. Unknown commands return exit code `2`.

## Version and project discovery

```bash
prumo version
prumo --json version
prumo status --path <project>
prumo --json status --path <project>
```

## Installation lifecycle

```bash
prumo setup
prumo --home <path> setup
prumo install connector <id>
prumo uninstall
prumo uninstall --connectors
prumo uninstall --purge-cache
prumo uninstall --purge-global-config
```

See [installation](../manual/installation.md) and [uninstallation](../manual/uninstallation.md).

## Project lifecycle

```bash
prumo init <path> --profile <profile.json> --non-interactive
prumo validate [path]
prumo doctor [path]
prumo framework-check
```

`init` requires `--profile` in non-interactive mode.

## Interactive Agent & Desktop IDE

```bash
# Launch interactive terminal TUI
prumo code-agent
prumo agent

# Launch desktop GUI
prumo native
```

## Goals

```bash
prumo goal new <id> <title> --phase <phase> [--objective <text>] [--path <path>]
prumo goal state <id> <state> [--reason <text>] [--path <path>]
prumo goal amend <id> [--file <path>] [--reason <text>] [--approved-by <actor>] [--path <path>]
prumo goal list [--path <path>]
```

States: `DRAFT`, `PLANNED`, `LOCKED`, `EXECUTING`, `VERIFYING`, `REVIEWING`, `BLOCKED`, `DONE`.

## Context and intelligence

```bash
prumo context plan <task> [--path <path>] [--json]
prumo report add <report.json> [--path <path>]
prumo report summary [--path <path>] [--json]
```

## Migration and snapshots

```bash
prumo migrate [path] [--dry-run] [--json]
prumo snapshot [path] [--output <archive.zip>]
```

## Documentation deltas & authority

```bash
prumo docs delta propose --goal <goal> [--path <project>] [changed ...] [--json]
prumo docs delta list [--path <project>] [--json]
prumo docs delta show --id <delta> [--path <project>] [--json]
prumo docs delta transition --id <delta> --state <state> [--evidence <id>]... [--path <project>] [--json]
prumo docs audit [--json]
prumo docs authority [--json]
prumo docs verify [--strict]
```

Delta states are `proposed`, `reviewed`, `accepted`, `rejected`, and `applied`. Applying a delta requires evidence and does not itself edit canonical documentation.

Migration should be previewed with `--dry-run`. Project data is not removed by uninstall.

## Resolution and explanation

```bash
prumo resolve <profile.json> [--json]
prumo explain workforce <profile.json> [--json]
prumo explain agent <id> [--json]
prumo explain skill <id> [--json]
prumo explain recipe <id> [--json]
prumo explain context <task-id> [--path <project>] [--json]
prumo explain model <role> [--path <project>] [--json]
prumo explain execution <profile> [--path <project>] [--json]
```

## Compiler targets

```bash
prumo compile --target generic [--path <project>] [--json]
prumo compile --target chatgpt [--path <project>] [--json]
prumo compile --target claude [--path <project>] [--json]
prumo compile --target kimi [--path <project>] [--json]
prumo compile --target codex [--path <project>] [--json]
prumo compile --target claude-code [--path <project>] [--json]
prumo compile --target traycer [--path <project>] [--json]
prumo compile --target antigravity [--path <project>] [--json]
```

## JSON envelope

Successful commands return:

```json
{
  "protocol_version": "1",
  "ok": true,
  "data": {},
  "diagnostics": [],
  "warnings": []
}
```

Exit codes:

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Validation or gate failure |
| `2` | Invalid command or arguments |
| `3` | Configuration error |
| `4` | Capability unavailable |
| `5` | Internal error |
| `6` | Project not found |
