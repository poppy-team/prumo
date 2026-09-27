# Motion Design Reference Guide

## 1. Physics over Easing
Interactive UI elements should feel like they have physical properties (mass, tension, friction).
- **Stiffness (Tension):** How quickly the spring moves toward its destination.
- **Damping (Friction):** How quickly the motion stops (prevents infinite oscillation).
- **Mass:** How much effort is required to move the element.

## 2. Easing Curves (Cubic Bézier)
For non-interactive or purely state-based animations, standardized curves are preferred.
- **Ease In:** `cubic-bezier(0.4, 0, 1, 1)` - Starts slow, ends fast (exits).
- **Ease Out:** `cubic-bezier(0, 0, 0.2, 1)` - Starts fast, ends slow (entrances).
- **Ease In-Out:** `cubic-bezier(0.4, 0, 0.2, 1)` - Slow at both ends (standard UI motion).

## 3. Disney's Principles in UI
- **Anticipation:** A slight negative movement before a positive action.
- **Follow Through (Overshoot):** Moving slightly past the target before settling.
- **Staging:** Dimming the background to focus on a modal animation.

## 4. Performance Golden Rules
Animating non-composited properties causes the browser to recalculate layout and repaint, dropping frames.
**GPU-Accelerated Properties:**
- `transform: translate()`
- `transform: scale()`
- `transform: rotate()`
- `opacity`

## 5. Accessibility
```css
@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
    scroll-behavior: auto !important;
  }
}
```
*Note: A complete removal of animations can remove feedback. Opt for opacity transitions where possible.*
