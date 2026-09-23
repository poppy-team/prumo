# Website Forensics — Technical Reference Guide

## 1. Core Concepts

**Scoped teardown**: at most 5 forensic questions answered with measurements — grid anatomy, type scale, color roles, interaction states — from public pages only. Open-ended browsing is not forensics.

**Respectful inspection**: browser devtools, throttled identified requests, robots.txt honored; paywalls, auth, and anti-bot measures are hard boundaries. The deliverable holds **patterns and measurements**, never redistributed fonts, images, or code.

**Viewport ladder**: measure at 360 / 768 / 1280 / 1920 px so breakpoints, container caps, and nav collapses are recorded, not guessed.

## 2. Patterns

- **Container + grid capture**: record max-width (e.g. 1200 px), column count, gutter (e.g. 24 px), and breakpoint where the grid collapses to single column.
- **Type role table**: family, weight, size px/rem, line-height per role (display/H1/body/caption); note subsetting (`latin` only) and `font-display: swap` strategy.
- **State inventory**: screenshot default/hover/focus/loading/empty/error for the primary flow; missing focus rings and unlabeled icon buttons go on the caution list.
- **Token mapping row**: each finding maps onto our scale (e.g. "their 8-pt rhythm matches our spacing base; their 6-px radius differs from our 8-px token — adopt or diverge explicitly").

## 3. Anti-Patterns

- Bulk-scraping content, media, or user data under a research pretext.
- Publishing findings with copied assets instead of measurements.
- Single-viewport conclusions ("the site uses 3 columns" — at which width?).
- Copying failing patterns (2.9:1 body contrast) because "the reference does it".

## 4. Worked Example

Target: competitor pricing page. Findings: container 1140 px max, 12-col grid collapsing at 768 px; H1 Inter 600 40 px/48 px, body 16 px/24 px; accent `#0066FF` on white at 4.5:1 (AA pass); FAQ accordion with visible `:focus-visible` rings; error state uses color alone (no icon/text prefix — flagged as caution). Mapped: type scale ratio 1.25 vs our 1.333 — recommend keeping ours; adopt their focus-ring treatment as an improvement ticket.

## 5. Verification Pointers

- Every claim carries a viewport width or a hex value; adjectives pair with numbers.
- Recompute one contrast ratio from sampled hex values; assert it matches the report.
- Confirm no redistributed assets in the deliverable (descriptions + measurements only).
