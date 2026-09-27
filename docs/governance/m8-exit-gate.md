# M8 History + Traceability — Exit Gate

Status: **COMPLETE (M8 Exit Gate Passed)**

## Exit gate criteria

Milestone M8 exit gate (per `docs/development/phases.md`):
Any code change is traceable to a decision and goal; the implementation journal is queryable;
experiments, rejections, and technical debt are tracked; the typed traceability graph links
requirements, decisions, goals, code, tests, documentation, and evidence.

---

### 1. Typed Traceability Graph (req ↔ dec ↔ goal ↔ code ↔ test ↔ doc ↔ evidence)

- Canonical graph model implemented in `internal/traceability/graph.go`:
  - Node kinds: `req`, `dec`, `goal`, `code`, `test`, `doc`, `evidence`, `experiment`, `rejection`, `debt`.
  - Edge kinds: `satisfies`, `derives_from`, `implements`, `verifies`, `documents`, `evidenced_by`, `rejects`, `incurs_debt`.
  - Bidirectional traversal (`TracePath`) resolves full upstream lineages (decisions, requirements) and downstream lineages (tests, evidence, documentation).
- Schema contract: `schemas/trace-graph.schema.json`.

---

### 2. Implementation Journal (Synthesis, Not Chain-of-Thought)

- Implemented in `internal/traceability/journal.go`:
  - Enforces Lean Progressive Context: stores structured implementation summaries, affected decisions, changed code, verification tests, and evidence hashes.
  - Invariant: Raw chain-of-thought, verbose conversation transcripts, and non-deterministic telemetry are prohibited from canonical journal storage.
  - Queryable by Goal, Decision, or File path.
- Schema contract: `schemas/journal-entry.schema.json`.

---

### 3. Experiment, Rejection, and Debt Registers

- Implemented in `internal/traceability/registers.go`:
  - **Experiment Register**: Captures hypotheses, methods, outcomes, and architectural conclusions.
  - **Rejection Register**: Tracks discarded proposals, evaluated alternatives, and authoritative rationale for rejection.
  - **Debt Register**: Tracks technical/architectural debt with severity (`low`, `medium`, `high`, `critical`), impacted contracts, and concrete remediation plans.

---

### 4. CLI Surface (`prumo-agent trace` and `prumo-agent journal`)

- Implemented in `cmd/prumo/trace_commands.go`:
  - `prumo-agent trace <ref>`: Traces any goal, decision, code file, or requirement, returning structured upstream/downstream lineages in human terminal format or JSON envelope.
  - `prumo-agent journal [--goal <goal>] [--decision <decision>] [--file <file>]`: Queries implementation history.
- Verified by automated tests in `cmd/prumo/trace_commands_test.go`.

---

## Evidence & Test Verification

- `go test -race ./...` — 100% pass across all packages.
- `go vet ./...` — clean.
- `gofmt -l .` — clean.
- `python -m pytest -q` — 136 pass (Python oracle preserved).
- Traceability suite (`internal/traceability`): all tests passing (100%).
- CLI suite (`cmd/prumo`): all tests passing (100%).

---

## Conclusion

Milestone M8 (History + Traceability) is **COMPLETE**. All criteria and exit gate invariants are satisfied.
