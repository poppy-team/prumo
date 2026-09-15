# Project Orchestration Protocol (POP)

POP defines how Prumo turns Goals into bounded agentic work while remaining independent of a specific orchestrator.

## Bootstrap rule AI-001

Never infer a new project's preferred model roster from another project. Every project defines its own roster.

## Flow

```text
Vision
→ Phase
→ Goal
→ Task DAG
→ context plan/budget
→ isolated execution
→ verification
→ independent review
→ documentation delta
→ project intelligence
→ evidence/gate
→ context GC
```

## Roles and models

Roles such as Architect, Implementer, Tester and Security Reviewer are stable abstractions. `model-policy.json` maps the explicit project roster onto roles.

Routing should consider:

- capability;
- risk;
- context size;
- expected cost;
- historical project scorecard;
- provider availability.

The cheapest model is not automatically best; the preferred policy is the cheapest reliable model that passes required gates.

## Context before execution

Each task should begin with a context strategy/budget. The orchestrator should prefer deterministic task maps, structural/symbol retrieval and known Context Packs before broad exploration.

Do not feed the entire project to every role.

## Fallbacks

1. Availability fallback.
2. Quality fallback after bounded repair attempts.
3. Context fallback: targeted expansion when evidence is insufficient.
4. Review diversity for high-risk work when possible.
5. Human escalation after fallback/budget exhaustion or policy conflict.

Retries and context expansion are bounded. No infinite loops or unbounded recursion.

## Work isolation

Independent tasks use branches/worktrees or equivalent isolation. Parallelization is permitted only when dependency analysis indicates safe independence.

Context isolation is separate from code isolation: a delegated role should receive a fresh, minimal context for its question.

## Source of truth

Orchestrator artifacts may mirror Goals/config but repository canonical files remain authoritative. Generated summaries/context/site content cannot silently modify locked acceptance criteria.

## Output discipline

Intermediate agent output should be compact and operational. Long reasoning narratives are not durable project history. Persist stable findings in canonical docs/ADR/evidence and discard the rest.

## Completion

An agent declaration has no authority. Required gates/evidence, documentation impact evaluation (pre-change `DocumentationPlan` at lock, post-change `DocumentationDelta`), intelligence update and cleanup determine completion.
