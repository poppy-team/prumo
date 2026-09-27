# Visual Reference Research — Technical Reference Guide

## 1. Core Concepts

**Moodboard with provenance**: 8–15 full-viewport captures, each with source URL, capture date, and viewport size. Fragments without source context are excluded — a board you cannot cite is a board you cannot defend.

**Token extraction** converts impressions into measurements: dominant palette (≤5 hex values per reference), type roles (display/body/mono with families, weights, scale ratio), spacing base (4 vs 8 pt), and radius/shadow language. Each direction must show WCAG AA-passing body-text pairs before recommendation.

**Convergence**: cluster into 2–3 directions, score against the brief adjectives, recommend exactly one. **Exclusions log** records every rejection with a reason so the decision survives personnel change.

## 2. Patterns

- **Brief-before-board**: lock 3–5 adjectives + hard constraints (existing brand color `#1D4ED8`, AA minimum) before collecting; prevents cherry-picking.
- **Scale-ratio check**: divide display size by body size across breakpoints; consistent ratios (≈1.25–1.333) signal a systematic scale worth borrowing.
- **Contrast-first palette**: compute body-text contrast for every candidate pair with a standard checker; park failing pairs in the exclusions log.
- **Runner-up preserved**: keep the second direction summarized so pivots don't restart research.

## 3. Anti-Patterns

- Unsourced Pinterest/Dribbble crops with no URL or date.
- Recommending a palette whose body text fails AA (3.2:1 gray-on-white is the classic trap).
- Presenting three equal options with no recommendation ("pick your favorite").
- Extracting adjectives ("clean", "modern") with no token measurements attached.

## 4. Worked Example

Brief adjectives: calm, clinical, editorial. Twelve references cluster into dense-utilitarian (5), airy-editorial (5), playful-rounded (2). Airy-editorial recommended: palette `#0F172A` text on `#FFFFFF` (15.6:1), accent `#0E7490` (4.6:1 on white), Inter 16 px body with 1.333 display scale, 8-pt spacing, 8 px radius. Playful direction rejected: body pair `#9CA3AF` on `#F9FAFB` at 2.3:1 fails AA. Token draft handed off with contrast table.

## 5. Verification Pointers

- Count sourced references (≥8) and rationale sentences (one per kept reference).
- Recompute two contrast pairs independently; assert AA pass.
- Assert the deliverable names one direction and lists exclusions with reasons.
