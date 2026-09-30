# Contributing and Framework Evolution

Prumo should learn from real projects without becoming a dumping ground for project-specific instructions.

A reusable change should identify:

- recurring problem;
- evidence;
- generic rule;
- affected protocol/schemas/catalog/adapters;
- compatibility/migration impact;
- token/cost/maintenance impact when context behavior changes.

## Checklist

- Keep core provider-neutral.
- Keep `ENTRYPOINT.md` and root `AGENTS.md` short.
- Maintain Markdown + JSON as the human-maintained format budget.
- A new persistent format requires ADR-level justification.
- Keep runtime/cache/derived data out of canonical Git state.
- Update schemas when contracts change.
- Add tests for behavior/migration changes.
- Do not bake current model rankings into role definitions.
- Prefer capabilities/risk selectors over technology duplication.
- Prefer semantic virtual chunks over documentation microfiles.
- Context changes need a token/quality benchmark and stopping rule.
- Deep recursion stays experimental unless evidence justifies promotion.
