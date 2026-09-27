# GitHub Repository Management — Verification Checklist

## 1. Branch Protection
- [ ] Protected branches require PR reviews (≥1; more for high-risk paths) with up-to-date-branch enforcement.
- [ ] Required status checks listed by exact job name; force pushes and deletions blocked.
- [ ] Release branches carry equal-or-stronger protection than `main`.

## 2. Secrets & Environments
- [ ] Secrets stored at the lowest workable scope (environment > repository > organization).
- [ ] Production environments require named reviewers; fork PRs cannot read secrets (negatively proven).
- [ ] Rotation ledger current (names + scopes + dates); no secret values in any evidence.

## 3. Ownership & Routing
- [ ] CODEOWNERS covers all protected paths with current owner teams; stale owners removed.
- [ ] Area labels mirror triage routing; repository defaults (visibility, features) match policy.

## 4. Verification & Evidence
- [ ] Protection and environment payloads re-fetched via API after change; before/after diff archived.
- [ ] Negative proofs executed where safe (direct push rejected, fork-PR env empty).
- [ ] `scripts/verify.sh` exits 0 from the repository root.
