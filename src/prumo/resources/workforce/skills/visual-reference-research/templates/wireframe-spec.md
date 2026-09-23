# Visual Direction Specification

## Brief
- Product: Prumo workspace navigation
- Adjectives: calm, clinical, editorial
- Screens: activity stream, Goals index, and detail panel
- Constraints: WCAG AA, existing brand accent `#1D4ED8`, light and dark themes

## Recommended Direction
- Pattern: airy editorial with a compact information rail.
- Source count: 12 full-viewport references captured on 2026-09-22.
- Rationale: strongest alignment with calm and clinical; the runner-up is dense-utilitarian.

## Token Draft
| Role | Value | Evidence |
|---|---|---|
| Text | `#0F172A` | 15.6:1 on white |
| Accent | `#0E7490` | 4.6:1 on white |
| Body | 16 px / 1.5 | readable at 360 px |
| Spacing | 8 px base | consistent across references |
| Radius | 8 px | restrained component language |

## Exclusions
- Reference 07: body contrast 3.2:1, below AA; do not use.
- Reference 11: playful rounded geometry conflicts with the clinical brief.

## Handoff
- Build the token draft in the design-token workflow and test keyboard focus visibility before implementation.
