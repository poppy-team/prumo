# Layout Patterns Verification Checklist

## Grid and Proportions
- [ ] Base unit (e.g., 4px, 8px) is strictly adhered to for all margins, paddings, and gaps.
- [ ] Selected column grid (4/8/12) is consistently applied across varying viewports.
- [ ] Golden ratio ($\varphi \approx 1.618$) or Rule of Thirds is employed for primary visual divisions where applicable.

## Composition and Flow
- [ ] Primary optical areas (top-left) contain critical anchors (Gutenberg diagram).
- [ ] Z-Pattern or F-Pattern flow aligns with the intended cognitive scan path of the user.
- [ ] Visual weight is balanced asymmetrically or symmetrically based on the established wireframe.

## Layout Primitives
- [ ] Vertical spacing is handled by a `Stack` primitive (using `gap` or `margin-block`).
- [ ] Horizontal grouping is handled by a `Cluster` primitive (using `flex-wrap` and `gap`).
- [ ] Main layout structures utilize a `Sidebar` or `Grid` primitive.
- [ ] Centering is managed by a `Center` primitive (using intrinsic `max-width`).
- [ ] Layout primitives do NOT contain domain-specific logic or cosmetic styling (colors, borders, typography).

## Responsive Strategies
- [ ] Container queries (`@container`) are preferred over viewport media queries (`@media`) for component-level adjustments.
- [ ] Fluid typography and spacing use `clamp()` or relative units.
- [ ] Breakpoints are defined by content failure points, not specific device models.

## Code Quality
- [ ] Avoidance of magic numbers (e.g., `margin-top: 37px`).
- [ ] Usage of design tokens (e.g., `var(--space-md)`).
- [ ] No fixed absolute widths (`width: 400px`) unless justified by strict asset requirements.
