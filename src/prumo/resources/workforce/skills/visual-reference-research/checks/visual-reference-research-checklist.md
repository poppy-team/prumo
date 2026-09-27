# Visual Reference Research — Verification Checklist

## 1. Brief & Scope
- [ ] 3–5 brand adjectives and hard constraints (existing brand color, WCAG AA floor) recorded before collection.
- [ ] Screens in scope listed; references outside scope excluded.

## 2. Collection & Provenance
- [ ] 8–15 full-viewport captures, each with source URL, capture date, and viewport size.
- [ ] No unsourced fragments; licensing permits internal reuse of captures.

## 3. Token Extraction & Accessibility
- [ ] Per-reference tokens extracted: palette hex values (≤5), type roles with scale ratio, spacing base, radius/shadow language.
- [ ] Body-text contrast pairs computed for every candidate direction; failing pairs logged in exclusions, never recommended.

## 4. Convergence & Decision
- [ ] References clustered into 2–3 directions, scored against brief adjectives.
- [ ] Exactly one direction recommended with per-reference rationale sentences; runner-up summarized.
- [ ] Exclusions log lists every rejection with a reason.

## 5. Handoff & Evidence
- [ ] Candidate token table (color roles, type scale, spacing, radius) emitted with contrast evidence.
- [ ] Moodboard file wired and reviewable; `scripts/verify.sh` exits 0 from the repository root.
