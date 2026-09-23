# Context Optimization Report Specification

## 1. Run Identity
- **Task**: Incident triage INC-4417, checkout service error spike
- **Loaded set**: 24,000 tokens; **target budget**: 12,000 tokens
- **Trailing mean**: 31,000 tokens per resolved goal over the last 10 triage runs

## 2. Per-Section Accounting

| Section | Before | Technique | After | Saved |
|---|---|---|---|---|
| app.log excerpt | 9,400 | Script dedup: 212 repeated frames to 1 plus count | 3,300 | 6,100 |
| Retrieved runbook chunks | 7,100 | Dedup 3 near-duplicate chunks, kept newest revision | 4,200 | 2,900 |
| Passing suite output | 4,200 | Summary: 214 passed, 0 failed, 41s; original at evidence/INC-4417/full-suite.log | 150 | 4,050 |
| Error signatures (6) | 2,100 | Untouched, verbatim | 2,100 | 0 |
| Acceptance criteria | 1,200 | Untouched, verbatim | 1,200 | 0 |
| **Total** | **24,000** | | **10,950** | **13,050 (54%)** |

## 3. Fidelity Checklist
- [ ] All 6 error signatures byte-identical (verified with diff against originals).
- [ ] Failing assertion from checkout_test.go lines 88-94 kept verbatim.
- [ ] Removal declarations present for log frames, doc chunks, and suite output.

## 4. Cache Annotations
- **Stable prefix**: 3,300 tokens (instructions + error schema + skill directives), byte-identical to run INC-4409.
- **Hit rate this run**: 85%; trailing mean 84%.
- **Volatile tail**: timestamps and request ids isolated after the prefix boundary.

## 5. Verification Evidence
- [ ] Optimized set addresses acceptance criteria; triage proceeded without reloads.
- [ ] Tokens per goal this run: 10,950 vs trailing mean 31,000.
- [ ] `scripts/verify.sh` exits 0.
