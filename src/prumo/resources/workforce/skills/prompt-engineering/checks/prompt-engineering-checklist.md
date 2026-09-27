# Prompt Engineering & Evaluation — Verification Checklist

## 1. Baseline & Harness Discipline
- [ ] Eval harness runs on the current prompt with a fixed seed before any edit.
- [ ] Sampling config (temperature, top_p, seed, max_tokens) is pinned in harness config.
- [ ] Eval set meets minimums: 30 gold cases for format gates, 100 for behavior gates.

## 2. Single-Variable Edits & Versioning
- [ ] Each version changes exactly one instruction, example, or constraint.
- [ ] Prompt saved as a new versioned file; previous version untouched.
- [ ] Changelog line states what changed and the hypothesis behind it.

## 3. Regression Gates
- [ ] Full eval set re-run with identical config; per-dimension deltas recorded.
- [ ] No dimension regresses more than 2 points; target dimension improves.
- [ ] Flaky eval cases (pass rate variance over 5 points across 3 seed-42 reruns) are quarantined, not averaged away.

## 4. Prompt Diff Review
- [ ] Second reviewer checks for contradictory instructions within the prompt.
- [ ] Few-shot examples use synthetic data only; no production secrets or customer data.
- [ ] No unbounded output directives; every enumeration carries an explicit cap.

## 5. Evidence & Rollback
- [ ] Baseline vs candidate score tables attached to the change record.
- [ ] Rollback pointer recorded: previous version path plus one-command revert.
- [ ] Automated verification script `scripts/verify.sh` executes with exit code 0.
