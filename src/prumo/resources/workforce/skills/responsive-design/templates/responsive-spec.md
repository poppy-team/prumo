# Responsive Component Specification

## Component Name: [Component Name]

### 1. Fluid Typography & Spacing
Define the fluid bounds for typography and spacing within this component.
- **Base Font Size:** `clamp([Min Size], [Preferred Fluid Size], [Max Size])`
- **Padding/Margin:** `clamp([Min Spacing], [Preferred Spacing], [Max Spacing])`

### 2. Container Queries
Describe how the component adapts to its container.
- **Container Name (Optional):** `[Container Name]`
- **Container Type:** `inline-size`
- **Breakpoints (Container-based):**
  - `@container (min-width: [Width A]):` [Description of changes, e.g., switch to row layout]
  - `@container (min-width: [Width B]):` [Description of changes, e.g., expand image size]

### 3. Responsive Images
Specify image requirements.
- **Aspect Ratios:** [e.g., 16:9 on mobile, 4:3 on desktop]
- **Art Direction:** [Yes/No - Describe if crop changes across breakpoints]
- **Loading Strategy:** [lazy / eager (if above fold)]

### 4. Interaction States
- **Hover/Pointer Rules:** [Describe states inside `@media (hover: hover) and (pointer: fine)`]
- **Touch Target Minimums:** [Confirm all interactive elements are ≥44x44px]

### 5. Fallbacks (Progressive Enhancement)
- **No-JS State:** [Describe behavior if JS is disabled]
- **No-Support State:** [Describe fallback if `@container` or `clamp()` is unsupported]
