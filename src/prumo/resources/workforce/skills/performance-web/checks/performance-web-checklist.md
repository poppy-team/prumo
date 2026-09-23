# Web Performance — Verification Checklist

## 1. Core Web Vitals & Field Data
- [ ] RUM p75 values captured for LCP, INP, CLS over the last 28 days plus CrUX history for the same URLs.
- [ ] LCP under 2.5 s, INP under 200 ms, CLS under 0.1 at p75; any breach has a named owner and dated remediation.
- [ ] Lab results (Lighthouse, WebPageTest) and field data agree in direction before a win is claimed.
- [ ] Checkout-flow p75 LCP has not regressed more than 100 ms versus the previous release.

## 2. Budgets & Bundle Discipline
- [ ] Lighthouse CI assertions enforce performance, accessibility, best-practices, and SEO minimums per route.
- [ ] Initial route JavaScript stays under 200 KB gzip; CI fails builds exceeding budget by more than 5 percent.
- [ ] Bundle visualizer report identifies the five largest modules with removal or lazy-load decisions.
- [ ] Unused CSS and dead locales are pruned; icon imports are tree-shaken per route.

## 3. Images, Fonts & Critical Path
- [ ] Hero images use AVIF/WebP under 100 KB with explicit dimensions, fetchpriority high, and responsive srcset.
- [ ] Fonts are subset, preloaded once, served with font-display swap; invisible-text duration is near zero.
- [ ] Critical CSS is inlined, non-critical JS deferred, preconnects limited to two origins.
- [ ] Below-fold widgets (chat, reviews, recommendations) lazy-load on interaction or intersection.

## 4. Network Waterfall & Third Parties
- [ ] WebPageTest waterfall at 4G throttled shows TTFB under 800 ms with server versus network split annotated.
- [ ] Every third-party script has an owner, a measured main-thread cost, and blocking versus async classification.
- [ ] No single third party exceeds 200 ms main-thread time; ownerless scripts are removed.
- [ ] scripts/verify.sh executes with exit code 0.
