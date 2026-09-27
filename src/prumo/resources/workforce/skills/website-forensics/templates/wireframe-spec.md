# Website Forensics Teardown

## Scope
- Sites: public pricing pages from three competitors
- Capture date: 2026-09-23
- Viewports: 360, 768, 1280, and 1920 px
- Questions: container grid, type scale, pricing state, and focus treatment

## Measurements
- Grid: 1140 px maximum, 12 columns, 24 px gutter, single column below 768 px.
- Type: H1 40/48 px, body 16/24 px, 1.25 scale ratio, `font-display: swap`.
- Color: accent `#0066FF` on white at 4.5:1; body text remains above AA.
- States: hover, focus, loading, empty, and error captured for the primary flow.

## Cautions
- The reference error state uses color alone; do not copy it.
- No fonts, images, or source code are redistributed; this teardown records measurements and patterns only.
- Inspection stopped when a site returned HTTP 429; no anti-bot bypass was attempted.

## Recommendation
Keep our 1.333 type scale and adopt the reference focus-ring treatment in a separate accessibility ticket.
