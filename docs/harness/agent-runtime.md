# Agent Runtime contracts (HA0) + Native Agent (HA2)

Canonical package: `internal/harness/agent` (contracts), `internal/harness/runtime` (state machine).

## Contracts

`AgentRuntime`, `NativeAgentState`, `ModelProvider`, `AgentProvider`,
`ModelRequest`, `ModelEvent`, `Turn`, `Message`, `Observation`, `ToolCall`,
`ToolResult`, `PermissionRequest`, `Checkpoint`, `Continuation`, `Handoff`,
`AgentEvent`, `DomainEvent`, `PendingEffect`.

Schemas: `schemas/harness-checkpoint.schema.json`,
`schemas/harness-handoff.schema.json`, `schemas/agent-event.schema.json`.

## State machine (reentrant, unrolled)

```text
Prepare -> CompileContext -> RequestModel -> ConsumeModelEvent
-> PlanToolCalls -> PermissionCheck -> ExecuteTool -> RecordObservation
-> EvaluateStop -> Checkpoint -> Yield / Continue / Complete
```

Each safe point supports persist/cancel/pause/resume/handoff/replay.
Correctness never depends on coroutine/stack. `Runner.Step` advances exactly
one phase; `RunUntilDone` loops with a runaway guard.

## Ports

ModelService, ContextService, ToolService, PermissionService,
WorkspaceService, BudgetService, CheckpointStore, EventSink,
EvidenceService, KnowledgeService. Adapters implement ports; domain
packages stay testable with stdlib/fakes.

## Implementation status

Implemented: contracts, runner, fake-driven tests, checkpoint integration,
permission lifecycle, budget hook, ACI executor wiring, `prumo agent run`.
`prumo agent run` persists a JSONL timeline (`run.started/finished` plus
permission/model lifecycle events) replayable via `prumo agent events`;
`prumo agent protocol` negotiates the versioned protocol kernel
(`internal/harness/protocol`, v0.1.0).
Partial: compaction/steering policies, daemon scheduling, PTY lifecycle.
