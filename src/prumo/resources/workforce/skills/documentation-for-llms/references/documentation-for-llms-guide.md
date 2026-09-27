# Documentation for LLM Consumption — Reference Guide

## 1. Core Concepts

### 1.1 Progressive Disclosure
Agents operate under token budgets, so documentation must be layered: a thin entry layer
(outcome + links, under 1500 tokens) and detail layers reachable on demand. Each layer must
be self-sufficient for a stop decision: after reading any layer, the agent knows whether to
stop or which link to follow next. Never bury the decision (which command to run, which file
is canonical) three sections deep.

### 1.2 The llms.txt Convention
Per llmstxt.org, a Markdown file at the site root gives LLMs a sitemap: sections with
one-line descriptions and links. Keep it flat, under 300 lines, and ordered by task
frequency rather than alphabet. Regenerate it from navigation config so it cannot drift.

### 1.3 Frontmatter Schema
Every page carries machine-readable metadata:

```yaml
---
title: Control Plane Loop
description: How the Prumo control plane schedules Goals and enforces evidence gates.
reviewed: 2026-09-10
audience: both
canonical: docs/runtime/control-plane.md
---
```

`canonical` declares the single source of truth for the topic and lets linters detect
duplicate coverage across pages.

### 1.4 Chunk Discipline
Retrieval systems slice pages into chunks. Sections of 150 to 600 tokens survive slicing
intact; a 2500-token section gets cut mid-argument. One heading, one idea, one section.

### 1.5 Anchor Stability
GitHub-style anchors derive from heading text, so heading renames break inbound links.
Treat published headings as API: rename only with a redirect note, and prefer adding a new
subsection over rewording a linked heading.

## 2. Patterns and Anti-Patterns

| Pattern (do this) | Anti-Pattern (never do this) |
|---|---|
| Hub page under 1500 tokens linking to detail pages | Single 9000-token mega-page covering install, config, API, and FAQ |
| One canonical page per fact, linked everywhere | Same install snippet pasted into 6 pages, drifting apart |
| Explicit unique headings (`## Retry policy for CDC consumers`) | Three `## Configuration` headings in one page (anchor collision) |
| Fenced code blocks with language tags | Bare indented code that linters and highlighters misparse |
| llms.txt regenerated from nav config in CI | Hand-edited index that silently rots after 3 nav changes |
| Tables with at most 5 columns | 9-column matrixes that explode chunk token counts |

## 3. Worked Example
A 7200-token `authentication.md` is split into a 900-token hub (`authentication.md`:
which flow to pick + links) and three detail pages (`oauth2-code-flow.md` at 1400 tokens,
`service-tokens.md` at 1100 tokens, `troubleshooting-auth.md` at 1800 tokens). Each page
gets the frontmatter schema above, llms.txt gains four entries, lychee confirms 23 of 23
anchors resolve, and markdownlint passes clean.
