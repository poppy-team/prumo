# Multi-Agent Orchestration — Verification Checklist

## 1. DAG Planning & Dispatch
- [ ] Executable DAG recorded with nodes, depends-on edges, and computed critical path.
- [ ] Parallel dispatch respects the pool limit of 4 concurrent agents.
- [ ] No node starts before its dependencies close with evidence.

## 2. Isolation
- [ ] Each implementer works in a dedicated git worktree (e.g. /tmp/prumo-wt/T-342-coder).
- [ ] Task-scoped file allowlists enforced structurally; no cross-worktree writes occurred.
- [ ] Merge-back protocol followed: rebase on current main, resolve, CI re-run.

## 3. Caps Enforcement
- [ ] Caps ledger tracks retries per node (cap 2) and delegation depth per chain (cap 3).
- [ ] Nodes exhausting caps are parked and escalated, never silently retried.
- [ ] Every cap hit appears in the post-run report with owner decision.

## 4. Reviewer Diversity & Evidence Gates
- [ ] Author != reviewer enforced by user id check on every diff.
- [ ] Security-sensitive diffs carry 2 reviewers, one with security scope.
- [ ] Every closed node attaches test, lint, or review evidence; bare "done" claims rejected.

## 5. Completion & Audit
- [ ] CI verdicts on every merge-back archived; CI overrides agent self-report on conflicts.
- [ ] Run bundle (DAG, caps ledger, reviews, CI logs) archived under evidence/<goal-id>/.
- [ ] Automated verification script `scripts/verify.sh` executes with exit code 0.
