# Information Architecture Verification Checklist

## Inventory and Ownership
- [ ] Screens, entities, actions, settings, help content, and cross-links are inventoried.
- [ ] Each item has an owner, canonical destination, and source of truth.
- [ ] Orphans, dead ends, duplicate destinations, and protected content are identified.

## Taxonomy and Labels
- [ ] Groups reflect validated user mental models rather than the organization chart.
- [ ] Labels use user vocabulary and avoid internal jargon or ambiguous icons.
- [ ] Rejected labels and naming conflicts are recorded.

## Navigation Structure
- [ ] Primary, settings, and transactional depth budgets are explicit.
- [ ] Critical tasks have a direct path and an efficient recovery path.
- [ ] Navigation remains usable under labels, narrow viewports, and assistive technology.

## Progressive Disclosure
- [ ] Common decisions are visible; advanced scope has an explicit and reversible affordance.
- [ ] Destructive consequences, consent, errors, and completion paths are never hidden.
- [ ] State transitions preserve context and provide a clear way back.

## Wayfinding Evidence
- [ ] Tree-test and first-click results meet declared success thresholds.
- [ ] Failed paths produce specific label or structure changes, not cosmetic-only edits.
- [ ] `scripts/verify.sh` exits with code 0 from the repository root.
