# UI/UX Heuristic Evaluation & Cognitive Walkthrough Checklist

## 1. Cognitive Walkthrough Gates
- [ ] **Goal Visibility**: User can immediately identify how to initiate their target task without tutorial intervention.
- [ ] **Affordance & Signifiers**: Interactive elements (buttons, links, inputs) visually communicate their clickability and interaction mode.
- [ ] **Feedback Immediacy**: Every user action produces visible or auditory feedback within 100ms.
- [ ] **Progress Clarity**: Multi-step operations indicate progress, steps remaining, and estimated completion.

## 2. Nielsen Norman 10 Heuristics Assessment
- [ ] **H1: Visibility of System Status**: Loading spinners, indeterminate progress bars, and status badges reflect actual background state.
- [ ] **H2: Match Between System & Real World**: Terminology, icons, and units match user's domain vernacular; zero internal technical jargon.
- [ ] **H3: User Control & Freedom**: Explicit Cancel buttons, Close icons, Undo capabilities, and non-destructive escape routes.
- [ ] **H4: Consistency & Standards**: UI adheres to standard platform conventions (button placement, icon meanings, keyboard shortcuts).
- [ ] **H5: Error Prevention**: Dangerous or irreversible actions (e.g. deletion) require explicit confirmation dialogs; inputs validate formatting defensively.
- [ ] **H6: Recognition Over Recall**: Instructions, labels, and contextual helpers are visible where needed rather than requiring memorization.
- [ ] **H7: Flexibility & Efficiency**: Power users can utilize keyboard shortcuts, tab navigation, and search accelerators.
- [ ] **H8: Aesthetic & Minimalist Design**: Visual hierarchy guides the eye to primary CTAs; unnecessary decorative clutter eliminated.
- [ ] **H9: Error Recovery**: Error messages describe what happened plainly and offer a 1-click recovery path.
- [ ] **H10: Help & Documentation**: Contextual tooltips, empty state guides, or documentation links assist users at points of friction.

## 3. Ergonomics & Interaction Quality
- [ ] **Touch Target Sizing**: Minimum clickable area is $\ge 44 \times 44\text{ px}$ (or $\ge 24\text{ px}$ with spacing per WCAG 2.2 SC 2.5.8).
- [ ] **Contrast Compliance**: Text contrast meets WCAG AA ($4.5:1$ for body, $3:1$ for large text).
- [ ] **Zero Dark Patterns**: No confirmshaming, pre-checked consent boxes, or hidden fees.
