# Game UI Testing — Verification Checklist

## 1. Focus Navigation & Input
- [ ] Every screen declares an initial focused control; focus is never homeless while a screen is active.
- [ ] Four-directional gamepad/keyboard sweeps (up/down/left/right) reach all focusable widgets with no traps.
- [ ] Focus indicator (highlight ring, scale, or color shift) is visible on every focus stop at 3 m TV viewing distance.
- [ ] Confirm/cancel bindings match platform convention (e.g., gamepad South = confirm, East = cancel on Xbox layout).
- [ ] Key repeat and gamepad button bounce are debounced: one press yields exactly one activation event.
- [ ] Mouse/touch hover states do not steal gamepad focus; last-input device arbitration is deterministic.

## 2. Safe Areas, Scaling & Aspect Ratios
- [ ] No interactive control lies outside the action-safe rectangle (3.5% inset) on any declared target resolution.
- [ ] No critical text or HUD readout lies outside the title-safe rectangle (5% inset).
- [ ] Layouts verified at 1920x1080, 1280x720, 3840x2160, plus ultrawide 3440x1440 and 4:3 1024x768 where supported.
- [ ] All dynamic positioning uses anchors, containers, or layout groups; no hard-coded pixel offsets for text-bearing widgets.
- [ ] Nine-patch/slice borders scale without distortion at 200% UI scale factor.

## 3. Localization & Text Robustness
- [ ] Pseudo-localized build (40% expansion, accented padding) shows zero clipped or overlapping labels.
- [ ] ja-JP build renders with zero missing-glyph tofu; font fallback chain covers CJK punctuation and emoji used in UI.
- [ ] ar-SA build mirrors navigation bars, progress bars, and breadcrumb order (RTL) while keeping numerals and health values LTR.
- [ ] No concatenated or order-dependent sentence fragments; all strings use positional placeholders (`{0}`, `{player}`).
- [ ] Longest-locale screenshots archived per screen with overflow annotations resolved or triaged.

## 4. State Transitions & Regression Evidence
- [ ] Pause/resume, modal stacking, and scene-change transitions complete within 500 ms of input at 50 ms clamped frame time.
- [ ] Exactly one screen owns input focus after every transition; background screens ignore input.
- [ ] Screenshot baselines exist per screen, resolution, and locale; pixel diff against baseline is at or below 0.5%.
- [ ] Transition tests run green in CI on the build under test; logs attached to the test specification.
- [ ] Automated verification script `scripts/verify.sh` executes with exit code 0.
