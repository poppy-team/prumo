# Untrusted Project Security — Technical Reference Guide

## 1. Core Concepts

**Zero-trust ingestion**: a foreign repository is untrusted data until quarantine inspection passes. No build, install, or test command runs during triage — analysis is static (parsers, AST, text/entropy scanners) with **network egress disabled**.

**Execution triggers** hide in plain sight: `.git/hooks/*`, `.vscode/tasks.json`, `.devcontainer/devcontainer.json`, GitHub Actions workflows, `Makefile`/`build.rs` targets, npm `preinstall`/`postinstall` scripts, `.env` sourcing in shell profiles. Every trigger is an allow-list decision, not an auto-run.

**Sandboxed exploration**: when dynamic execution is finally justified, it happens in an ephemeral non-root container (or microVM) with a read-only project mount, tmpfs scratch space, CPU/memory caps, and no network — requiring explicit human authorization first.

## 2. Patterns

- **Static manifest parsing**: read `go.mod`, `package.json`, `Cargo.toml`, `requirements.txt` as text; map dependency names/versions without resolving or downloading.
- **Entropy + hook sweep**: flag base64/hex blobs > 200 chars, zero-width Unicode, and any file under hook/trigger paths; report each with path + byte offset.
- **Read-only triage mount**: `mount -o ro,nodev,noexec` for inspection; compile artifacts go to an isolated tmpfs never re-ingested.
- **Clearance report**: emit `.prumo/history/quarantine-report.json` with findings, risk score, and an explicit APPROVED / QUARANTINED verdict before onboarding.

## 3. Anti-Patterns

- Running `npm install`, `cargo build`, `make`, or test suites on first contact.
- Allowing egress "just to fetch dependencies" during triage.
- Trusting file extensions (`.txt` containing ELF binaries, polyglot scripts).
- Skipping human authorization because "the container looks safe".

## 4. Worked Example

Repository `acme-widgets` arrives as a tarball: static sweep finds a `postinstall` hook in `package.json` curling an external URL plus a 4 KB base64 blob in `assets/logo.svg`. Both flagged; network stays off; no install runs. After maintainer confirmation the hook is a telemetry exfiltrator, verdict QUARANTINED with findings, risk score 9/10, and the exact hook excerpt in the report.

## 5. Verification Pointers

- List all auto-executed paths (hooks, tasks, workflows, install scripts) with disposition per path.
- Prove zero egress: firewall/cgroup counters show no outbound connections during triage.
- Confirm verdict file exists with schema-valid risk score before any build command is authorized.
