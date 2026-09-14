# Context v2, Knowledge, Documentation Compiler

## Context Compiler v2 (`internal/harness/contextv2`)

Pipeline: authority/trust/privacy gates → freshness → exact/structured
retrieval → FTS/BM25 (+optional embeddings + typed graph) → RRF → MMR/dedup
→ dependency/coverage constraints → marginal-utility-per-token packing →
progressive disclosure L0–L4 → replayable Manifest v2 (relevance, authority,
freshness/rev, provenance, token estimate, reason, method, compression,
full-content pointer). No-LLM path is first-class; embeddings are
derived/optional. Extends (not replaces) `internal/contextcompiler` baseline.

Wired into runs (`contextv2.CompileWorkspace`): every `agent run` and daemon
run compiles goal + entrypoints + git-modified files + bounded tree listing
through gates→dedup→packing, persists `context-<run>.json`, and emits
`context.compiled` (included/tokens/pressure/level). Budgets/levels via
`--context-budget/--context-level`. Compilation never fails a run: on bad
roots it degrades to the goal item.

Retrieval beyond listing: BM25-lite over workspace text fused as score
bonus, and the project Memory Atlas (`.prumo/atlas.json`) recalled as
pointer-sized candidates — never dumped wholesale. Code intelligence:
`repomap` (bounded Go/md symbol index + compact render) always on; LSP
(`gopls` et al.) when a server binary exists, repomap fallback otherwise;
top symbol hits enter as `symbol:` pointers. Token estimates use the
versioned `tokens-v1` table.

Cross-project memory moves only through `Promote` (allowlisted targets,
volume cap, `restricted`/`confidential` never cross, provenance stamped,
copy-not-move).

## Knowledge Runtime (`internal/harness/knowledge`)

Typed records with stable IDs: Source, Section, Claim, Finding, Decision,
Requirement, Constraint, Risk, Assumption, OpenQuestion, ResearchRecord,
Evidence, MemoryRecord. Each carries authority/trust/freshness/status,
provenance, lineage/supersession, typed relationships. Canonical records stay
simple; indexes/summaries/embeddings are derived.

Knowledge API: Locate/Get/Search/Related/Explain/Context/Affected +
ValidateDelta/CommitDelta/Supersede/Promote/Refresh/Snapshot. Mutation is
`KnowledgeDelta → Validate (policy/review) → Commit → KnowledgeChanged →
derived rebuild` with optimistic-concurrency-friendly supersession.

Run seeding (`knowledge.SeedRequirement/SeedEvidence`): every `agent run`
and daemon run persists `knowledge-<run>.json` — Requirement from the goal
at start, Evidence linked at finish — so coverage/readiness hold per run
without manual bookkeeping. Global promotion still requires Delta review.

## Contradiction / Coverage / Readiness

Separate engines. Typed `contradicts` edges (never substring heuristics);
relationship-driven coverage (requirement ← evidences/covers); revision-aware
evidence; readiness policy = no contradictions + no uncovered reqs + no
blocker open-questions. LLM proposes candidates only; it never decides
authority.

## Documentation Compiler (`internal/harness/doccompile`)

Knowledge IR → Document IR → dependency DAG → incremental evaluator →
renderer → canonical serializer → atomic writer. Stable IDs, SHA-256
fingerprints, reverse invalidation via DAG order, red-green digests,
CAS/no-op writes, whole-file generation, atomic replace, artifact manifests,
provenance. AST-aware Markdown regions + JSON Pointer/Patch are the next
increment (interfaces reserved).

## Human Documentation Runtime (HD0–HD4 baseline)

`DocumentationSpec`, profiles, `DocumentationUnit`/Page Graph, deterministic
planner, README/reference generation, basic docs tree, coverage/readiness —
this `docs/harness/` tree + `promotion-report` are the first instance.
Website/Starlight (HD5+) follows the dependency roadmap and never blocks the
Harness.

## Local Intelligence

Decoupled by design: generation/rerank → llama.cpp/GGUF worker, embeddings →
ONNX worker, OpenVINO → optional adapter. Workers are supervised processes,
never mandatory Core bindings. Concrete models stay benchmark-driven;
No-LLM remains first-class.
