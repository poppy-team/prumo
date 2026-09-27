# Reference: Color Science for Design Systems

## Core Theory: OKLCH & OKLAB
**OKLCH** stands for Lightness, Chroma, and Hue in the Oklab color space.
- **Lightness (L):** 0 to 1 (or 0% to 100%). Perceptually uniform, meaning `L=0.5` looks exactly half-bright regardless of hue.
- **Chroma (C):** Technically unbounded but practically 0 to ~0.4 for standard displays. Measures the intensity/purity of the color.
- **Hue (H):** 0 to 360 degrees. 

**Advantage over HSL:** HSL is not perceptually uniform. `hsl(60, 100%, 50%)` (yellow) appears much brighter to the human eye than `hsl(240, 100%, 50%)` (blue), despite having the exact same lightness value in HSL. In OKLCH, all colors with `L=0.7` will have the exact same perceived brightness.

## Wide Gamut & Display P3
- **sRGB:** The legacy standard for the web.
- **Display P3:** Developed by Apple, offers ~25% more colors than sRGB, specifically in vibrant reds, greens, and oranges.
- **Gamut Mapping:** When a color specified in OKLCH exceeds the capability of a display, the browser algorithmically "maps" it down to the nearest displayable color.

## Contrast Algorithms

### WCAG 2.x Contrast
Based on relative luminance ($Y$). 
- **4.5:1** for regular text.
- **3.0:1** for large text (usually 18pt/24px normal, or 14pt/18.5px bold) and UI components.

### APCA (Accessible Perceptual Contrast Algorithm)
The future of WCAG 3.0. Based on human visual perception rather than simple math.
- Outputs Lightness Contrast ($L_c$).
- $L_c$ 90: Preferred for body text.
- $L_c$ 75: Minimum for regular text.
- $L_c$ 60: Minimum for large text.

## CSS Implementation Techniques

**Dynamic Variants with `color-mix`:**
```css
/* Mix primary color with 20% white in OKLCH space */
.surface-light {
  background: color-mix(in oklch, var(--primary) 20%, white);
}
```

**Relative Color Syntax:**
```css
/* Derive a new color by altering lightness of an existing one */
.hover-state {
  background: oklch(from var(--primary) calc(l - 0.1) c h);
}
```

## Further Reading
- [oklch.fyi](https://oklch.fyi) - Evaluator and visualizer.
- Evil Martians: "OKLCH in CSS: why we moved from RGB and HSL"
- W3C CSS Color Module Level 4
- APCA Documentation
