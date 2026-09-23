# Visual Quality Assurance (Visual QA) Checklist

## 1. Spatial Grid & Spacing Fidelity
- [ ] **4pt / 8pt Baseline Grid**: All paddings, margins, and gap properties snap strictly to 4px/8px design token steps.
- [ ] **Zero Hardcoded Off-Grid Spacing**: No arbitrary values (`11px`, `17px`, `23px`) in component styles.
- [ ] **Optical Alignment**: Asymmetric icons (arrows, play buttons, badges) are optically aligned relative to text baselines.
- [ ] **Container Padding**: Consistent horizontal gutters preserved across screen edges.

## 2. Typography & Vertical Rhythm
- [ ] **Font Family & Fallbacks**: Correct font loaded (`Inter`, `Roboto`, system stack); fallback stack does not cause layout shifts (CLS).
- [ ] **Typographic Scale**: Font sizes, line heights, and weights match Figma redlines.
- [ ] **Text Overflow Handling**: Long strings wrap gracefully or truncate cleanly (`text-ellipsis`, `line-clamp`) without clipping.
- [ ] **Letter Spacing**: Heading and small-caps tracking match token values.

## 3. Colors, Elevation & Surfaces
- [ ] **Token Parity**: Surface backgrounds, text colors, and borders resolve directly to semantic design tokens.
- [ ] **Borders & Dividers**: Border widths (1px, 2px), radii, and colors match spec across all sides.
- [ ] **Elevation & Shadows**: Multi-layered box shadows match design tokens for blur, spread, offset, and opacity.

## 4. Assets & Screen Density
- [ ] **Vector Iconography**: All icons and symbols render as sharp SVGs without pixelation on high-DPI (Retina) screens.
- [ ] **Image Sharpness**: Raster graphics use `srcset` with `@2x` assets for high-DPI displays.
- [ ] **Aspect Ratio**: Images preserve correct aspect ratios with `object-fit: cover` or `contain` without distortion.

## 5. Responsive Layout Stability
- [ ] **No 320px Overflow**: Page reflows cleanly without horizontal scrollbars at 320px width.
- [ ] **Breakpoint Adaptability**: Grid columns and flex containers transition smoothly across mobile, tablet, and desktop breakpoints.
