# Color Science Validation Checklist

## OKLCH Primitives
- [ ] All base palettes defined in OKLCH format.
- [ ] Lightness ($L$) values are strictly locked across all hues for the same step (e.g., all 500-level colors share $L=0.6$).
- [ ] Chroma ($C$) tapers appropriately at high ($L > 0.9$) and low ($L < 0.2$) lightness to avoid clipping.
- [ ] Gradients/interpolations utilize OKLAB space to avoid hue shifts.

## Semantic Tokens
- [ ] Semantic intents (`primary`, `success`, `danger`, `surface`, `text`) are mapped to primitives, not hardcoded.
- [ ] Dark mode scales invert lightness curves symmetrically (or use scientifically adjusted dark mode perceptual curves).

## Contrast & Accessibility
- [ ] Normal text against backgrounds meets WCAG 2.2 4.5:1 ratio (or APCA $L_c \ge 75$).
- [ ] Large text and UI components meet WCAG 2.2 3:1 ratio (or APCA $L_c \ge 60$).
- [ ] Focus indicators and borders meet non-text contrast requirements.

## CSS Implementation & Gamut
- [ ] Output utilizes CSS Module Level 4 syntax (e.g., `oklch(L C H)`).
- [ ] Wide-gamut (Display P3) colors are gracefully mapped for sRGB-only displays.
- [ ] Fallbacks exist for older browsers (if dictated by project requirements).
