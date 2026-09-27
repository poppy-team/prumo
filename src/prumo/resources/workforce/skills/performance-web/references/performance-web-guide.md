# Web Performance Reference Guide

## 1. Core Concepts

### 1.1 The Vitals and Their Math
Largest Contentful Paint (LCP) marks when the largest content element renders; good is under 2.5 s at
p75. Interaction to Next Paint (INP) measures worst-case event-to-paint latency across a visit; good is
under 200 ms. Cumulative Layout Shift (CLS) sums shift scores (impact fraction times distance fraction);
good is under 0.1. Time to First Byte (TTFB) bounds server plus network responsiveness; good is under
800 ms. First Contentful Paint (FCP) under 1.8 s confirms the critical render path is alive. All field
thresholds apply at the 75th percentile of real users, segmented by device and connection, never as a
single lab number.

### 1.2 Lab vs Field
Lab (Lighthouse, WebPageTest) is deterministic and debuggable but simulates one device on one network.
Field (RUM, CrUX) captures real distributions including the long tail of low-end devices. Method: use lab
to find causes, field to judge outcomes. A 300 ms lab LCP gain means nothing until RUM p75 over 7 days
confirms it; seasonal traffic and device mix routinely overturn lab-only verdicts.

### 1.3 Critical Render Path
HTML parses, CSS blocks rendering, JS blocks parsing unless deferred. The critical path is the minimal
set of bytes needed for first paint: HTML, critical CSS, hero image, fonts. Everything else (analytics,
chat, recommendations) must be async, deferred, or lazy. Each additional render-blocking request on 4G
adds roughly 150–300 ms; chains of three or more blocking scripts are the classic 2-second regression.

### 1.4 Waterfall Reading
Read top to bottom: DNS, TLS, TTFB, content download, then dependencies fanning out. Orange (waiting) on
the root document means server latency; long blue (download) bars mean oversized assets; wide gaps mean
parser-blocking scripts. Annotate the three longest bars with owner and fix before touching anything else:
waterfalls reward the biggest bar, opinions reward the loudest engineer.

### 1.5 Third-Party Cost Model
Each external script pays DNS plus TLS plus download plus main-thread execution, often 200–600 ms total
on mid-tier Android. Tag managers hide multiplicative cost: one container loading six vendors. Budget rule:
page keeps at most 5 synchronous third parties, each under 200 ms main-thread time, every one with a named
owner and a quarterly removal review. Facade patterns (click-to-load chat) convert 400 ms automatic costs
into zero until the user asks.

## 2. Patterns and Anti-Patterns

| Area | Pattern (do) | Anti-Pattern (avoid) |
|---|---|---|
| Hero | AVIF 80 KB, fetchpriority high, explicit size | 2.4 MB PNG discovered late by the parser |
| Fonts | Subset WOFF2, preload one weight, swap | Five weights blocking text for 1.8 s |
| JS | Route-split, defer, 200 KB gzip budget | 1.1 MB synchronous bundle on every page |
| Third party | Facade chat, async analytics, owner each | Six synchronous pixels in head |
| Caching | Immutable hashed assets, 1-year max-age | Unversioned bundle.js cached 5 minutes |
| Images | srcset plus sizes, lazy below fold | 3000 px desktop image on 360 px phones |
| Measurement | 3 lab runs plus 7-day RUM p75 | Single Lighthouse run declared victory |

## 3. Code Example: Lighthouse CI Budgets that Fail the Build

```json
{
  "ci": {
    "collect": {
      "url": ["https://shop.acme.example/", "https://shop.acme.example/checkout/review"],
      "numberOfRuns": 3,
      "settings": { "preset": "desktop", "formFactor": "mobile" }
    },
    "assert": {
      "assertions": {
        "largest-contentful-paint": ["error", { "maxNumericValue": 2500 }],
        "interaction-to-next-paint": ["error", { "maxNumericValue": 200 }],
        "cumulative-layout-shift": ["error", { "maxNumericValue": 0.1 }],
        "resource-summary:script:size": ["error", { "maxNumericValue": 204800 }]
      }
    }
  }
}
```

Budgets run on every pull request against preview deploys: LCP, INP, and CLS thresholds mirror the field
gates, and the 204,800-byte script cap enforces the 200 KB gzip discipline before merge, not after release.
