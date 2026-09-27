# ADR 009: Performance Budgets and Pressure Policy

# Status

Accepted (2026-09-17) as policy. The thresholds stay deliberately unstated until a measured distribution exists — that is what this ADR decides, not a gap in it.

# Context

- GAP-036 asked for performance budgets and TTL/memory-pressure decisions.
- Measured baselines exist (2026-09-11, i7-3632QM): context compile ~6.8ms,
  BM25 search ~0.78ms, NativeAgent run-step ~7µs; fuzz 1.5M execs clean.
- A redraw baseline was added on the same reference hardware (2026-09-17,
  i7-3632QM) for the H10 criterion 5 window of 200 event rows:
  `tui.Model.View` 4.38 ms/frame (83 KB, 1846 allocs), `tui.Timeline.Append`
  146 µs/op (2 allocs), `tui.Session.Poll` with a full log and an advanced
  cursor 45 µs/op (1 alloc). Two facts the numbers state: the frame is linear
  in visible rows, which is why the window is capped; and an append re-indexes
  the window, so one new event on a full timeline costs a full-window copy —
  negligible against the 250 ms poll, and worth knowing rather than assuming.
  Measured by `tui/redraw_test.go`; no threshold has been derived from it yet.
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
