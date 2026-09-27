# Documentation for LLM Consumption — Verification Checklist

## 1. Frontmatter & Metadata
- [ ] Every touched page carries title, description under 160 characters, reviewed date in ISO format, and audience tag.
- [ ] No draft, deprecated, or tombstoned page is linked from llms.txt.
- [ ] Audience tags are accurate: agent-facing pages avoid unexplained project jargon or define it inline.
- [ ] Reviewed dates older than 180 days are flagged for re-review, not silently kept.

## 2. Progressive Disclosure & Token Discipline
- [ ] Entry page states the outcome and links onward in under 1500 tokens.
- [ ] No single page exceeds 6000 tokens; oversized pages are split with a hub page.
- [ ] Each sub-page opens with a summary of 3 lines or fewer enabling early stopping.
- [ ] Token counts before and after restructuring are recorded in the evidence log.

## 3. Anchors & Cross-Links
- [ ] No duplicate headings exist within any single page.
- [ ] Cross-links use full relative paths plus explicit fragments.
- [ ] Renamed headings with inbound links leave a redirect note for one release cycle.
- [ ] lychee (or equivalent) reports zero broken links and zero broken anchors.

## 4. Single Source of Truth
- [ ] Each fact has exactly one canonical page; all other pages link rather than restate.
- [ ] No paragraph is copy-pasted across two or more pages.
- [ ] llms.txt entries point at canonical pages, never at mirrors or forks.
- [ ] llms.txt stays under 300 lines and matches current navigation structure.

## 5. Verification Gates
- [ ] markdownlint passes with zero errors on all touched pages.
- [ ] Automated verification script `scripts/verify.sh` executes with exit code 0.
- [ ] Chunk audit confirms retrievable sections fall between 150 and 600 tokens.
- [ ] Tables wider than 5 columns were converted to definition lists or split.
