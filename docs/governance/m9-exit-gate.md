# M9 Experience Layer — Exit Gate

Status: **COMPLETE (M9 Exit Gate Passed)**

## Exit gate criteria

Milestone M9 exit gate (per `docs/development/phases.md`):
An agent can hand off to another agent or session with full structured context;
session events are structured semantic units without conversational transcripts;
summaries are generated per session and per Goal; the Experience Provider Contract
governs episodic storage; experience proposals are validated before promotion;
and retention policies manage storage expiration.

---

### 1. Structured Session Events (Not Transcript)

- Implemented in `internal/experience/events.go`:
  - Defined event types: `session_started`, `goal_selected`, `decision_made`, `checkpoint_saved`, `tool_executed`, `blocker_occurred`, `session_completed`, `handoff_created`.
  - Invariant: Only semantic events are recorded; conversational transcripts and raw LLM token dumps are prohibited from episodic storage.
- Schema contract: `schemas/experience-event.schema.json`.

---

### 2. Session and Goal Summaries

- Implemented in `internal/experience/summary.go`:
  - `SynthesizeSessionSummary`: aggregates completed tasks, current focus, active blockers, and decisions.
  - `SynthesizeGoalSummary`: aggregates across all sessions contributing to a Goal, deriving progress status (`planned`, `in_progress`, `blocked`).

---

### 3. Handoff Protocol (State Transfer Between Agents / Sessions)

- Implemented in `internal/experience/handoff.go`:
  - Allows seamless context transfer between autonomous agents without passing full conversation transcripts.
  - Handoff package encapsulates:
    - Structured session summary.
    - Active decisions and open questions.
    - Evidence hashes.
    - Full lifecycle states (`pending`, `transferred`, `acknowledged`, `rejected`).
- Schema contract: `schemas/experience-handoff.schema.json`.

---

### 4. Experience Provider Contract & File Provider

- Defined in `internal/experience/provider.go`:
  - Interface `ExperienceProvider` with methods for recording events, saving/loading summaries, and creating/acknowledging handoffs.
  - `FileProvider` implements thread-safe file storage under `.prumo/experience/`.

---

### 5. Experience Proposals (Review Governed)

- Implemented in `internal/experience/proposal.go`:
  - When agents observe reusable heuristics or recurring patterns, they generate an `ExperienceProposal`.
  - Invariant: Experience proposals are review-governed (`review_required: true`) and cannot be promoted to canonical rules or workforce skills without governance approval.

---

### 6. Retention Policies

- Implemented in `internal/experience/retention.go`:
  - Configurable max age hours and max events per session.
  - Automated pruning (`FileProvider.Prune`) cleans expired raw events while preserving synthesized summaries indefinitely.

---

### 7. CLI Surface

- Implemented in `cmd/prumo/experience_commands.go` and `cmd/prumo/main.go`:
  - `prumo-agent experience status`
  - `prumo-agent experience handoff create --id <id> --from <from> --to <to> --goal <goal>`
  - `prumo-agent experience handoff show <id>`
  - `prumo-agent experience handoff ack <id> --actor <actor>`
  - `prumo-agent experience events --session <id>`
- Verified by automated tests in `cmd/prumo/experience_commands_test.go`.

---

## Evidence & Test Verification

- `go test -race ./...` — 100% pass across all packages.
- `go vet ./...` — clean.
- `gofmt -l .` — clean.
- `python -m pytest -q` — 136 pass (Python oracle preserved).
- Experience suite (`internal/experience`): all tests passing (100%).
- CLI suite (`cmd/prumo`): all tests passing (100%).

---

## Conclusion

Milestone M9 (Experience Layer) is **COMPLETE**. All criteria and exit gate invariants are satisfied.
