# HARNESS_IMPLEMENTATION_REPORT

Goal: Prumo Harness headless production-grade (HA0–HA11 baseline).

## Goal result

Baseline COMPLETE as an operable foundation; production-hardening continues.
`prumo agent run|resume|handoff` works end-to-end (Goal → Context → Model →
Tool → Permission → Sandbox/Workspace → edit → tests → Evidence → Gate →
Checkpoint → completion; kill → restart → resume verified by tests).

## Architecture implemented

- HA0 contracts (`internal/harness/agent`): AgentRuntime, NativeAgentState,
  ModelProvider/AgentProvider, ModelRequest/Event, Turn/Message/Observation,
  ToolCall/Result, PermissionRequest/Resolution, Checkpoint/Continuation/
  Handoff, AgentEvent/DomainEvent, PendingEffect.
- HA1 FakeProvider (scripted response/tool/permission/usage/error/complete/
  cancel) + shared conformance suite; first real adapter OpenAI-compat
  (SSE, tool_calls, usage, retryable 429/5xx, cancel, health).
- HA2 NativeAgent reentrant state machine (`internal/harness/runtime`):
  Prepare→…→Checkpoint→Yield/Complete; `Step` + `RunUntilDone` + runaway guard.
- HA3 ToolGateway+Permission: ACI catalog wired through perm engine in Runner.
- HA4 checkpoint/restart/resume: atomic store + `Latest(run)` + idempotent
  side-effect journal (duplicate rate target zero).
- HA5 Coding ACI (`internal/harness/aci`) + Environment baseline (local/
  worktree, path containment, safe-mode); container/gVisor future.
- Budgets (`internal/budget` + `ConsumeBudget` hook) + `AgentEvent` timeline.
- Model Gateway (`internal/harness/gateway`): healthy-first routing, retry,
  circuit breaker, post-side-effect Handoff rule.
- External AgentProviders (`internal/harness/extagent`): OpenCode server,
  Codex CLI, FakeAgent; external state normalized, never canonical.
- Handoff v2 (`internal/harness/handoff`): typed pointer-first bundle + CLI.
- Context v2 (`internal/harness/contextv2`): gates→freshness→RRF→MMR→
  dependency closure→utility-per-token→L0–L4 manifest. No-LLM first-class.
  Wired into `agent run` + daemon via `CompileWorkspace` (goal, entrypoints,
  git-modified, bounded tree; persisted `context-<run>.json`;
  `context.compiled` event; `--context-budget/--context-level`).
- Knowledge (`internal/harness/knowledge`): typed records, Delta
  Validate→Commit, typed contradiction/coverage/readiness.
- Doc compiler (`internal/harness/doccompile`): DAG order, render, CAS/no-op
  atomic writes, manifests.
- Workforce (`internal/harness/team`): roles, delegation modes, ownership
  guard, worktree allocation, least-context review input.
- Eval suite (`internal/harness/eval`): 13 deterministic scenarios, no tokens.

## Documentation migration result

See `docs/harness/promotion-report.md` (DocumentationMigrationReport).
Snapshot `2026-09-11-d41f18bb65f5` (43 pages, 0 broken links) reconciled;
`docs/harness/` (7 pages) promoted; `docs/PRUMO.md`,
`docs/reference/schemas.md`, `docs/SOURCE_MAP.json` updated. Post-promotion:
repository canonical docs > Notion snapshot.

## Packages/modules added

`internal/harness/{agent,model,perm,checkpoint,runtime,gateway,extagent,handoff,contextv2,knowledge,doccompile,team,aci,eval,runlayer,protocol,daemon,acp,mcp,humandocs,localintel,schemareg,repomap,lsp}`;
`cmd/prumo/agent_commands.go`; schemas harness-checkpoint/harness-handoff/
agent-event; `docs/harness/*`.

## Old modules migrated

Reused, not replaced: `internal/runtime` (Run/Checkpoint shapes),
`internal/toolgateway` (descriptor policy), `internal/environment` (local/
worktree), `internal/budget`, `internal/modelregistry`, `internal/
contextcompiler` (v1 baseline kept), `internal/experience` (handoff shapes).
v0.3/v0.4 conformance preserved.

## Providers implemented

Model: fake (deterministic), openai-compat (real HTTP/SSE), anthropic
(real `/v1/messages` SSE: text, tool_use+partial_json merge, usage,
overloaded/rate-limit retryability, ctx cancel; httptest-covered, live keys
environmental via `--api-key`/`PRUMO_MODEL_API_KEY`).
Agent: opencode-server (adapter rewritten against the measured 1.18.30 API:
abort/permissions-shape/SSE events/usage; live lifecycle+resume+usage+
handoff-attach passed 2026-09-11), codex-cli (invocation shape validated
read-only against 0.153.4; creds present), fake-agent adapters + availability
probe (`prumo agent providers`; measured 2026-09-11).

