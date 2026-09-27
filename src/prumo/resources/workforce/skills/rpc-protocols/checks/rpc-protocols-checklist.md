# RPC & Message Protocols — Verification Checklist

## 1. IDL Modeling & Toolchain
- [ ] Service is authored in a checked-in IDL with package version, language options, and streaming modes declared.
- [ ] Stubs are generated with pinned `buf generate`; hand-edited generated code is absent.
- [ ] Toolchain pins for buf, protoc, grpcurl, and capnp where used are recorded.

## 2. Numbering & Evolution Discipline
- [ ] No field or method number is reused or repurposed; removed fields are `reserved` with reason comments.
- [ ] Changes within a major version are purely additive; renames and type changes ship as a new package version.
- [ ] A v2 package coexists with v1 under a documented 90-day migration window.

## 3. Error Contract & Retry Policy
- [ ] Every failure maps to a canonical gRPC status code with structured `google.rpc.Status` details.
- [ ] Each method is labeled idempotent or non-idempotent; retries apply only to idempotent methods.
- [ ] Retry policy uses capped exponential backoff with jitter: base 100 ms, max 3 attempts.

## 4. Time, Size & Streaming Budgets
- [ ] Per-method deadlines are set and documented, for example 800 ms reads and 5 s exports.
- [ ] Responses over 256 KB paginate or stream; unary blobs above the ceiling are absent.
- [ ] Streaming choices use unary, server-streaming, or bidirectional modes with backpressure notes.

## 5. Compatibility & Performance Evidence
- [ ] `buf lint` and `buf breaking` against main report zero findings.
- [ ] `grpcurl` live listing confirms the deployed surface matches the IDL.
- [ ] Load test at 500 RPS shows p99 under 120 ms on golden methods.
- [ ] v1 clients replay cleanly against v2 servers and `scripts/verify.sh` exits 0.
