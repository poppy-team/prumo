# Wireframe Fidelity Levels & Layout Engineering Guide

## 1. Fidelity Spectrum: From Conceptual Sketch to Code

| Dimension | Low-Fidelity (Lo-Fi) | Mid-Fidelity (Mid-Fi) | High-Fidelity (Hi-Fi) |
| :--- | :--- | :--- | :--- |
| **Visual Appearance** | Rough boxes, placeholder text | Grayscale, exact 8pt spacing, real copy | Full brand colors, assets, tokens |
| **Primary Goal** | Information hierarchy, layout concept | Interaction behavior, responsive reflow | Visual polish, final dev handoff |
| **Grid Alignment** | Approximate | Strict 4/8/12-column grid | Pixel-perfect design token grid |
| **Key Audience** | Product managers, early brainstorm | Engineers, UX researchers, clients | Final engineering implementation |

---

## 2. Responsive 12-Column Grid Architecture

A standard 12-column grid provides maximum mathematical divisibility (divisible by 1, 2, 3, 4, 6, and 12):

$$\text{Container Width} = 12 \times W_{\text{col}} + 11 \times W_{\text{gutter}} + 2 \times W_{\text{margin}}$$

```
Desktop (1200px+): 12 Columns
| Col 1 | Col 2 | Col 3 | Col 4 | Col 5 | Col 6 | Col 7 | Col 8 | Col 9 | Col 10 | Col 11 | Col 12 |
[────── Sidebar: 3 Cols ──────][────────────────────── Main Content: 9 Cols ──────────────────────]

Tablet (768px): 8 Columns
[───────── Main Content: 6 Cols ─────────][── Aside: 2 Cols ──]

Mobile (320px - 480px): 4 Columns (Stacked)
[────────────────────────────────── Main Content: 4 Cols ──────────────────────────────────]
[────────────────────────────────── Sidebar / Drawer (Off-Canvas) ────────────────────────]
```

---

## 3. Graybox Schematic Conventions

To prevent color bias during structural layout reviews, adhere to standard graybox notation:
- **Image / Media Placeholder**: Neutral gray rectangle with an 'X' diagonal cross (`background: #e2e8f0;`).
- **Body Text**: Subtle horizontal gray rounded bars or realistic truncated text.
- **Button / Interactive**: Dark neutral pill or rectangle (`background: #0f172a; color: #ffffff;`).
- **Container / Card**: Light gray border with subtle background surface (`border: 1px solid #cbd5e1; background: #ffffff;`).

---

## 4. Semantic HTML5 Landmark Mapping

```
+------------------------------------------------------------------------+
| <header role="banner">                                                 |
|   Logo, Global Search Input, User Avatar & Profile Menu                |
+------------------------------------------------------------------------+
| <nav role="navigation">                                                |
|   Primary Navigation Menu / Breadcrumb Trail                           |
+------------------------------------------------------------------------+
| <main role="main">                        | <aside role="complementary">|
|   Unique Primary Page Content              |   Filters, Related Links,   |
|   - Hero Card                             |   Contextual Help Widget    |
|   - Data Table / Results Feed             |                             |
|   - Primary Action Forms                  |                             |
+-------------------------------------------+----------------------------+
| <footer role="contentinfo">                                            |
|   Legal Disclaimers, Status Links, Version Metadata                    |
+------------------------------------------------------------------------+
```
