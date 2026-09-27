# Observability Report — Checkout Telemetry Pass

## 1. Metadata
- **Skill**: observability (Observability)
- **Date**: 2026-09-06
- **Author / Agent**: platform-team
- **Target Goal / Phase**: GOAL-095 checkout-telemetry

## 2. Executive Summary
Instrumented the checkout path with structured JSON logs, W3C trace context propagation, RED metrics per endpoint, and secrets redaction at the sink. P99 checkout latency visible per span (p99 890 ms, budget 1,200 ms); card-number redaction verified (0 leaks in 50,000-log scan); log volume held to 4.1 KB/request against the 5 KB budget.

## 3. Inputs & Scope
- **Inputs Evaluated**: checkout service endpoints (6), log schema v2, OpenTelemetry SDK 1.28, redaction allowlist v4
- **Artifacts Modified**: `checkout/*.go` (instrumentation), `otel/collector.yaml`, `logging/redactor.go`

## 4. Key Findings & Implementation Details
- **Logs**: JSON schema v2 (`ts`, `level`, `svc`, `trace_id`, `span_id`, `msg`, `fields`); levels used: DEBUG gated by flag, INFO for business events, WARN for degraded, ERROR for failures (level audit: 0 `fmt.Print` strays).
- **Tracing**: W3C `traceparent` propagated across checkout → payments → ledger; span attributes include `merchant_id`, `amount_cents` (never PAN); tail sampling 10% + 100% on errors.
- **Metrics**: RED per endpoint (`requests_total`, `duration_histogram`, `errors_total`); alert on p99 > 1,000 ms for 5 min and error-rate > 0.5%.
- **Correlation**: Every log line carries `trace_id`/`span_id`; incident drill: trace → logs → metrics joined in 2 queries (runbook updated).
- **Redaction**: Sink-side redactor strips PAN/CVV/tokens via allowlist; scan of 50,000 production-shape logs: 0 secret leaks; overhead +0.8% CPU.
- **Budgets**: 4.1 KB/request log volume; cardinality audit: 0 unbounded label values (user IDs excluded from labels).

## 5. Verification & Evidence
- **Evidence Type**: test
- **Test Results**: Passed — redaction scan 0/50,000 leaks; level audit clean; incident-drill join 2 queries; volume under budget
- **Static Analysis Status**: Pass — instrumentation lint clean; no unbounded metric labels

## 6. Next Steps & Handoff
- Extend RED + tracing to refund path (GOAL-096); owner: platform-team.
- Tune tail-sampling rate after 2 weeks of volume data; owner: sre-oncall.
