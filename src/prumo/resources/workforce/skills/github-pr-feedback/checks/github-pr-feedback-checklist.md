# GitHub PR Feedback Handling — Verification Checklist

## 1. Thread Inventory
- [ ] Every unresolved thread listed with file/line, reviewer, and classification (fix/test/question/dispute).
- [ ] Blocking (`request-changes`) threads separated from nits and ordered first.
- [ ] Zero threads left unclassified.

## 2. Targeted Resolution
- [ ] Fixes use minimal diffs scoped to the reviewer's ask; no opportunistic refactoring bundled.
- [ ] Reviewer-found bugs land with failing-first regression tests.
- [ ] Disputes documented in-thread with technical rationale and evidence, never resolve-and-ignore.

## 3. Replies & Re-Review
- [ ] Each thread answered (commit SHA for fixes, direct answers for questions).
- [ ] CI green on the final push; no unmapping force-push without a note.
- [ ] Re-review explicitly re-requested from each blocking reviewer with a thread→commit summary.

## 4. Closure & Evidence
- [ ] Zero unresolved threads confirmed via `gh pr view` before sign-off.
- [ ] Final verdict (approved/merged) recorded with evidence.
- [ ] `scripts/verify.sh` exits 0 from the repository root.
