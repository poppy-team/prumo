# Responsive Design Checklist

## 1. Content-First Breakpoints
- [ ] Breakpoints are based on content flow, not specific device dimensions.
- [ ] Layout maintains readability and hierarchy between breakpoints.
- [ ] No arbitrary media queries tied to specific phone/tablet models.

## 2. Fluid Design & Typography
- [ ] `clamp()` is used for fluid typography scaling.
- [ ] Spacing relies on relative units (`rem`, `em`) and fluid functions.
- [ ] Viewport units (`vw`, `vh`, `dvh`, `svh`, `lvh`) are used appropriately for layout sizing.

## 3. Container Queries
- [ ] Component adaptivity uses `@container` instead of viewport `@media` queries where applicable.
- [ ] Container elements define `container-type: inline-size`.
- [ ] Components are thoroughly tested in various container contexts.

## 4. Layout & Grid
- [ ] Mobile: 4-column grid, minimum 16px margins.
- [ ] Tablet: 8-column grid, minimum 24px margins.
- [ ] Desktop: 12-column grid, minimum 32px margins.
- [ ] Wide: Constrained max-width container with centered content.

## 5. Responsive Images & Media
- [ ] `srcset` and `sizes` attributes are implemented for resolution switching.
- [ ] `<picture>` element is used for art direction when necessary.
- [ ] Next-gen formats (WebP/AVIF) are provided with fallbacks.
- [ ] `loading="lazy"` is applied to below-the-fold images.
- [ ] `fetchpriority="high"` is applied to LCP (Largest Contentful Paint) images.

## 6. Interaction & Accessibility
- [ ] Touch targets are at least 44x44px.
- [ ] Hover styles are wrapped in `@media (hover: hover)`.
- [ ] Logical properties (`margin-inline`, `padding-block`) are used instead of physical directions.
- [ ] Layout reflows successfully at 400% zoom (320px equivalent) without horizontal scrolling (WCAG 1.4.10).

## 7. Progressive Enhancement
- [ ] Core content and functionality work without CSS or JavaScript.
- [ ] Advanced CSS features are wrapped in `@supports` queries when fallbacks are needed for target browsers.
- [ ] Print stylesheets are provided or `@media print` is implemented.
