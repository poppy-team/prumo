# Wireframe Specification: {{Screen.Name}}

## 1. Overview & Screen Purpose
- **Screen ID**: `{{screen_id}}`
- **User Role**: `{{user_persona}}`
- **Primary Action (North Star)**: {{Define the single primary task user must accomplish}}
- **Fidelity Level**: [Lo-Fi Schematic | Mid-Fi Grayscale Prototype]
- **Target Container Width**: 1200px (Desktop), 768px (Tablet), 375px (Mobile)

---

## 2. Layout Structure & Responsive Grid

```text
+------------------------------------------------------------------------+
| HEADER (Banner): Logo, Search Bar, Notifications, User Menu           |
+-------------------+----------------------------------------------------+
| SIDEBAR (Nav):    | MAIN CONTENT (role="main"):                        |
| - Dashboard       | 1. Page Header (Title + Primary Action Button)     |
| - Projects        | 2. Metric Stat Cards (4 columns -> 2 col -> 1 col) |
| - Settings        | 3. Data Table / List Feed (Filter bar + Pagination)|
|                   +----------------------------------------------------+
|                   | ASIDE (role="complementary"): Filter Drawer        |
+-------------------+----------------------------------------------------+
| FOOTER (Contentinfo): Copyright, Status indicator, Documentation link  |
+------------------------------------------------------------------------+
```

### Responsive Column Grid Mapping
- **Desktop ($\ge 1200\text{px}$)**: 12 Columns. Sidebar = 3 cols, Main Content = 9 cols.
- **Tablet ($768\text{px} - 1024\text{px}$)**: 8 Columns. Sidebar collapses to icons (1 col), Main = 7 cols.
- **Mobile ($320\text{px} - 480\text{px}$)**: 4 Columns. Sidebar moves off-canvas (hamburger menu), Main = 4 cols (stacked).

---

## 3. Semantic Landmark Map
- `<header role="banner">`: Contains logo, global search input, quick action icons.
- `<nav role="navigation">`: Contains primary sidebar links and breadcrumbs.
- `<main role="main">`: Contains primary unique page content.
- `<aside role="complementary">`: Contains contextual filter drawer.
- `<footer role="contentinfo">`: Contains copyright and status links.

---

## 4. Component States Matrix

| State | Visual Treatment & Behavior | Technical Implementation |
| :--- | :--- | :--- |
| **Default** | Standard populated cards and table rows | Populated with real sample records |
| **Loading** | Skeleton blocks with pulsing gray opacity | Skeleton components match exact layout height |
| **Empty** | Illustration placeholder, "No records found", CTA button | "Create New Item" button shifts focus |
| **Error** | Red inline banner with retry action | Error boundary captures failure; retry refetches |

---

## 5. Engineering Annotations
- **Form Tab Order**: Search -> Filter Dropdowns -> Table Header Sorts -> Row Action Buttons.
- **Text Truncation**: Table column 2 (Title) clamps to single line with `text-ellipsis` and native tooltip.
- **Sticky Elements**: Header remains sticky (`position: sticky; top: 0; z-index: 10;`).
- **Touch Targets**: All mobile buttons and links enforce minimum $44 \times 44\text{ px}$ hit areas.
