# Motion Specification: [Component Name]

## 1. Overview
**Purpose:** Describe the interaction and its intended user feedback.
**Trigger:** e.g., Hover, Click, Page Load, Gesture.

## 2. Motion Tokens Applied
- **Duration:** [e.g., `motion.duration.small` (150ms)]
- **Easing / Spring:** [e.g., `motion.easing.productive` or `motion.spring.bouncy`]

## 3. Properties Animated
- **Start State:** `opacity: 0, transform: translateY(10px)`
- **End State:** `opacity: 1, transform: translateY(0)`
- **Properties:** [e.g., `opacity`, `transform`]

## 4. Choreography (if applicable)
- **Stagger Delay:** [e.g., 50ms per item]
- **Sequence:** [Describe entrance/exit order]

## 5. Reduced Motion Fallback
- **Fallback Behavior:** [e.g., Instant appearance, fade without translation]

## 6. Performance Notes
- Will this trigger layout recalculation? [Yes/No - if Yes, justify]
- Expected Frame Rate: 60fps
