# Installing Prumo

Prumo distributes a single static binary. Python is not required to run the Go CLI.

## Development Build

```bash
go build -o /tmp/prumo ./cmd/prumo
/tmp/prumo-agent --json version
```

## Release Build

```bash
VERSION=0.4.0 sh scripts/release.sh
ls dist/
cat dist/checksums.txt
```

## Setup and Portable Home

```bash
prumo-agent --home ~/.prumo setup
prumo-agent --home ./project-home setup
PRUMO_HOME=./project-home prumo setup
```

Setup records the installation manifest and detects available harnesses. Running setup twice converges to the same manifest.

## Install Connectors

```bash
prumo-agent --home ~/.prumo install connector opencode
```

Connector state lives under `PRUMO_HOME/connectors/<id>/cleanup.json`.

## Uninstall Safely

```bash
prumo-agent --home ~/.prumo uninstall
prumo-agent --home ~/.prumo uninstall --connectors
prumo-agent --home ~/.prumo uninstall --purge-cache
prumo-agent --home ~/.prumo uninstall --purge-global-config
```

Uninstall never deletes repository data: `.ai/`, docs, goals, plans, evidence, and all project files remain untouched. Purge flags only remove cache, derived runtime state, or global configuration.
