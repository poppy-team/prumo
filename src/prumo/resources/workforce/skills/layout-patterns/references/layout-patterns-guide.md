# Layout Patterns Guide

## 1. Grid Systems
A grid system provides spatial rhythm and logical structure to information design, grounded in principles established by designers like Josef Müller-Brockmann.

### 1.1 Column & Modular Grids
- **Column Grids**: Subdivide vertical space into standard units (e.g., 12 columns). Excellent for general web layout.
- **Modular Grids**: Create a matrix of cells (rows × columns) for complex data, dashboards, and image galleries.
- **Hierarchical Grids**: Used for asymmetric, organically grouped content with disparate proportions.

### 1.2 Mathematical Foundations
- **Golden Ratio ($\varphi$)**: Two quantities are in the golden ratio if their ratio is the same as the ratio of their sum to the larger of the two quantities.
  $$ \varphi = \frac{1 + \sqrt{5}}{2} \approx 1.6180339887 $$
- **Rule of Thirds**: A simplified compositional approach dividing the viewport into a 3×3 matrix. Primary focal points reside at the intersections.

## 2. Composition Principles

### 2.1 Viewing Patterns
Cognitive psychology dictates how users scan interfaces based on content density:
- **Z-Pattern**: Optimal for landing pages and marketing material. The eye traces a horizontal line across the top, diagonally down to the bottom left, and horizontally across the bottom right.
- **F-Pattern**: Characteristic of text-heavy blocks (articles, search results). High fixation at the top and left edges, dropping off aggressively as the eye moves right and down.
- **Gutenberg Diagram**: Splits the screen into 4 quadrants. Primary optical area (top-left) to Terminal area (bottom-right).

## 3. Layout Primitives (Agnostic Layout Components)
Inspired by "Every Layout" (Heydon Pickering & Andy Bell):

*   **Stack**: Injects consistent vertical margin (`gap` or `margin-block-start`) between adjacent sibling elements.
*   **Cluster**: Aligns elements horizontally wrapping onto multiple lines as space decreases.
*   **Sidebar**: A two-column layout with one column of fixed or intrinsic width, and another taking up the remaining space.
*   **Center**: A container setting an intrinsic max-width and utilizing `margin-inline: auto` to center content.
*   **Switcher**: A flexbox layout that switches from a horizontal row to a vertical column once a threshold is crossed (`calc((var(--threshold) - 100%) * 999)` technique).
*   **Reel**: A horizontally scrollable container, often with `scroll-snap-type` applied for carousel-like interfaces.
*   **Cover**: Vertically centers a principal element, dynamically pushing a header and footer to the extremities.

## 4. Responsive Strategies
- **Container Queries (`@container`)**: Allows components to adapt based on their parent container's width, ensuring portability.
- **Fluid Sizing**: Utilizing `clamp(MIN, VAL, MAX)` to scale typography and spacing linearly between minimum and maximum viewports.
- **Content-First Breakpoints**: "Start with the small screen first, then expand until it looks like shit. TIME FOR A BREAKPOINT!" — Stephen Hay.

## References
- *Grid Systems in Graphic Design* — Josef Müller-Brockmann
- *Every Layout* — Heydon Pickering & Andy Bell
- *Responsive Web Design* — Ethan Marcotte
- *CSS Grid Layout Module Level 2 (Subgrid)* — W3C Standards
