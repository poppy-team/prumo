# GitHub Issue Refinement — Verification Checklist

## 1. Intent Preservation
- [ ] Reporter intent restated in one sentence and matches the original problem.
- [ ] Ambiguities resolved by asking the reporter, not by reinterpreting.
- [ ] Before/after refinement summary recorded with intent-preservation note.

## 2. Testable Acceptance Criteria
- [ ] Every checkbox names an observable verification method (command, metric threshold, error code).
- [ ] No adjectives without measurements ("fast", "correct", "user-friendly" converted or removed).
- [ ] Criteria reference defined fixtures, environments, and tooling only.

## 3. Dependencies & Sizing
- [ ] Blocking/blocked-by links, required ADRs, and area labels mapped.
- [ ] Hidden second deliverables split into cross-linked issues; one deliverable per issue.
- [ ] Type, severity (P0–P3), milestone, and labels consistent; contradictions removed.

## 4. Scope Control & Evidence
- [ ] Explicit non-goals section declares where implementation stops.
- [ ] No new scope added without reporter/owner acknowledgment in a comment.
- [ ] `scripts/verify.sh` exits 0 from the repository root.
