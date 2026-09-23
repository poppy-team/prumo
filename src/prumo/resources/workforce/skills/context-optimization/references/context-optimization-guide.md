# Context Optimization — Reference Guide

## 1. Core Concepts

### 1.1 Tokens per Resolved Goal
The north-star metric: total tokens spent divided by goals completed. A run spending
48,000 tokens to close one goal scores 48k/goal against a trailing mean of 31k/goal
and triggers optimization. Per-section accounting tells you where the 17k excess lives.

### 1.2 Relevance Tiers
Critical tier (error signatures, failing assertions, acceptance criteria) is immutable.
Supporting tier (surrounding code, passing neighbors) compresses first. Noise tier
(unrelated passing suites, duplicated headers) is dropped outright. Tiering is decided
from acceptance criteria, never from length: a long section can still be critical.

### 1.3 Lossless vs Lossy Compression
Dedup is lossless: collapsing 212 identical frames to 1 plus a count loses nothing.
Summarization is lossy and therefore restricted to verbose successes, with a removal
declaration pointing at the original. Paraphrasing a critical fact is neither; it is
corruption, because the model can no longer quote the exact string for grep or retry.

### 1.4 Prompt Caching Geometry
Providers cache stable prefixes. Keep instructions, schemas, and skill directives as
byte-identical leading blocks across runs in a workflow; put volatile content (logs,
timestamps) at the tail. Measured effect in production pilots: hit rates of 82 to 88%
on repeated triage loops, roughly halving billed input tokens.

## 2. Patterns and Anti-Patterns

| Pattern (do this) | Anti-Pattern (never do this) |
|---|---|
| Measure per-section cost, cut the largest first | Trimming prose while a 9,400-token log sits untouched |
| Script-collapsed dedup with counts | Hand-deleting "some" repeated lines by feel |
| Failing assertions byte-identical | Summarizing the one failing test into "a test failed" |
| Summaries declaring removals + original location | Silent compression nobody can audit |
| Stable cache prefixes, volatile tail | Rewording instructions every run, zero cache hits |

## 3. Worked Example
Incident triage working set of 24,000 tokens: app.log excerpt 9,400 (212 repeated
frames collapsed to 1, saves 6,100), retrieved docs 7,100 (3 near-duplicate chunks
deduped, saves 2,900), passing suite output 4,200 (replaced by "214 passed, 0 failed,
41s", saves 4,050). Total after: 10,950 tokens, a 54% saving; all 6 error signatures
verbatim; cache prefix stable at 3,300 tokens with an 85% hit rate.
