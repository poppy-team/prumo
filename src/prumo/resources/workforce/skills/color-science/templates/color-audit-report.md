# Color Audit Report

**Date:** YYYY-MM-DD  
**Auditor:** [Name/Agent]  
**Target Standard:** [WCAG 2.2 AA / APCA]  

## 1. Palette Consistency Analysis
| Hue Scale | Lightness Lock (L) | Max Chroma (C) | Notes |
|-----------|--------------------|----------------|-------|
| Blue      | e.g., 500 = 0.62   | 0.25           |       |
| Red       | e.g., 500 = 0.62   | 0.28           |       |

## 2. Semantic Contrast Matrix
| Foreground Token | Background Token | Calculated Ratio (WCAG) | APCA Lc | Pass/Fail |
|------------------|------------------|-------------------------|---------|-----------|
| `text-primary`   | `surface-base`   | x.x:1                   | xx      | ✅ / ❌    |
| `text-inverse`   | `action-primary` | x.x:1                   | xx      | ✅ / ❌    |
| `border-focus`   | `surface-base`   | x.x:1                   | xx      | ✅ / ❌    |

## 3. Gamut Mapping Verification
- **Wide Gamut Used:** [Yes/No] (e.g., Display P3 via OKLCH)
- **Fallback Strategy:** [CSS `@supports`, Relative Color Syntax, Build Tool]
- **Notes on Clipping:** [Any noticeable chroma clipping in sRGB mode?]

## 4. Remediation Items
- [ ] Issue 1...
