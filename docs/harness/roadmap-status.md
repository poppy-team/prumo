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
| HA5 | Coding ACI + Sandbox baseline | ACCEPTED | implemented (ACI catalog + containment + local/worktree + container execution with limits; live-daemon run PROVEN 2026-09-14 via TestContainerLive, gVisor future) |
| HA6 | 2nd ModelProvider + Gateway/fallback | ACCEPTED | code-complete (openai-compat + anthropic adapters, gateway routing/fallback/CB; live keys environmental) |
| HA7 | 1st external AgentProvider | ACCEPTED | implemented (OpenCode adapter verified against real 1.18.30 API incl. live lifecycle/resume/usage; Codex invocation shape validated read-only; REAL model send verified live via cursor-cli 2026-09-14 with own auth — opencode/codex sends blocked by vendor quota, Prumo boundary verified) |
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
- [~] versioned public protocol (envelope + 3 schemas + negotiation kernel v0.2.0 + `prumo agent protocol` + checked-in IDL manifest with code-conformance tests; typed public SDK `sdk/prumo` with boundary test; v0.2.0 added the `approve`/`deny` ops and a run waiting for a decision is observable as `awaiting_approval`; generated bindings for other languages pending)
- [~] reconnect/replay (resume+handoff CLI done; JSONL event replay via `prumo agent events` done; local daemon `serve/ps/logs` with restart-safe reconnect done; remote TCP+TLS+token done nos limites — CA corporativa e hardening futuro)

Verdict: NOT READY for split — see `HARNESS_IMPLEMENTATION_REPORT.md`.
The first Prumo Code must build with zero `internal/` imports; that
conformance is now pinned for the Go SDK (`TestBoundaryNoInternalImports`)
and remains to be extended to remote transport + generated bindings.

## Post-harness: Implementation Waves

The split gate is READY (conditional). Before the H10 TUI spike,
implementation waves W0–W21 address structural gaps identified by deep
research analysis and the Documentation Control Plane Deep Audit.
See `docs/development/waves.md` for canonical wave definitions and entry/exit criteria.

**Wave progress** (2026-09-15): **✅ W0–W18, W20** · **🟡 W19** (example execution,
DOC-GAP-020) · **🟡 W21** (workforce routing by impact type, DOC-GAP-024; full
Markdown AST, DOC-GAP-026) · **⬜ none**. W21.2/W21.3 (query/intent telemetry) are
**refused by ADR 011** because the trust model forbids collecting what users
retrieve; the refusal is recorded, not left as an open gap.
TUI gate: every P0/P1 contract wave is complete and
`prumo docs verify --strict` passes.

Gate evidence: `conformance/schema-runtime` (W1), `prumo docs authority` (W0),
`internal/knowledge`, `internal/gauntlet`, W12 client-facing reconnect/replay
conformance, `evals/documentation` (W13, 13 cases), semantic readiness v2
(W15: `internal/documentation/semantic.go`, `TestSemanticReadinessDogfood`),
the agent surface compiler (W16: `internal/agentsurface`, `docs/agents/instruction-ir.json`,
`prumo docs agents verify`), the semantic impact graph and Goal preflight/postflight
(W17: `internal/documentation/impact_semantic.go`, `plan.go`), the publishing and
AI-retrieval plane (W18: `internal/docpublish`, `prumo docs site verify`), the
continuous verification and Gauntlet runtime (W19: `VerifyDocs`, `VerifyDocsStrict`,
`internal/gauntlet/run.go`, new CI steps), documentation lifecycle
(W20: `internal/doclifecycle`) and documentation intelligence plus brownfield
adoption (W21: `internal/docintel`, `prumo docs adopt`).
Full repo suite: 73 packages ok, 0 failures.

W15 changed a verdict on purpose and then earned it back honestly: the first draft
of the repository's own bindings cited a document the authority map classifies as
historical, and the semantic evaluator rejected it as `non-canonical-evidence`
instead of accepting a green result. W15.10 was then authored properly, so
`prumo docs readiness` is now green because 7 contracts genuinely carry accepted
claims with verified evidence — not because the check was weakened.

| Priority | Waves | Focus |
|----------|-------|-------|
| P0 | W0–W7, W15 | Authority cleanup, schema conformance, KnowledgeUnit, ContextManifest, UI contracts, locale, Gauntlet schema, Semantic Readiness v2 |
| P1 | W8–W14, W16–W19 | Shared Product Contract, design tokens, doc compiler, accessibility, reconnect/replay, evals, Framework update, Agent surfaces, Impact graph, Publishing/AI retrieval, Continuous verification |
| P1/P2 | W20–W21 | Documentation lifecycle (i18n/media/versioning) & Intelligence, adoption & scale |
| Gate | — | P0 complete + W8/W9/W11/W12/W15 → H10 TUI spike may begin (**met**) |
