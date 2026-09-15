# Project Intelligence

Project Intelligence records how a project is evolving, not only what the project currently is.

It covers:

- LLM/token cost;
- compute/CI cost where measurable;
- human-equivalent effort estimates;
- development velocity;
- quality and bug repair cost;
- documentation impact/coverage;
- technical debt;
- risk;
- release/component history.

## Measurement honesty

Never present an estimate as an observed value.

Every monetary/effort metric should support provenance such as:

```text
observed
estimated
allocated
unknown
```

Estimates should prefer ranges and include confidence.

## Task report

Every completed implementation task should produce a compact report, then update project-level intelligence.

Example fields:

```text
id
type
component
status
risk
complexity
started/finished
models/providers
input/output/cached tokens
intermediate output
direct/estimated cost
effort range
files changed
tests
documentation delta
debt introduced/removed
evidence pointers
confidence
```

Do not create one permanent report file per task by default. Store compact task entries in the project intelligence data set.

## Durable storage

The recommended durable source is one JSON data file, for example:

```text
.prumo/history/project-intelligence.json
```

Runtime raw traces/caches belong in SQLite and are not canonical.

A future storage adapter may scale this without changing the conceptual contract.

## Project-level aggregation

The project report should support filtering/grouping by:

- period;
- component;
- task type;
- feature;
- release/milestone;
- agent role;
- model/provider;
- complexity;
- risk;
- status.

## Token economy metrics

Track input and output separately:

- source context available/estimated;
- retrieved tokens;
- injected tokens;
- API input;
- cached input;
- intermediate output;
- final output;
- documentation output.

Useful derived signals:

### Context Reduction Ratio

How much available source was avoided while preserving success.

### Output Efficiency

Whether generated output is mostly useful work/result rather than narration.

### Strategy efficiency

Compare Direct, Context Pack, Progressive Retrieval and Isolated Delegation by success, tokens, cost and latency.

## Cost dimensions

### Compute

LLM API, CI, builds, storage or other measured compute.

### Effort

Human-equivalent implementation/review effort, clearly marked estimated unless observed.

### Context

Input, output and cache use; savings from selective retrieval.

### Maintenance

Debt/complexity introduced, dependency surface and future remediation estimate.

## Subscription/local models

Support billing modes:

- `api`;
- `subscription`;
- `local`;
- `free`;
- `unknown`.

Do not invent a per-task direct dollar cost for a fixed subscription. If a project chooses to allocate subscription cost, store it separately from direct incremental cost.

## Estimate vs actual

Where possible store pre-task estimate and post-task observation/estimate to measure variance and calibrate future planning.

Start with simple statistics:

- medians;
- percentiles;
- variance;
- task/component similarity;
- risk multiplier.

Do not add ML until the historical data demonstrates a need.

## Technical debt

Accepted shortcuts should be explicit:

- debt ID;
- reason;
- component;
- remediation estimate;
- task that introduced it;
- task that removed it.

## Cost of quality

When a defect can be linked safely to earlier work, record repair effort/cost. The goal is component/system learning, not blame.

## Dashboard

The documentation site may expose a simple generated Project Intelligence dashboard:

```text
Overview
Costs
LLM Usage
Context Efficiency
Documentation
Quality / Debt
Development History
```

The dashboard reads durable JSON/derived aggregates. It is replaceable and never canonical.

## Task completion lifecycle

```text
Documentation Plan (preflight)
→ Implement
→ Test
→ Review
→ Documentation Delta (postflight)
→ Measure
→ Update Project Intelligence
→ Update durable knowledge/history
→ Context GC
→ Complete
```
