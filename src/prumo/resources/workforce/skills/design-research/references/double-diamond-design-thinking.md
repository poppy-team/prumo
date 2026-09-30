# Reference: Design Research Methodologies & Aesthetic Taxonomy

## 1. The Double Diamond Framework (Design Council UK)

The Double Diamond model maps the divergent and convergent stages of the design process:

```
      DIVIDE & EXPLORE             DELIVER & REFINE
         (Divergent)                 (Convergent)

    /─────────────────\         /─────────────────\
   /   1. DISCOVER     \       /    3. DEVELOP     \
  /  (Market research,  \     /   (Wireframing,     \
 /    user interviews,   \   /    prototyping, tests \
X     analogous audits)   X X     multi-device QA)    X
 \                       /   \                       /
  \    2. DEFINE        /     \     4. DELIVER      /
   \  (Synthesize data, /      \  (Final spec, tokens/
    \  problem thesis) /        \   implementation)  /
     \────────────────/          \──────────────────/
      PROBLEM SPACE                 SOLUTION SPACE
```

### Four Operational Modes:
1. **Discover (Divergent):** Unpack the problem space. Gather qualitative user insights, competitor forensics, and academic UX papers without rushing to premature solutions.
2. **Define (Convergent):** Cluster observations into actionable problem statements (How Might We statements, Jobs To Be Done).
3. **Develop (Divergent):** Brainstorm multiple exploratory directions, graybox wireframes, and divergent visual styles.
4. **Deliver (Convergent):** Converge onto the validated pattern, specify design tokens, and verify usability.

---

## 2. Stanford d.school Design Thinking

A five-phase iterative framework emphasizing deep human empathy:
1. **Empathize:** Conduct semi-structured interviews, observe user friction in situ, map emotional highs and lows.
2. **Define:** Unpack insights into an actionable point-of-view (POV) formula: `[User] needs [Need] because [Surprising Insight]`.
3. **Ideate:** Generate divergent solution candidates using Crazy Eights, SCAMPER, or analogical transfer.
4. **Prototype:** Build quick representations to answer specific questions (Goldilocks fidelity).
5. **Test:** Observe real users navigating the prototype without intervening or defending design choices.

---

## 3. Digital Aesthetics Taxonomy & Chronology

| Movement | Origin / Peak | Defining Visual Traits | Best Applied When |
|---|---|---|---|
| **Bauhaus** | 1919–1933 | Primary colors, geometric primitives, form follows function, asymmetric balance | Conceptual branding, artistic tools, minimal foundational identities |
| **Swiss / International Typographic** | 1950s–Present | Strict mathematical grids, neutral sans-serif (Helvetica), objective clarity, generous whitespace | Documentation systems, editorial platforms, enterprise dashboards |
| **Skeuomorphism** | 2007–2012 | Real-world physical metaphors, beveled edges, drop shadows, leather/wood textures | Novel mental models where users need physical analogies for digital actions |
| **Flat Design** | 2012–2015 | Pure 2D shapes, zero elevation/shadows, bright primary hues, geometric glyphs | High-performance low-bandwidth web interfaces, compact icons |
| **Material Design (1–3)** | 2014–Present | Elevation through shadows ($1\text{dp}$ to $24\text{dp}$), physical paper metaphors, dynamic color (Material You) | Mobile applications, Android ecosystems, consistent enterprise apps |
| **Neumorphism (Soft UI)** | 2019–2021 | Dual soft shadows (light + dark), minimal color contrast, debossed surfaces | Ambient audio software, personal dashboards (Caution: WCAG risk) |
| **Glassmorphism** | 2020–2023 | `backdrop-filter: blur()`, multi-layered transparency, subtle $1\text{px}$ frosted glass borders | Hero card overlays, modal dialogs, premium SaaS landing pages |
| **Neubrutalism** | 2022–Present | High-contrast black outlines ($2\text{px}$–$4\text{px}$), hard non-blurred drop shadows, unapologetic typography | Developer tools, Gen-Z / youth platforms, bold non-corporate products |
| **Minimalist Maximalism** | 2024–Present | Dense information layout with Swiss rigor combined with vibrant high-chroma OKLCH accents | Modern developer dashboards, technical documentation, creative tools |
