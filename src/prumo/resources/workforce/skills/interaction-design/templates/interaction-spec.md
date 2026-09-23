# Interaction Specification: {{Interaction.Title}}

## 1. Overview & Interaction Intent
- **Component / Surface**: `{{component_name}}`
- **Target Modalities**: [Pointer / Mouse | Touch Gestures | Keyboard Focus | Hybrid]
- **Primary Objective**: {{Describe the interaction goal and behavioral affordance}}
- **Fitts's Law Hit Area**: {{hit_area_dimensions}} (Minimum: $44 \times 44\text{ px}$)

---

## 2. Micro-Interaction Four-Part Model

### 1. Trigger
- **User Action**: [Click | Hover | Long-Press | Drag | Swipe Left/Right | Keydown]
- **Target Selector**: `{{css_selector_or_ref}}`

### 2. Rules & Preconditions
- **Conditions**: {{e.g., input must be valid, drawer must not be locked}}
- **State Changes**: `{{current_state}}` $\longrightarrow$ `{{target_state}}`

### 3. Feedback & Latency Budget
- **Instant Response ($<100\text{ms}$)**: Visual active state / scale transform ($0.98$).
- **Intermediate Response ($<1000\text{ms}$)**: Inline loading spinner, button disabled.
- **Completion Feedback**: Success checkmark badge, sensory confirmation, haptic vibration.

### 4. Loops & Modes
- **Persistence**: [Auto-dismisses in 4s | Persistent until user dismiss | Modal focus trap]

---

## 3. Motion Timing & Easing Curves

| State Transition | Duration | Easing Curve (CSS) | Properties Animated |
| :--- | :---: | :--- | :--- |
| **Hover / Focus Ring** | $150\text{ ms}$ | `cubic-bezier(0.2, 0.0, 0.0, 1.0)` | `box-shadow`, `background-color` |
| **Pressed / Active** | $100\text{ ms}$ | `ease-out` | `transform: scale(0.97)` |
| **Drawer Slide In** | $300\text{ ms}$ | `cubic-bezier(0.0, 0.0, 0.2, 1.0)` | `transform: translateX(0)` |
| **Drawer Slide Out** | $250\text{ ms}$ | `cubic-bezier(0.4, 0.0, 1.0, 1.0)` | `transform: translateX(100%)` |

---

## 4. Accessibility & Fallbacks
- **Keyboard Equivalence**:
  - `Space` / `Enter`: Activates control.
  - `Escape`: Closes overlay and returns focus to triggering element.
- **Prefers-Reduced-Motion Fallback**:
  ```css
  @media (prefers-reduced-motion: reduce) {
    .interactive-element {
      transition: opacity 150ms ease !important;
      transform: none !important;
    }
  }
  ```

---

## 5. Verification & Acceptance Criteria
- [ ] Hit area satisfies $44 \times 44\text{ px}$ minimum bounding box.
- [ ] Visual feedback occurs within $100\text{ ms}$ of pointerdown.
- [ ] Maximum animation duration $\le 400\text{ ms}$.
- [ ] Complete keyboard control verified with visible `:focus-visible` ring.
