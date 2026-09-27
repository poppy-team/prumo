# Documentation Release Specification

## 1. Release Identity
- **Release**: v0.4.0, cut 2026-09-23
- **Generator**: Astro Starlight 4.14.2, Node 22.11.0 per .nvmrc
- **Base URL**: https://docs.prumo.dev
- **Hosting**: Cloudflare Pages, production branch `release-docs`
- **Visibility mode**: production build with PUBLIC_DOCS_ONLY=true

## 2. Route Changes

| Route | Change | Redirect |
|---|---|---|
| /docs/runtime/control-plane/ | New page from docs/runtime/control-plane.md | none |
| /docs/architecture/dependency-rules/ | New page, 4 sections | none |
| /docs/guides/migration-v03-v04/ | New migration guide | none |
| /docs/start/install/ (renamed from /docs/start/installation/) | Rename for nav consistency | 301 /docs/start/installation/ -> /docs/start/install/ |
| /docs/api/goals/ (renamed from /docs/api/goal-objects/) | Rename to match Go package name | 301 /docs/api/goal-objects/ -> /docs/api/goals/ |
| /docs/v03/legacy-auth/ | Removed, superseded by service tokens page | 301 /docs/v03/legacy-auth/ -> /docs/security/service-tokens/ |

## 3. Sitemap & Index
- **Public routes before**: 41; **after**: 47; internal routes leaked: 0.
- **Sitemap**: sitemap.xml regenerated; diff reviewed 2026-09-23, all 6 additions match the changelog.
- **Search index**: rebuilt; all 6 new pages return results for their title queries.
- **Canonical tags**: spot-checked 12 pages, all point at https://docs.prumo.dev canonical URLs.

## 4. Verification Evidence
- [ ] `npm run build` clean log from pinned toolchain, build time 96 seconds.
- [ ] lychee over built output: 312 of 312 links valid, zero broken anchors.
- [ ] Preview deploy smoke test: top 10 routes HTTP 200 with correct canonical tags.
- [ ] `scripts/verify.sh` exits 0.
