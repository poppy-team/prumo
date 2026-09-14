# Providers: Model vs Agent + Model Gateway

## ModelProvider (Prumo owns the loop)

Package `internal/harness/model`. Contract: Capabilities, Models,
Stream→EventStream, Cancel, Usage, Health. Normalized events: TextDelta,
ReasoningSummaryDelta, ToolCallDelta/Ready, UsageUpdated, Warning, Error,
Completed, Cancelled.

- `FakeModelProvider`: deterministic scripts (`response|tool call|permission|
  tool result|continuation|complete`), shared conformance suite.
- `OpenAICompat`: first real adapter (`/chat/completions` SSE, tool_calls
  normalization, usage, retryable 429/5xx, cancel via ctx). No vendor types
  escape; swap the library without changing domain contracts.
- `Anthropic`: second real adapter (`/v1/messages` SSE: text deltas,
  `tool_use` blocks with `partial_json` merge, usage, overloaded/rate-limit
  retryability, cancel via ctx). httptest covers text, tool-use merge,
  retryable errors and cancel. Live keys remain environmental
  (`--api-key` / `PRUMO_MODEL_API_KEY`); wire mapping is code-complete.

## AgentProvider (external runtime owns the loop)

Package `internal/harness/extagent`. Contract: Discover/Status/Capabilities/
Models, CreateSession/ResumeSession, Send/Cancel, Approve/Deny, Events, Close.

- `OpenCodeServer`: verified against the real `opencode serve` API
  (measured 1.18.30): `POST /session {title?}`, `GET /session`,
  `GET /session/:id/message`, `POST /session/:id/message {parts:[{type,
  text}]}` (SSE turn stream), `POST /session/:id/abort`, `POST
  /session/:id/permissions/:perID {response: once|always|reject}`,
  `DELETE /session/:id`, `GET /event` SSE. Rich contract
  (`ResumeSession` via list-verify, `Usage` from session tokens/cost).
  Stub tests mirror these shapes; live lifecycle tests
  (`PRUMO_LIVE_OPENCODE_URL`) passed 2026-09-11.
- `CodexCLI`: structured `codex exec --json` adapter; invocation shape
  validated read-only against codex-cli 0.153.4 (`exec [PROMPT]`, `--json`,
  `resume`, `--output-schema` confirmed present).
- `FakeAgent`: conformance double (session/events/permissions/usage/cancel/resume).

External state is normalized, never canonical.

## ACP agent server (`internal/harness/acpserver`)

The Harness itself speaks ACP v1 as an agent over stdio
(`prumo agent acp`, backed by the local daemon): `initialize`,
`session/new|load|resume|list|delete|close`, `session/prompt` with
`session/update` streaming (`agent_message_chunk`, stop reasons
`end_turn|cancelled`), `session/cancel` notification. Per-session
`mcpServers` fail loudly (daemon-level `--mcp` instead); authenticate,
elicitation and terminals are out of the v1 subset by decision (GAP-032).
Prompts steer live runs or start fresh ones — editors never lose input.

## Bounds (explicit, not gaps-in-disguise)

- Model-invoking `Send` on either external runtime spends user quota: live
  send stays stub-tested until the user approves model spend. Everything up
  to the model call is live-verified.
- Codex credentials are present on the dev host (`~/.codex/auth.json`);
  live `exec` is ready to run the moment spend is approved.

## Availability matrix (`prumo agent providers`)

`internal/harness/extagent/probe.go` reports honest availability: binaries +
versions + server reachability, never assumed interop. Measured 2026-09-11
(dev host):

```text
fake             model  available   builtin      deterministic double
openai-compat    model  unconfigured             needs PRUMO_MODEL_BASE_URL
anthropic        model  unconfigured             needs PRUMO_MODEL_API_KEY
opencode-cli     agent  available   1.18.30      binary present
opencode-server  agent  probed live              lifecycle verified 2026-09-11 vs 1.18.30 (send pending spend approval)
codex-cli        agent  available   0.153.4      binary + credentials present; live exec pending spend approval
acp-generic      agent  unconfigured             needs PRUMO_ACP_URL
```

Live session interop against those CLIs/servers is the next matrix step;
the adapters + probe are the conformance baseline it will run against.

## Model Gateway (internal, separate from Workforce)

Package `internal/harness/gateway`. `ModelRoute/RouteTarget/ProviderHealth/
QuotaState`, healthy-first deterministic routing, retry, circuit breaker
(3 failures → open + 30s cooldown), cost/latency/capability/privacy inputs.

Rule: transparent fallback ONLY before observable side effects; after side
effects the gateway refuses and the caller must do an explicit Handoff.
