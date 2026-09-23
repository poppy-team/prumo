# Visual QA Audit Report: {{Feature.Title}}

## 1. Audit Metadata
- **View / Component**: `{{component_or_page}}`
- **Figma Specification**: [Figma Link or Frame ID]
- **Auditor**: `{{auditor_id}}`
- **Date**: {{YYYY-MM-DD}}
- **Overall Parity Score**: {{parity_percentage}}% (Target: $\ge 95\%$)
- **Sign-Off Status**: [APPROVED | APPROVED WITH PUNCH LIST | REJECTED]

---

## 2. Category Parity Scorecard

| Audit Dimension | Target Standard | Observed Status | Score (%) |
| :--- | :--- | :--- | :---: |
| **Spatial Rhythm** | 4pt/8pt grid compliance | {{spatial_status}} | {{spatial_score}}% |
| **Typography Scale** | Font, weight, line-height parity | {{typo_status}} | {{typo_score}}% |
| **Color & Elevation** | Design token resolution | {{color_status}} | {{color_score}}% |
| **Asset Sharpness** | Vector SVG / Retina `@2x` | {{asset_status}} | {{asset_score}}% |
| **Responsive Fluidity** | Zero 320px overflow, clean reflow | {{responsive_status}} | {{responsive_score}}% |

---

## 3. Visual Punch List & Remediation Log

### Issue #1: [Brief Defect Title]
- **Category**: [Spacing | Typography | Color | Elevation | Asset | Alignment]
- **Severity**: [Blocker | High | Medium | Cosmetic]
- **Component / Element**: `{{css_selector_or_component}}`
- **Design Spec (Expected)**: `{{expected_token_or_value}}`
- **Observed in Build (Actual)**: `{{observed_value}}`
- **Remediation CSS**:
  ```css
  /* Recommended Fix */
  .target-element {
    margin-bottom: var(--space-md); /* 16px (was 11px) */
    line-height: var(--line-height-tight);
  }
  ```

---

## 4. Breakpoint Inspection Log
- **320px (Mobile Min)**: [PASS / No horizontal overflow, touch targets $\ge 44\text{px}$]
- **768px (Tablet)**: [PASS / Navigation collapses cleanly, columns wrap]
- **1440px (Desktop)**: [PASS / Container bounded to max-width, balanced gutters]

---

## 5. Design Sign-Off Actions
- [ ] Punch list items assigned to sprint backlog.
- [ ] Token overrides aligned with central design tokens.
- [ ] Retest scheduled upon punch list completion.