## Security/sandbox state

Deterministic allow/ask/deny with persisted resolutions; destructive always
gated; ACI path containment + output bounds; safe-mode denylist; secrets stay
at execution boundary (model never receives them by design). Container
execution: `--sandbox container` routes command tools into `run --rm
--network none` with memory/CPU/PID limits (workspace mounted, file tools on
host); missing runtimes fail fast. Live-daemon proof pending
(`TestContainerLive` behind `PRUMO_LIVE_DOCKER=1`).

## Context/Knowledge state

Baselines above; embeddings optional; Memory Atlas cross-project = retrieval
source (not prompt dump); planner/site/i18n deferred.

## Tests/evals results

- `go test ./internal/harness/...` — 15 packages green (incl. conformance).
- `go test ./cmd/prumo/ -run TestAgent` — headless run/resume/handoff/
  events/protocol/daemon-ps-logs/steer/schedule/stop/promote/budget/strict/
  egress/providers/context/knowledge/sandbox/anthropic-stub + SIGKILL recovery.
- Gateway: per-class backoff + jitter, quota cooldowns/exclusion, cost/privacy
  policy routing, tool-schema enforcement + response_format.
- Live opencode interop (`PRUMO_LIVE_OPENCODE_URL`): lifecycle, resume,
  usage, handoff-attach — all passing, zero model spend (no `Send` invoked).
- Fuzz: `FuzzCompilePacking` 1.15M execs/40s + `FuzzFingerprintStable`
  374K execs/40s, zero failures.
- Full `go test ./...` — see below (must be green before merge).
- Headless acceptance: `prumo agent run --goal …` → complete; resume +
  handoff verified in test.

## Performance measurements available

None yet beyond test timings. Budgets tracked (tokens via usage events);
latency/throughput benchmarks for providers/sandbox are open (see 19).

## Known limitations

- Model-invoking external sends (opencode `Send`, codex `exec`) deliberately
  untested live: they spend user quota and need explicit approval. All
  lifecycle paths around them are live-verified.
- Live vendor model keys untested here (adapters code-complete with
  stub-server tests; `TestAgentRunAnthropicAgainstStub` proves CLI wiring).
- External matrix is adapter-level; live Codex/OpenCode interop untested here.
- Daemon is local-socket only; no auth, no remote transport, no
  background supervision (systemd/launchd units future).
- Compaction/steering policies minimal; PTY lifecycle basic.
- Protocol IDL/SDK generation pending (envelope + 3 schemas shipped).
- Container/gVisor sandbox, fuzz/chaos, local-inference workers deferred.

## Deferred items

Prumo Code repo, Desktop/TUI, remote/cloud execution, microVM, complex A2A,
docs website, visual polish — all explicitly out of this Goal.

## Protocol status

Versioned protocol kernel: JSON envelopes + `harness-checkpoint`,
`harness-handoff`, `agent-event` schemas + `internal/harness/protocol`
v0.1.0 negotiation (`prumo agent protocol [--client X]`). Timeline replay:
JSONL event log per run via `prumo agent events`. Local daemon
(`internal/harness/daemon`, Unix socket, `serve/ps/logs`): persisted run
records + restart-safe reconnect; cancellation at safe points; provider
factory shared via `model.ForName`. Public typed SDK `sdk/prumo` (stdlib
only: Start/Status/List/Events/Cancel/Protocol/Wait) with boundary
conformance (`TestBoundaryNoInternalImports`) + live roundtrip test. Sandbox ladder is honest
(local/worktree available; container availability detected, execution
pending). Fuzz seeds: context packing budget invariant + fingerprint
determinism. Remote transport + full IDL + generated bindings remain.

## Prumo Code split readiness

NOT READY. Gate: HA0–HA4 done, HA5 baseline usable, headless e2e done;
missing: hardened sandbox provider, 2nd live vendor adapter, live external
matrix proof, versioned protocol IDL/SDK, daemon reconnect/replay. Creating
`raillen/prumo-code` now would force `internal/` imports — forbidden by the
boundary invariant.

## Remaining open questions

From page 19 still open: ACP client/server subset order; MCP Go SDK choice;
Docker vs Podman first + rootless mandate; SQLite driver; LSP sharing;
plugin isolation; perf budgets; local-model benchmarks/thresholds/TTL;
model-package versioning. Plus: CLI rule (`cmd` may import only app+protocol)
contradicts reality — needs ADR, not silent fix.
