# Icon Specification Template

## Icon Name
`{category}/{action-or-object}` (e.g., `navigation/arrow-right`)

## Metadata
- **Category:** [Navigation | Action | Status | Content | ...]
- **Aliases:** [List of alternative names, e.g., chevron-right, next]
- **Introduced in Version:** [e.g., v1.2.0]
- **Status:** [Active | Deprecated | Draft]

## Visual Specification
- **Grid Size:** [e.g., 24x24]
- **Live Area:** [e.g., 20x20]
- **Primary Keyline Used:** [Circle | Square | Vertical Rect | Horizontal Rect | None]
- **Style:** [Stroke | Fill]
- **Stroke Width:** [e.g., 2px]

## Accessibility
- **Default Role:** [Decorative | Informative]
- **Recommended `aria-label` (if informative):** "[Action or meaning, e.g., 'Go to next page']"

## SVG Source (Optimized)
```xml
<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
  <!-- path data -->
</svg>
```

## Usage Guidelines
- **Use When:** [Describe the context for using this icon]
- **Do Not Use When:** [Describe anti-patterns]
- **Replaces:** [If deprecating an older icon, list it here]
