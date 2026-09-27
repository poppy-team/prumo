# Sample Interaction Specification: Slide-Over Detail Drawer

## 1. Overview
- **Component**: `SlideOverDrawer.tsx`
- **Modalities**: Touch swipe, Mouse click, Keyboard navigation (`Escape`, `Tab`)
- **Fitts's Law Hit Target**: Close button maintains $48 \times 48\text{ px}$ hit area; backdrop covers full viewport.

---

## 2. Micro-Interaction Definition

### 1. Trigger
- **Open Trigger**: User clicks any table row (`TableRow[role="button"]`) or presses `Enter` on row.
- **Dismiss Trigger**:
  - Click backdrop overlay.
  - Click close icon button (`Button[aria-label="Close drawer"]`).
  - Press `Escape` key.
  - Touch swipe right with velocity $>0.5\text{ px/ms}$ or displacement $>30\%$ of drawer width.

### 2. Rules
- While open, body scroll is locked (`overflow: hidden`).
- Focus is trapped within drawer nodes (`focus-trap`).
- Background clicks are intercepted by backdrop.

### 3. Feedback
- Immediate visual feedback on row click ($<50\text{ms}$ row highlight).
- Drawer slides in from right edge using CSS transform.
- Backdrop fades in from `opacity: 0` to `opacity: 0.5`.

### 4. Loops & Modes
- Modal mode: Background content marked with `inert` or `aria-hidden="true"`.
- Upon closing, focus is restored to the triggering table row.

---

## 3. Motion Timing & Easing Curves

```css
/* Drawer Container */
.drawer-panel {
  position: fixed;
  top: 0;
  right: 0;
  height: 100vh;
  width: min(100vw, 480px);
  transform: translateX(100%);
  transition: transform 300ms cubic-bezier(0.0, 0.0, 0.2, 1.0); /* Entering */
  will-change: transform;
}

.drawer-panel[data-state="open"] {
  transform: translateX(0);
}

.drawer-panel[data-state="closing"] {
  transition: transform 250ms cubic-bezier(0.4, 0.0, 1.0, 1.0); /* Exiting */
  transform: translateX(100%);
}

/* Reduced Motion Override */
@media (prefers-reduced-motion: reduce) {
  .drawer-panel {
    transition: opacity 150ms ease !important;
    transform: none !important;
  }
}
```
