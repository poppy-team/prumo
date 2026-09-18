# Living Plan sample — zero-to-ready in one review

This project demonstrates the complete M6 Living Plan loop end to end: a
project starts **blocked** (its `project.scope` contract is unbound), and a
planning session resolves the two open questions, governs the resulting
documentation delta, and exits with a **passing readiness** report and an
acceptance/test goal.

It ships in the state the loop reaches after step 6: both contracts bound and
ready, with the applied delta recorded in
[`docs/governance-delta-applied.json`](docs/governance-delta-applied.json).

## Artifacts

| File | Role |
|------|------|
| `prumo.json` | canonical project configuration |
| `docs/profiles/builtin.json` | project-scoped mini registry: one `core-software` profile |
| `docs/contracts/builtin.json` | the two applicable contracts (`product.vision`, `project.scope`) |
| `docs/contracts/bindings.json` | contract → source bindings, ownership and answered questions |
| `docs/product/vision.md` | canonical intent document (bound and ready) |
| `docs/project/scope.md` | canonical scope document authored during the loop (step 5) |
| `docs/governance-delta-applied.json` | the applied documentation delta recording both decisions |

## Replay the loop

Build the CLI (from the framework root):

```sh
go build -o /tmp/prumo ./cmd/prumo
```

Copy the sample to a scratch directory, then seed a planning session with the
two open questions that tripped docs readiness (this is the harness/goal-focus
step that produces open questions from the coverage audit):

```sh
prumo --json docs readiness --path ./copy --goal G-SAMPLE
# {"ready": false, "blocking_contracts": ["project.scope"]}

prumo --json plan questions --session PLAN-SAMPLE --path ./copy
# Q1 [blocker]   Which capabilities are out of scope for the first delivery?
# Q2 [high-risk] Which target users drive the primary outcome?
```

1. **Answer the blocker** — an explicit user decision, owner attributed.

   ```sh
   prumo-agent plan answer --session PLAN-SAMPLE --path ./copy \
     --question Q1 --classification explicit-decision --actor owner \
     --statement "Networking, observability and storage backends are out of scope for the first delivery."
   # resolved Q1 (explicit-decision): open questions now 1
   ```

2. **Answer the remaining question** — a constraint for the vision contract.

   ```sh
   prumo-agent plan answer --session PLAN-SAMPLE --path ./copy \
     --question Q2 --classification constraint --actor owner \
     --statement "CLI operators and automation harnesses define the primary outcome; reviewers are secondary."
   # resolved Q2 (constraint): open questions now 0
   ```

3. **Review the canonical decisions** recorded in the session.

   ```sh
   prumo-agent plan decisions --session PLAN-SAMPLE --path ./copy
   ```

4. **Document impacts** — the accepted decisions imply a proposed delta touching
   both contracts. Applying it records governance but never writes canonical
   docs; that is the authoring step below.

   ```sh
   prumo-agent plan delta --session PLAN-SAMPLE --path ./copy
   ```

5. **Author the scope document and bind it** (repository governance). In this
   sample the files already exist; erase `docs/project/scope.md` and the second
   binding to replay this step truthfully.

   ```sh
   prumo-agent plan delta --apply --session PLAN-SAMPLE --path ./copy
   # delta DD-… state=applied, evidence=[DP-Q1-dec, DP-Q2-dec], ready=true
   ```

6. **Readiness re-passes** — derived coverage, not an assertion.

   ```sh
   prumo --json docs readiness --path ./copy --goal G-SAMPLE
   # {"ready": true, "coverage": [["product.vision","implementation-ready"],["project.scope","implementation-ready"]]}
   ```

7. **Propose the goal** — acceptance criteria plus the task DAG when requested.

   ```sh
   prumo-agent plan blueprint --plan --session PLAN-SAMPLE --path ./copy
   ```

## Why it matters

- Blocking comes from **unbound contracts**, not from unanswered questions —
  answering questions closes them; binding sources clears the contract state.
- The loop never mutates canonical documentation: `plan delta [--apply]` only
  records the delta; a human (or an agent under repository governance) authors
  the documents and updates `bindings.json`.
- Agent answers are never promoted to decisions: a `agent-suggestion`
  classification stays proposed and the question remains open.