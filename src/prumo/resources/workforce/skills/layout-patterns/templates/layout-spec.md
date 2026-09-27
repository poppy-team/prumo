# Layout Specification Document

## 1. System Foundation
**Base Unit**: [e.g., 8px]  
**Typography Base**: [e.g., 16px (1rem)]  

## 2. Spacing Scale (Geometric Progression)
- `--space-xs`: [e.g., 0.5rem / 8px]
- `--space-sm`: [e.g., 1rem / 16px]
- `--space-md`: [e.g., 1.5rem / 24px]
- `--space-lg`: [e.g., 2rem / 32px]
- `--space-xl`: [e.g., 3rem / 48px]
- `--space-xxl`: [e.g., 4rem / 64px]

## 3. Grid Definition
- **Mobile** (Intrinsic > ~320px): [e.g., 4 Columns, 16px Gutter, 16px Margin]
- **Tablet** (Intrinsic > ~768px): [e.g., 8 Columns, 24px Gutter, 32px Margin]
- **Desktop** (Intrinsic > ~1024px): [e.g., 12 Columns, 24px Gutter, Auto Margin]

## 4. Primary Layout Primitives Used
| Primitive | Purpose | Key CSS Variables |
|-----------|---------|-------------------|
| Stack     | [e.g., Vertical flow] | `--stack-space` |
| Cluster   | [e.g., Tag groups] | `--cluster-space` |
| Sidebar   | [e.g., Main content + Nav] | `--sidebar-width` |
| ...       | ...     | ... |

## 5. Composition Strategy
**Dominant Scan Pattern**: [e.g., F-Pattern / Z-Pattern]
**Focal Points**:
1. [Primary action/element]
2. [Secondary information]

## 6. Container Query Breakpoints
- `@container (min-width: 40ch)` -> [Behavior change]
- `@container (min-width: 60ch)` -> [Behavior change]
