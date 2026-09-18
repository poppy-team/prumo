# Troubleshooting and Operations

See the [usage manual](../manual/usage.md) for supported workflows and the [installation manual](../manual/installation.md) for verified installation paths.

## Common failures

### `prumo: command not found`

Use a locally built binary or installation path:

```bash
go build -trimpath -o ./prumo ./cmd/prumo
./prumo version
export PATH="$PWD:$PATH"
prumo-agent version
```

Do not infer a broken global installation from a missing `PATH` entry.

### `missing prumo.json` or `project root not found`

The command was executed outside an Prumo project:

```bash
prumo-agent validate ./my-project
prumo-agent status --path ./my-project
prumo-agent init ./my-project --profile examples/brasa/project-profile.json --non-interactive
```

### Goal lock digest mismatch

The locked Goal was changed outside `goal amend`. Inspect the Goal file and create an approved amendment:

```bash
prumo-agent goal state P00-G01 LOCKED --path ./my-project
```

Never bypass a lock by copying or rewriting files manually.

### Invalid goal transition

Follow the protocol order:

```text
DRAFT → PLANNED → LOCKED → EXECUTING → VERIFYING → REVIEWING → DONE
```

`DONE` requires evidence. Use `--reason` for transitions that need audit context.

### Adapter output looks stale

Regenerate from canonical resources:

```bash
prumo-agent compile --target codex --path ./my-project
prumo-agent compile --target claude-code --path ./my-project
```

Do not edit generated files such as `AGENTS.md`, `.codex/`, `.claude/`, or runtime ENTRYPOINT files.

### Conformance mismatch

Compare commands, exit codes, JSON, stdout/stderr, filesystem effects, and canonical artifacts between Python and Go. Golden outputs are stored under `conformance/golden/`. Regenerate a golden snapshot only after reviewing the intentional protocol change.

## Recovery order

1. `ENTRYPOINT.md` or the generated harness adapter.
2. `prumo.json`.
3. `PROJECT_STATE.md`.
4. `docs/PRUMO.md`.
5. Active Goal under `.ai/goals/`.
6. Only relevant canonical docs, symbols, and tests selected by the context strategy.

## Incident and deployment runbooks

- `incident-response.md`
- `monitoring.md`
- `recovery.md`
- `deployment.md`

The legacy Python-specific troubleshooting and deployment instructions were intentionally retired as user-facing paths. Python remains relevant only as the v0.3 compatibility oracle documented under `docs/migration/`.
