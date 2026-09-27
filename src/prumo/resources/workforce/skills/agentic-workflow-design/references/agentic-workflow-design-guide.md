# Agentic Workflow Design — Reference Guide

## 1. Core Concepts

### 1.1 Stages with Single Owners
A recipe is a sequence of stages, each owned by exactly one role. The implement-review-
merge pipeline has three stages and three owners: implementer produces a diff plus test
log, reviewer produces an approval or rejection with reasons, merger produces the merge
commit. When a stage fails, exactly one role is accountable for the retry or the
escalation.

### 1.2 Coordination Topologies
Pipeline suits linear dependencies (each stage needs the previous output). Fan-out/
fan-in suits independent shards (5 files refactored in parallel, merged by one stage
with conflict rules). Supervisor suits dynamic routing (a triage role assigns tasks to
specialists). Blackboard suits opportunistic collaboration (agents post findings to
shared state). Default to pipeline; choose fancier topologies only when dependencies
demand them.

### 1.3 Handoff Contracts
A handoff is a typed message: task_id, producer stage, artifact paths, evidence
pointers, schema version. Version the contract (handoff v2) so producers and consumers
can evolve independently. A consumer receiving an unknown version rejects loudly with
the version mismatch in the log; silent coercion is how pipelines corrupt data.

### 1.4 Idempotency and Depth Caps
Retries are safe only when stages are idempotent: re-running produces the same result
with no duplicate side effects (file writes are overwrites, notifications carry
dedupe keys). Delegation depth counts handoffs from the root; at depth 3 the chain
stops and escalates. Unbounded delegation is how two agents politely ping-pong a task
forever.

## 2. Patterns and Anti-Patterns

| Pattern (do this) | Anti-Pattern (never do this) |
|---|---|
| One owner per stage, named in the recipe | "The team" owns the review stage |
| Versioned handoff schemas beside the recipe | Verbal handoffs in chat threads |
| Retry cap 2, depth cap 3, then escalate | Unlimited retries "until it works" |
| Idempotent stages with dedupe keys | Notify-customer stage that double-sends on retry |
| Explicit transition table | Implied states nobody wrote down |

## 3. Worked Example
Recipe "fix-and-merge" v1.4.0 for bug BR-2091: pipeline of scout (owner researcher),
implement (owner coder), test (owner runner), review (owner reviewer, read-only).
Handoff v2 carries task_id, files_changed, test_log path. Transition table has 6
allowed edges; retry cap 2 per stage, depth cap 3. Failure walkthrough: reviewer
rejection returns to implement with reasons (retry 1 of 2); second rejection escalates
to the Goal owner. No stall path survives review.
