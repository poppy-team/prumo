# Prumo

[![Release](https://img.shields.io/github/v/release/poppy-team/prumo?color=2563eb&label=release)](https://github.com/poppy-team/prumo/releases/tag/v0.6.1)
[![Go Version](https://img.shields.io/badge/go-1.22+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Workforce](https://img.shields.io/badge/workforce-189%20skills%20%7C%2039%20agents%20%7C%2020%20recipes-blueviolet)](docs/workforce/index.md)
[![Docs](https://img.shields.io/badge/docs-online-06b6d4.svg)](https://prumo-framework.vercel.app)

**Prumo** is a Git-native protocol and engineering harness for software projects built collaboratively by human developers and autonomous AI agents.

The repository is the durable source of truth. Prumo stores canonical project state in Markdown, JSON, JSON Schema, and Git commits; generated adapters, caches, indexes, and runtime state remain derived and disposable.

---

## Documentation Website

The full documentation website, inspired by modern developer documentation portals, is available at **[prumo-framework.vercel.app](https://prumo-framework.vercel.app)** (or browseable locally via local documentation server).

---

## Names & Ecosystem

- **Framework & CLI**: **`Prumo`** (binary/command: `prumo`, built from `cmd/prumo`).
- **Interactive Coding Agent (TUI)**: **`Prumo Agent`** (launched via `prumo agent` or `prumo tui`).
- **Desktop Graphical IDE (GUI)**: **`Prumo IDE`** (launched via `prumo native` or `prumo-viewer`).

---

## Release Line v0.6.1

Prumo v0.6 is a pure Go distribution ([ADR 002](docs/adr/002-retire-python-runtime.md)).

- **Go v0.6.1** is the official single-binary CLI and Core implementation.
- **Zero External Runtime Dependencies**: No Python, pip, or virtualenv required.
- **Embedded Assets**: All canonical schemas, catalog, workforce packages, and adapter templates are embedded directly into the binary.
- **Headless Harness & ACI**: Deterministic sandboxed tool gateway, process execution with hard timeouts, output boundaries, and AST syntax parsing.
- **Canonical Workforce**: **189 skills**, **39 specialized agents**, and **20 deterministic recipes** across systems, architecture, design systems, UI/UX, vector art, motion, and security.
- **Universal Connectors**: Transparent adapters for Google Antigravity, Claude Code, OpenAI Codex, Cursor, Windsurf, OpenCode, Cline, and Roo Code.

---

## What Prumo Provides

- **Deterministic Project Initialization**: Canonical profiles, capabilities, and repository metadata.
- **Cryptographic Goal Lifecycle**: SHA-256 integrity locks and formal, non-bypassable amendments (`goal amend`).
- **Lean Progressive Context (LPC)**: Smallest sufficient context, progressive expansion, and bounded outputs to prevent token exhaustion and hallucinations.
- **Plan DAG Validation & Execution**: Graph-based task dependencies, gate evaluations, and automated evidence collection.
- **Agent-Computer Interface (ACI)**: Sandboxed command runners, atomic file replacers, and multi-language AST inspection tools.
- **Surgical Adapter Compilers (`doccompile`)**: Preserves user-maintained configuration regions across compile, install, and uninstall cycles.
- **Repository Diagnostics (`prumo doctor`)**: Automated health checks for schemas, lock digests, DAG cycles, and toolchains.
- **Deterministic Machine Interface**: Standard JSON envelope (`protocol_version: "1"`) for seamless CI/CD and automation scripting.

---

## One-Link Installation

### Linux & macOS

Downloads the release binary, verifies SHA-256 checksums, installs to `~/.local/bin/prumo`, and configures your shell `PATH` automatically:

```bash
curl -fsSL https://raw.githubusercontent.com/poppy-team/prumo/main/scripts/install.sh | sh
```

### Windows (PowerShell)

Downloads the Windows binary, verifies SHA-256 checksums, installs to `%LOCALAPPDATA%\Programs\prumo\prumo.exe`, and permanently configures user `PATH`:

```powershell
powershell -ExecutionPolicy ByPass -c "irm https://raw.githubusercontent.com/poppy-team/prumo/main/scripts/install.ps1 | iex"
```

### Direct Binary Downloads (v0.6.1)

Precompiled release binaries are available on the [GitHub Release v0.6.1](https://github.com/poppy-team/prumo/releases/tag/v0.6.1):

- **Linux x86_64**: [`prumo-0.6.1-linux-amd64.tar.gz`](https://github.com/poppy-team/prumo/releases/download/v0.6.1/prumo-0.6.1-linux-amd64.tar.gz)
- **Linux ARM64**: [`prumo-0.6.1-linux-arm64.tar.gz`](https://github.com/poppy-team/prumo/releases/download/v0.6.1/prumo-0.6.1-linux-arm64.tar.gz)
- **macOS Apple Silicon (ARM64)**: [`prumo-0.6.1-darwin-arm64.tar.gz`](https://github.com/poppy-team/prumo/releases/download/v0.6.1/prumo-0.6.1-darwin-arm64.tar.gz)
- **macOS Intel (x86_64)**: [`prumo-0.6.1-darwin-amd64.tar.gz`](https://github.com/poppy-team/prumo/releases/download/v0.6.1/prumo-0.6.1-darwin-amd64.tar.gz)
- **Windows x86_64**: [`prumo-0.6.1-windows-amd64.zip`](https://github.com/poppy-team/prumo/releases/download/v0.6.1/prumo-0.6.1-windows-amd64.zip)

---

## Quick Start (5 Minutes)

### 1. Build and verify from source (Optional)

```bash
git clone https://github.com/poppy-team/prumo.git
cd prumo
go run ./cmd/prumo version
```

### 2. Initialize a project

```bash
prumo init ./my-project --profile ./examples/brasa/project-profile.json --non-interactive
prumo validate ./my-project
prumo doctor ./my-project
```

`prumo init` sets up canonical project files (`prumo.json`, `docs/PRUMO.md`, `.prumo/history/`).

### 3. Manage Goals with cryptographic locks

```bash
# Create a new Goal
prumo goal new P01-G01 "Core Systems" \
  --phase P01 \
  --objective "Implement foundation services" \
  --path ./my-project

# Transition state to PLANNED and then LOCKED
prumo goal state P01-G01 PLANNED --path ./my-project
prumo goal state P01-G01 LOCKED --path ./my-project
prumo goal list --path ./my-project
```

Locked Goals require formal amendments (`prumo goal amend`) to modify; direct manual edits break the SHA-256 lock digest.

### 4. Compile harness adapters

```bash
prumo compile --target claude-code --path ./my-project
prumo compile --target antigravity --path ./my-project
prumo compile --target cursor --path ./my-project
```

### 5. Execute tasks with the Harness

```bash
prumo run --path ./my-project
```

---

## Command Reference

| Command | Action |
|---|---|
| `prumo init` | Initialize a new repository with a canonical profile |
| `prumo goal` | Manage Goals (`new`, `state`, `amend`, `list`) with SHA-256 locks |
| `prumo plan` | Formulate and validate task dependency DAGs |
| `prumo run` | Execute directives and plans via sandboxed Harness |
| `prumo compile` | Compile provider-neutral rules to harness adapters |
| `prumo doctor` | Deep repository and harness health diagnostics |
| `prumo validate` | Validate repository state against JSON Schemas |
| `prumo agent` | Launch interactive Prumo Agent TUI (alias: `prumo tui`) |
| `prumo native` | Launch Prumo IDE desktop GUI |
| `prumo docs audit` | Audit documentation authority, drift, and links |
| `prumo docs authority` | Validate authority map and routing integrity |

---

## Machine Output (`--json`)

Every automation-ready command accepts `--json`:

```bash
prumo --json doctor ./my-project
prumo --json framework-check
```

Output envelope:

```json
{
  "protocol_version": "1",
  "ok": true,
  "data": {},
  "diagnostics": [],
  "warnings": []
}
```

Standard output (`stdout`) is reserved strictly for JSON when `--json` is enabled. Diagnostics and logs are piped to `stderr` with zero ANSI formatting.

---

## Running the Documentation Site Locally

The documentation website is powered by VitePress:

```bash
# Start development server
pnpm run docs:dev

# Build static website
pnpm run docs:build

# Preview static build
pnpm run docs:preview
```

---

## Documentation Map

- **[Documentation Site](https://prumo-framework.vercel.app)**
- **User Manuals**:
  - [Installation Manual](docs/manual/installation.md)
  - [Uninstallation Manual](docs/manual/uninstallation.md)
  - [Usage Manual & CLI Reference](docs/manual/usage.md)
  - [Connectors & Adapters](docs/manual/connectors.md)
- **Getting Started**:
  - [First Project Walkthrough](docs/getting-started/first-project.md)
  - [Core Concepts](docs/getting-started/concepts.md)
- **Architecture & Design**:
  - [Product Vision](docs/product/vision.md)
  - [Architecture Overview](docs/architecture/overview.md)
  - [Dependency Rules](docs/architecture/dependency-rules.md)
  - [ADR 001: Go Core](docs/adr/001-go-core.md)
  - [ADR 002: Retire Python Runtime](docs/adr/002-retire-python-runtime.md)
  - [ADR 016: Split Agent & Harness](docs/adr/016-product-split-agent-and-harness.md)
- **Harness & Workforce**:
  - [Harness Overview](docs/harness/index.md)
  - [Workforce Catalog](docs/workforce/index.md)
- **Governance & Methodology**:
  - [Lean Progressive Context (LPC)](docs/governance/lpc.md)
  - [Authority & Projection Policy](docs/governance/authority.md)
  - [Testing Strategy](docs/development/testing-strategy.md)
  - [Coding Standards](docs/development/coding-standards.md)
  - [Development Blueprint](docs/development/implementation-blueprint.md)
  - [Documentation Router](docs/PRUMO.md)

---

## License and Contribution

Prumo is released under the [MIT License](LICENSE). Third-party code notices are cataloged in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

Before contributing, review `AGENTS.md`, the [coding standards](docs/development/coding-standards.md), [dependency rules](docs/architecture/dependency-rules.md), and [testing strategy](docs/development/testing-strategy.md).
