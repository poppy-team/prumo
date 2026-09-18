# Documentation Control Plane

**Status**: accepted architecture (ADR 010); implementation tracked by W15–W21 in
`docs/development/waves.md`.

## What this plane is

The Documentation Control Plane is the lifecycle that coordinates existing
documentation mechanisms so that documentation is an executable, verified part
of project state instead of prose produced after development. It is a
coordination concept over modules — never a second Knowledge Runtime and never
a monolith.

**Core invariant**: canonical knowledge and typed contracts are the source;
documentation, agent instructions, context packs, site pages, translations and
AI-retrieval surfaces are **governed projections** with explicit authority,
provenance, lifecycle and verification.

```
Goal / intent
   → impact preview            (W17)
   → DocumentationPlan         (W17)
   → implementation + evidence
   → KnowledgeDelta
   → semantic readiness        (W15)
   → DocumentationDelta        (W17)
   → deterministic projections (W10, W18)
   → verification + gauntlet   (W19)
   → publish / agent surfaces  (W16, W18)
   → intelligence + freshness  (W21)
```

## Layers

| Layer | Name | Responsibility | Where it lives | Wave |
|-------|------|----------------|----------------|------|
| A | Canonical knowledge | Authoritative facts: `docs/`, `schemas/`, ADRs, contracts, knowledge records | repository + `internal/harness/knowledge` | W3 |
| B | Documentation contracts | What knowledge a project type must document; which document answers which contract | `docs/contracts/builtin.json`, `docs/contracts/bindings.json` | W5, W10 |
| C | Documentation planning | Per-Goal documentation obligations before implementation; pre/post impact reconciliation | W17 | W17 |
| D | Projection & compiler | Deterministic generation, managed regions, manifests, HumanDocs HD0–HD4 | `internal/harness/doccompile`, `internal/harness/humandocs` | W10 |
| E | Agent surfaces | Provider-neutral instruction IR plus per-tool adapters, fingerprinted and freshness-checked | `docs/agents/instruction-ir.json`, `internal/agentsurface` | W16 ✅ |
| F | Publication | Human site, raw Markdown, `llms.txt`, generated contract/schema reference, MCP read surfaces | `internal/docpublish` | W18 ✅ |
| G | Verification | Semantic readiness, drift/authority gates, docs gauntlet, terminology and version policy, CI groups | `internal/documentation`, `internal/doclifecycle` | W15, W19, W20 ✅ |
| H | Observability | Documentation cost/coverage/freshness metrics, brownfield adoption, context GC | `internal/docintel` | W21 ✅ |
| I | Lifecycle | Translation/media currency by digest, glossary, deprecation, version policy, release readiness | `internal/doclifecycle`, `docs/lifecycle.json`, `docs/glossary.json` | W20 ✅ |

## Readiness model (layer G)

Readiness is **requirement → claim → evidence**, not keyword presence and not
evidence count:

```
Contract.required_knowledge
   └── Requirement (stable ID, via binding.requirement_map)
          └── Claim (accepted, authoritative, at a revision)
                 └── Evidence (bound by target = claim ID, verified, current, artifact exists)
```

A contract is `verified` only when every required knowledge item resolves to at
least one accepted, authoritative claim whose evidence names that claim,
targets the current revision, is marked verified, and whose artifact exists.

### States

| State | Meaning | Blocks readiness |
|-------|---------|------------------|
| `missing` | No bound source at all | yes |
| `partial` | Obligation unbound (no claim, no requirement mapping) | yes |
| `unverified` | Bound but unproven, or only a lexical match | yes |
| `stale` | Bound knowledge older than the current revision | yes |
| `verified` | Every obligation semantically proven | no |
| `not-applicable` | Contract declares no required knowledge | no |

### Findings

| Kind | Trigger |
|------|---------|
| `unmapped-requirement` | Required knowledge has no requirement ID |
| `missing-claim` | No claim satisfies the requirement |
| `claim-not-accepted` | Claim status is not `accepted` |
| `unauthorised-claim` | Claim authority is missing or `external` |
| `superseded-claim` | Claim was replaced by an accepted claim |
| `contradiction` | Claim contradicts an accepted claim |
| `stale-evidence` | Evidence targets an older revision, or is flagged stale |
| `unverified-claim` | No verified evidence bound to the claim |
| `invalid-evidence` | Evidence has no stable ID |
| `missing-artifact` | Evidence artifact does not exist |

### Lexical evaluation is never authoritative

`internal/documentation` retains its legacy word-matching evaluator for
migration visibility only. A lexical result is reported with
`mode: lexical` and `authoritative: false`, and its state can never exceed
`unverified`. `prumo-agent docs readiness` therefore cannot pass merely because the
required words appear somewhere in a bound document.

## Package boundaries

| Package | Owns | May import |
|---------|------|------------|
| `internal/documentation` | Contract registry, bindings, lexical + semantic evaluation, impact, delta, authority/drift, eval corpus | stdlib, `internal/protocol` only |
| `internal/harness/knowledge` | Typed knowledge records, relations, contradictions, memory atlas | harness internals |
| `internal/harness/humandocs` | HD0–HD4 plans, generation, coverage | contracts/profiles data |
| `internal/harness/doccompile` | Deterministic DAG builds, fingerprints, managed regions | stdlib |
| `cmd/prumo` | Composition root; wires the plane's commands | all of the above |

