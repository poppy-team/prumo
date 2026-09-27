# Responsive Design Guide

## 1. Container Queries
Container queries (`@container`) allow components to adapt to their parent container's size rather than the viewport. This is essential for reusable components that might be placed in a sidebar or the main content area.
- **Requirement:** The parent must have a container type defined (e.g., `container-type: inline-size;`).
- **Usage:** `@container (min-width: 400px) { ... }`

## 2. Fluid Design
Fluid design uses mathematical functions to scale values smoothly between a minimum and maximum bound.
- **`clamp(min, preferred, max)`:** Ideal for typography and spacing.
  - Example: `font-size: clamp(1rem, 2vw + 1rem, 2rem);`
- **Viewport Units:** Use modern viewport units like `dvh` (dynamic viewport height) to account for mobile browser UI changes.

## 3. Content-First Breakpoints
Do not hardcode breakpoints to specific device dimensions (e.g., 320px, 768px, 1024px). Instead, add a breakpoint when the content becomes unreadable or the layout breaks.

## 4. Responsive Images
Serve the right image to the right device to save bandwidth and improve performance.
- **Resolution Switching:** Use `srcset` and `sizes` to let the browser choose the best image based on viewport width and pixel density.
- **Art Direction:** Use `<picture>` and `<source>` to crop or change the image entirely at different breakpoints.

## 5. Viewport Strategies
General guidance for grid systems:
- **Mobile (320-767px):** 4-column grid, 16px margins, touch targets ≥44×44px.
- **Tablet (768-1023px):** 8-column grid, 24px margins.
- **Desktop (1024-1439px):** 12-column grid, 32px margins.
- **Wide (1440px+):** Max-width container, centered.

## 6. Touch vs Pointer
Prevent hover states from sticking on touch devices by wrapping hover styles in media queries:
```css
@media (hover: hover) and (pointer: fine) {
  .button:hover { background-color: var(--hover-color); }
}
```

## 7. Logical Properties
Use logical properties to support Left-to-Right (LTR) and Right-to-Left (RTL) languages seamlessly:
- `margin-inline-start`, `margin-inline-end`
- `padding-block-start`, `padding-block-end`
- `inline-size`, `block-size`
