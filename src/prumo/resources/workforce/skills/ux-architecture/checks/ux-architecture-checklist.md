# UX Information Architecture Checklist

## 1. Structural Hierarchy & Depth
- [ ] **Top-Level Breadth**: Primary navigation contains $5 \pm 2$ items (no more than 7 top-level buckets).
- [ ] **Task Depth**: Primary user tasks reachable within 3 levels from dashboard root.
- [ ] **Zero Orphaned Views**: Every view has a defined ancestor trail and navigation link.
- [ ] **Balanced Tree**: Information density is distributed evenly across categories without lopsided clusters.

## 2. Navigation Systems & Wayfinding
- [ ] **Current State Indicator**: The active category/page is prominently highlighted (`aria-current="page"`).
- [ ] **Breadcrumbs**: Hierarchical paths over 2 levels deep display functional, accessible breadcrumbs.
- [ ] **Consistent Global Anchor**: Logo or home icon returns user predictably to root dashboard.
- [ ] **Mobile Responsiveness**: Desktop navigation maps cleanly to mobile drawer, tab bar, or hierarchical drill-down list.

## 3. Taxonomy & Labeling
- [ ] **User-Centric Language**: Category labels reflect user terminology rather than internal business jargon.
- [ ] **Mutual Exclusivity**: Primary categories are clearly differentiated to prevent ambiguous choice between sibling options.
- [ ] **Canonical Naming**: Terminology is consistent across navigation menus, page titles, and breadcrumbs.

## 4. Search & Filtering
- [ ] **Faceted Filters**: High-volume catalogs provide multi-attribute filtering with real-time result counts.
- [ ] **Zero-Results Recovery**: Empty search states suggest spelling alternatives, broader categories, or filter reset options.
