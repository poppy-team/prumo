# Goal System v2

Goals represent outcome-based milestones.

## Goal Lifecycle
1. `DRAFT`: Initial drafting of objective, constraints, and acceptance criteria.
2. `PLANNED`: Reviewed and mapped into task dependencies.
3. `LOCKED`: Acceptance criteria are cryptographically locked via SHA256 digest.
4. `EXECUTING`: Tasks running under active Plan.
5. `VERIFYING`: Evidence being collected and verified against gates.
6. `REVIEWING`: Cross-provider review of changes.
7. `DONE`: All acceptance criteria verified by recorded evidence.

## Goal Amendments
Once `LOCKED`, criteria cannot be silently edited. Changes require formal amendments:
```bash
prumo-agent goal amend P01-G01 --file amendment.json
```
