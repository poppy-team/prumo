# Documentation Publishing — Verification Checklist

## 1. Toolchain & Reproducible Builds
- [ ] Generator version is pinned in package.json and `npm ci` reproduces the build.
- [ ] `npm run build` succeeds on a clean checkout with no local-only plugins.
- [ ] Node version matches .nvmrc or volta pin; no uncommitted config overrides.

## 2. Source-to-Route Mapping
- [ ] Every canonical Markdown file maps to exactly one route in nav config.
- [ ] Zero orphan pages: every built page is reachable from nav, index, or sitemap.
- [ ] Public vs internal visibility is decided by frontmatter plus build flags, never by copied files.

## 3. Links, Anchors & Redirects
- [ ] Build plus lychee report zero broken internal links and zero broken anchors.
- [ ] Every moved or renamed public route ships a 301 redirect in the same release.
- [ ] No redirect chains longer than 2 hops; redirect targets return HTTP 200.

## 4. Sitemap, SEO & Search Index
- [ ] sitemap.xml lists every public route and zero internal routes.
- [ ] Canonical link tags point at the production base URL on all public pages.
- [ ] Search index rebuilds after content changes; new pages appear in site search.

## 5. Release & Smoke Tests
- [ ] Version snapshot (sidebar + changelog entry) is recorded for the release.
- [ ] Top 10 routes return HTTP 200 on the preview deploy before promotion.
- [ ] Automated verification script `scripts/verify.sh` executes with exit code 0.
