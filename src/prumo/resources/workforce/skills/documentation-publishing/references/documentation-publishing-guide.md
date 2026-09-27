# Documentation Publishing — Reference Guide

## 1. Core Concepts

### 1.1 Single-Source Builds
The generator compiles the same Markdown files developers edit. Content config (Astro
content collections with a zod schema, or Docusaurus docs plugin with frontmatter
validation) rejects pages missing required fields at build time, so drift is a build
failure rather than silent rot.

### 1.2 Public/Internal Views
One file, one route, with `visibility: public|internal` in frontmatter. The production
build filters internal pages from nav, sitemap, and search index via an env flag such as
`PUBLIC_DOCS_ONLY=true`. Preview builds render everything for reviewers. Forked copies
are the failure mode this replaces: they always diverge within two releases.

### 1.3 Versioned Docs
On each minor release, snapshot the sidebar and freeze the route tree under
`/docs/v0.4/`. The changelog entry lists added, moved, and removed routes. Redirects for
renames ship in the same release, or external bookmarks die the day you publish.

### 1.4 Link Integrity
Relative Markdown links are validated at build time by both Astro and Docusaurus.
Anchors are not: a renamed `##` heading silently breaks `#fragment` links. Lychee over
the built HTML output closes that gap because it resolves fragments against rendered
heading ids.

### 1.5 Sitemap Discipline
sitemap.xml is the contract with search engines and with agents that crawl the site.
After every release, diff it: additions must match the changelog, removals must each
have a redirect, and internal routes must never appear.

## 2. Patterns and Anti-Patterns

| Pattern (do this) | Anti-Pattern (never do this) |
|---|---|
| Pinned generator + `npm ci` reproducible build | Floating `latest` tag; builds differ per machine |
| Visibility via frontmatter + build flags | `docs-public/` copy-pasted fork of `docs/` |
| Redirect shipped in the same release as the move | Rename now, "redirects later" (later never comes) |
| Lychee over built HTML catching bad anchors | Only checking Markdown links, missing fragment rot |
| Sitemap diff reviewed per release | Sitemap ignored for a year, full of dead routes |

## 3. Worked Example
The Prumo docs site (Astro Starlight 4.14.2, base URL https://docs.prumo.dev) cuts
release v0.4.0: 6 new pages, 3 renames each with a 301 redirect, sitemap grows from 41
to 47 public routes with zero internal leaks, lychee reports 312 of 312 links valid,
and the top-10 route smoke test returns HTTP 200 across the board before promotion.
