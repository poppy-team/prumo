# Generative SVG Art Quality & Determinism Checklist

## 1. Mathematical & Structural Invariants
- [ ] **Deterministic Seed:** The artwork algorithm accepts an integer or string seed and produces a byte-for-byte identical output given the same input.
- [ ] **ViewBox Normalization:** Root SVG declares `viewBox="0 0 W H"` with appropriate aspect ratio (`preserveAspectRatio="xMidYMid meet"` or `"none"` where intended).
- [ ] **Scale Independence:** No hardcoded outer `width` / `height` pixels that break fluid responsive layouts.
- [ ] **Path Consolidation:** Repetitive procedural curves or strokes are concatenated into consolidated `<path d="...">` elements to minimize DOM nodes.

## 2. Filter Pipeline & Visual Integrity
- [ ] **Filter Bounding Box:** Filters specify extended bounds (`x="-20%" y="-20%" width="140%" height="140%"`) to eliminate displacement clipping at boundaries.
- [ ] **Color Interpolation:** All `<filter>` tags declare `color-interpolation-filters="sRGB"` for uniform perceptual blending across WebKit, Gecko, and Blink.
- [ ] **Fallback Degradation:** Visual artwork maintains aesthetic coherence if SVG filters are unsupported or disabled by high-contrast OS settings.
- [ ] **Zero Artifacts:** No unwanted seams, jagged lines, or clipping boxes on high-DPI displays.

## 3. DOM & Performance Budget
- [ ] **Element Count:** Total DOM nodes within the SVG strictly $\le 2,500$ elements.
- [ ] **File Size Budget:** Uncompressed SVG payload $\le 250$ KB ($\le 50$ KB gzipped).
- [ ] **Coordinate Precision:** Floating-point numbers rounded to at most 2 decimal places (`d="M12.34,56.78..."`) to minimize payload bloat.

## 4. Theme & Accessibility
- [ ] **Color Token Integration:** Stroke, fill, and gradients reference semantic CSS custom properties or OKLCH tokens rather than rigid hex literals.
- [ ] **Dark & Light Mode Adaptation:** Graphic maintains adequate contrast ratios against dark and light container backgrounds.
- [ ] **A11y Semantics:** 
  - Decorative art: `aria-hidden="true"` and `focusable="false"`.
  - Semantic/Infographic art: `role="img"` with `<title>` and `<desc>` elements matching internationalized context.
