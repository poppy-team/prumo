# Workflow Recipe Specification

## 1. Recipe Identity
- **Recipe**: fix-and-merge v1.4.0 for bug BR-2091 (token-refresh race)
- **Topology**: pipeline (linear dependencies: scout -> implement -> test -> review -> merge)
- **Rationale**: each stage strictly requires the previous stage output; no parallelizable shards exist.
- **Failure budget**: delegation depth cap 3, retries cap 2 per stage.

## 2. Stages & Owners

| Stage | Owner role | Capabilities | Evidence produced | Gate |
|---|---|---|---|---|
| scout | researcher | filesystem.read | scout note: root cause + files to change | Names files and test command |
| implement | coder | filesystem.read, filesystem.write | diff + implement log | Diff touches only listed files |
| test | runner | process.spawn | go test ./auth -race log, 41s | Zero failures, race detector clean |
| review | reviewer (read-only) | filesystem.read | approval or rejection with reasons | No unresolved findings |
| merge | merger | process.spawn | merge commit 9f3ac21 | CI green on the merge commit |

## 3. Handoff Contract v2

| Field | Type | Example |
|---|---|---|
| task_id | string | T-342 |
| producer_stage | string | implement |
| files_changed | string array | auth/refresh.go, auth/refresh_test.go |
| test_log | path | evidence/T-342/test-race.log |
| schema_version | string | handoff-v2 |

## 4. State Transitions (allowed only)
- pending -> running -> in_review -> done
- running -> failed -> retrying -> running (retry counter max 2)
- in_review -> running (rejection with reasons, counts as retry)
- any -> escalated (depth 3 reached or retries exhausted) -> aborted or done by owner

## 5. Failure Walkthrough
- **Test failure**: runner reports log excerpt; implement retries (1 of 2) with narrowed mutex scope; passes on retry.
- **Reviewer rejection**: returns to implement with 2 findings; second rejection escalates to Goal owner Mara Chen.
- **Handoff mismatch**: consumer rejects handoff-v1 payload citing version; producer re-emits v2; incident logged.

## 6. Verification Evidence
- [ ] Recipe reviewed 2026-09-22, no silent-stall path found.
- [ ] `scripts/verify.sh` exits 0.
