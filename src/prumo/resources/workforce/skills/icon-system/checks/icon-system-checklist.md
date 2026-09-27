# Icon System Validation Checklist

## Grid & Geometry
- [ ] Base grid is defined (e.g., 24×24px).
- [ ] Live area (padding) is established (e.g., 20×20px).
- [ ] Keylines are defined (Circle, Square, Vertical Rectangle, Horizontal Rectangle).
- [ ] Icons align to keylines for consistent optical weight.
- [ ] Optical adjustments are applied (e.g., triangles breaking grid by 1px).

## Styling
- [ ] Stroke width is mathematically consistent across the library (e.g., 2px).
- [ ] Corner radius (border-radius) is consistent.
- [ ] Cap and join styles are uniformly set (e.g., `stroke-linecap="round"`, `stroke-linejoin="round"`).
- [ ] No mixing of solid and outlined styles within the same semantic tier.

## Naming & Categorization
- [ ] Naming follows strict `{category}/{action-or-object}` structure.
- [ ] File names are entirely lowercase kebab-case.
- [ ] Icons are placed in standard categories (Navigation, Action, Status, Content, Communication, Social, Media, File, Device, Editor, Toggle, Alert).

## Technical & SVG Optimization
- [ ] SVGs are processed with SVGO or similar optimization tool.
- [ ] Unnecessary groups (`<g>`), `<defs>`, and design tool metadata are removed.
- [ ] `<path>` data is minified.
- [ ] Stroke/Fill colors use `currentColor` where dynamic styling is required.

## Accessibility (A11y)
- [ ] Decorative icons are marked with `aria-hidden="true"`.
- [ ] Informative icons include appropriate text alternatives (e.g., `aria-label`, `title`).
- [ ] No icon acts as the sole indicator of status or action without fallback context.

## Governance & Distribution
- [ ] Semantic versioning rules are documented.
- [ ] Distribution method (NPM, Sprite, Components) is defined.
- [ ] Deprecation policy is established (warning period before removal).
