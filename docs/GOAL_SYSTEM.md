# Goal System

A roadmap says what should happen over time. A Goal defines an outcome that can be independently verified.

## Hierarchy

`Vision → Project Goals → Phase → Goals → Subgoals → Task DAG → Evidence`

## Required fields

A Goal has an ID, phase, objective, constraints, non-goals, acceptance criteria, gates, dependencies, evidence and state history.

Goals are stored as JSON under `.ai/goals/<phase>/<id>.goal.json`.

## State machine

```text
DRAFT → PLANNED → LOCKED → EXECUTING → VERIFYING → REVIEWING → DONE
          ↑          ↓          ↓             ↓
          └──────── BLOCKED ─────┴─────────────┘
```

`DONE` requires evidence.

## Locking

Once LOCKED, acceptance criteria are protected. Discoveries that genuinely change the outcome require an explicit amendment with rationale/review; a test failure is not a reason to relax acceptance.

## Context/Cost planning

A locked Goal may include or reference:

- task risk/complexity;
- context budget profile;
- required docs/tests;
- cost/effort estimate;
- special review gates.

These are planning controls, not completion substitutes.

## Evidence

Evidence may include test/CI reports, benchmarks, screenshots, logs, artifacts, reviews, migration verification and manual gates. Store stable paths/URLs rather than dumping large evidence payloads into the Goal.

## Completion integration

Goal completion should trigger:

- final documentation impact check;
- compact task/project intelligence update;
- durable evidence link update;
- context garbage collection.

Documentation impact is computed **before and after** behavior-changing work:
a typed `DocumentationPlan` at Goal lock (expected affected units, translations,
media, agent surfaces, and explicit N/A decisions) and a `DocumentationDelta` at
completion (actual impacts, new/resolved gaps). A Goal is not complete while its
required documentation projections are stale.

Specification: W17 in `docs/development/waves.md`. Architecture: ADR 010.
