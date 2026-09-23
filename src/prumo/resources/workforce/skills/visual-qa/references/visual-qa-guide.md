# Visual Quality Assurance (Visual QA) Technical Reference Guide

## 1. Design-to-Code Parity Methodology

Visual QA bridges the gap between static design deliverables (Figma, Sketch) and dynamic browser rendering:

```
+-------------------+       Overlay Comparison        +-------------------+
|    Figma Spec     | <-----------------------------> |  Implemented DOM  |
|  (Vector Tokens)  |                                 | (Rendered Canvas) |
+-------------------+                                 +-------------------+
          |                                                     |
          +───────────────► Audit Dimensions ◄──────────────────+
                            - Spatial Grid (4pt/8pt)
                            - Typographic Vertical Rhythm
                            - Color Token Parity
                            - Asset Sharpness (SVG / @2x)
```

---

## 2. The 4pt / 8pt Spatial Grid System

Spatial consistency reduces visual noise and cognitive fatigue:
- **8pt Macro Grid**: Used for layout containers, section paddings, column gutters, and major component margins ($8, 16, 24, 32, 48, 64, 80\text{ px}$).
- **4pt Micro Grid**: Used for small element internals: icon-to-label gaps, button padding, badge height, border radius ($4, 8, 12, 16\text{ px}$).

### Why Ban Arbitrary Values?
Values such as `margin: 11px;` break modular cadence, cause sub-pixel rounding errors on non-integer device pixel ratios (e.g. 1.25x or 1.5x scaling in Windows/Android), and degrade visual rhythm.

---

## 3. Typographic Vertical Rhythm

To maintain harmonious vertical rhythm:
1. **Unitless Line Heights**: Declare `line-height` as unitless ratios (e.g. `line-height: 1.5` or `1.25`), allowing line height to scale proportionally with font size.
2. **Cap Height Alignment**: Align icons to the font's *cap height* (the height of uppercase letters like 'H' or 'T') rather than the full bounding box of the font, preventing icons from looking vertically sunken next to text.
3. **Leading Trim**: When using modern CSS (`text-box-trim: both; text-box-edge: cap alphabetic;`), remove extraneous font leading to achieve pixel-perfect alignment against adjacent buttons or icons.

---

## 4. High-DPI (Retina) & Vector Sharpness

| Screen Type | Device Pixel Ratio (DPR) | Physical Pixels per CSS Pixel | Asset Requirement |
| :--- | :---: | :---: | :--- |
| Standard Desktop | 1.0x | 1 | Scalable SVG or 1x raster |
| Modern Laptop / Tablet | 2.0x | 4 ($2 \times 2$) | SVG or `@2x` raster (`image-set` / `srcset`) |
| Flagship Mobile / Pro Display | 3.0x | 9 ($3 \times 3$) | SVG or `@3x` raster |

- **Sub-Pixel Border Blurring**: On 1.5x displays, a `1px` border can render as 1.5 physical pixels, causing a blurred, washed-out line. Use `box-shadow: 0 0 0 1px ...` or transform scaling when razor-sharp hair-lines are critical.
