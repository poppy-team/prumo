# SVG Engineering Reference Guide

## 1. The Power of `viewBox`
The `viewBox` attribute specifies the internal coordinate system of the SVG. 
Syntax: `viewBox="<min-x> <min-y> <width> <height>"`
For standard UI icons, the convention is `viewBox="0 0 24 24"`. Removing hardcoded `width` and `height` attributes and relying solely on the `viewBox` makes the SVG fluid and responsive, sizing perfectly to its parent container.

## 2. Using `currentColor`
Setting `fill="currentColor"` or `stroke="currentColor"` enables the SVG to inherit the computed CSS `color` property from its parent context. This is essential for theming (e.g., light mode/dark mode) and text-alignment matching.

## 3. Accessibility Standards
SVGs are not inherently accessible.
- **Decorative**: An icon used purely for visual flair next to text that already conveys the meaning.
  *Implementation*: `<svg aria-hidden="true" focusable="false" ...>` (focusable=false prevents IE11 bugs).
- **Informative**: An icon that stands alone (e.g., a magnifying glass button with no text).
  *Implementation*:
  ```html
  <svg role="img" aria-labelledby="icon-title" ...>
    <title id="icon-title">Search</title>
    <path ... />
  </svg>
  ```

## 4. SVGO Configuration for Engineers
SVGO (SVG Optimizer) is the standard tool for cleaning up SVGs.
A typical safe `.svgo.yml` configuration:
```yaml
plugins:
  - removeDoctype: true
  - removeXMLProcInst: true
  - removeComments: true
  - removeMetadata: true
  - removeEditorsNSData: true
  - cleanupAttrs: true
  - inlineStyles: true
  - minifyStyles: true
  - convertStyleToAttrs: true
  - cleanupIDs: true
  - removeRasterImages: false
  - removeUselessDefs: true
  - cleanupNumericValues: true
  - cleanupListOfValues: true
  - convertColors: true
  - removeUnknownsAndDefaults: true
  - removeNonInheritableGroupAttrs: true
  - removeUselessStrokeAndFill: true
  - removeViewBox: false      # CRITICAL: Always false
  - cleanupEnableBackground: true
  - removeHiddenElems: true
  - removeEmptyText: true
  - convertShapeToPath: true
  - moveElemsAttrsToGroup: true
  - moveGroupAttrsToElems: true
  - collapseGroups: true
  - convertPathData: true
  - convertTransform: true
  - removeEmptyAttrs: true
  - removeEmptyContainers: true
  - mergePaths: true
  - removeUnusedNS: true
  - sortAttrs: true
  - removeTitle: false        # CRITICAL for A11y
  - removeDesc: false         # CRITICAL for A11y
  - removeDimensions: true    # CRITICAL for Responsiveness
```

## 5. Advanced: SVG Filters and Procedural Generation
SVG provides built-in GPU-accelerated image processing capabilities:
- `<feTurbulence>`: Generates Perlin noise (great for textures).
- `<feDisplacementMap>`: Displaces pixels based on another input (glitch effects).
- `<feColorMatrix>`: Advanced color manipulation.
Procedural SVGs rely on programmatic generation of `<path d="..." />` attributes to create dynamic charts, data visualizations, and generative art.
