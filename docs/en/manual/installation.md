# Installation Manual

## Choose the method and paths (Installation paths)

| Scenario | Method |
|----------|--------|
| End user | Binary from a published release |
| Prumo development | `go build` or `go run` |
| CI, tests, and USB/devbox | `--home` or `PRUMO_HOME` |
| Automation and scripts | Compiled or packaged Go binary |

## One-Link Install

### Linux and macOS
The script detects the operating system and architecture (`amd64`/`arm64`), downloads the binary and the release checksums, validates SHA-256 integrity, installs to `~/.local/bin/prumo`, automatically configures the `PATH` in your shell's startup files (`~/.bashrc`, `~/.zshrc`, `~/.config/fish/config.fish`, or `~/.profile`), and runs `prumo-agent setup`.

```bash
curl -fsSL https://raw.githubusercontent.com/raillen/prumo/main/scripts/install.sh | sh
```

To review before running:
```bash
curl -fsSL https://raw.githubusercontent.com/raillen/prumo/main/scripts/install.sh -o install.sh
less install.sh
sh install.sh
```

### Windows (PowerShell)
The PowerShell script detects the architecture (`amd64`/`arm64`), downloads the `prumo-windows-*.exe` binary, validates SHA-256 integrity, installs to `%LOCALAPPDATA%\Programs\prumo\prumo.exe`, permanently configures the user's `PATH` in the Windows Registry, and runs `prumo-agent setup`.

```powershell
powershell -ExecutionPolicy ByPass -c "irm https://raw.githubusercontent.com/raillen/prumo/main/scripts/install.ps1 | iex"
```

Optional variables for customization:
```bash
PRUMO_VERSION=v0.6.0 PRUMO_INSTALL_DIR="$HOME/.local/bin" sh install.sh
PRUMO_HOME="$HOME/.prumo" sh install.sh
```


## Development from the Repository

```bash
git clone git@github.com:raillen/prumo.git
cd prumo
go version
go run ./cmd/prumo-agent version
```

Build a local binary:

```bash
go build -trimpath -o ./prumo ./cmd/prumo
./prumo version
```

The Go binary does not require Python, Node, or CGO.

## Release Build

The official script produces Linux amd64/arm64, macOS amd64/arm64, and Windows amd64/arm64:

```bash
VERSION=0.5.0 sh scripts/release.sh
cat dist/checksums.txt
cat dist/release.json
```

A release installation must verify the checksum before replacing the binary. The rollback strategy preserves the previous binary until the new one passes `version`, `framework-check`, and smoke tests; on failure, restore the previous binary without changing project data. Release signatures remain an operational gate before the final public distribution.

## Global State and Portable Mode

By default, Prumo uses `~/.prumo` on Unix-like systems. Use `PRUMO_HOME` or `--home` to isolate the state:

```bash
PRUMO_HOME="$PWD/.prumo-home" ./prumo setup
./prumo-agent --home "$PWD/.prumo-home" setup
```

The global state contains:

```text
PRUMO_HOME/
├── config/installation.json
├── cache/
├── logs/
├── connectors/
└── runtimes/
```

Do not confuse this state with `.prumo/` inside a project. The latter belongs to the project and may contain local derived state.

## Setup

```bash
prumo-agent --home ~/.prumo setup
```

`setup` is idempotent and records:

- Prumo version;
- binary path;
- harnesses detected on the `PATH`;
- connector state;
- paths managed by Prumo.

The command does not automatically change harnesses' global settings.

## Install a Connector

```bash
prumo-agent --home ~/.prumo install connector opencode
```

The connector's state must have a cleanup manifest before any automatic removal.

## Verify the Installation

```bash
prumo-agent version
prumo-agent --json version
prumo-agent framework-check
prumo-agent --json framework-check
```

Project verification:

```bash
prumo-agent validate ./my-project
prumo-agent doctor ./my-project
```

## Python Deprecation and Retirement (ADR 002)

The Python v0.3 (legacy) runtime and tests have been completely removed (see [ADR 002](../../adr/002-retire-python-runtime.md)). Prumo v0.6 is distributed exclusively as a single compiled Go binary, with no dependency on external interpreters, virtual environments, or Python package managers.

## Common Problems

### `go: command not found`
Install Go 1.22 or later and check `go version`.

### `prumo: command not found`
Use the binary's absolute path or add the binary's directory to the `PATH`:

```bash
export PATH="$PWD/bin:$PATH"
```

### `missing prumo.json`
The command was run outside a Prumo project. Provide the project path or run `prumo-agent init` first.

### Checksum failure
Do not use the artifact. Download it again and compare it with `dist/checksums.txt` or with the checksum published by the release.

### Corrupted global state
Use a new portable home to isolate the diagnosis:

```bash
prumo-agent --home "$PWD/prumo-recovery-home" setup
```
