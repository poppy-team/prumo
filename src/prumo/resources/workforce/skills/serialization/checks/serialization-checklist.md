# Binary & Data Serialization — Verification Checklist

## 1. Schema Definition & Versioning
- [ ] Schemas live in versioned sources (.proto, JSON Schema) with owners, units, and changelog entries.
- [ ] Retired field numbers are reserved forever; no reuse appears in any schema revision.
- [ ] buf breaking (or equivalent) reports zero breaking findings against the previous released schema.
- [ ] Codegen plugins (protoc, plugins per language) are pinned to exact versions in CI.

## 2. Deterministic Canonical Encoding
- [ ] Map keys are sorted, field order is fixed, timestamps are int64 epoch millis in UTC.
- [ ] Golden fixture orders_golden_v3.bin encodes to a stable SHA-256 across runs and machines.
- [ ] No encoder iterates host-ordered maps or pointers into the byte stream.
- [ ] Custom binary layouts declare little-endian order with magic bytes and a version prefix.

## 3. Compatibility & Robust Decoding
- [ ] Readers ignore unknown fields and apply documented defaults for missing optional fields.
- [ ] Unknown enum values map to an explicit UNRECOGNIZED branch, never to a silent zero default.
- [ ] Decode limits enforced: recursion depth max 32, message cap 2 MB, UTF-8 validated on strings.
- [ ] Backward and forward compatibility proven by decoding new payloads with the oldest supported reader and vice versa.

## 4. Testing & Budgets
- [ ] Property fuzz of 100,000 random messages proves decode(encode(m)) equals m with zero mismatches.
- [ ] Cross-language corpus (Go vs Python minimum) round-trips with zero mismatches.
- [ ] Event envelope under 512 bytes p99; decode under 25 microseconds p99 on the CI runner class.
- [ ] scripts/verify.sh executes with exit code 0.
