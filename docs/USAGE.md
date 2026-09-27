# Usage

## 1. Discuss and stabilize decisions

Use any model/tool to explore product, architecture and constraints. Promote only stable reusable knowledge to the repository.

## 2. Create a JSON project profile

Record project type, stack, features, risks, orchestration/autonomy and explicit model roster.

See:

```text
examples/brasa/project-profile.json
```

## 3. Initialize

```bash
prumo-agent init ./project --profile ./project-profile.json --non-interactive
```

New projects receive:

- `prumo.json`;
- `PROJECT_STATE.md`;
- `docs/PRUMO.md`;
- JSON workforce/model manifests;
- `.ai/goals/`;
- durable Project Intelligence seed;
- gitignore rules for runtime/cache state.

## 4. Validate

```bash
prumo-agent validate ./project --schemas ./schemas
```

## 5. Define and lock Goals

Replace placeholder acceptance criteria before LOCKED. Agents execute against locked scope and recorded project decisions.

## 6. Use progressive context

Before broad work:

1. start from PRUMO + active Goal;
2. use direct/symbol/structural context first;
3. load a known Context Pack when available;
4. expand only if evidence is insufficient;
5. use isolated depth-1 delegation only when beneficial;
6. stop when the task is sufficiently grounded.

Do not generate permanent task context files.

## 7. Compile adapters only when needed

```bash
prumo-agent compile --target codex --path ./project
prumo-agent compile --target traycer --path ./project
```

Generic/chat compiled context is runtime state.

## 8. Finish a task

```text
implementation
→ tests
→ review
→ documentation delta
→ evidence
→ task/project intelligence
→ context cleanup
```

Keep reports compact and distinguish observed costs from estimates.

## 9. Publish docs

Canonical Markdown remains source of truth. Projects may continuously build a documentation site/dashboard from the same sources; do not maintain a second documentation copy.

## 10. Recover anywhere

A fresh model/tool reads `ENTRYPOINT.md`, `prumo.json`, `PROJECT_STATE.md`, `docs/PRUMO.md`, the active Goal and then expands context on demand.

```bash
prumo-agent snapshot ./project
```

Snapshots exclude runtime/cache state.

## Legacy v0.1

YAML profiles/projects remain readable for migration where supported. Run a documented migration before editing them as v0.2 canonical state.
