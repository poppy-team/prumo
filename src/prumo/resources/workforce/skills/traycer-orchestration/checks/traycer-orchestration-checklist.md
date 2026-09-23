# Traycer Orchestration — Verification Checklist

## 1. Plan Graph & Canonical Goals
- [ ] Plan holds at most 25 ordered steps, each declaring inputs, expected artifacts, and its verification gate.
- [ ] Plan hash is recorded; mid-run edits create a new plan version and re-baseline downstream checkpoints.
- [ ] Repository acceptance criteria are attached per step; adapter output contradicting them is rejected.

## 2. Dispatch & Adapter Boundary
- [ ] Steps dispatch with repository paths and acceptance text only; Goal reinterpretation by the adapter is absent.
- [ ] No plan, ledger, or checkpoint references Traycer-internal identifiers; adapter swap replays existing ledgers.
- [ ] Adapter interface version is recorded in the spec.

## 3. Step Ledger & Trace Completeness
- [ ] Ledger at `.prumo/tracer/ledger.jsonl` is append-only with step id, operation, artifact hashes, timestamps, and gate verdicts.
- [ ] Every executed step has a ledger entry; trace completeness measures 100%.
- [ ] Steps without ledger entries are treated as not executed.

## 4. Gates, Checkpoints & Retry Discipline
- [ ] Verification gates run immediately after each step; checkpoints write only on pass, retaining the last 20.
- [ ] Failed gates retry at most 3 times from checkpointed state with failure logs attached.
- [ ] A step failing 3 gates marks blocked and halts downstream dispatch.

## 5. Resume, Replay & Success Metrics
- [ ] Resume drill from the latest checkpoint replays the ledger without re-executing green steps.
- [ ] Replay report pairs every artifact with its creating ledger entry.
- [ ] Step success rate exceeds 90% and `scripts/verify.sh` exits 0.
