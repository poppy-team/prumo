# Motion Library Checklist

## Token Definition
- [ ] Motion durations are defined as semantic tokens (e.g., `motion.duration.fast`).
- [ ] Easing curves are defined for both productive and expressive states.
- [ ] Spring configurations (mass, stiffness, damping) are standardized.

## Implementation & Patterns
- [ ] Reusable motion primitives (`fadeIn`, `slideIn`, etc.) are implemented and exported.
- [ ] Spring physics are used for gesture-driven interactions.
- [ ] Complex sequences use standardized choreography (staggering).

## Accessibility
- [ ] All animations respect `prefers-reduced-motion`.
- [ ] Reduced motion alternatives use crossfades or instant transitions.
- [ ] Critical interaction feedback is preserved even when motion is reduced.

## Performance
- [ ] Animations target only `transform` and `opacity` properties.
- [ ] Layout-triggering properties (width, height, etc.) are avoided in animations.
- [ ] The `will-change` property is used only when absolutely necessary and cleaned up.
- [ ] Animations maintain 60fps on target devices.
