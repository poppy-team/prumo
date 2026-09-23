# Asset Reference Research — Verification Checklist

## 1. Brief Completeness & Source Sweep
- [ ] Every asset need states type, quantity, technical budget, style anchors, and placeholder slot.
- [ ] Trusted sources are swept in documented order before any aggregator or search-engine find.
- [ ] Each candidate logs source URL, author, stated license, and file hash.

## 2. License Matrix & Forbidden Terms
- [ ] Matrix classifies every candidate as approved, conditional, or forbidden with rationale.
- [ ] CC0 and CC-BY 4.0 approvals power commercial builds; CC-BY-SA passes only with compatibility review.
- [ ] NC, ND, unknown, and ripped-game-content licenses are forbidden in commercial paths with zero exceptions.

## 3. Placeholder Mapping & Technical Fit
- [ ] Every grey-box placeholder pairs with a replacement; orphan placeholders are absent.
- [ ] Pivots, tile sizes, audio loop points, and font glyph coverage for pt-BR and en-US verify per mapping.
- [ ] Mismatches resize the brief explicitly; silent stretching at import is absent.

## 4. Attribution Pack & Vendor Archives
- [ ] CREDITS.md entries plus in-game credits lines carry title, author, source URL, license, and modification notes.
- [ ] Source archives with SHA-256 hashes live under `assets/vendor/` for re-verification.
- [ ] Re-downloaded files with differing hashes are treated as new candidates, not silent swaps.

## 5. Coverage & Verification Evidence
- [ ] Placeholder coverage reaches 100% of briefed needs.
- [ ] 100% of shipped assets carry license proof with zero forbidden entries in commercial paths.
- [ ] `scripts/verify.sh` executes with exit code 0.
