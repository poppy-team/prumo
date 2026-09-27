# Frontend Web Page Specification Template

## 1. Page Overview
- Page: Checkout review at https://shop.acme.example/checkout/review owned by Storefront team
- Stack: React 18 with Vite 6, React Query 5, React Router 7, served by Fastify SSR shell
- Device matrix: Pixel 8, Moto G54 on 4G throttled, MacBook Air M2, NVDA on Firefox plus VoiceOver on Safari
- Budgets: initial JS under 200 KB gzip, LCP under 2.5 s on Moto G54, CLS under 0.05, Lighthouse 90+ all categories

## 2. Component Inventory
- OrderSummary card: semantic section with h2, line items as list, totals in description list, no layout shift
- AddressForm: six labeled inputs, inline errors via aria-describedby, submit validation, 250 ms city autocomplete debounce
- PayButton: native button, disabled while processing, aria-live polite announces Payment authorized for order ord_9f31
- ErrorPanel: role alert on gateway failure with Retry payment button preserving all entered data

## 3. Accessibility Verification
- axe-core 4.10 run: zero serious, zero critical, two moderates fixed (duplicate landmark, low-contrast helper text raised to 4.6:1)
- Keyboard pass: full checkout completable in 41 Tab stops, modal traps verified, skip link first stop
- Screen reader pass: NVDA 2026.1 reads item quantities and totals correctly; VoiceOver rotor shows single h1 and six landmarks

## 4. Performance & Test Evidence
- Lighthouse mobile: performance 94, accessibility 98, best practices 100, SEO 100; desktop performance 99
- Bundle: initial route 184 KB gzip, payment widget split to 62 KB lazy chunk, hero image AVIF 38 KB with explicit 1200x630
- Playwright: 26 flows green on Chromium 131, WebKit 26, Firefox 133; vitest 188 passed
- verify.sh: exit code 0 on 2026-09-23 run by storefront pipeline job 882
