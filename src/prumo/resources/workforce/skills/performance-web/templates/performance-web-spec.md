# Web Performance Audit Specification Template

## 1. Scope & Budgets
- Property: shop.acme.example storefront, audited 2026-09-18 by performance guild, owner Paulo Henrique
- Flows: homepage, search listing with 48 items, checkout review, order confirmation
- Budgets: LCP under 2.5 s, INP under 200 ms, CLS under 0.1, TTFB under 800 ms, initial JS under 200 KB gzip
- Rigs: Lighthouse 12 mobile emulation 3 runs, WebPageTest Dulles 4G throttled, RUM provider SpeedCurve, CrUX 28-day window

## 2. Findings by Cost
- Chat widget vendor TalkFast: 480 ms main-thread, synchronous, owner missing → facade behind click-to-chat, saves 430 ms LCP on listing
- Hero JPEG 1.9 MB product collage: AVIF 82 KB replacement with fetchpriority high, saves 1.1 s LCP on homepage
- Three font weights render-blocking: keep 400 plus 700 subset, swap display, saves 600 ms invisible text on 4G
- Recommendations carousel hydrating 240 KB JS below fold: intersection-lazy, saves 180 KB initial JS on homepage

## 3. Before / After Evidence
- Lighthouse mobile homepage: performance 61 → 93, LCP 4.8 s → 2.1 s, CLS 0.22 → 0.03, TTFB 690 ms unchanged
- WebPageTest waterfall: blocking chain 7 → 2 requests, total bytes 3.4 MB → 1.1 MB, finish 9.2 s → 3.8 s
- RUM p75 over 7 days post-flag at 10 percent: LCP 3.9 s → 2.4 s, INP 240 ms → 170 ms, CLS 0.18 → 0.04
- CrUX 28-day trend: good-URL share for LCP rises 54 percent → 81 percent, checkout p75 LCP stable at 1.9 s

## 4. Gates & Follow-Up
- Lighthouse CI assertions merged for 4 routes; bundle cap 200 KB gzip enforced on pull requests
- TalkFast facade ships behind flag chat_facade_v2, full rollout after 7-day RUM confirmation ending 2026-09-30
- verify.sh: exit code 0 on 2026-09-23 run by performance pipeline job 207
