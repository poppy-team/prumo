# Binary & Data Serialization Reference Guide

## 1. Core Concepts

### 1.1 Why Determinism Matters
Caches hash payloads for keys, networks sign them for authenticity, games snapshot them for rollback.
If the same logical message can encode to different bytes, hashes flap, signatures fail intermittently,
and snapshots diverge. Determinism requires canonical rules: sorted map keys, fixed field order, single
timestamp representation, no pointer addresses. Test it directly: encode the same golden message 1,000
times across two machines and compare SHA-256 digests.

### 1.2 Tag-Length-Value and Varint Math
Protobuf encodes each field as tag (field number shifted 3 bits plus 3-bit wire type) followed by payload.
Varints use 7 payload bits per byte with a continuation bit: values under 128 take 1 byte, values under
16,384 take 2 bytes. Signed integers must use sint32/sint64 with ZigZag mapping (0 → 0, -1 → 1, 1 → 2) so
small negatives stay 1–2 bytes instead of 10. Size model: message bytes ≈ sum of tag bytes + varint or
fixed payload bytes; prefer packed repeated numeric fields to collapse N tags into one.

### 1.3 Endianness and Cross-Language Types
Network order is big-endian by convention (TCP headers, UUID bytes); most CPUs and Protobuf fixed32 are
little-endian. Hand-rolled formats must declare one order in the header and convert explicitly: Python
struct '<I' versus '>I' differs silently and correctly parses garbage the other way. Cross-language traps:
Go int is 64-bit, TypeScript numbers lose precision past 2^53 (use string or int64 wrappers for IDs and
cents), Rust u64 JSON-serializes as number at your peril. Fix widths in the schema: int32, int64, fixed64.

### 1.4 Schema Evolution Rules
Backward compatible: new reader accepts old data (add optional fields with defaults). Forward compatible:
old reader accepts new data (never require what old writers omit). Compatible changes: add optional field,
add enum value with UNRECOGNIZED handling, widen int32 to int64 on tolerant readers. Breaking changes:
renumber fields, change wire type, make optional required, delete values consumers switch on. buf breaking
encodes these rules as CI law; major schema versions gate the breaking remainder.

### 1.5 Format Selection
Protobuf: typed schemas, 3–10x smaller than JSON, fastest decode, needs codegen. MessagePack: schemaless,
compact, good for dynamic payloads with JSON-like ergonomics. JSON: universal and debuggable, 2–5x larger,
parse cost highest; acceptable for control planes, wrong for 60 Hz game packets. Zero-copy (FlatBuffers,
Cap'n Proto) mmap straight into structs for read-heavy caches at the cost of builder complexity.

## 2. Patterns and Anti-Patterns

| Area | Pattern (do) | Anti-Pattern (avoid) |
|---|---|---|
| Numbers | sint32 with ZigZag, fixed64 for IDs | int32 holding -1 encoded in 10 bytes |
| Timestamps | int64 epoch millis UTC | Local-time strings without offset |
| Enums | Zero reserved for UNKNOWN, UNRECOGNIZED branch | Exotic zero meaning plus silent default |
| Strings | UTF-8 validated, length-capped | Unbounded strings into fixed buffers |
| Maps | Sorted keys, bounded size | 100 k-entry map hashed nondeterministically |
| JSON | String IDs past 2^53, cents as integers | Float money and float equality checks |
| Custom binary | Magic, version, length prefix, CRC32 | Raw struct dump across architectures |

## 3. Code Example: Versioned Protobuf Event with Safe Evolution

```proto
syntax = "proto3";
package acme.billing.v1;

message OrderPaid {
  string order_id = 1;        // public id, e.g. ord_9f31
  int64 total_cents = 2;      // minor units, never float
  string currency = 3;        // ISO 4217, e.g. BRL
  int64 paid_at_millis = 4;   // UTC epoch millis
  PaymentMethod method = 5;   // new optional enum, defaults UNKNOWN

  enum PaymentMethod {
    PAYMENT_METHOD_UNKNOWN = 0;
    PAYMENT_METHOD_CARD = 1;
    PAYMENT_METHOD_PIX = 2;
  }

  reserved 6, 7;              // retired fraud_score, coupon_code numbers
  reserved "fraud_score", "coupon_code";
}
```

Adding method as an optional enum with an UNKNOWN zero keeps old readers working (they ignore field 5)
and new readers safe (missing field means UNKNOWN, never a guessed CARD). Retired numbers 6 and 7 can
never be reassigned, so rolling deploys mixing v1.3 and v1.4 binaries decode without corruption.
