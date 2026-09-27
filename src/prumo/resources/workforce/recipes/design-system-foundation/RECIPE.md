# Design System Foundation Architecture Recipe

## Purpose
Establishes the foundation of an enterprise design system: W3C Design Tokens Community Group (DTCG) token tiers, perceptual OKLCH color palettes, modular typography, headless component architecture, and WCAG accessibility conformance.

## Preconditions
- Strategic brand vision and core design identity defined.
- Primary web/application frameworks identified (React, Vue, Web Components).

## Required Inputs
- Brand guidelines & color swatches.
- Component inventory requirements (inputs, buttons, overlays, data displays).

## Step DAG & Dependencies

1. **Define Multi-Tier W3C Design Tokens** (`token-architecture`)
   - **Role:** `design-system-engineer`
   - **Skills:** `design-tokens`, `color-science`, `typography-system`
   - **Input:** Brand aesthetic guidelines
   - **Output:** Global, semantic, and component tokens JSON
   - **Evidence Required:** `test`
   - **Gates:** `token-validation`
   - **Dependencies:** None (Entry step)

2. **Specify Atomic UI Component Contracts** (`primitive-specification`)
   - **Role:** `design-system-engineer`
   - **Skills:** `component-specification`, `design-system`
   - **Input:** Semantic tokens
   - **Output:** Component props, states, and variant specifications
   - **Evidence Required:** `review`
   - **Dependencies:** `token-architecture`

3. **Build Headless / Accessible UI Component Primitives** (`component-implementation`)
   - **Role:** `ui-component-engineer`
   - **Skills:** `ui-implementation`, `clean-code`
   - **Input:** Component specifications and tokens
   - **Output:** Component source code and test suite
   - **Evidence Required:** `test`
   - **Gates:** `component-test`
   - **Dependencies:** `primitive-specification`

4. **Author Living Styleguide & Documentation** (`living-styleguide`)
   - **Role:** `creative-director`
   - **Skills:** `design-system`, `visual-communication`
   - **Input:** Components and token manifests
   - **Output:** Interactive documentation website & playground
   - **Evidence Required:** `review`
   - **Dependencies:** `component-implementation`

5. **Exhaustive WCAG & Keyboard Navigation Audit** (`accessibility-conformance`)
   - **Role:** `accessibility-reviewer`
   - **Skills:** `visual-regression`
   - **Input:** Rendered components and documentation views
   - **Output:** WCAG conformance audit report
   - **Evidence Required:** `test`
   - **Gates:** `wcag-aaa`
   - **Dependencies:** `living-styleguide`

## Exit Gates & Verification
- **`token-validation`**: Tokens adhere strictly to W3C DTCG schema (2025.10); transforms to CSS custom properties produce zero syntax errors.
- **`component-test`**: All components have 100% test coverage across interactive states (`idle`, `hover`, `active`, `focus-visible`, `disabled`, `loading`).
- **`wcag-aaa`**: Zero accessibility violations; automated testing via Axe-core passes with contrast ratios $\ge 7:1$ for normal text and $\ge 4.5:1$ for large text.
