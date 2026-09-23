# Competitor Analysis Verification Checklist

## Scope and Evidence
- [ ] Every competitor has an exact product version, platform, workflow, and capture date.
- [ ] Each observation is atomic and links to a screenshot, recording timestamp, or dated source.
- [ ] Captures older than the declared recency window are marked stale and excluded from ranking.

## Fair Comparison
- [ ] The same starting state and task are used for every competitor in a workflow.
- [ ] A fixed heuristic and accessibility rubric is applied without ad hoc criteria.
- [ ] Failures, recovery paths, interaction cost, and platform constraints remain visible.

## Evidence and Inference
- [ ] Observations, inferences, and validated findings are stored in separate fields.
- [ ] Every inference names a confidence level and a validation plan.
- [ ] Contradictory evidence is retained and explained rather than discarded.

## Synthesis and Opportunities
- [ ] Patterns are grouped by user problem rather than visual resemblance.
- [ ] Opportunities are ranked by impact, frequency, confidence, and implementation effort.
- [ ] Each recommendation links back to the evidence IDs that justify it.

## Release Gate
- [ ] The matrix is complete for every scoped workflow and competitor.
- [ ] Ethical and terms-of-service constraints are recorded.
- [ ] `scripts/verify.sh` exits with code 0 from the repository root.
