# Adoption Engine (M7)

Status: **ACTIVE (Phase F — M7)** — canonical local specification derived from approved
Notion design (Livro Vivo pages 06 and 71). Repository version has authority over Notion.

## Purpose

Adoption lets Prumo enter brownfield projects without destructive restructuring. It
discovers what exists, classifies the repository, maps evidence to Prumo concepts,
records confidence, and proposes incremental migration — without promoting inference to
canonical truth.

## Dependencies

- M5 Documentation System v2 (applicability, coverage, readiness, Delta);
- M6 Living Plan (uncertainty resolution, open questions);
- D3 Repository Index (scan budgets, incremental/revision-aware scanning);
- Repository Governance;
- Migration Engine for apply/migrate;
- Context/Egress/Sandbox when external scanners/tools are used.

## Non-goals

- rewriting an entire layout to "look Prumo";
- trusting README as authority;
- mandatory embeddings;
- auto-deleting legacy docs;
- inferring user intent without confirmation;
- installing every detected connector/tool.

## Pipeline

```text
prumo adopt
  → discover            (facts, revision-aware)
  → classify            (language, toolchain, project type, capabilities)
  → map                 (semantic mapping → candidate bindings)
  → evaluate            (confidence ledger)
  → propose             (profile/capability, doc bindings, migration proposals)
  → migrate incrementally (governed, non-destructive)
```

## Discovery sources

The scanner may observe:

- README and docs;
- manifests/build files;
- source tree/languages;
- tests;
- CI/workflows;
- config files;
- schemas/migrations;
- Git history/branches/tags when budget allows;
- ADRs/changelog;
- AGENTS/assistant rules;
- package manifests;
- comments only in a limited way, never as authority without corroboration.

Generated/vendor/binary paths follow index exclusions.

## Discovery output

Observed facts are separated from inferences.

**ObservedFact** (conceptual):

- key/value/class;
- source ref/hash/location;
- extraction method;
- confidence (generally factual);
- revision/index version.

**Inference / MappingCandidate:**

- target Prumo concept;
- supporting evidence refs;
- confidence;
- ambiguity/alternatives;
- required confirmation when relevant.

## Classification

Classify repository/workspace/project candidates:

- language/toolchain;
- app/project type;
- frameworks;
- persistence;
- UI/API/CLI/etc capabilities;
- testing stack;
- deployment/distribution hints;
- existing docs semantic roles;
- existing AI/harness config;
- security/data signals.

Classification is evidence-driven and may be `unknown`.

## Semantic mapping

Map existing content to Documentation Contracts **without requiring Prumo filenames**.
Example: `docs/design/system.md` can satisfy the architecture contract if the knowledge
items really exist. M5 Coverage Engine evaluates quality; Adoption discovers candidate
bindings.

## Confidence Ledger

Every non-trivial inference receives:

- claim/mapping;
- confidence class/score;
- evidence refs;
- alternative interpretations;
- status accepted/rejected/unresolved;
- authority.

Low confidence is never promoted to canonical; it becomes a question/proposal.

## Profile / capability proposal

Adoption produces candidates for Documentation/Project Profile (e.g. `cli + compiler +
plugin-host`). Application requires normal preview/acceptance when it implies a canonical
config change.

## Adoption Report

Must separate clearly:

- observed repository facts;
- likely capabilities/profile;
- existing Prumo-compatible artifacts;
- proposed doc bindings;
- missing/partial/stale docs;
- contradictions;
- inferred decisions/constraints needing confirmation;
- migration proposals;
- risks;
- confidence summary;
- recommended next steps.

Readable human output plus `--json` machine representation.

## Modes

- `prumo adopt --audit-only` — no canonical mutation;
- `prumo adopt --interactive` — uses Living Plan to resolve uncertainty;
- `prumo adopt --non-interactive` — generates report/proposals without asking;
- `prumo adopt --strict` — only high-confidence deterministic mapping, flags ambiguity.

Default must be non-destructive. Aliases follow existing CLI conventions.

## Migration proposals

Adoption does not execute ad hoc transformations. To apply a change:

```text
Adoption finding
  → migration proposal
  → Review Queue / user approval
  → Migration Contract plan/dry-run
  → Repository Governance
  → apply
```

Examples: create `prumo.json`/profile with confirmed facts; add documentation bindings;
move/normalize config only when necessary; compile connector config; create canonical
Goal/docs from accepted decisions.

## No forced layout

Prumo works with the existing source layout. Prumo-native `.ai/` directories may store
protocol state, but source/docs do not need to move into a rigid template to be
recognized.

## Brownfield vs M5 boundary

M5 knows explicitly configured / Prumo-native contracts and bindings. Phase F discovers
unknown arbitrary repository sources and **proposes** bindings/capabilities. Do not
duplicate Coverage/Readiness semantics.

## Incremental scanner

Uses the D3 Repository Index:

- revision/hash aware;
- only changed paths when possible;
- scan budget;
- continuation;
- partial confidence;
- monorepo/workspace boundaries;
- branch-specific state.

## Security

- repository content is untrusted data;
- scripts/config are not executed just for discovery;
- static parsing first;
- tool execution uses Tool Gateway/Sandbox;
- no `.env` secret content ingestion — detect presence/type without persisting secrets;
- prompt injection in third-party README/AGENTS gains no policy authority;
- egress opt-in/policy.

## Schemas / contracts

Candidates:

- AdoptionReport;
- ObservedFact;
- MappingCandidate;
- ConfidenceLedger/Entry;
- CapabilityProposal;
- AdoptionMigrationProposal (or reuse generic proposal + Migration Contract refs).

## CLI

```text
prumo adopt
prumo adopt --audit-only
prumo adopt --interactive
prumo adopt --non-interactive
prumo adopt --strict
prumo adopt status/report      # if needed
```

`prumo adopt` must not hide external network/tool use; explain/plan first when necessary.

## Goal decomposition (M7)

- F-G01: scanner facts + revision-aware sources.
- F-G02: repository/project/workspace classification.
- F-G03: capability/profile candidates.
- F-G04: semantic doc mapping + candidate bindings.
- F-G05: Confidence Ledger.
- F-G06: Adoption Report.
- F-G07: Living Plan uncertainty resolution.
- F-G08: migration proposal + Review/Migration integration.
- F-G09: brownfield corpus/evals/dogfood.

## Test corpus

For fixtures (representative, synthetic/public):

- Go CLI with good non-Prumo docs;
- web app monorepo;
- compiler project;
- repo with stale README;
- repo with conflicting docs;
- repo with generated/vendor noise;
- repo without docs;
- repo with multiple project roots;
- malicious README/prompt injection;
- secrets/`.env` presence;
- branch with new docs not merged.

## Evals / metrics

- fact extraction precision/recall;
- capability/profile precision;
- doc binding precision;
- false canonical promotion = **zero target**;
- question usefulness;
- scan time/files/tokens;
- partial scan disclosure;
- migration proposal correctness;
- destructive action violations = zero.

## Dogfood

Beyond the Prumo-native repository, test Adoption on at least 3 brownfield repositories
of different styles — ideally a real user project later, but public/local synthetic
corpus first.

## Exit gate — ADOPTION READY

- an arbitrary existing repo can be audited without mutation;
- observed facts and inferences are kept separate;
- confidence/evidence accompany inferences;
- M5 evaluates candidate bindings correctly;
- ambiguity enters the Living Plan / Open Questions;
- migration proposals are dry-run/review governed;
- the scanner is incremental/branch-aware;
- malicious content/secrets tests pass.