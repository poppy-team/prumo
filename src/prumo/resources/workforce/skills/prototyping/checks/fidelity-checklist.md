# Prototyping & Interaction Verification Checklist

## 1. Scope & Hypothesis
- [ ] **Core Persona & Scenario:** Target user role, prior knowledge, and environmental context explicitly defined.
- [ ] **Testable Hypothesis:** Stated in format *"We believe [user segment] will achieve [task outcome] within [time/clicks] because [interface affordance]"*.
- [ ] **Critical Path Demarcation:** The happy path and primary error paths are fully clickable; dead-end placeholder links are labeled or disabled.

## 2. State & Interaction Completeness
- [ ] **Complete State Matrix:** All interactive controls specify `idle`, `hover`, `focus-visible`, `active`, `loading`, `success`, and `error` states.
- [ ] **Empty & Error States:** First-time onboarding views, zero search results, network timeout, and input validation failures are prototyped.
- [ ] **Transition Feedback:** Action feedback appears within $100$ ms; loading spinners appear for operations taking $> 300$ ms.
- [ ] **Spring / Easing Parameters:** Motion physics use non-linear curves (`cubic-bezier(0.16, 1, 0.3, 1)` or standard spring dynamics).

## 3. Ergonomics & Accessibility
- [ ] **Touch Target Size:** Interactive areas are at least $44 \times 44$ px (or $48 \times 48$ px on mobile, meeting WCAG 2.5.5).
- [ ] **Keyboard Flow:** Tab key traverses interactive elements in logical visual reading order without trapping focus.
- [ ] **Modal Traps:** Dialog overlays trap focus within the modal while open and return focus to the trigger on `Esc` or close.
- [ ] **Screen Reader Labels:** Form inputs and icon-only buttons declare `aria-label` or visible labels.

## 4. Usability Testing Preparedness
- [ ] **Task Scripts:** Scripted scenarios written without leading instructions or terminology clues.
- [ ] **Success Metrics:** Time on task, task completion rate ($\ge 80\%$), and System Usability Scale (SUS $\ge 75$) targets recorded.
- [ ] **Observer Logging Sheet:** Sheet prepared with columns for Timestamp, Observed Behavior, User Quotation, and Friction Severity.
