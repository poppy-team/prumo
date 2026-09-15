# Testing

Prumo emphasizes rigorous testing to ensure deterministic behavior, contract conformance, and backward compatibility.

## Test Suite Organization (Go)

The test suite is written in pure Go and organized by package:

- `cmd/prumo`: CLI end-to-end and entrypoint tests (`main_test.go`).
- `internal/validation`: Draft 2020-12 JSON Schema validation and `$ref` resolution tests.
- `internal/goals`: Goal lifecycle, SHA-256 lock integrity, and amendment evaluation logic.
- `internal/resolver`: Deterministic agent, skill, recipe, risk, and model-policy resolution.
- `internal/scaffold`: Project initialization and directory bootstrapping.
- `internal/compiler`: Multi-target compiler adapters (Generic, Codex, Claude Code, Traycer, OpenCode).
- `internal/cliops`: Full CLI operations, differential conformance, and execution pipeline.
- `internal/runtime`: Agent runtime control plane, Living Plan, checkpoints, tool gateway, and session harness.
- `internal/adoption`: Brownfield repository scanning, classification, recipe synthesis, and adoption engine.
- `internal/install`: Portable installation, connector setup, and safe uninstallation.

## Running Tests

Run all unit, integration, and race tests:

```bash
go test -v -race ./...
```

Run tests for a specific package:

```bash
go test -v ./internal/adoption/...
```

## Quality Gates

Before submitting changes or merging pull requests:

```bash
diff -u <(echo -n) <(gofmt -d .)
go vet ./...
go test -v -race ./...
```

## Conformance Suite & Golden Fixtures

Differential conformance fixtures are stored under `conformance/golden/` (such as `conformance/golden/m3/compiler-parity.json` and `conformance/golden/cli/resolve-brasa-json.json`). Tests verify that Go compiler outputs and CLI behaviors adhere strictly to canonical schemas and frozen contracts.

## Python Runtime Retirement (ADR 002)

The legacy Python v0.3 test suite (`tests/`) and runtime modules have been fully retired (ADR 002). No Python interpreter is required to build, test, or execute Prumo.
