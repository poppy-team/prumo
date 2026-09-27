# Brand Creation & Identity System Recipe

## Purpose
Engineering a complete, mathematically sound, and accessible brand identity system from initial aesthetic positioning to vector logo crafting, OKLCH tokenization, and brand guidelines compilation.

## Preconditions
- Strategic product brief, company charter, or mission statement available.
- Target market, user personas, and core company values identified.

## Required Inputs
- Brand charter and positioning objectives.
- Competitive landscape references.

## Step DAG & Dependencies

1. **Aesthetic Research & Semiotic Positioning** (`aesthetic-strategy`)
   - **Role:** `creative-director`
   - **Skills:** `aesthetic-analysis`, `visual-communication`
   - **Input:** Brand brief & company charter
   - **Output:** Aesthetic strategy & moodboard spec
   - **Evidence Required:** `review`
   - **Dependencies:** None (Entry step)

2. **OKLCH Color Science & Accessible Palette Generation** (`color-foundation`)
   - **Role:** `brand-designer`
   - **Skills:** `color-science`, `brand-identity`
   - **Input:** Aesthetic strategy
   - **Output:** OKLCH color palette tokens JSON
   - **Evidence Required:** `test`
   - **Gates:** `contrast-audit`
   - **Dependencies:** `aesthetic-strategy`

3. **Fluid Typography Hierarchy & Variable Fonts Spec** (`typography-architecture`)
   - **Role:** `brand-designer`
   - **Skills:** `typography-system`, `brand-identity`
   - **Input:** Aesthetic strategy
   - **Output:** Typography system tokens JSON
   - **Evidence Required:** `review`
   - **Dependencies:** `aesthetic-strategy`

4. **Mathematical SVG Vector Logo & Mark Crafting** (`logo-engineering`)
   - **Role:** `svg-artist`
   - **Skills:** `svg-engineering`
   - **Input:** Brand tokens and aesthetic direction
   - **Output:** Responsive SVG logo pack
   - **Evidence Required:** `test`
   - **Gates:** `vector-conformance`
   - **Dependencies:** `color-foundation`, `typography-architecture`

5. **Compile Formal Brand Identity Guidelines** (`guidelines-compilation`)
   - **Role:** `brand-designer`
   - **Skills:** `brand-identity`, `visual-communication`
   - **Input:** Vector logo assets and token manifests
   - **Output:** Brand guidelines manual markdown
   - **Evidence Required:** `review`
   - **Dependencies:** `logo-engineering`

6. **Comprehensive Visual Identity & Multi-surface Audit** (`identity-audit`)
   - **Role:** `visual-identity-auditor`
   - **Skills:** `design-critique`
   - **Input:** Brand guidelines and vector assets
   - **Output:** Brand audit scorecard and sign-off
   - **Evidence Required:** `review`
   - **Gates:** `brand-critique`
   - **Dependencies:** `guidelines-compilation`

## Exit Gates & Verification
- **`contrast-audit`**: All primary, accent, and neutral token pairs meet WCAG AAA (7:1) or AA (4.5:1).
- **`vector-conformance`**: SVG assets validate against XML schemas, specify responsive viewBox, and include semantic a11y attributes.
- **`brand-critique`**: Formal sign-off by visual identity auditor confirming brand consistency across digital and print touchpoints.
