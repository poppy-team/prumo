# Prumo

Prumo is a Git-native protocol and CLI for software projects built with humans and AI agents.

The repository is the durable source of truth. Prumo stores canonical project state in Markdown, JSON, JSON Schema, and Git; generated adapters, caches, indexes, and runtime state remain derived.

## Names

The framework is **Prumo**. The command you run is **`prumo-agent`**, and the terminal client is launched with **`prumo-agent tui`** — a separate binary (`prumo-agent-tui`) that `prumo-agent tui` finds and runs. Documents that predate the rename and say `prumo <command>` mean the same binary.

## Current release line

Prumo v0.6 is a pure Go distribution (ADR 002).

- **Go v0.6** is the official single-binary CLI and Core implementation.
- Python v0.3 has been completely retired (ADR 002).
- Zero external runtime dependencies (no Python, pip, or virtualenv required).
- All canonical assets (schemas, catalog, workforce, adapters) are embedded directly into the Go binary.

## What Prumo provides

- Project initialization from a profile.
- Deterministic agent, skill, recipe, risk, and model-policy resolution.
- Goal lifecycle with SHA-256 lock integrity and formal amendments.
- Plan DAG validation, Events, Evidence, Gates, and Doctor diagnostics.
- JSON Schema Draft 2020-12 validation with local `$ref` resolution.
- Lean Progressive Context planning and project intelligence reports.
- Compiler adapters for Generic, ChatGPT, Claude, Kimi, Codex, Claude Code, Traycer, Gemini, OpenCode, and Google Antigravity.
- Machine-readable JSON envelopes for automation and harness integrations.
- Conformance tests comparing Go behavior with golden specification baselines.
- Portable installation state, connector ownership, setup, and safe uninstall.

## Quick start from source

Requirements:

- Go 1.22+;
- Git.

```bash
git clone git@github.com:raillen/prumo.git
cd prumo

go run ./cmd/prumo-agent version
go run ./cmd/prumo-agent --json version
```

Expected version output:

```text
0.5.0
```

Run `prumo-agent --help` or `prumo-agent <command> --help` to view all available commands, options, and quick examples. You can also consult the [CLI reference](docs/manual/usage.md#command-reference).

## Initialize a project

Create a profile with at least one preferred model, then initialize a project:

```bash
go run ./cmd/prumo-agent init ./my-project \
  --profile examples/brasa/project-profile.json \
  --non-interactive

go run ./cmd/prumo-agent validate ./my-project
go run ./cmd/prumo-agent doctor ./my-project
```

`prumo-agent init` creates canonical project files such as `prumo.json`, `.ai/`, `docs/PRUMO.md`, `PROJECT_STATE.md`, and `.prumo/history/`. It does not install a harness globally.

## Work with Goals

```bash
go run ./cmd/prumo-agent goal new P00-G01 "Foundation" \
  --phase P00 \
  --objective "Establish the project foundation." \
  --path ./my-project

go run ./cmd/prumo-agent goal state P00-G01 PLANNED --path ./my-project
go run ./cmd/prumo-agent goal state P00-G01 LOCKED --path ./my-project
go run ./cmd/prumo-agent goal list --path ./my-project
```

Locked Goals must be changed through `goal amend`; direct edits are detected by the lock digest.

## Compile a harness adapter

```bash
go run ./cmd/prumo-agent compile --target generic --path ./my-project
go run ./cmd/prumo-agent compile --target codex --path ./my-project
go run ./cmd/prumo-agent compile --target claude-code --path ./my-project
```

Generated artifacts are derived. The canonical project files and workforce packages remain the source of truth.

## One-Link Install (Linux, macOS & Windows)

### Linux & macOS
Downloads the release binary, verifies SHA-256 checksums, installs to `~/.local/bin/prumo`, and configures your shell `PATH` automatically:

```bash
curl -fsSL https://raw.githubusercontent.com/raillen/prumo/main/scripts/install.sh | sh
```

### Windows (PowerShell)
Downloads the Windows binary, verifies SHA-256 checksums, installs to `%LOCALAPPDATA%\Programs\prumo\prumo.exe`, and permanently configures user `PATH`:

```powershell
powershell -ExecutionPolicy ByPass -c "irm https://raw.githubusercontent.com/raillen/prumo/main/scripts/install.ps1 | iex"
```


For a reversible uninstall, review the script before running it:

```bash
curl --fail --location https://raw.githubusercontent.com/raillen/prumo/main/scripts/uninstall.sh -o uninstall.sh
sh uninstall.sh --mode pure --dry-run
sh uninstall.sh --mode pure
```

Use a portable installation home when testing or working in CI:

```bash
go run ./cmd/prumo-agent --home ./prumo-home setup
go run ./cmd/prumo-agent --home ./prumo-home install connector opencode
go run ./cmd/prumo-agent --home ./prumo-home uninstall --connectors --purge-cache --purge-global-config
```

Uninstall never removes project files, `.ai/`, docs, Goals, Plans, Evidence, or other repository data. Read the [installation manual](docs/manual/installation.md) and [uninstall manual](docs/manual/uninstallation.md).

## Machine output

Commands that support automation accept `--json`:

```bash
go run ./cmd/prumo-agent --json version
go run ./cmd/prumo-agent --json doctor ./my-project
go run ./cmd/prumo-agent --json framework-check
```

The envelope is:

```json
{
  "protocol_version": "1",
  "ok": true,
  "data": {},
  "diagnostics": [],
  "warnings": []
}
```

stdout is reserved for JSON when `--json` is used. Diagnostics belong on stderr. JSON output contains no ANSI formatting.

## Development checks

```bash
gofmt -l cmd internal embedded_assets.go
go test ./... -race
go vet ./...
```

The Go suite includes comprehensive tests for initialization, resolver behavior, Goals, Plans, Doctor, Explain, migration, snapshots, all seven compiler targets, connectors, and adoption engine.

## Documentation map

- [Install manual](docs/manual/installation.md)
- [Uninstall manual](docs/manual/uninstallation.md)
- [Usage manual](docs/manual/usage.md)
- [CLI reference](docs/manual/usage.md#command-reference)
- [First project](docs/getting-started/first-project.md)
- [Core concepts](docs/getting-started/concepts.md)
- [v0.4 boundary scope (historical)](docs/product/scope-v0.4.md)
- [Architecture](docs/architecture/overview.md)
- [ADR 001: Go Core](docs/adr/001-go-core.md)
- [ADR 002: Retire Python Runtime](docs/adr/002-retire-python-runtime.md)
- [Migration status](docs/migration/v0.3-to-v0.4-go.md)
- [Conformance strategy](docs/migration/conformance-strategy.md)
- [Development blueprint](docs/development/implementation-blueprint.md)
- [Testing strategy](docs/development/testing-strategy.md)
- [Security and trust model](docs/security/trust-model.md)
- [Runtime Control Plane](docs/runtime/control-plane.md)
- [Documentation System v2](docs/governance/documentation-system.md)
- [Documentation source map](docs/SOURCE_MAP.json)

Use `docs/PRUMO.md` as the repository documentation router. Do not load the entire documentation tree for a single task.

## License and contribution

Prumo is released under the [MIT License](LICENSE). Code derived from other
projects, and the revisions it was taken from, are recorded in
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

Before contributing, read `AGENTS.md`, the [coding standards](docs/development/coding-standards.md), the [dependency rules](docs/architecture/dependency-rules.md), and the [testing strategy](docs/development/testing-strategy.md).
