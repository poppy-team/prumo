# Contributing to Prumo

Start with `AGENTS.md`, `ENTRYPOINT.md`, and `docs/PRUMO.md`.

## Quick contribution flow

```bash
git switch main
git pull --ff-only
git switch -c feat/g042-example

# make changes

gofmt -w .
go test ./...
go vet ./...
pytest

git add <files>
git commit -m "feat(example): add capability"
git push -u origin feat/g042-example
```

Open a Pull Request against `main`. Do not push directly to `main`.

## Prerequisites

- Go 1.22 or later.
- Git.
- Python 3.10 or later only for the v0.3 oracle and Python tests.
- `gh` only when performing authorized remote repository administration.

## Branch naming

Use one category per branch:

```text
feat/<goal>-<slug>
fix/<issue>-<slug>
docs/<slug>
refactor/<slug>
test/<slug>
chore/<slug>
release/<version>
hotfix/<slug>
```

Automation branches such as `dependabot/*` remain valid.

## Commit format

Use Conventional Commits:

```text
type(scope): imperative summary
```

Valid examples:

```text
feat(repo): add repository policy evaluation
fix(goal): preserve lock during amendment
docs(install): document portable installation
```

Invalid subjects include `update`, `wip`, `final`, and untracked prose without a type.

## Checks before a PR

```bash
gofmt -l cmd internal
go test ./...
go vet ./...
go run ./cmd/prumo-agent repo policy check
pytest
```

Include validation, evidence, documentation delta, migration impact, and rollback notes in the PR template.

## Review and merge

`main` requires a Pull Request. The repository uses squash merge. Merged work branches are deleted automatically.

Risk determines verification:

- Low: PR, CI, resolved conversations, human merge decision.
- Team: at least one approval is recommended.
- High: independent verification and evidence are required.
- Critical: explicit human approval and security or evidence gates are required.

## Release policy

Releases use semantic tags such as `v0.5.0-alpha.1` and `v0.5.0`. Published release tags are immutable. See `docs/governance/release-policy.md`.

## Security reporting

Report vulnerabilities privately through `SECURITY.md`. Do not open public Issues for exploits.

## Troubleshooting

- `prumo-agent repo policy check` reports the local repository state.
- `prumo-agent repo policy plan --json` reports deterministic remediation.
- Use `prumo-agent repo policy apply --dry-run` before any remote remediation.
- Repository state changes require explicit commands and, when remote, appropriate credentials.
