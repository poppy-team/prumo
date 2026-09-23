# RPC & Message Protocols — Technical Reference Guide

## 1. Core Concepts

### 1.1 IDL-First Service Modeling
The schema file is the contract; everything else derives from it. A billing invoice service in proto3 declares its package version, language options, and streaming mode up front:

```proto
syntax = "proto3";
package billing.v1;

option go_package = "example.com/billing/v1;billingv1";
option java_package = "com.example.billing.v1";

service InvoiceService {
  rpc GetInvoice(GetInvoiceRequest) returns (GetInvoiceResponse);
  rpc ExportLedger(ExportLedgerRequest) returns (stream ExportLedgerChunk);
}

message GetInvoiceRequest {
  string invoice_id = 1;
  bool include_lines = 2;
}

message GetInvoiceResponse {
  Invoice invoice = 1;
  reserved 2, 4 to 6;
  reserved "legacy_total_cents", "legacy_currency";
}
```

`reserved` numbers and names document what was removed and block accidental reuse forever.

### 1.2 Evolution Without Breakage
Wire compatibility rests on three habits: only add optional fields, never change a field type or number meaning, and ship incompatible semantics as a new package such as `billing.v2`. `buf breaking --against '.git#branch=main'` enforces this mechanically in CI — zero findings is the merge gate, not an aspiration.

### 1.3 Errors Clients Can Act On
gRPC defines 17 canonical status codes. Map business outcomes onto them — `NOT_FOUND` for missing invoices, `FAILED_PRECONDITION` for closed accounting periods, `INVALID_ARGUMENT` for malformed ids — and attach machine-readable details with `google.rpc.Status`. Retryability follows idempotency: safe to retry `GetInvoice`, never safe to blindly retry a charge method.

### 1.4 Budgets for Time and Bytes
Deadlines compose down the call chain, so set them per method: 800 ms for point reads, 5 s for exports. Cap unary responses at 256 KB and paginate or stream beyond that. Retries use capped exponential backoff with jitter from a 100 ms base and stop after 3 attempts, which bounds amplification during partial outages.

## 2. Cap'n Proto Notes
Cap'n Proto suits zero-copy paths where decode cost dominates: schemas version with explicit field ordinals, builders avoid intermediate allocations, and promise pipelining cuts round trips. Use it for hot internal data planes; keep gRPC plus Protobuf for polyglot service surfaces where tooling breadth matters more than raw decode speed.

## 3. Live Probing with grpcurl
- `grpcurl -plaintext billing:50051 list` enumerates deployed services for drift checks.
- `grpcurl -plaintext billing:50051 describe billing.v1.InvoiceService` dumps the live method set.
- Any method present live but absent from the IDL is a contract violation, not a feature.

## 4. Common Pitfalls
- Reusing a deleted field number for an unrelated type, which resurrects ghost data in old clients.
- Returning free-text errors without status codes, forcing clients to string-match failure modes.
- Retrying non-idempotent methods, which turns one outage into duplicate charges.
