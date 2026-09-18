# Installation and Deployment

This is the operational index for Prumo deployment. The user-facing procedure is maintained in the [installation manual](../manual/installation.md). The uninstall procedure is maintained in the [uninstallation manual](../manual/uninstallation.md).

## Release requirements

- Static Go binary with `CGO_ENABLED=0`.
- Supported artifacts: Linux amd64/arm64, macOS amd64/arm64, Windows amd64/arm64.
- Checksums published with every release.
- Signature/provenance verification required before public production distribution.
- Python is not a runtime dependency of the Go CLI.

## Build from source

```bash
go build -trimpath -o ./prumo ./cmd/prumo
./prumo version
```

## Build release artifacts

```bash
VERSION=0.4.0 sh scripts/release.sh
cat dist/checksums.txt
```

## CI checks

```bash
gofmt -l cmd internal embedded_assets.go
go test ./... -race
go vet ./...
```

Python v0.3 has been completely retired (ADR 002). CI executes pure Go checks and CLI smoke tests.


## Project deployment checks

```bash
prumo-agent validate ./project
prumo-agent doctor ./project
prumo-agent framework-check
```

Do not deploy a project with failing validation or unresolved critical Doctor findings. Generated adapters must be regenerated from canonical resources rather than hand-edited.

## Installation ownership

The installation manifest lives under `PRUMO_HOME/config/installation.json`. It records the binary path, connectors, managed fragments, and paths created by Prumo. Package-manager-owned binaries must not be overwritten silently.

## Rollback

1. Keep the previous binary until the new binary passes `version`, `framework-check`, and a project `doctor` run.
2. Replace the binary atomically.
3. Preserve the previous installation manifest.
4. If a connector update fails, use its cleanup manifest and report leftovers instead of deleting unknown files.

## Historical Python v0.3

The Python runtime remains documented in `docs/migration/` and `pyproject.toml` for compatibility testing only. Do not use it as the default v0.4 installation path.
