# DocumentationMigrationReport — Notion snapshot → canonical docs

Snapshot: `.prumo/imports/notion/prumo-code-agent/2026-09-11-d41f18bb65f5/`
(manifest + source-map + validation: 43 pages, 0 missing, 0 broken links).
Snapshot is IMMUTABLE provenance; it is not canonical after promotion.

## Workflow executed

inventory → normalize → reconcile → plan → generate/update → validate → promote
(per page 41; no giant-prompt rewrite).

## Sources inspected

- Snapshot root + pages 00–08, 18–19, 21–26, 27–34 (selective), 35–41.
- Repo: README, ENTRYPOINT, AGENTS, FRAMEWORK, docs/, schemas/, conformance/,
  cmd/, internal/ (runtime, toolgateway, environment, budget, modelregistry,
  contextcompiler, experience, documentation), src/prumo/resources/, .prumo/,
  prompts/.

## Decisions promoted (ACCEPTED → target architecture)

Prumo-native Go harness; ModelProvider!=AgentProvider; reentrant agent state
machine; FakeProvider conformance-first; ToolGateway+ACI; deterministic
permissions; gradual sandbox; crash-safe checkpoints + idempotent effects;
budget+observability; internal Model Gateway with post-side-effect Handoff
rule; Codex/OpenCode/ACP-first external adapters; typed pointer-first
Handoff v2; Context Compiler v2 pipeline; Knowledge Runtime + Delta API;
typed contradiction/coverage/readiness; deterministic doc compiler; HD0–HD4
first; local inference in supervised workers; workforce least-context +
worktree ownership; reference→spike→decision→contract/test flow; repo split
gate HA0–HA5+headless+protocol+replay.

## Implementation drifts (accepted + missing → now baselined)

HA0–HA4 had partial foundations (runtime Run/Checkpoint, toolgateway
descriptors, budget envelopes, modelregistry routing, context baseline,
experience handoff) but no unified NativeAgent, no ModelProvider boundary,
no FakeProvider suite, no reentrant loop, no idempotent journal, no gateway
with Handoff rule, no typed Handoff v2, no Context v2/RRF/MMR, no Knowledge
Delta engine, no doc DAG/CAS, no team ownership guard, no `prumo agent`.
All now have baseline implementations under `internal/harness/*` (see
roadmap-status for partial/deferred marks). No fake-completion: docs state
partial/deferred explicitly.

## Files created

- `docs/harness/{overview,agent-runtime,providers,security,context-knowledge,workforce-handoff,roadmap-status}.md`
- `internal/harness/{agent,model,perm,checkpoint,runtime,gateway,extagent,handoff,contextv2,knowledge,doccompile,team,aci}/*`
- `cmd/prumo/agent_commands.go`, `cmd/prumo/agent_commands_test.go`
- `schemas/{harness-checkpoint,harness-handoff,agent-event}.schema.json` (next commit)
- `internal/harness/eval/*` (next commit)

## Files modified

- `cmd/prumo/main.go` (agent route), `cmd/prumo/help.go` (agent help)
- `docs/PRUMO.md` (router entry), `docs/SOURCE_MAP.json` (provenance)

## Files superseded

None removed. Stale claims are corrected in place; legacy docs kept where
behavior is still newer than the notebook.

## Reconciliation classes observed

- accepted+implemented/aligned: run lifecycle basics, budget envelopes, tool descriptors.
- accepted+implementation missing → baselined this Goal: HA0–HA4 core, Handoff v2, Context v2, Knowledge Delta, doc DAG.
- accepted+repo docs stale → updated: architecture now points at harness tree.
- repo newer than notebook: CLI help system v0.4.1/0.4.2, connectors (antigravity etc.).
- true contradictions: none P0. CLI layer rule (cmd may import only app+protocol)
  contradicts current cmd/* reality — recorded, not silently fixed.
- historical/superseded: Python runtime (oracle only), pre-Harness loop sketches.

## Validation

- `go test ./internal/harness/... ./cmd/prumo/` green (incl. conformance + evals).
- Schemas validate against sample payloads (see eval).
- Links: harness tree internally consistent; PRUMO router points at it.
- No P0 contradictions remain; open gaps listed in roadmap-status + final report.

## Remaining blockers

2nd real vendor adapter live creds, provider live matrix, daemon reconnect,
protocol IDL/SDK generation, container sandbox provider, fuzz/chaos passes.
