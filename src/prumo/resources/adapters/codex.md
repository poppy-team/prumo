# AGENTS.md

This project uses Prumo v0.2.

- Read `ENTRYPOINT.md`, `prumo.json`, the active Goal and `docs/PRUMO.md`.
- Treat Prumo as an external CLI utility available in PATH (`prumo`). Run `prumo <command>` or `prumo --help` for project operations and lifecycle. Do not inspect or search for internal framework development source code.
- Use Lean Progressive Context: minimum sufficient context, progressive expansion, pointer over payload.
- Do not scan/read the entire repository by default.
- Respect context/output budgets and stop when evidence is sufficient.
- Keep delegation bounded; deep recursion is disabled unless the project explicitly says otherwise.
- Implement only locked Goal scope; tests/evidence determine completion.
- Compute Documentation Delta; patch only impacted canonical docs.
- Do not create task-specific `CONTEXT.md`/`SUMMARY.md` files.
- Record compact Project Intelligence where the project workflow supports it.
- Generated/runtime context is not canonical and should be garbage-collected.
