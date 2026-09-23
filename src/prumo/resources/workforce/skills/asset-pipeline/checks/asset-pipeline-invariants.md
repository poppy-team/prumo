# Asset Pipeline Verification Checklist

## Source and Contract Validation
- [ ] Source bytes, metadata, target profile, and cooker version are recorded.
- [ ] Dimensions, codec, color space, and per-asset limits are validated before conversion.
- [ ] Over-budget or malformed assets fail at cook time with an actionable diagnostic.

## Deterministic Cooking
- [ ] Artifact identity hashes canonical source bytes, settings, target, and cooker version.
- [ ] Timestamps, absolute paths, iteration order, and random seeds do not change output.
- [ ] Two clean cooks of the same revision produce byte-identical artifacts.

## Artifact and Cache Safety
- [ ] Compressed outputs, mips or LODs, and memory estimates match target support.
- [ ] Concurrent workers publish the same artifact key through one atomic result.
- [ ] Failed or partial cooks never replace a valid cache entry.

## Hot Reload and Streaming
- [ ] Cooking, decoding, and disk I/O stay off the frame thread.
- [ ] Immutable resource revisions are swapped safely and old revisions retire after readers finish.
- [ ] Every streamed asset has a resident low-detail fallback and bounded bandwidth use.

## Budgets and Evidence
- [ ] Per-asset, per-level, and resident-memory budgets pass on every target tier.
- [ ] Determinism, hot-swap, streaming, and frame-time benchmarks are recorded.
- [ ] `scripts/verify.sh` exits with code 0 from the repository root.
