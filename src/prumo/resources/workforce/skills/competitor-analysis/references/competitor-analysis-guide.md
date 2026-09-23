# Competitor Analysis Reference Guide

## Evidence Unit

Capture one atomic observation per workflow step. Each record contains competitor and version, platform, task, step number, observed behavior, friction cost, capture date, and evidence reference. Interpretation belongs in a separate field.

## Sampling

Choose products that address the same user goal, then bound the work to named workflows and platforms. For each workflow, record the starting state and perform the same task without skipping steps. Capture failures, recovery paths, and time or interaction cost when observable.

## Comparison Rubric

Use a fixed 1–5 scale for visibility, match to user language, user control, consistency, error prevention, recognition, efficiency, simplicity, recovery, and help. Add accessibility spot-checks for keyboard operation, focus, names, and contrast. Do not change the rubric between competitors.

## Synthesis

Group observations by user problem, not visual resemblance. Label each conclusion as observed, inferred, or validated. Rank opportunities with impact, frequency, confidence, and implementation effort; preserve links back to the underlying evidence IDs.

## Patterns and Anti-Patterns

| Do | Avoid |
|---|---|
| Record version, platform, and capture date | Write timeless claims about a changing product |
| Quote the observed interaction | Infer motivation from one screenshot |
| Use identical tasks for every product | Compare different workflows and call it fair |
| Keep contradictory evidence visible | Delete inconvenient observations |
| Exclude stale captures from ranking | Blend old and current evidence silently |

## Short Example

Observed: `OBS-17`, participant reaches export in 6 clicks and cannot identify the format control. Inferred: `INF-04`, the format label may not match user vocabulary; unvalidated. Opportunity: test a visible file-type summary before the export action.
