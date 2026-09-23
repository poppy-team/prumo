# Information Architecture (IA) & Navigation Strategy: Technical Reference Guide

## 1. The Four Foundational Systems of IA (Rosenfeld & Morville)

Information Architecture is organized into four interconnected systems:
1. **Organization Systems**: How content is structured and categorized (hierarchical, hyperlinked, database-driven, chronological, topical).
2. **Labeling Systems**: How information is represented to users (clear terminology, consistent tone, iconographic signifiers).
3. **Navigation Systems**: How users browse and move through content (global, local, contextual, supplemental/sitemap).
4. **Search Systems**: How users discover information by issuing queries (indexing, tokenization, facets, auto-complete, zero-result handling).

---

## 2. Structural Hierarchy: Depth vs. Breadth

```
Shallow & Broad (Ideal: 5-7 items, max 3 levels deep)
[Root] ──► [Cat 1] ──► [Item]
       ──► [Cat 2] ──► [Item]
       ──► [Cat 3] ──► [Item]

Deep & Narrow (Anti-Pattern: Buries content, high cognitive recall burden)
[Root] ──► [Level 1] ──► [Level 2] ──► [Level 3] ──► [Level 4] ──► [Item]
```

### 2.1 The "3-Click Rule" Pragmatic Reality
While modern UX research indicates users will click more than 3 times if information scent is strong, high tree depth increases error rates and backtracking. Maintain high **information scent**: every choice must clearly indicate what content lies ahead.

---

## 3. Navigation Typologies

### 3.1 Global Navigation
Persistent across every screen. Provides macro context and brand identity. Typically houses high-frequency workflows (Dashboard, Projects, Analytics, Settings).

### 3.2 Local & Contextual Navigation
- **Local Navigation**: Tab bars, secondary sidebars, or sub-headers reflecting the current sibling tree.
- **Contextual Navigation**: Inline hyperlinks, "Related Articles", "Frequently Bought Together", or bidirectional cross-entity links.

### 3.3 Breadcrumb Trails
Conforms to W3C ARIA Authoring Practices Guide (APG):
```html
<nav aria-label="Breadcrumb">
  <ol class="breadcrumbs">
    <li><a href="/">Home</a></li>
    <li><a href="/catalog">Catalog</a></li>
    <li><a href="/catalog/sensors">Sensors</a></li>
    <li><span aria-current="page">Temperature Sensor V2</span></li>
  </ol>
</nav>
```

---

## 4. IA Validation Methodologies

1. **Card Sorting**:
   - *Open Sort*: Users organize unlabeled cards into custom buckets; uncovers organic user mental models.
   - *Closed Sort*: Users sort cards into pre-defined categories; tests categorization clarity.
2. **Tree Testing**:
   - Strips all visual design, styling, and search to evaluate purely the text-based hierarchy.
   - Measures: *Directness* (% of users who found the target without backtracking), *Success Rate* (% of users who found the correct destination), and *Time on Task*.
