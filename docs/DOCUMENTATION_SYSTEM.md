# Documentation System

Prumo treats documentation as a product surface for four audiences and as the canonical knowledge source for agents.

## Audiences

### Users

Need to install, learn, perform tasks, understand concepts and recover from problems.

### Developers / contributors

Need onboarding, codebase tours, build/test/debug workflows, task-oriented change guides and extension/API references.

### Operators / maintainers

Need deployment, configuration, observability, backup/recovery, incidents, runbooks, compatibility, migration and release guidance.

### Agents

Need a short entrypoint, applicable invariants, task maps/context packs, source pointers, evidence and stopping rules—not a duplicate documentation universe.

## Documentation modes

User-facing docs should distinguish:

- Tutorials — learning path.
- How-to — perform a concrete task.
- Reference — exact options/APIs/formats.
- Explanation — concepts/rationale.

Do not collapse all four into one giant `USER_GUIDE.md`.

## Recommended project taxonomy

Use only the sections that the project genuinely needs:

```text
docs/
├── PRUMO.md
├── product/
├── user/
├── onboarding/
├── architecture/
├── data/
├── api/
├── ui-ux/
├── developer/
├── testing/
├── security/
├── operations/
├── support/
├── release/
├── decisions/
└── reference/
```

This is a semantic taxonomy, not a requirement to create an empty file in every directory.

## PRUMO as intent router

`docs/PRUMO.md` should answer "what are you trying to do?" before exposing the full document tree.

Typical routes:

```text
Understand the product
Install/use it
Learn a feature
Troubleshoot
Set up development
Understand the codebase
Implement a feature
Add a test
Deploy/operate
Perform a release
Work as an AI agent
```

Sub-Prumos are allowed when a project is large, but they are maps rather than duplicate truth.

## Developer documentation essentials

High-value low-maintenance docs include:

- `CODEBASE_TOUR.md` or equivalent;
- development environment/build/test instructions;
- debugging playbook;
- "how to add X" task guides for recurring extension points;
- testing conventions;
- API/plugin extension guide;
- release workflow.

## User/support essentials

When applicable:

- getting started;
- interface/feature tutorials;
- supported platforms/formats;
- keyboard shortcuts/reference;
- troubleshooting by symptom;
- known issues;
- diagnostics/log collection;
- support/reporting policy.

## Operational essentials

When applicable:

- configuration;
- observability;
- runbooks;
- backup/restore;
- disaster recovery;
- migrations;
- compatibility matrix;
- deprecation/support lifecycle;
- rollback.

## Stable architectural knowledge

Projects should document important invariants and rejected approaches/ADRs so agents do not repeatedly rediscover or violate boundaries.

## Documentation economy

Rules:

- **patch before proliferation**;
- create a new file only for a stable concept, distinct audience/task or independent specification;
- do not create task-specific context files;
- do not persist generated summaries per document;
- prefer virtual structural chunks over physical microfiles;
- do not duplicate the same truth for the site and repository.

## Documentation Delta

At the end of a behavior-changing task, compute documentation impact before writing:

```text
User docs: none
Architecture: patch timeline section
Spec: patch transitions
Developer: none
Migration: required
```

Then update only affected canonical sources.

## Documentation Control Plane

Canonical knowledge and typed contracts are the source; every human page, agent
instruction file, context pack, site page, translation/media artifact and
AI-retrieval surface is a **governed projection** with explicit authority,
provenance, lifecycle and verification. The lifecycle is:

```text
Goal/intent → pre-change impact → DocumentationPlan → implementation + evidence
→ KnowledgeDelta → semantic readiness → DocumentationDelta → deterministic
projections → verification → publish/agent surfaces → freshness → Context GC
```

Rules:

1. Readiness is semantic (requirement → claim → evidence), never keyword presence
   or evidence count.
2. Missing knowledge creates a gap; documentation is never invented to fill it.
3. Agent instructions are compiled from canonical knowledge with fingerprints and
   freshness checks; vendor formats are replaceable adapters.
4. "Not applicable" is explicit; silent omission is not.
5. Deterministic verification runs before model-assisted critique, and a model
   critic cannot override a deterministic failure.
6. Historical/legacy docs are excluded from current agent context by default.

Architecture decision: `docs/adr/010-documentation-control-plane.md`.
Delivery waves: W15–W21 in `docs/development/waves.md`.

## Documentation site

The framework goal is a documentation site that grows with the project.

Rules:

1. Canonical Markdown/JSON remains source of truth.
2. Site navigation is generated from the documentation model/PRUMO where practical.
3. Public and internal visibility can differ without duplicating documents.
4. Search, versioning, breadcrumbs, related docs and API references are derived views.
5. Project Intelligence dashboards consume durable project data.
6. Broken links, invalid metadata and orphaned required docs should be CI-detectable.
7. The site must be replaceable; framework knowledge cannot depend on one static-site generator.

## Documentation coverage

Projects may track coverage by capability/feature:

```text
Feature       Code  Tests  User  Dev  Ops
Transitions    ✓      ✓     ✓     ✓    n/a
```

Coverage is an engineering signal, not a reason to generate low-value prose.

## Metadata

Keep document metadata minimal. Prefer deterministic inference from path/structure over repeated fields.

If explicit metadata is needed, use simple Markdown fields or a project JSON index. Do not introduce another maintained format solely for metadata.

## Definition of Done impact

A task evaluates, as applicable:

- user docs;
- developer docs;
- API/reference;
- migration/compatibility;
- operations/support;
- changelog/release;
- PRUMO links.

"Not applicable" is valid; silent omission is not.