Dependency direction points inward: adapters (site generators, tool-specific
instruction files, MCP servers) depend on the core, never the reverse. The
documentation engine does not import harness packages, so the plane is
composed at the CLI/application layer through typed JSON contracts rather than
by coupling the harness to the documentation engine.

## Authority rules

- Every document is classified `canonical`, `projection` or `historical` in
  `docs/AUTHORITY_MAP.json`; order and policy live in
  `docs/governance/authority.md`.
- A projection must resolve to a canonical source. Projection→projection chains
  are reported as cycles.
- Active documents may not cite a release line older than the current version
  without a historical annotation.

## Agent surfaces (layer E)

Instruction knowledge is compiled, never hand-maintained per tool:

```
docs/agents/instruction-ir.json   (canonical, provider-neutral)
        │  precedence: scope depth → precedence → rule ID
        ├── agents-md  → managed region `agents-core` inside AGENTS.md; nested scopes → <scope>/AGENTS.md
        ├── copilot    → .github/copilot-instructions.md, .github/instructions/*.instructions.md
        ├── cursor     → .cursor/rules/*.mdc (glob-scoped)
        ├── claude     → CLAUDE.md importing @AGENTS.md; scoped .claude/rules/*.md
        └── skill-md   → .agents/skills/*/SKILL.md
```

Standalone vendor surfaces are written to `.prumo/runtime/agents/` (gitignored)
by default, so a projection never becomes a canonical repository artifact. The
Claude root surface imports `AGENTS.md` instead of duplicating it. Every surface
carries a generated marker with adapter, scope, source fingerprint and source
units, and a `manifest.json` sidecar records the compiled set.

Gates: `prumo-agent docs agents verify` (and `TestAgentSurfaceContextRotGate`) fail
when a compiled surface is stale, when instruction budgets are exceeded, when
two rules conflict, or when any instruction document references a path,
subcommand, test or symbol that no longer exists. Nested YAML frontmatter is
permitted **only** inside generated vendor projections; canonical artifacts stay
Markdown/JSON.

## CLI surfaces

The whole surface is implemented: `prumo-agent docs audit`, `readiness`, `impact`,
`plan`, `delta`, `explain`, `contradictions`, `authority`,
`agents build|verify|explain`, `verify [--strict]`, `gauntlet`,
`build|manifest|query|doctor`, `site build|verify`, `translate|media|release`,
`glossary`, `version`, `metrics`, `resources read`, `mutations` and
`adopt inspect|propose` (see the CLI surface table in
`docs/development/waves.md`).

## Non-goals

A general CMS, a Notion replacement, a WYSIWYG editor, a mandatory vector
database or LLM, a mandatory site generator, and a translation SaaS.

## Stopping condition

The plane is complete when readiness is semantic, every gate is deterministic,
and projections are regenerated or explicitly marked N/A on change.
Model-assisted critique is optional and may never override a deterministic
failure. New context mechanisms must report token/cost impact and define what
stops them.

## Migration status

Semantic readiness v2 (W15) is implemented and enforced by the eval corpus
(`evals/documentation/`, D001–D011). The repository's own bindings are no longer
lexical: seven contracts (`product.vision`, `project.scope`,
`architecture.system`, `testing.strategy`, `security.trust`,
`installation.lifecycle`, `cli.reference`) carry accepted claims with verified
evidence at revision 1, and `prumo-agent docs readiness` reports `ready: true` with no
warnings.

The first draft of those bindings cited `docs/migration/conformance-strategy.md`,
which the authority map classifies as **historical**. The evaluator rejected it as
`non-canonical-evidence` rather than accept a green result. The binding was then
corrected instead of the check being loosened — the same misbinding the D002 eval
case exists to catch, caught on the real repository.

What is *not* yet semantic is deliberate: an artifact the authority map does not
classify is accepted on existence and digest evidence rather than condemned by
inference, and no agent-inference path exists to promote a claim. Both are
fail-open on facts the repository has not stated, and fail-closed on facts it has.

Downstream layers that consume this model: W16 compiles agent surfaces from the
canonical IR; W17 turns impact into a Goal preflight/postflight; W18 projects the
same graph to site and AI retrieval; W19 verifies with a deterministic-first gate
whose `--strict` mode requires semantic readiness; W20 manages translation, media
and release lifecycle by content digest and adds the terminology registry and
version policy; W21 measures the result and supports brownfield adoption without
duplicating truth.

## What the plane deliberately does not do

Layer H measures the **repository**, never the reader. There is no telemetry
channel and no query log: `prumo-agent docs query` and `prumo-agent docs resources read` are
read surfaces that record nothing, and `prumo-agent docs metrics` computes every
figure from repository state. Retrieval-success and missing-intent metrics
(W21.2/W21.3) were proposed by the audit and are **refused** — see ADR 011.
The limitation is real and accepted: Prumo cannot say which existing page people
fail to find, and buying that answer would cost the framework's local-first
trust posture.
