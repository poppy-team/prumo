# Portable Continuation

D1 separates durable Prumo Runs from temporary executor sessions.

```text
Run != ExecutorSession
```

A Run remains the same when model, harness, or session changes. The current baseline stores derived runtime records under `.prumo/runtime/` and exposes:

```bash
prumo run --run R001 --path ./project
prumo continue --run R001 --path ./project --json
prumo continue --run R001 --path ./project --prompt
```

The portable continuation record must contain enough structured state to rehydrate a fresh executor: Run identity, Goal/Task refs, branch/revision/dirty state, completed/current/pending work, decisions, evidence, blockers, pending side effects, and next steps.

Do not persist transcript, chain-of-thought, credentials, or raw sensitive tool output.

Before D1 continuation is fully wired to real Run/Checkpoint persistence, use `.prumo/runtime/bootstrap-continuation.md` as temporary derived state. It must remain ignored and contain only engineering facts.

Resume safety requires:

1. inspect current branch and working tree;
2. compare revision with checkpoint;
3. reconcile unknown side effects;
4. recompile context for the new executor;
5. avoid replaying completed or uncertain side effects.
