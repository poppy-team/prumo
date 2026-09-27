# SVG Component Specification

**Component Name:** `[IconName]`
**Category:** `[Category e.g., UI, Illustration, Branding]`

## Dimensions
- **Base Grid:** `[e.g., 24x24]`
- **ViewBox:** `viewBox="0 0 [width] [height]"`

## Styling
- **Fill Mode:** `[currentColor | Multi-color variables]`
- **Stroke Width (if applicable):** `[e.g., 2px]`

## Accessibility Configuration
- **Type:** `[Decorative | Informative]`
- **Title (if informative):** `[Concise title]`
- **Description (if informative):** `[Detailed description or N/A]`

## Source Data
```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true" focusable="false">
  <!-- Paste optimized SVG paths here -->
</svg>
```

## Component Wrapper (Example: React)
```tsx
import React from 'react';

export const [IconName]Icon = ({ className = '', ...props }) => (
  <svg 
    xmlns="http://www.w3.org/2000/svg" 
    viewBox="0 0 24 24" 
    fill="currentColor" 
    className={`w-6 h-6 ${className}`}
    aria-hidden="true"
    focusable="false"
    {...props}
  >
    <!-- Paste paths -->
  </svg>
);
```
