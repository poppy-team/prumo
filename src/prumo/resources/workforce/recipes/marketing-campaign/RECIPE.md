# Marketing & Advertising Campaign Pipeline Recipe

## Purpose
Orchestrates the creation, design, and implementation of high-conversion multi-channel marketing campaigns using the AIDA framework (Attention, Interest, Desire, Action), cognitive psychology, and responsive web technologies.

## Preconditions
- Brand identity guidelines and design tokens established.
- Target marketing objective, core messaging, and distribution channels identified.

## Required Inputs
- Campaign strategic brief & value proposition.
- Platform format specifications (social, display network, email, landing view).

## Step DAG & Dependencies

1. **AIDA Framing & Psychological Hook Strategy** (`campaign-strategy`)
   - **Role:** `creative-director`
   - **Skills:** `design-psychology`, `advertising-design`
   - **Input:** Campaign brief & value proposition
   - **Output:** Campaign strategy & copy matrix
   - **Evidence Required:** `review`
   - **Dependencies:** None (Entry step)

2. **Design Multi-Format Advertising Creatives** (`ad-collateral`)
   - **Role:** `advertising-designer`
   - **Skills:** `advertising-design`, `marketing-collateral`
   - **Input:** Campaign strategy & copy matrix
   - **Output:** Multi-format ad units (SVG, HTML5, CSS)
   - **Evidence Required:** `review`
   - **Gates:** `aida-evaluation`
   - **Dependencies:** `campaign-strategy`

3. **Layout Architecture & Cognitive Hierarchy** (`landing-architecture`)
   - **Role:** `advertising-designer`
   - **Skills:** `layout-patterns`, `responsive-design`
   - **Input:** Campaign strategy
   - **Output:** Landing page layout wireframe spec
   - **Evidence Required:** `review`
   - **Dependencies:** `campaign-strategy`

4. **Implement Responsive High-Conversion Landing View** (`landing-implementation`)
   - **Role:** `frontend-engineer`
   - **Skills:** `responsive-design`, `clean-code`
   - **Input:** Layout wireframe and creative assets
   - **Output:** Production-ready responsive landing view
   - **Evidence Required:** `test`
   - **Gates:** `responsive-check`, `accessibility`
   - **Dependencies:** `ad-collateral`, `landing-architecture`

5. **Full Campaign Quality & Conversion Review** (`conversion-qa`)
   - **Role:** `quality-reviewer`
   - **Skills:** `clean-code`
   - **Input:** Landing view and creative assets
   - **Output:** Campaign launch readiness scorecard
   - **Evidence Required:** `review`
   - **Dependencies:** `landing-implementation`

## Exit Gates & Verification
- **`aida-evaluation`**: Visual hierarchy clearly leads attention, establishes interest, fosters desire with social proof, and terminates in prominent CTAs.
- **`responsive-check`**: Landing page layout verified across 5 breakpoints (320px, 640px, 768px, 1024px, 1440px) with zero horizontal overflow.
- **`accessibility`**: Landing view meets WCAG 2.2 AA standards with verified focus indicators and screen reader landmarks.
