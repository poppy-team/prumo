# CLI Reference

The active CLI is the Go binary built from `cmd/prumo`.

```bash
prumo-agent <command> [options]
```

Use the [usage manual](../manual/usage.md) for workflows and examples. This page records the current command surface.

## Global options

| Option | Meaning |
|--------|---------|
| `--json` | Emit the machine-readable envelope on stdout |
| `--home <path>` | Use an isolated global Prumo home |

`prumo-agent --help` is not implemented yet. Unknown commands return exit code `2`.

## Version and project discovery

```bash
prumo-agent version
prumo-agent --json version
prumo-agent status --path <project>
prumo-agent --json status --path <project>
```

## Installation lifecycle

```bash
prumo-agent setup
prumo-agent --home <path> setup
prumo-agent install connector <id>
prumo-agent uninstall
prumo-agent uninstall --connectors
prumo-agent uninstall --purge-cache
prumo-agent uninstall --purge-global-config
```

See [installation](../manual/installation.md) and [uninstallation](../manual/uninstallation.md).

## Project lifecycle

```bash
prumo-agent init <path> --profile <profile.json> --non-interactive
prumo-agent validate [path]
prumo-agent doctor [path]
prumo-agent framework-check
```

`init` requires `--profile` in non-interactive mode.

## Goals

```bash
prumo-agent goal new <id> <title> --phase <phase> [--objective <text>] [--path <path>]
prumo-agent goal state <id> <state> [--reason <text>] [--path <path>]
prumo-agent goal amend <id> [--file <path>] [--reason <text>] [--approved-by <actor>] [--path <path>]
prumo-agent goal list [--path <path>]
```

States: `DRAFT`, `PLANNED`, `LOCKED`, `EXECUTING`, `VERIFYING`, `REVIEWING`, `BLOCKED`, `DONE`.

## Context and intelligence

```bash
prumo-agent context plan <task> [--path <path>] [--json]
prumo-agent report add <report.json> [--path <path>]
prumo-agent report summary [--path <path>] [--json]
```

## Migration and snapshots

```bash
prumo-agent migrate [path] [--dry-run] [--json]
prumo-agent snapshot [path] [--output <archive.zip>]
```

## Documentation deltas

```bash
prumo-agent docs delta propose --goal <goal> [--path <project>] [changed ...] [--json]
prumo-agent docs delta list [--path <project>] [--json]
prumo-agent docs delta show --id <delta> [--path <project>] [--json]
prumo-agent docs delta transition --id <delta> --state <state> [--evidence <id>]... [--path <project>] [--json]
```

Delta states are `proposed`, `reviewed`, `accepted`, `rejected`, and `applied`. Applying a delta requires evidence and does not itself edit canonical documentation.

Migration should be previewed with `--dry-run`. Project data is not removed by uninstall.

## Resolution and explanation

```bash
prumo-agent resolve <profile.json> [--json]
prumo-agent explain workforce <profile.json> [--json]
prumo-agent explain agent <id> [--json]
prumo-agent explain skill <id> [--json]
prumo-agent explain recipe <id> [--json]
prumo-agent explain context <task-id> [--path <project>] [--json]
prumo-agent explain model <role> [--path <project>] [--json]
prumo-agent explain execution <profile> [--path <project>] [--json]
```

## Compiler targets

```bash
prumo-agent compile --target generic [--path <project>] [--json]
prumo-agent compile --target chatgpt [--path <project>] [--json]
prumo-agent compile --target claude [--path <project>] [--json]
prumo-agent compile --target kimi [--path <project>] [--json]
prumo-agent compile --target codex [--path <project>] [--json]
prumo-agent compile --target claude-code [--path <project>] [--json]
prumo-agent compile --target traycer [--path <project>] [--json]
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
