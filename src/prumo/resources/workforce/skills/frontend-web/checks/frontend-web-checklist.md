# Frontend Web Engineering — Verification Checklist

## 1. Semantic Markup & Landmarks
- [ ] Page uses header, nav, main, footer landmarks exactly once each where applicable, with a single h1.
- [ ] Heading levels never skip (h2 follows h1, h3 follows h2) and interactive elements use native button, a, input, select, dialog.
- [ ] axe-core reports zero serious or critical violations on every route.
- [ ] Decorative images carry empty alt; informative images carry concise alt under 125 characters.

## 2. Keyboard & Focus Management
- [ ] All functionality is reachable and operable by keyboard alone with no traps; Tab order matches visual order.
- [ ] Focus indicator is visible with at least 3:1 contrast against adjacent colors on every control.
- [ ] Modals trap focus, close on Escape, and restore focus to the triggering element.
- [ ] Skip-to-content link is the first Tab stop and becomes visible on focus.

## 3. Forms, ARIA & Screen Readers
- [ ] Every input has an explicit associated label; errors are inline and linked via aria-describedby.
- [ ] aria-live regions announce async results (polite) and errors (assertive) without double announcements.
- [ ] Custom widgets implement correct roles, states, and keyboard patterns per APG (combobox, tabs, dialog).
- [ ] Color is never the sole carrier of meaning; contrast meets 4.5:1 for text and 3:1 for large text and UI components.

## 4. Data Fetching & Resilient States
- [ ] Loading skeletons, empty states, error panels with retry, and offline banners are implemented for every fetch.
- [ ] Search inputs debounce at 250 ms; requests time out at 2 s with typed user-facing messages.
- [ ] Optimistic updates roll back on failure and surface the error without losing user input.
- [ ] Pagination cursors and error codes match the backend-api contract exactly.

## 5. Performance Budgets & Test Evidence
- [ ] Initial route JavaScript is under 200 KB gzip; route-level code splitting is verified in the bundle report.
- [ ] Images use AVIF/WebP with explicit dimensions; CLS stays under 0.1 in Lighthouse mobile emulation.
- [ ] Playwright flows pass on Chromium, WebKit, and Firefox; Lighthouse scores 90+ in all four categories.
- [ ] scripts/verify.sh executes with exit code 0.
