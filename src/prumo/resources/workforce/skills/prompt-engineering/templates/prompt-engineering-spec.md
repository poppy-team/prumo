# Prompt Evaluation Report Specification

## 1. Change Identity
- **Prompt**: prompts/triage-v3.3.0.md (supersedes prompts/triage-v3.2.0.md)
- **Hypothesis**: capping file lists at 10 paths reduces context bloat and raises tool-call validity.
- **Single variable**: added constraint "list at most 10 file paths per response" plus one max-length few-shot example.
- **Harness**: evals/triage-harness.yaml, 120 gold cases, seed 42, temperature 0.2, top_p 0.9, max_tokens 2048.

## 2. Score Comparison

| Dimension | v3.2.0 baseline | v3.3.0 candidate | Delta | Gate |
|---|---|---|---|---|
| Format adherence | 96% | 97% | +1 | pass |
| Tool-call validity | 91% | 93% | +2 | pass |
| Task success | 78% | 81% | +3 | pass, target improved |
| Refusal correctness | 100% | 100% | 0 | pass |
| Median tokens / solved case | 3,420 | 3,112 | -9% | pass |

## 3. Review Record
- **Reviewer**: Mara Chen, 2026-09-22.
- **Conflicts**: none; new cap consistent with existing "prefer smallest sufficient changeset" instruction.
- **Secrets scan**: examples use synthetic user id 1007 and token sk-test-0000; rg for prod prefixes clean.
- **Bounds**: enumeration capped at 10 paths; no unbounded directives remain.

## 4. Rollout & Rollback
- **Rollout**: canary to triage worker pool, 10% traffic for 24h starting 2026-09-23.
- **Rollback**: one-command revert to prompts/triage-v3.2.0.md, rollback tested 2026-09-22.

## 5. Verification Evidence
- [ ] Baseline and candidate runs share identical harness config and seed.
- [ ] No dimension regresses more than 2 points.
- [ ] `scripts/verify.sh` exits 0.
