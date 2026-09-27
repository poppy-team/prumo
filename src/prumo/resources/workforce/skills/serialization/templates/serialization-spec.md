# Serialization Schema Specification Template

## 1. Schema Overview
- Domain: billing events for OrdersAPI, owner billing-platform, contact billing@acme.example
- Sources: proto/billing/v1/order_paid.proto, proto/billing/v1/order_refunded.proto, codegen protoc 26.1 with Go, Python, TypeScript plugins
- Compatibility: v1 readers back to 2025-11 release must decode all v1.4 payloads; major v2 planned for 2026-12
- Budgets: event envelope under 512 bytes p99, decode under 25 microseconds p99 on c6i.large class runners

## 2. Message Inventory
- OrderPaid v1.4: 5 fields, golden SHA-256 9f31c4..., p99 size 96 bytes Protobuf, 188 bytes JSON fallback
- OrderRefunded v1.2: 7 fields including reason enum with UNKNOWN zero, p99 size 132 bytes Protobuf
- Cache profile payloads use MessagePack with sorted keys for session snapshots averaging 1.8 KB
- Debug and webhook payloads use JSON with string IDs and integer cents, never floats

## 3. Evolution & Compatibility Proof
- buf breaking v1.3 → v1.4: zero breaking, one addition (PaymentMethod method = 5) with UNRECOGNIZED handling in all runtimes
- Oldest reader (billing-worker 1.1.0) decodes v1.4 golden fixtures ignoring field 5; newest reader defaults missing field 5 to UNKNOWN
- Retired numbers 6 and 7 reserved in both messages; reuse audit clean on 2026-09-20

## 4. Test & Benchmark Evidence
- Fuzz round-trip: 100,000 random OrderPaid cases, decode(encode(m)) equals m, zero mismatches, seed 20260923
- Cross-language corpus: Go encode → Python decode and reverse over 5,000 vectors, zero mismatches
- Benchmarks: Protobuf decode p99 11 microseconds, MessagePack 19 microseconds, JSON 44 microseconds; envelope p99 96 bytes
- verify.sh: exit code 0 on 2026-09-23 run by billing-platform pipeline job 334
