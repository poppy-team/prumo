# Color Theory & Perceptual Color Spaces in Modern Design Systems

A robust design system relies heavily on a mathematically structured color palette. Understanding modern color science ensures accessible, aesthetically cohesive, and cross-platform functional interfaces.

## 1. Color Models & Color Spaces

### 1.1 The OKLCH Color Space (Perceptually Uniform)
Traditional HSL and RGB color models fail perceptual uniformity: in HSL, pure yellow (`hsl(60, 100%, 50%)`) has vastly higher perceived lightness than pure blue (`hsl(240, 100%, 50%)`), producing unpredictable contrast calculations when swapping hues.

**OKLCH solves this:**
- **$L$ (Lightness):** $0.0$ to $1.0$ (or $0\%$ to $100\%$). A constant $L = 0.70$ reflects identical perceptual luminance to the human visual cortex across all hues.
- **$C$ (Chroma):** $0.0$ to $\sim 0.37$ (unbounded). Measures color intensity / purity.
- **$H$ (Hue):** $0^\circ$ to $360^\circ$ color circle angle.

### 1.2 Wide Gamut & Display P3
Modern mobile displays and monitors display colors beyond standard sRGB:
- **sRGB:** Standard web legacy gamut.
- **Display P3:** $\approx 25\%$ wider gamut than sRGB, delivering significantly richer cyans, emeralds, and fiery oranges.
- **CSS Color Module Level 4:** Allows authoring wide-gamut colors natively with automatic browser gamut mapping:
  ```css
  :root {
    --color-brand: oklch(0.62 0.24 255); /* Deep rich indigo in P3 */
  }
  ```

---

## 2. Multi-Tier Semantic Palette Architecture

Never reference raw color values in component markup. Organize tokens into three strict tiers:

```
Global / Primitive Tokens   (e.g., --color-blue-500: oklch(0.60 0.20 250))
      │
      ▼
Semantic Tokens             (e.g., --color-action-primary: var(--color-blue-500))
      │
      ▼
Component-Scoped Tokens     (e.g., --button-primary-bg: var(--color-action-primary))
```

### Semantic Token Roles:
- **Interactive / Brand (`--color-action-*`):** Primary triggers, active states, focus rings.
- **Surface & Backgrounds (`--color-surface-*`):** Base canvas, elevated card surfaces, overlay modals.
- **Typography & Content (`--color-text-*`):** Primary headings, secondary body, subtle captions, inverted text.
- **Dividers & Borders (`--color-border-*`):** Subtle card boundaries, high-contrast input outlines.
- **Feedback & Semantics:**
  - **Success (`--color-feedback-success`):** Confirmations, positive status, verified badges.
  - **Warning (`--color-feedback-warning`):** Attention required, transient hazards, pending approval.
  - **Danger / Error (`--color-feedback-danger`):** Destructive actions, validation errors, critical failures.
  - **Info (`--color-feedback-info`):** Neutral informational callouts, documentation banners.

---

## 3. Accessibility & Contrast Standards

### 3.1 WCAG 2.2 Contrast Matrix (Relative Luminance)
- **AA Level:** Minimum $4.5:1$ for body text; $3.0:1$ for large text ($\ge 24\text{px}$ regular or $\ge 18.5\text{px}$ bold) and essential graphical UI controls.
- **AAA Level:** Minimum $7.0:1$ for body text; $4.5:1$ for large text.

### 3.2 APCA (Accessible Perceptual Contrast Algorithm - WCAG 3.0)
APCA measures contrast as Lightness Contrast ($L_c$) considering polarity (light-on-dark vs dark-on-light):
- $L_c \ge 90$: High-legibility continuous reading text.
- $L_c \ge 75$: Standard body text and form labels.
- $L_c \ge 60$: Headlines and secondary labels.
- $L_c \ge 45$: Non-text UI controls and active border indicators.
