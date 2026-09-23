# Multi-Agent Orchestration — Reference Guide

## 1. Core Concepts

### 1.1 Dependency DAG Execution
Work is a directed acyclic graph: nodes are tasks with declared inputs, edges are
depends-on relations. The orchestrator dispatches nodes whose dependencies are closed,
up to 4 concurrent agents, and the critical path (longest dependency chain) determines
the earliest finish. Cycles are rejected at plan time by `prumo plan --dag`, not
discovered at runtime when two agents wait on each other forever.

### 1.2 Structural Isolation
Each implementer gets a git worktree plus a file allowlist. Isolation by structure
beats isolation by instruction: an agent cannot "accidentally" edit outside files it
cannot see. Merge-back rebases the worktree on current main and re-runs CI, so
integration breaks surface at merge time with the responsible diff identified.

### 1.3 Caps Ledger
Two counters per run: retries per node (cap 2) and delegation depth per chain (cap 3).
The ledger is append-only and checked before every dispatch. Caps convert infinite
loops into finite, owned decisions: park the node, notify the Goal owner, record the
outcome. A run with zero cap hits is normal; a run with unlogged retries is a defect.

### 1.4 Evidence-Gated Completion
Close requires evidence: a test log path, a lint log, or a review approval id
attached to the node. CI verdicts on merge-back override agent claims; when an agent
says done and CI says red, the node reopens automatically with the CI log attached.

### 1.5 Reviewer Diversity
The reviewer of a diff must differ from its author by user id, checked mechanically.
Security-sensitive paths (auth/, crypto/, secrets handling) require two reviewers with
one holding security scope. Self-review is not review, even when the agent insists
it was thorough.

## 2. Patterns and Anti-Patterns

| Pattern (do this) | Anti-Pattern (never do this) |
|---|---|
| DAG with critical path before dispatch | Starting agents, "figuring out order later" |
| Worktrees + allowlists enforced by tooling | "Please only edit your files" in prose |
| Park + escalate at retry 3 / depth 4 | Silent 11th retry at 3am burning budget |
| Author != reviewer checked by user id | Author reviewing own diff "for speed" |
| CI verdict overrides agent done-claims | Closing the Goal on agent assertion with red CI |

## 3. Worked Example
Goal G-118 runs 7 nodes across a 4-agent pool: scout and fixtures in parallel, two
implement shards in isolated worktrees (/tmp/prumo-wt/T-342-coder-a and -b), test node
after both shards, review by two non-authors, merge-back with CI green in 6 minutes.
One implement node hits retry 2 on a flaky test, parks, and the owner reroutes after
quarantining the flake. Run bundle archived under evidence/G-118/ with DAG, caps
ledger (1 cap hit), reviewer record, and CI logs.
