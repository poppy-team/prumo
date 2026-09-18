# Monitoring Guide

Monitor the Go CLI through deterministic health checks, exit codes, JSON envelopes, and repository state.

## Framework health

```bash
prumo-agent framework-check
prumo-agent --json framework-check
```

The command validates canonical adapters, schemas, catalog references, and workforce packages.

## Project health

```bash
prumo-agent validate ./project
prumo-agent doctor ./project
prumo-agent --json doctor ./project
```

Run after Goal transitions, migrations, compiler changes, and before release. Treat `ERROR` findings as blockers. Warnings require review according to the project risk policy.

## CLI diagnostics

JSON output is stable for automation:

```bash
prumo-agent --json version
prumo-agent --json validate ./project
prumo-agent --json doctor ./project > doctor.json
```

Use stderr for operational diagnostics and preserve stdout when consuming JSON.

## Compiler checks

```bash
for target in generic chatgpt claude kimi codex claude-code traycer; do
  prumo-agent compile --target "$target" --path ./project
  echo "$target: $?"
done
```

Compare generated artifacts against the canonical project and source workforce. Do not hand-edit generated adapters.

## Repository state

```bash
git status --short
du -sh ./project/.ai ./project/.prumo ./project/.codex ./project/.claude 2>/dev/null
```

Investigate unexpected generated files, unbounded `.prumo/runtime` growth, or changes outside the task scope.

## CI health gates

```bash
gofmt -l cmd internal embedded_assets.go
go test ./... -race
go vet ./...
```

Python v0.3 has been completely retired (ADR 002). CI health gates run pure Go checks.

## Suggested cadence

- Per change: `go test ./...`, `go vet ./...`, targeted `doctor`.
- Before merge: race tests, `framework-check`, all compiler targets.
- Before release: release build, checksums, install/setup/uninstall smoke tests, project-data preservation test.

