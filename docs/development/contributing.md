# Contributing

The short contribution guide is maintained at [`CONTRIBUTING.md`](../../CONTRIBUTING.md). This page contains repository-internal engineering rules.

## Repository entrypoints

Read `AGENTS.md`, `ENTRYPOINT.md`, `FRAMEWORK.md`, and `docs/PRUMO.md` before changing behavior. New work must preserve the provider-neutral core policy and Lean Progressive Context discipline.

## Development setup

```bash
git clone git@github.com:raillen/prumo.git
cd prumo
go test ./...
go vet ./...
gofmt -l cmd internal embedded_assets.go
```

The Go test suite includes comprehensive unit, integration, and differential conformance tests:

```bash
go test -v -race ./...
```

## Code and package conventions

- Follow [coding standards](coding-standards.md).
- Respect [dependency rules](../architecture/dependency-rules.md).
- Do not introduce `utils`, `common`, `helpers` or other semantic-free packages.
- Keep the CLI thin: business behavior belongs in application services, not `main.go`.
- Return explicit, wrapped errors; never use `panic` for user/config failures.
- Treat JSON key order as an output-serialization detail; compare generated JSON semantically in conformance tests.
- Keep `embedded_assets.go` at the module root because Go `//go:embed` cannot traverse outside `internal/`.

## Schemas, workforce, and CLI

- Update the canonical schema in `schemas/` before changing a machine contract.
- Keep agents, skills, and recipes under `src/prumo/resources/` until the migration explicitly moves canonical content.
- Add differential tests for resolver, Goals, compiler, validation, scaffolding, migration, doctor, and explain behavior.
- Validate all compiler targets after changing shared compiler logic.
- Run `prumo framework-check` behavior coverage after catalog/workforce changes.

## Migration discipline

- The Python v0.3 runtime was retired in ADR 002; the repository is 100% pure Go.
- Do not change the Prumo protocol only to make Go implementation easier.
- New capabilities require closed Goals, documentation, and acceptance criteria.
- Lock content, ADRs, and schemas must change explicitly, not silently.

## Pull request process

- Keep changes review-sized and scoped to one Goal or milestone.
- Report `go test -race ./...`, `go vet ./...`, and `gofmt -l` results.
- Record protocol-affecting decisions in the canonical document and ADR first.
- Request review from maintainers before merging protocol, security, or release changes.
