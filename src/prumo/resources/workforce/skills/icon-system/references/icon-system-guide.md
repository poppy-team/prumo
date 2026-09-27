# Icon System Reference Guide

## 1. Grid and Keylines
A robust icon system relies on a consistent foundational grid.
- **Base Grid:** Typically 24×24px.
- **Padding:** 2px safe zone padding.
- **Live Area:** 20×20px. Icons should rarely break out of this area, except for optical alignment.

### Keylines
Keylines guide the general proportions of icons to ensure they carry the same visual weight.
- **Circle:** 20px diameter.
- **Square:** 18×18px.
- **Vertical Rectangle:** 16×20px.
- **Horizontal Rectangle:** 20×16px.

## 2. Optical Alignment
Geometric shapes of equal metric size do not always appear to have equal mass.
- **Triangles:** A triangle drawn to the exact height of a square will appear smaller. Extend the vertices of triangles by ~1px beyond the standard grid to balance the weight.
- **Circles vs Squares:** A circle must touch the absolute edges of the live area, whereas a square should be slightly smaller to compensate for its larger surface area.

## 3. Stroke vs. Fill
Consistency is the most important factor in an icon family.
- Decide early on a core style: Outlined (Stroke) or Solid (Fill).
- **Outlined Specs:** Standardize on a stroke width (e.g., 2px), cap style (`round` or `square`), and join style (`round`, `bevel`, `miter`).
- Mixing styles within the same context creates visual noise. Use filled icons specifically to denote active/selected states if the base is outlined.

## 4. Naming Convention
A scalable library must use a predictable namespace.
Format: `{category}/{action-or-object}` (kebab-case)
- **Categories:** `navigation`, `action`, `status`, `content`, `communication`, `social`, `media`, `file`, `device`, `editor`, `toggle`, `alert`.
- **Examples:**
  - `navigation/arrow-left`
  - `status/check-circle`
  - `file/document`

## 5. Accessibility
Icons are visual communication, but must be accessible to assistive technologies.
- **Decorative Icons:** purely visual, redundant, or paired with text. Always use `aria-hidden="true"`.
- **Informative Icons:** standalone icons that convey meaning or action. Use `aria-label="Action description"` and `role="img"`.
- **Rule of Thumb:** Never use icons as the *sole* indicator of important information without a text alternative or visible text fallback.

## 6. Versioning and Governance
Icon libraries should be versioned using Semantic Versioning (SemVer).
- **Patch (`1.0.x`):** Fixing an SVG path bug, optimizing an existing file without changing its appearance.
- **Minor (`1.x.0`):** Adding new icons to the library.
- **Major (`x.0.0`):** Removing icons, renaming existing icons, changing the overall grid, or changing the standard stroke width.
- Always provide deprecation warnings (e.g., via console warnings in React components) for at least one minor cycle before a major removal.
