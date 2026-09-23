# GitHub Repository Management — Technical Reference Guide

## 1. Core Concepts

**Branch protection as code**: required reviews, required status checks (exact names), up-to-date-branch enforcement, force-push/deletion blocks — snapshotted before change via `gh api repos/{owner}/{repo}/branches/main/protection` and re-fetched after.

**Secret scoping**: environment > repository > organization is the wrong order — secrets live at the *lowest* scope that works (environment secrets for prod, repository for shared CI, organization only for truly global tokens). Fork PRs never see secrets; production environments require named reviewers.

**CODEOWNERS routing**: every protected path maps to an owner team; area labels mirror triage routing so issues land with humans, not voids.

## 2. Patterns

- **Snapshot-diff-verify**: export protection JSON → apply change → re-fetch → diff; the diff is the evidence.
- **Negative proofs**: attempt direct push to `main` (expect rejection), inspect a fork-PR run's env (expect empty secrets), request a production deployment as non-approver (expect gate).
- **Check-name exactness**: required checks reference the job's full name (`test (ubuntu-22.04, node 20)`), not the workflow name — renames break protection silently.
- **Rotation ledger**: secret names + scopes + rotation dates recorded; values never appear in logs or reports.

## 3. Anti-Patterns

- Undocumented click-ops ("I think I set it in the UI").
- Organization-wide secrets for single-repo needs.
- Required checks pointing at renamed/deleted jobs (protection theater).
- Stale CODEOWNERS routing reviews to departed teams.

## 4. Worked Example

Repo `acme/api`: snapshot shows `main` requires 1 review but zero required checks and allows force-push. Change: 2 reviews for `src/auth/**` (via CODEOWNERS + ruleset), required checks `lint`, `test (ubuntu-22.04, node 20)`, block force pushes and deletions, production environment with 2 named approvers, `NPM_TOKEN` moved from repository to production-environment scope. Negative proof: direct push rejected (`GH006`), fork-PR run shows no secrets. Before/after payloads archived.

## 5. Verification Pointers

- Re-fetched protection payload matches the intended policy field-for-field.
- Every secret lists scope + rotation date; no values in evidence.
- CODEOWNERS covers all protected paths with no stale owners.
