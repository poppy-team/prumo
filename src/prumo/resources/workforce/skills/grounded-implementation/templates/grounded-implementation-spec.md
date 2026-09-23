# Grounded Implementation Record — Checkout Retry Wiring

## 1. Task & Scope Boundaries
- **Task**: Wire `MaxRetries` through the checkout client without inventing retry budgets.
- **Date**: 2026-09-16
- **Author**: payments-team
- **IN**: `checkout/client.go`, `checkout/options.go`, `checkout/client_test.go`
- **OUT**: `payments/ledger/*` (locked contract), PSP budget constants (awaiting authority)
- **INCIDENTAL (read-only)**: `docs/architecture/overview.md`, PSP timeout profile

## 2. Reality Inspection Findings
- `checkout.NewClient` constructor confirmed at `checkout/client.go:41`; `Options` struct at `checkout/options.go:12` with fields `Timeout`, `BaseURL` — no retry field exists (verified by read, not assumed).
- 4 call sites confirmed: `grep -rn "NewClient(" --include="*.go"` → `cmd/api:88`, `worker:31`, `e2e:12`, `bench:5`.
- PSP retry budget: UNKNOWN — no ADR or spec found (`grep -rni "retry" docs/ ADRs/` empty). Recorded as assumption below; constants isolated behind a single `RetryPolicy` struct for one-line correction.

## 3. Explicit Assumptions
- `ASSUMPTION(payments-team, expires 2026-10-07)`: max 4 attempts, backoff 200 ms × 2^n + jitter. Mechanical wiring proceeds; budget values flagged for confirmation before release cut.

## 4. Diff Summary (Grounded Slices Only)
- `Options.MaxRetries int` added with validation (`0 ≤ n ≤ 8`); constructor threads it into the HTTP layer; 6 new tests (validation, single-attempt, retry-exhaustion with mock PSP).
- No changes to ledger, PSP constants, or public API surface beyond the additive field.

## 5. Verification & Evidence
- `go test ./checkout/...` 31/31 green; `grep` audit shows zero references to invented symbols.
- Blocked slice tracked as GOAL-079 (PSP retry budget authority); this record ships without it.
