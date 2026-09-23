# LLM Documentation Plan Specification

## 1. Corpus Identity
- **Corpus root**: docs/
- **Audience**: both (human developers and coding agents)
- **Corpus size at plan time**: 48 pages, 61,400 words
- **Entry page**: docs/index.md, currently 2100 tokens, target under 1500 tokens
- **Index file**: llms.txt at repository root, currently absent, to be created under 300 lines

## 2. Canonical Source Map

| Topic | Canonical page | Pages currently duplicating it |
|---|---|---|
| Control plane loop | docs/runtime/control-plane.md | docs/development/implementation-blueprint.md intro section |
| Token budget policy | docs/architecture/dependency-rules.md | docs/development/coding-standards.md appendix |
| Auth for automation tokens | docs/security/service-tokens.md | docs/runtime/living-plan.md setup notes |

## 3. Restructure Plan

| Page | Current tokens | Action | Target tokens |
|---|---|---|---|
| docs/security/authentication.md | 7200 | Split into hub + oauth2-code-flow.md + service-tokens.md + troubleshooting-auth.md | hub 900, details 1400 / 1100 / 1800 |
| docs/index.md | 2100 | Trim history section to link, keep outcome + links | 1200 |
| docs/runtime/living-plan.md | 5400 | Extract setup notes to canonical auth page, link instead | 3900 |

## 4. Frontmatter Rollout
- **Schema**: title, description max 160 chars, reviewed ISO date, audience tag, canonical path.
- **Example**: title "Service Tokens", description "Issue and rotate machine tokens for CI and automation clients.", reviewed 2026-09-10, audience both.
- **Enforcement**: markdownlint MD041 plus scripts/check-frontmatter.sh in CI; zero tolerance on new pages.

## 5. Verification Evidence
- [ ] markdownlint: zero errors across docs/ (baseline run 2026-09-18 showed 14 errors, all fixed).
- [ ] lychee: 23 of 23 anchors resolve, zero broken links.
- [ ] Token counts before/after recorded in evidence log evidence/llm-docs-2026-09-23.md.
- [ ] `scripts/verify.sh` exits 0.
