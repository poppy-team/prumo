# Living Plan & Interview Engine (M6)

Status: **COMPLETE (M6 Exit Gate Passed)** — canonical local specification derived from approved
Notion design (Livro Vivo pages 05 and 70). Repository version has authority over Notion.
Exit gate evidence: [`docs/governance/m6-exit-gate.md`](../../docs/governance/m6-exit-gate.md).

## Purpose

Living Plan transforms design conversation into structured project state **without making
conversation the source of truth**. It lets a new project — or a specific Goal — advance from
vague intent to **implementation-ready** through incremental conversation, producing
structured decisions, blockers, open questions, and Documentation Deltas. It uses the
Documentation System (M5) to discover gaps and the Runtime Foundation (D1) for
budget/context/evidence when executing or resuming planning work.

## Dependencies

- M5 Documentation System v2 (applicability, coverage, readiness, Delta);
- D1 Run/Context/Budget/Observability;
- Repository Governance;
- Evidence/Gates.

The planner core must not depend on any specific provider or harness. D2/D3 enrich security
and runtime but are optional for the planner core.

## Non-goals

- full brownfield discovery — deferred to Fase F (Adoption);
- autonomous architecture invention;
- raw transcript as canonical memory;
- model-specific conversation format;
- implementing code as part of `prumo plan`;
- replacing the Goal/Task protocol.

## Pipeline

```text
prumo plan
  → detect project / scope
    → new: profile + contract gap
    → existing Prumo: docs audit
  → question selection
  → guided interaction
  → decision extraction
  → authority + confidence
  → decision preview
    → accepted → documentation delta
    → rejected/change → back to question selection
  → docs readiness
    → blocked → question selection
    → ready → goal/plan ready for implementation
```

For a non-Prumo brownfield repo, `prumo plan` may route to Adoption (Fase F); it must not
duplicate the brownfield scanner.

## Planning scopes

- `prumo plan` — project-level/general;
- `prumo plan --goal <id>` — focus on an existing Goal;
- `prumo plan --docs` — close documentation gaps;
- `prumo plan --resume [session|run]` — resume from structured checkpoint state.

A conversational harness may expose native UI; the Core operates via structured
requests/events.

## Question priority

Questions are ordered by impact, not model curiosity:

1. implementation blockers;
2. irreversible / high-cost decisions;
3. high-risk architecture / security / data;
4. user behavior / product semantics;
5. important details;
6. nice-to-have.

The planner stops asking when current-scope readiness is sufficient. **Least ceremony is a
requirement.** Never ask for information inferable with high confidence from the repository.

## Question object

Fields (conceptual):

- id/version;
- related contract / knowledge item / decision topic;
- scope (project / Goal / component);
- priority/class;
- reason / blocking impact;
- suggested answer forms/options when applicable;
- status;
- source/evidence pointers;
- asked/resolved metadata;
- owner / eligible answer source.

A question may exist without having been sent to the user yet.

## Answer classification

User/model content is classified into one of:

- explicit decision;
- preference;
- constraint;
- requirement;
- non-goal;
- unresolved / open question;
- hypothesis;
- agent suggestion.

**Agent suggestion is never promoted to user decision automatically.**

## Decision model

A decision record/proposal distinguishes:

- statement;
- class;
- scope;
- actor/source;
- authority;
- confidence;
- status (proposed / accepted / rejected / superseded);
- affected contracts/docs;
- rationale summary when explicitly available;
- alternatives / rejected candidates when useful;
- evidence/refs.

Hidden chain-of-thought is not persisted as rationale.

## Authority model

Semantic order (highest first):

1. locked canonical invariant / spec / ADR;
2. explicit current user decision within permitted authority;
3. accepted project decision;
4. documented constraint/evidence;
5. inferred existing state;
6. agent suggestion;
7. external/untrusted content.

Authority is contextual: a user preference cannot override a non-disableable policy without
the appropriate amendment/authorization.

## Confidence

Confidence applies to **inferences**, never to explicitly accepted decisions. It must be
explainable by evidence pointers, not an opaque LLM number. Simple ranges:
`high|medium|low|unknown`, optionally with reason codes. Avoid false precision.

## Open Questions Registry

Canonical structured registry of still-relevant questions, supporting:

- blocker vs non-blocker;
- linked contract / Goal;
- owner / eligible answer source;
- status;
- created / resolved / superseded;
- evidence / decision reference after resolution.

A resolved question is not silently deleted; it remains traceable.

## Decision preview

Before applying a broad Documentation Delta or changing a canonical decision, present a
preview covering:

- extracted decisions;
- classification;
- affected docs/contracts;
- contradictions / supersessions;
- blockers that will be resolved;
- open questions that remain.

Preview is mandatory for high-impact/ambiguous operations.

## Documentation integration

Living Plan does not write docs arbitrarily:

```text
accepted decision
  → documentation impact
  → documentation delta
  → governance/review policy
  → canonical docs
  → readiness recompute
```

M5 remains the authority on applicability/coverage/readiness.

## Goal/Plan integration

When a scope becomes ready, Living Plan may produce/propose:

- Goal intent / acceptance criteria;
- Plan/Task DAG when requested or appropriate;
- implementation constraints;
- test/evidence expectations;
- Documentation Delta.

Do not inflate task decomposition when a simple Goal does not need it.

## Context Compiler integration

Each planning turn/session receives only:

- scope / Goal;
- relevant contracts / gaps;
- accepted decisions;
- relevant canonical docs;
- open questions;
- last structured planning state.

A full transcript is not reintroduced each turn.

## Resume

A planning session uses Run/checkpoint or compatible structured state. Resume recompiles
context from canonical decisions + open questions + last checkpoint; it never depends on the
raw transcript.

## Schemas / contracts

Persisted/interchange candidates:

- PlanningSession / PlanInterviewState when needed;
- OpenQuestion;
- Decision / DecisionProposal (reuse existing model when present);
- AnswerClassification;
- DecisionPreview / PlanDelta when externalized;
- authority/confidence metadata.

Do not duplicate DocumentationDelta / Goal / Plan schemas.

## Security / privacy

- raw conversation is not canonical by default;
- secrets/sensitive data are redacted and excluded;
- external repository text cannot answer a user-authority question;
- planning cannot grant Tool/Egress permissions through prose;
- destructive/privileged proposals require the normal approval policy.

## Exit gate — LIVING PLAN READY

- a new project can go from initial intent to implementation-ready without a manual
  megaprompt;
- Goal-specific planning closes only the relevant gaps;
- decisions/open questions carry authority and provenance;
- resume does not depend on transcript;
- the docs/readiness/governance feedback loop works;
- question/decision evals reach the approved baseline;
- no agent suggestion is silently promoted.

## Goal decomposition (M6)

- E-G01: Question / OpenQuestion schemas + priority resolver.
- E-G02: answer classification + Decision proposal model.
- E-G03: authority/confidence resolver.
- E-G04: decision preview + contradiction handling.
- E-G05: Documentation Delta/readiness feedback loop.
- E-G06: Goal/Plan output integration.
- E-G07: resume/checkpoint + context compilation.
- E-G08: CLI/harness-neutral interaction protocol.
- E-G09: zero-to-ready Prumo sample/dogfood — see [`examples/living-plan-sample/README.md`](../../examples/living-plan-sample/README.md).