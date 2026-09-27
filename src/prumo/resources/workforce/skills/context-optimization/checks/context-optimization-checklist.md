# Context Optimization — Verification Checklist

## 1. Measurement First
- [ ] Working set tokenized per section before any cut; largest sections identified.
- [ ] Target budget stated in writing (e.g. 24,000 loaded, 12,000 target).
- [ ] Critical facts enumerated from acceptance criteria before compression starts.

## 2. Dedup & Pruning
- [ ] Repeated frames, headers, and configs collapsed by script with kept/dropped counts.
- [ ] Sections tiered critical / supporting / noise; noise fully dropped.
- [ ] No critical-tier content paraphrased, truncated, or moved to a summary.

## 3. Summarization Fidelity
- [ ] Passing test output replaced by counts; failing assertions kept byte-identical.
- [ ] Every summary states what was removed and where the original lives.
- [ ] Error signatures, ids, and acceptance criteria verified verbatim post-compression.

## 4. Cache Discipline
- [ ] Immutable instruction/schema blocks annotated as cache candidates.
- [ ] Cacheable prefixes byte-stable across runs; hit rate recorded (target above 80%).
- [ ] Rewording a cacheable prefix invalidates and re-baselines the hit-rate record.

## 5. Verification Gates
- [ ] Token accounting shows per-section before/after and total savings percentage.
- [ ] Tokens-per-resolved-goal recorded against the trailing mean.
- [ ] Automated verification script `scripts/verify.sh` executes with exit code 0.
