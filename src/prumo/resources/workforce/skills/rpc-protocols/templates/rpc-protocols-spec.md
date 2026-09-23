# RPC Protocol — Deliverable Specification

## 1. Service & Toolchain
- **Service**: InvoiceService in billing/v1 and billing/v2 packages, 9 RPCs total
- **IDL files**: billing/v1/invoice.proto with 14 messages, billing/v2/invoice.proto with 16 messages
- **Toolchain**: buf 1.50.0, protoc 27.3, grpcurl 1.9.1
- **Namespaces**: go_package example.com/billing/v1 plus v2 sibling, java_package com.example.billing.v1 plus v2 sibling
- **Migration window**: v1 deprecated 2026-09-10, removal no earlier than 90 days later

## 2. Numbering & Evolution Log
- **Reserved in v1**: field numbers 2, 4 to 6 and names legacy_total_cents, legacy_currency on GetInvoiceResponse
- **Added in v2**: field 7 line_items_json on Invoice, method RefundInvoice, zero reused numbers
- **Breaking gate**: `buf breaking --against '.git#branch=main'` reports 0 findings on 2026-09-12
- **Lint gate**: `buf lint` reports 0 findings on 2026-09-12

## 3. Error Taxonomy
- **GetInvoice**: NOT_FOUND for unknown invoice_id, INVALID_ARGUMENT for malformed ids, idempotent, retryable
- **ChargeInvoice**: FAILED_PRECONDITION for closed periods, ALREADY_EXISTS for duplicate idempotency keys, non-idempotent without key, not retried blindly
- **ExportLedger**: RESOURCE_EXHAUSTED for ranges over 366 days, idempotent, retryable with backoff
- **Details**: all failures attach google.rpc.Status with error code string and invoice_id

## 4. Time & Size Budgets
- **Deadlines**: GetInvoice 800 ms, ChargeInvoice 2 s, ExportLedger 5 s with server-streaming chunks of 64 KB
- **Payload ceiling**: 256 KB unary, pagination cursor beyond, largest golden response measured 182 KB
- **Retry policy**: capped exponential backoff with jitter, base 100 ms, max 3 attempts, idempotent methods only

## 5. Measured Results
- **Load test**: 500 RPS mixed golden methods for 10 minutes, p99 96 ms against 120 ms budget
- **Replay**: v1 client suite of 214 calls against v2 server, 214 decodes correct, 0 field drops
- **Live probe**: grpcurl list matches IDL service set exactly on staging billing-staging:50051

## 6. Verification Evidence
- buf reports archived at .prumo/rpc/buf-2026-09-12.log
- Load percentiles at .prumo/rpc/load-2026-09-12.log
- Replay log at .prumo/rpc/replay-v1-v2-2026-09-12.log
- `scripts/verify.sh` exit code 0 on 2026-09-12
