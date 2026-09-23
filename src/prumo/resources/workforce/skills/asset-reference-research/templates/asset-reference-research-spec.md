# Asset Reference Research — Deliverable Specification

## 1. Brief & Project Status
- **Project**: Lumen Drift vertical slice, commercial release planned, NC and ND forbidden
- **Needs briefed**: 24 rows across sprites, tiles, audio, fonts, and UI, frozen 2026-09-08
- **Budgets**: 512 px hero sprites, 256 px tiles on 16 px grid, props under 5k triangles, audio 48 kHz loops
- **Locales**: pt-BR and en-US glyph coverage required for both fonts

## 2. Source Sweep Results
- **Kenney.nl**: 11 candidates from platformer-art-deluxe and space-shooter-redux, all CC0 1.0
- **OpenGameArt**: 9 candidates, 7 CC0, 2 CC-BY 4.0 with authors confirmed on profile pages
- **Poly Pizza**: 5 low-poly prop candidates, all CC0 1.0
- **Freesound**: 4 loop candidates, 3 CC0, 1 CC-BY 4.0 by author fieldrecordist with profile URL logged
- **Quarantined**: 2 aggregator re-uploads rejected for missing origin proof

## 3. License Matrix Summary
- **Approved**: 27 candidates, CC0 1.0 and CC-BY 4.0 with attribution lines prepared
- **Conditional**: 1 CC-BY-SA 4.0 track reviewed and rejected for share-alike risk in the proprietary bundle, replaced with a CC0 alternate
- **Forbidden**: 0 shipped, 2 quarantined re-uploads plus 1 NC track excluded before mapping
- **Matrix file**: .prumo/assets/license-matrix-2026-09-10.csv with 30 rows and rationale per row

## 4. Placeholder Mapping
- **Coverage**: 24 of 24 briefed needs mapped, zero orphan placeholders
- **Fit checks**: pivots verified on 6 sprite rows, 16 px tile grid verified on 8 tileset rows, loop points verified on 4 audio rows, pt-BR glyph coverage verified on both fonts
- **Resized brief rows**: 2, namely UI panel nine-slice margins widened from 8 to 12 px with producer sign-off

## 5. Attribution Pack
- **CREDITS.md**: 27 entries with title, author, source URL, license identifier, and modification notes
- **In-game credits**: matching 27 lines under Extras plus Credits screen, verified in build 0.4.12
- **Vendor archives**: 27 archives under assets/vendor/ with SHA-256 hashes listed in .prumo/assets/vendor-hashes-2026-09-10.log

## 6. Verification Evidence
- Sweep log at .prumo/assets/sweep-2026-09-10.log with URLs, authors, licenses, and hashes
- Commercial-path scan: 0 forbidden licenses across the shipping asset folder
- `scripts/verify.sh` exit code 0 on 2026-09-10
