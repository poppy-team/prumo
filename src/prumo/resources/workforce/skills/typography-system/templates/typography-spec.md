# Typography Specification: [Project Name]

## 1. Typefaces
- **Primary (Sans-Serif):** [Font Name] - used for [Body/Headers]
- **Secondary (Serif):** [Font Name] - used for [Headers/Quotes]
- **Monospace:** [Font Name] - used for [Code/Data]

## 2. Scale & Rhythm
- **Modular Scale Ratio:** [e.g., 1.25 (Major Third)]
- **Base Size:** [e.g., 16px]
- **Baseline Grid:** [e.g., 8px]

## 3. Typographic Hierarchy
| Level | Font Family | Fluid Clamp Value | Line Height | Letter Spacing | OpenType |
|-------|-------------|-------------------|-------------|----------------|----------|
| Display | Primary | `clamp(...)` | 1.1 | -0.02em | `liga` |
| H1 | Primary | `clamp(...)` | 1.2 | -0.01em | |
| Body | Primary | `clamp(1rem, 0.8rem + 1vw, 1.125rem)` | 1.5 | 0 | `kern` |
| Caption | Secondary | `clamp(0.75rem, 0.7rem + 0.2vw, 0.875rem)` | 1.4 | 0.01em | |
| Code | Monospace | `1rem` | 1.5 | 0 | `tnum` |
