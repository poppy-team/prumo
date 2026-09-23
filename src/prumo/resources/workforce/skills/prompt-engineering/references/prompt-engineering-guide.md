# Prompt Engineering & Evaluation — Reference Guide

## 1. Core Concepts

### 1.1 Eval Dimensions
Score prompts on orthogonal dimensions rather than a single vibe score: format
adherence (output parses against schema), tool-call validity (arguments match function
signatures), task success (acceptance criteria met), refusal correctness (declines
out-of-scope requests), and token efficiency (median tokens per solved case). A prompt
that gains 5 points of task success while losing 6 points of tool validity is a
regression wearing a costume.

### 1.2 The Regression Bar
Ship rule: target dimension improves AND no dimension drops more than 2 points. The
2-point band absorbs harness noise without hiding real damage. Track rolling means over
the last 5 runs; a slow 1-point-per-week bleed across a month is also a regression.

### 1.3 Few-Shot Design
Examples teach format, not facts. Two to five examples suffice; each shows input,
reasoning sketch, and exact output shape. Prefer edge cases (empty result, max-length
list) over happy paths, because the model already knows the happy path. Rotate one
example per quarter to prevent overfitting to fixed ids.

### 1.4 Sampling Discipline
Evals run cold and pinned: temperature 0.2, top_p 0.9, seed 42, max_tokens 2048. Any
comparison across different sampling configs measures the sampler, not the prompt.
Production may sample hotter; evals never do.

## 2. Patterns and Anti-Patterns

| Pattern (do this) | Anti-Pattern (never do this) |
|---|---|
| One variable changed per version | Rewriting half the prompt and "seeing how it goes" |
| Baseline scores frozen before editing | Editing first, evaluating only the candidate |
| Synthetic example data (id 1007) | Pasting a real customer ticket as an example |
| Explicit output caps ("at most 5 items") | "List all relevant items" with no bound |
| Quarantining flaky eval cases | Averaging flakiness into a passing mean |

## 3. Worked Example
Triage prompt v3.2.0 baselines at format 96%, tool-validity 91%, task success 78% on
120 gold cases, seed 42. Candidate v3.3.0 adds one constraint ("cap file lists at 10
paths") plus a max-length few-shot example: format 97%, tool-validity 93%, task success
81%, tokens per case down 9%. No dimension regresses; the diff passes review with
synthetic data confirmed, and v3.3.0 ships with v3.2.0 retained as rollback.
