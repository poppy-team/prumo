# Traycer Orchestration — Technical Reference Guide

## 1. Core Concepts

### 1.1 Goals Canonical, Adapter Replaceable
The repository Goal is the authority; Traycer is one execution engine among possible others. This separation means plans, ledgers, and checkpoints must be adapter-neutral JSON that any conforming adapter can replay. If swapping adapters invalidates history, the history format is wrong, not the new adapter.

### 1.2 Plan Graphs with Gates
A plan is an ordered list of steps where each step names its inputs, the artifacts it must produce, and the gate that judges it. A migration plan fragment looks like:

```json
{
  "plan": "ledger-sqlite-migration",
  "version": 3,
  "steps": [
    {
      "id": "m1-extract-schema",
      "inputs": ["ledger/schema.sql"],
      "artifacts": [".prumo/tracer/m1/schema.json"],
      "gate": "schema parses and covers 14 tables"
    },
    {
      "id": "m2-backfill",
      "inputs": [".prumo/tracer/m1/schema.json"],
      "artifacts": [".prumo/tracer/m2/backfill-report.log"],
      "gate": "row counts match source within 0 delta"
    }
  ]
}
```

The plan hash covers this whole document; editing it mid-run forks a new version instead of silently rewriting history.

### 1.3 Append-Only Step Ledger
Every execution appends one line per step to `.prumo/tracer/ledger.jsonl` with the step id, adapter operation, SHA-256 of each artifact, timestamps, and the gate verdict. Appends never mutate earlier lines, so audits replay cause and effect exactly as they happened.

### 1.4 Checkpoints and Resume
A checkpoint snapshots artifacts plus the ledger offset after each green gate. Resume loads the latest checkpoint and replays only subsequent ledger entries, which is why green steps are never re-executed: their artifacts are already proven and their ledger lines already closed.

## 2. Retry and Blocking Policy
- A gate failure retries the step from checkpointed state, up to 3 attempts, each with the failure log attached to the ledger entry.
- The third failure marks the step blocked and halts the plan; downstream steps never dispatch past a blocked step.
- Human re-planning unblocks by issuing a new plan version, preserving the blocked attempt in history.

## 3. Common Pitfalls
- Letting adapter-reported success override a failing gate, which launders unverified work into the trace.
- Checkpointing before the gate passes, so resume restores artifacts that never proved themselves.
- Dispatching steps in parallel when their artifacts overlap, creating ledger entries that replay nondeterministically.
