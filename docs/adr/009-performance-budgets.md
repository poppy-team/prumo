# ADR 009: Performance Budgets and Pressure Policy

# Status

Proposed (thresholds pending benchmark corpus)

# Context

- GAP-036 asked for performance budgets and TTL/memory-pressure decisions.
- Measured baselines exist (2026-09-11, i7-3632QM): context compile ~6.8ms,
  BM25 search ~0.78ms, NativeAgent run-step ~7µs; fuzz 1.5M execs clean.
- Budgets are already enforced for agent work (tokens/money/time/tool-calls,
  hard/soft modes, `internal/budget`) — GAP-036 is about the harness's own
  performance envelopes, not agent budgets.
- No representative corpus exists for provider latency or sandbox overhead
  yet; live vendor sends were only recently proven (GAP-039).

# Decision

1. Policy now: every user-facing operation must report honest latency
   against recorded baselines; regressions visible in benchmarks are treated
   like test failures (investigate before merge).
2. Hard thresholds are set only from measured distributions on the reference
   hardware, not invented: a threshold proposal cites its benchmark run.
3. Memory pressure/TTL for derived runtime state (WCC, indexes, traces):
   default TTL 7 days, prune-on-start, hard memory ceiling per component
   only after the first SQLite-backed component lands (ADR 007) and can be
   measured.
4. The stopping condition invariant applies: any context mechanism's budget
   reports token/cost impact alongside latency, with a bounded maximum.

# Consequences

- No fake precision: thresholds arrive with evidence, not guesses.
- Benchmarks join the eval matrix (GAP-012) as the regression floor.
- Derived-state growth gets a deterministic, explainable policy instead of
  ad-hoc cleanup.
