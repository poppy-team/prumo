# Prumo — Agent Entrypoint

If you are an AI system using this framework:

1. Read `FRAMEWORK.md` for invariants.
2. Read `docs/PRUMO.md` as the intent router; do not read the whole repository by default.
3. Read `docs/LEAN_PROGRESSIVE_CONTEXT.md` before planning broad context ingestion.
4. For an actual project, read `prumo.json`, `PROJECT_STATE.md` and `docs/PRUMO.md`.
5. Read the active Goal, applicable invariants and only the linked canonical sections needed for the task.
6. Use the selected Agents, Skills and Recipes from `.ai/`; never activate the entire registry indiscriminately.
7. Prefer pointers and structural slices over full documents/files. Expand context only when evidence is insufficient.
8. Keep intermediate output compact. Do not narrate exploration unless the user needs it.
9. Never silently weaken locked acceptance criteria.
10. Treat orchestrators and model providers as replaceable execution adapters.
11. Update only documentation actually impacted by stable behavior changes.
12. Before completion, record evidence, task/project intelligence where available, then garbage-collect temporary context.
13. CI, tests and recorded evidence—not model confidence—determine completion.
14. Go v0.6 is the active implementation. Python v0.3 remains a read-only compatibility oracle.

Legacy v0.1 projects may still contain YAML. Read it only for compatibility/migration; new canonical output must use Markdown + JSON.
