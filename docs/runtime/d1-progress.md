# D1 Runtime Foundation Progress

Implemented baseline capabilities:

- `Run` lifecycle with explicit transition validation.
- `ExecutorSession` separated from Run identity.
- Checkpoint and ContinuationRecord v1 schemas/models.
- `prumo-agent run` and `prumo-agent continue` human/JSON/prompt paths.
- Derived continuation under `.prumo/runtime/`, ignored by Git.
- Hierarchical Budget Envelope package with reservations, soft/hard limits, and usage.
- Deterministic Context Compiler package with source ordering, deduplication, token budget, and pressure states.
- Structured Observability event append baseline.
- Failure taxonomy and side-effect journal baseline.
- Bounded retry classification helper.

Remaining D1 gate work:

- None. All D1 gate deliverables are complete and verified:
  - Persisted Run lifecycle (`internal/runtime`) with explicit transitions and validation.
  - CLI operations: `prumo-agent run`, `prumo-agent run show`, `prumo-agent run cancel`, `prumo-agent run resume`, `prumo-agent run context`.
  - Continuation and prompt paths: `prumo-agent continue`, `prumo-agent continue --json`, `prumo-agent continue --prompt`.
  - Hierarchical Budget Envelope (`internal/budget`) and budget CLI commands (`prumo-agent budget`, `prumo-agent budget explain`).
  - Deterministic Context Compiler (`internal/contextcompiler`) with source ordering, token limits, pressure states, and CLI manifest recompilation.
  - Observability events append (`internal/observability`) and sanitized debug bundle export (`prumo-agent debug bundle`).
  - Failure taxonomy and side-effect journal baseline (`internal/runtime/failure`, `internal/runtime/journal`).
  - Portable continuation dogfood: fresh-executor fixture (`TestFreshExecutorContinuationDogfood`) and multi-process dogfood test (`TestPortableContinuationAcrossProcesses`).

Status: **COMPLETE (D1 Exit Gate Passed)**.

