# Security: Permissions, Sandbox, Egress

## Permission Engine (`internal/harness/perm`)

Deterministic policy over `PermissionRequest` (agent/role, action, resource,
args summary, fs scope, network dests, credential scopes, data class,
reversibility, risk, run/turn). Outcomes `allow|ask|deny` with scope
(once|turn|session|run|project), reason, constraints, expiry. Resolutions are
persisted (`PermissionApproved/PermissionDenied`). Destructive kinds always
require approval unless explicitly listed. The LLM never enforces.

## ToolGateway + Coding ACI (`internal/harness/aci`)

Catalog baseline: `fs.read/list/search`, `code.symbols/diagnostics`,
`edit.patch/create/delete/move` (+guarded), `process.exec`, `test.run`,
`git.status/diff`. `edit.patch` applies unified diffs via `git apply`
(atomic, context-validated) with an optional `base_rev` guard refusing
stale trees; re-apply fails instead of duplicating. `fs.search` prefers
`rg`, falling back to `grep` on hermetic hosts. Outputs boundados passam por
redação default (API keys, bearer, password=...) antes de alcançar o modelo.
MCP servers attach via `--mcp "<bin> [args]"` (stdio, untrusted-by-default):
tools advertised as `mcp.*` with server schemas, execution policy-gated
(allowlist, scope, safe-mode), specs sent to capable models.
Egress fail-closed opt-in: `--egress-deny` nega destinos fora de
`--egress-allow` (sufixos `.dominio` valem); `restricted` nunca sai mesmo
em modo permissivo. Container nega rede estruturalmente.
Execution respects trust, fs scope (cleaned + contained,
symlink/`..` escapes rejected), network scope, credentials, budget,
idempotency, side-effect journal, permissions. Output is bounded + truncatable.

## Environment / Sandbox

`internal/environment` hardened baseline: local workspace + isolated worktree
contracts, path safety, process execution with safe-mode denylist, workdir
containment. `SandboxProvider` interface is gradual: LocalTrusted → Worktree
→ Container (Docker/Podman, rootless preferred) → Strong (gVisor) → Remote →
microVM (future). `internal/harness/aci/sandbox.go` ships the honest ladder:
`LocalProvider`/`WorktreeProvider` always available, `ContainerProvider`
reports availability via runtime detection (`DetectContainerRuntime` prefers
podman over docker; bogus runtimes report unavailable instead of pretending).

Container execution (`internal/harness/aci/container.go`, `--sandbox
container --sandbox-image <img>` on `agent run`/`serve`): command tools
(`process.exec`, `test.run`, `code.diagnostics`) run via `run --rm
--network none` with memory/CPU/PID limits and the workspace mounted;
file tools stay on the host against the same workspace. Missing/unreachable
runtimes fail fast — never silent host fallback. Live-daemon verification is
pending (CI has no reachable daemon; `TestContainerLive` runs with
`PRUMO_LIVE_DOCKER=1`). gVisor/strong isolation remains future.

## Checkpoint / Resume / Side effects (`internal/harness/checkpoint`)

Atomic writes (tmp+rename), SHA-256 fingerprints, `Latest(run)` recovery.
`RecordIntent` BEFORE effects with `idempotency_key`; `RecordOutcome` AFTER.
Replay with a seen key after `applied` is skipped: duplicate observable side
effect rate target = zero. `kill prumo → restart → resume` is covered by
`prumo agent run|resume` + `checkpoint_test` + eval suite.

## Budget + Observability

`internal/budget` envelopes (tokens/money/time/tool-calls/retry/concurrency)
with hard/soft modes + reservations. Model usage feeds `ConsumeBudget`; hard
exhaustion fails the turn deterministically. Timeline: `AgentEvent` stream
(model/provider/agent/tools/permissions/context/handoff/failures) projected
to `DomainEvent` for clients. Every important execution is explainable.
