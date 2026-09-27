# SVG Engineering Checklist

## 1. Structure and Optimization
- [ ] ViewBox is present and standard (e.g., `viewBox="0 0 24 24"` for icons).
- [ ] Hardcoded `width` and `height` attributes are removed (enabling CSS sizing).
- [ ] SVG processed through SVGO (removing editor metadata, empty groups, unnecessary whitespace).
- [ ] No raster images (base64 `image/png` or `image/jpeg`) embedded unless strictly required.
- [ ] Node coordinates utilize whole numbers wherever possible to prevent sub-pixel blurring.

## 2. Styling and Theming
- [ ] Single-color icons utilize `fill="currentColor"` or `stroke="currentColor"`.
- [ ] Presentation attributes (like `fill`, `stroke-width`) are hoisted or standardized where possible.
- [ ] CSS Custom Properties (variables) are used for multi-color themeable illustrations.

## 3. Accessibility (A11y)
- [ ] **Decorative Icons**: Include `aria-hidden="true"` and `focusable="false"`.
- [ ] **Informative Icons**: Include `role="img"`.
- [ ] **Informative Icons**: Include `<title>` with a unique `id`.
- [ ] **Informative Icons**: Include `<desc>` with a unique `id` if a detailed description is necessary.
- [ ] **Informative Icons**: Include `aria-labelledby="[title-id] [desc-id]"`.

## 4. Animation and Performance
- [ ] Complex procedural paths are optimized (fewer path nodes).
- [ ] Animations (CSS or SMIL) respect `prefers-reduced-motion` media queries.
- [ ] Sprites utilize `<symbol id="icon-name">` correctly for efficient `use` referencing.
