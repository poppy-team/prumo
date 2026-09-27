# Information Architecture Specification: {{Product.Name}}

## 1. Executive Summary & Domain Scope
- **Product Area**: `{{product_area}}`
- **Primary User Archetypes**: `{{user_personas}}`
- **Maximum Hierarchy Depth**: {{max_depth}} (Target: $\le 3$)
- **Top-Level Category Count**: {{top_level_count}} (Target: $5 \pm 2$)

---

## 2. Global Sitemap Hierarchy

```mermaid
graph TD
    Root([Home Dashboard])
    
    %% Level 1 Buckets
    Root --> SectionA[1.0 {{Primary Feature A}}]
    Root --> SectionB[2.0 {{Primary Feature B}}]
    Root --> SectionC[3.0 {{Primary Feature C}}]
    Root --> SectionD[4.0 {{Management / Settings}}]
    
    %% Level 2 Sub-Sections
    SectionA --> SubA1[1.1 Overview & Lists]
    SectionA --> SubA2[1.2 Detailed Inspector]
    
    SectionB --> SubB1[2.1 Create / Edit Wizard]
    SectionB --> SubB2[2.2 History & Analytics]
    
    %% Level 3 Details
    SubA2 --> LeafA1[1.2.1 Item Configuration]
```

---

## 3. Navigation Schema & Wayfinding Model

| Navigation Tier | UI Pattern | Items / Rules | Active State Representation |
| :--- | :--- | :--- | :--- |
| **Tier 1 (Global)** | Left Persistent Sidebar | Top 5 primary modules | `aria-current="page"` with primary border |
| **Tier 2 (Local)** | Top Horizontal Tab Bar | Sub-sections within module | Underline indicator with bold weight |
| **Tier 3 (Contextual)**| Master-Detail Split View | Filtered items list | Selected row highlighted |
| **Breadcrumbs** | Top Utility Bar | Full hierarchical trail | Last item unlinked with `aria-current="page"` |

---

## 4. Taxonomy & Content Labeling Dictionary

| Concept / Entity | Canonical User Label | Deprecated / Prohibited Terms | Description |
| :--- | :--- | :--- | :--- |
| Workspace Unit | **Project** | Workspace, Folder, Bucket | Root container for developer resources |
| Execution Node | **Runner** | Worker, Daemon, Agent | Background computing process |
| Diagnostic Record| **Audit Log** | History, Activity, Traces | Immutable chronological security log |

---

## 5. Faceted Search & Filtering Model
- **Target Entity**: `{{entity_type}}`
- **Available Facets**:
  - *Status*: `[Active | Paused | Archived]`
  - *Date Range*: `[Last 24h | 7 Days | 30 Days | Custom]`
  - *Author / Owner*: `[Dropdown search with multi-select]`
- **Zero Results Fallback**: Display clear "No items match your filter criteria" message with a 1-click "Clear All Filters" CTA.

---

## 6. Verification & Validation Metrics
- [ ] Hierarchy tree depth does not exceed 3 levels for nominal tasks.
- [ ] No orphaned screens; all endpoints trace to root.
- [ ] Breadcrumb markup complies with W3C ARIA APG breadcrumb pattern.
