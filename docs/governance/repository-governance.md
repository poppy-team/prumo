# Repository Governance

Repository changes are governed by `.prumo/repository/policy.json`, validated by `schemas/repository-policy.schema.json`, and enforced through code, CI, GitHub settings, and review.

## Key concepts

- A **branch** is a movable pointer to a line of commits. Work happens on short-lived branches.
- A **Pull Request** proposes branch changes into `main` for validation and review.
- **Squash merge** combines one Pull Request into one logical commit on `main`.
- A **protected branch** requires validation and review before integration.
- A **ruleset** is a GitHub rule collection protecting branches or tags.
- A **force push** rewrites published history. Agents must not force push protected branches.

## Local policy

```bash
go run ./cmd/prumo-agent repo policy check
go run ./cmd/prumo-agent repo policy check --json
go run ./cmd/prumo-agent repo policy explain
go run ./cmd/prumo-agent repo policy plan --json
```

- `check` inspects the policy, local Git state, branch naming, working tree, and HEAD subject. It never mutates state.
- `explain` shows the rule, expected state, actual state, and enforcement reason.
- `plan` produces deterministic, ordered remediation. It never mutates state.
- `prumo-agent repo policy apply --dry-run` reports what would change.
- `prumo-agent repo policy apply` requires `GITHUB_TOKEN` or `GH_TOKEN`, re-reads remote state before mutation, changes only settings/rulesets supported by the adapter, and remains privileged.
- `check` and `plan` never mutate remote state.

Checks currently cover:

- `local.branch.naming`
- `local.working_tree.clean`
- `local.commit.subject`
- `github.repository.default_branch`
- `github.merge.squash_only`
- `github.branches.delete_after_merge`
- `github.ruleset.main`

## Remote policy

The GitHub adapter uses the standard library and REST API. It does not require `gh` at runtime. Credentials come from `GITHUB_TOKEN` or `GH_TOKEN`. Tokens are never stored in `prumo.json`, repository policy, Git, logs, or evidence.

Remote comparison distinguishes:

- `compliant`
- `non_compliant`
- `unavailable`
- `unsupported`
- `error`

Planned remote actions classify side effect, reversibility, permission, and risk. Re-reading remote state before privileged mutations prevents stale-plan overwrites.

## Agent permissions

Default Prumo agent behavior:

- Git read operations: allow.
- Local branch and commit in project scope: allow.
- Push work branch: allow with logging.
- Create Issue or Pull Request: allow with logging when requested by workflow or policy.
- Merge into `main`: explicit capability or approval policy.
- Modify rulesets: privileged.
- Publish release: privileged.
- Force push `main`, delete `main`, move or delete release tags: deny.

## Emergency bypass

Emergency bypass is a documented break-glass record, not a CLI shortcut. It requires reason, actor, timestamp, operation, scope, risk, evidence, and post-event review status. Never bypass rules because CI is inconvenient, tests are flaky, or faster completion is desirable.

## Reviews and merge

The repository uses the Solo profile: Pull Request, CI, conversation resolution, and a human merge decision. High and critical risk changes require independent verification and evidence. Critical changes require explicit human approval.

The GitHub default is squash merge. Merge commits and rebase merges are disabled. Merged branches are deleted automatically. The active `main` ruleset requires the `governance` status check; the repository's CI matrix also currently passes all Python and Go jobs.

## M5 dependency

```text
M4.75 Repository Governance Foundation
               ↓
M5 Documentation System v2
```

M5 agents inherit this governance for documentation deltas: branch, commit, Pull Request, validation, review, and merge remain governed from the beginning.
