# Icon Library Engineering & Design System Integration Recipe

## Purpose
Systematic design, vector optimization, token integration, and multi-framework distribution of an enterprise iconography library adhering to pixel grid constraints, optical weight balancing, and WCAG accessibility standards.

## Preconditions
- Visual design language and brand guidelines defined.
- Master coordinate grid specified (canonical: $24 \times 24$ px canvas with $2$ px keyline padding and $2$ px uniform stroke).

## Required Inputs
- Icon list with domain definitions and semantic aliases.
- Target framework requirements (React, Vue, Web Components, raw SVG symbols).

## Step DAG & Dependencies

1. **Establish Geometric Grid & Optical Alignment Keys** (`grid-specification`)
   - **Role:** `svg-artist`
   - **Skills:** `icon-system`
   - **Input:** Base pixel grid spec and design language
   - **Output:** Grid geometry and key shape guidelines
   - **Evidence Required:** `review`
   - **Dependencies:** None (Entry step)

2. **Vector Path Crafting & Optical Sizing** (`vector-crafting`)
   - **Role:** `svg-artist`
   - **Skills:** `svg-engineering`, `icon-system`
   - **Input:** Icon taxonomy and grid geometry
   - **Output:** Raw SVG vector icon glyphs
   - **Evidence Required:** `review`
   - **Gates:** `grid-conformance`
   - **Dependencies:** `grid-specification`

3. **Path Simplification & SVGO Bytecode Optimization** (`path-optimization`)
   - **Role:** `svg-artist`
   - **Skills:** `svg-engineering`
   - **Input:** Raw SVG glyphs
   - **Output:** Normalized, zero-overhead SVG assets
   - **Evidence Required:** `test`
   - **Gates:** `path-optimization`
   - **Dependencies:** `vector-crafting`

4. **Package Typed UI Components & Token Integration** (`component-packaging`)
   - **Role:** `design-system-engineer`
   - **Skills:** `design-tokens`, `icon-system`
   - **Input:** Optimized SVGs and design tokens
   - **Output:** Component packages & metadata catalog
   - **Evidence Required:** `test`
   - **Dependencies:** `path-optimization`

5. **Audit Accessibility & Multi-size Visual Conformance** (`accessibility-audit`)
   - **Role:** `accessibility-reviewer`
   - **Skills:** `icon-system`
   - **Input:** Component packages and rendered test matrix
   - **Output:** Iconography a11y audit report
   - **Evidence Required:** `test`
   - **Gates:** `a11y-audit`
   - **Dependencies:** `component-packaging`

## Exit Gates & Verification
- **`grid-conformance`**: 100% of vertices snap to grid or exact half-pixels; bounding box conforms to keyline shapes (circle, square, landscape, portrait).
- **`path-optimization`**: Zero extraneous `<g>` wrapper tags, no editor metadata (Inkscape/Illustrator), `fill="none"`, `stroke="currentColor"`.
- **`a11y-audit`**: Every component properly handles decorative (`aria-hidden="true"`) vs semantic (`role="img"` with title) contexts.
