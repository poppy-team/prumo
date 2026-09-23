# Wireframing & Layout Prototyping Checklist

## 1. Information Hierarchy & Grid Scaffolding
- [ ] **Content Priority**: Primary user action is immediately obvious above the fold; secondary content chunked logically.
- [ ] **12-Column Responsive Grid**: Layout adheres to standard column grids (4-col mobile, 8-col tablet, 12-col desktop).
- [ ] **Grayscale Focus**: Blueprint uses neutral grayscale tones without visual styling bias.
- [ ] **Scannability**: Text-heavy layouts leverage F-patterns; landing views leverage Z-patterns.

## 2. Accessibility Landmarks & Structure
- [ ] **HTML5 Semantic Landmarks**: All content resides within `<header>`, `<nav>`, `<main>`, `<aside>`, or `<footer>`.
- [ ] **Single Main Region**: Exactly one `<main>` landmark exists per view.
- [ ] **Logical Tab Order**: Interactive controls flow in natural reading order without unexpected tab jumps.
- [ ] **Heading Structure**: Sequential heading hierarchy (`h1` -> `h2` -> `h3`) without skipping levels.

## 3. Component States & Edge Cases
- [ ] **Default State**: Realistic data volume with long strings and edge cases.
- [ ] **Loading State**: Skeleton geometry matches populated component dimensions to prevent layout shifts (CLS).
- [ ] **Empty State**: Clear explanation of why content is missing and actionable button to create or import data.
- [ ] **Error State**: Inline or full-page error boundary with retry mechanisms.

## 4. Engineering Handoff Annotations
- [ ] **Responsive Breakpoints**: Column collapse and element stacking rules defined for $\le 480\text{px}$, $\le 768\text{px}$, and $\ge 1200\text{px}$.
- [ ] **Truncation Rules**: Single-line vs multi-line clamp behavior documented for dynamic text fields.
- [ ] **Touch Target Safety**: Mobile touch zones maintain minimum $44 \times 44\text{ px}$.
