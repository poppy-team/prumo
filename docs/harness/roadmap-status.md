# Harness roadmap status (HA0–HA11) + split gate

Snapshot: `2026-09-11-d41f18bb65f5`. ACCEPTED != implemented. Truth table
(HA-level). **Detalhe por gap: [gap-register](gap-register.md) — canônico,
atualizado a cada incremento e impresso no relatório da rodada.**

Snapshot: `2026-09-11-d41f18bb65f5`. ACCEPTED != implemented. Truth table:

| ID | Scope | Design | Implementation |
|----|-------|--------|----------------|
| HA0 | AgentRuntime contracts/events | ACCEPTED | implemented (`internal/harness/agent` + schemas) |
| HA1 | FakeProvider + conformance + first ModelProvider | ACCEPTED | implemented (Fake + OpenAI-compat + suite) |
| HA2 | Native Agent state machine | ACCEPTED | implemented (reentrant Runner, safe points) |
| HA3 | ToolGateway + Permission integration | ACCEPTED | implemented (ACI catalog + perm engine wired in Runner) |
| HA4 | checkpoint/restart/resume + budgets + observability | ACCEPTED | implemented (atomic store, idempotent journal, budget hook, AgentEvent) |
| HA5 | Coding ACI + Sandbox baseline | ACCEPTED | implemented (ACI catalog + containment + local/worktree + container execution with limits; live-daemon run pending, gVisor future) |
| HA6 | 2nd ModelProvider + Gateway/fallback | ACCEPTED | code-complete (openai-compat + anthropic adapters, gateway routing/fallback/CB; live keys environmental) |
| HA7 | 1st external AgentProvider | ACCEPTED | implemented (OpenCode adapter verified against real 1.18.30 API incl. live lifecycle/resume/usage; Codex invocation shape validated read-only; model-invoking send pending spend approval) |
| HA8 | Handoff cross-provider dogfood | ACCEPTED | implemented (typed bundle + full fake transfer eval + live opencode attach/abort/delete; model-mediated continuation pending spend approval) |
| HA9 | Context v2 / Knowledge integration | ACCEPTED | implemented (workspace v2 compilation wired into run+daemon with persisted manifests; per-run knowledge seeding with coverage/readiness; global promotion via delta review) |
| HA10 | multi-agent/worktrees | ACCEPTED | implemented (concurrent roles, fail-fast cancel, hard budgets, snapshot diffs, least-context review gate, bounded-auto with cap) |
| HA11 | compatibility/eval matrix | ACCEPTED | implemented (unit+conformance+CLI evals + live availability/session matrix + 1.5M fuzz execs, zero failures; generated non-Go bindings pending) |
| HD0–HD4 | Human Docs Runtime baseline | ACCEPTED | implemented (spec+composition, deterministic planner, README/reference/tree via CAS, assisted briefs, coverage/readiness gates; site/i18n HD5+ future) |
| Local inference | workers (llama.cpp/ONNX) | ACCEPTED | deferred (interfaces reserved; No-LLM first-class) |

## Split gate (`raillen/prumo-code` NOT created)

- [x] HA0 contracts
- [x] HA1 FakeProvider + first ModelProvider
- [x] HA2 Native Agent
- [x] HA3 Tool/Permission integration
- [x] HA4 checkpoint/restart/resume
- [~] HA5 Coding ACI + Sandbox baseline (baseline usable, hardening continues)
- [x] headless coding Run end-to-end (`prumo agent run` → tool → checkpoint)
- [~] versioned public protocol (envelope + 3 schemas + negotiation kernel v0.1.0 + `prumo agent protocol` + checked-in IDL manifest with code-conformance tests; typed public SDK `sdk/prumo` with boundary test; generated bindings for other languages pending)
- [~] reconnect/replay (resume+handoff CLI done; JSONL event replay via `prumo agent events` done; local daemon `serve/ps/logs` with restart-safe reconnect done; remote TCP+TLS+token done nos limites — CA corporativa e hardening futuro)

Verdict: NOT READY for split — see `HARNESS_IMPLEMENTATION_REPORT.md`.
The first Prumo Code must build with zero `internal/` imports; that
conformance is now pinned for the Go SDK (`TestBoundaryNoInternalImports`)
and remains to be extended to remote transport + generated bindings.
