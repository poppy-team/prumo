# Interaction Design (IxD) Engineering Checklist

## 1. Motor Ergonomics & Fitts's Law
- [ ] **Touch Target Sizing**: Interactive touch targets maintain minimum $44 \times 44\text{ px}$ (or $\ge 24\text{ px}$ with spacing).
- [ ] **Thumb Zone Optimization**: Primary mobile actions positioned in the lower two-thirds of the screen for natural thumb reach.
- [ ] **Target Spacing**: Adjacent clickable elements have $\ge 8\text{ px}$ separation to prevent accidental taps.

## 2. Micro-Interaction Choreography
- [ ] **Complete Four-Part Model**: Every micro-interaction defines Trigger, Rules, Feedback, and Loops/Modes.
- [ ] **Instant Feedback**: Visible interaction feedback (active state, ripple, depression) occurs in $<100\text{ ms}$.
- [ ] **Loading Continuity**: Operations taking $>1.0\text{ s}$ display progress bars or skeleton loaders; operations taking $>10\text{ s}$ provide background queuing.

## 3. Animation Timing & Physics
- [ ] **Bounded Duration**: Transitions range between $100\text{ ms}$ (micro) and $400\text{ ms}$ (macro); zero sluggish transitions $>400\text{ ms}$.
- [ ] **Asymmetric Easing**: Entering elements use deceleration curve (`ease-out`); exiting elements use acceleration curve (`ease-in`).
- [ ] **Reduced Motion Override**: Layouts support `@media (prefers-reduced-motion: reduce)` by substituting spatial motion with instantaneous opacity cross-fades.

## 4. Affordance & Error Prevention
- [ ] **Clear Signifiers**: Clickable elements display visible affordances (borders, elevation, cursor pointer).
- [ ] **Undo Over Confirmation**: Transient reversible actions provide non-blocking Undo toasts instead of modal interruptions.
- [ ] **Destructive Safeguards**: Irreversible actions require explicit two-step verification.
- [ ] **Keyboard Equivalence**: All mouse/touch interactions can be triggered via keyboard navigation.
