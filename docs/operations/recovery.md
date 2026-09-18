# Recovery Guide

Canonical Prumo projects recover from Git, snapshots, and deterministic regeneration. Reinstallation is rarely the right first action.

## Recovery order

1. `ENTRYPOINT.md` or the generated harness adapter.
2. `prumo.json`.
3. `PROJECT_STATE.md`.
4. `docs/PRUMO.md`.
5. Active Goal under `.ai/goals/`.
6. Relevant canonical docs, symbols, and tests selected by the context strategy.

## Restore a project

Do not hand-edit locked Goals or generated adapters. Prefer regeneration and approved amendments.

```bash
git status --short
prumo-agent doctor ./my-project
prumo-agent validate ./my-project
prumo-agent goal list --path ./my-project
```

Create a portable snapshot before manual recovery:

```bash
prumo-agent snapshot ./my-project --output ./my-project-recovery.zip
```

Restore from a known-good Git commit when local state diverged:

```bash
git log --oneline --decorate -10
git restore --source=<commit> -- <paths>
```

Regenerate adapters from canonical resources after restoring the source files:

```bash
prumo-agent compile --target codex --path ./my-project
prumo-agent compile --target claude-code --path ./my-project
```

## Restore global installation state

Use a clean portable home to isolate corrupted global state:

```bash
prumo-agent --home ./recovery-home setup
```

Then reinstall only connectors that provide cleanup manifests. Data loss in a repository is never fixed by deleting the project directory.

## Uninstall behavior

Uninstall follows `docs/manual/uninstallation.md`. It removes installation state, connectors, cache, or global configuration according to the flags. It never deletes `.ai/`, docs, Goals, Plans, Evidence, or other project files.

## Migration recovery

Preview migrations before applying them:

```bash
prumo-agent migrate ./my-project --dry-run --json
prumo-agent migrate ./my-project
```

The migration creates a timestamped pre-migration snapshot under `.prumo/snapshots/`.
