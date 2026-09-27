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
(`prumo-agent agent acp`, backed by the local daemon): `initialize`,
`session/new|load|resume|list|delete|close`, `session/prompt` with
`session/update` streaming (`agent_message_chunk`, stop reasons
`end_turn|cancelled`), `session/cancel` notification. Per-session
`mcpServers` fail loudly (daemon-level `--mcp` instead); authenticate,
elicitation and terminals are out of the v1 subset by decision (GAP-032).
Prompts steer live runs or start fresh ones — editors never lose input.

## `opencode` — a delegated model provider

`--provider opencode` hands the turn to the opencode CLI, which runs it with its
own tools, its own permission policy and **its own authentication** — including
the models it serves for free. It is a `ModelProvider` with one difference that
is declared rather than hidden: `Capabilities().ToolCalls` is **false**, because
the tools are opencode's. Forwarding its tool calls would make Prumo execute a
tool that was already executed; reporting them as text would put words in the
model's mouth. So the run gets the answer and the spend, and says so.

```bash
prumo-agent agent run --provider opencode --model opencode/mimo-v2.5-free --goal "..."
prumo-agent agent providers          # opencode is listed as a model provider
prumo-agent agent models --provider opencode
```

The wire format is `opencode run --format json`, one JSON object per line:
`{type, timestamp, sessionID, ...extra}`. The types are `text`, `reasoning`,
`tool_use`, `step_start`, `step_finish` (which carries `tokens` and `cost`) and
`error`. These shapes were read from the installed binary (1.18.31) rather than
guessed; an event the adapter does not recognise is reported as unrecognised and
never invented. A provider error is surfaced with the provider's own words — an
exhausted free quota says so, instead of failing as "the run failed".

`PRUMO_OPENCODE_BIN` overrides the binary; `PRUMO_OPENCODE_DIR` the working
directory (the daemon sets it to the workspace for runs it hosts).

## What each model can do (`.prumo/models.json`)

No provider publishes its models' features in a form a client can read: OpenAI's
`/models` returns ids, Anthropic's returns ids and display names. So the harness
reports **what the workspace declares** and marks everything else as undeclared —
it never infers `vision` from a model's name.

```json
{
  "version": 1,
  "models": {
    "some-model": {"text": true, "vision": true, "tools": true, "context_tokens": 200000},
    "another-model": {"text": true, "reasoning": true}
  }
}
```

The `models` operation answers with both faces: `models` carries bare ids (for a
client that only picks one), `model_info` carries
`{"id", "declared", "capabilities"}`. A model nobody declared arrives with
`declared: false` — the difference between "nobody said" and "it cannot see" is
the whole point, and a client that erases it is lying about a model. Schema:
`schemas/model-declarations.schema.json`.

`vision` is the flag the image-attachment work gates on (`GAP-090`): a model that
does not declare it receives an image reference as text, not as pixels.

## Bounds (explicit, not gaps-in-disguise)

- Model-invoking `Send` on either external runtime spends user quota: live
  send stays stub-tested until the user approves model spend. Everything up
  to the model call is live-verified.
- Codex credentials are present on the dev host (`~/.codex/auth.json`);
  live `exec` is ready to run the moment spend is approved.

## Availability matrix (`prumo-agent agent providers`)

`internal/harness/extagent/probe.go` reports honest availability: binaries +
versions + server reachability, never assumed interop. Measured 2026-09-11
(dev host):

```text
fake             model  available   builtin      deterministic double
openai-compat    model  unconfigured             needs PRUMO_MODEL_BASE_URL
anthropic        model  unconfigured             needs PRUMO_MODEL_API_KEY
opencode-cli     agent  available   1.18.30      binary present
opencode-server  agent  probed live              lifecycle+resume+usage+handoff verified 2026-09-11/14 vs 1.18.30
codex-cli        agent  available   0.154.0      binary + credentials present; ChatGPT usage limit 2026-09-14 (retry after 19/09)
cursor-cli       agent  available   2026.07.23   binary + login present; REAL model turn verified 2026-09-14 (own auth, no API key)
acp              agent  spawn-based              any ACP v1 agent binary via ACPClient (stdio JSON-RPC); dogfood live vs `agent acp` 2026-09-14
acp-generic      agent  unconfigured             needs PRUMO_ACP_URL
```

Live send proofs (2026-09-14, user-approved spend):

- `cursor-agent -p --output-format json --trust`: real turn completed
  (`TestCursorLiveSend`), adapter `CursorCLI` in `extagent`.
- ACP client dogfood (`TestACPLiveDogfood`, `PRUMO_LIVE_ACP=1`): Prumo
  spawns `agent acp` (real binary) against a live daemon backed by the fake
  provider — full external loop with zero API keys: session/new →
  session/prompt → streamed update → end_turn → session/delete.
- `opencode serve` + `OpenCodeServer.Send`: request reached the server; the
  default model (openrouter free tier) failed with provider-side wallet/rate
  errors — Prumo boundary performed correctly (session, message, typed
  provider error surfaced).
- `codex exec --json`: invocation shape correct; ChatGPT account hit usage
  limit (resets 2026-09-19).
- Container live proof: `TestContainerLive` PASS (docker, `--network none`,
  memory/CPU/PID limits).

Live session interop against those CLIs/servers is the next matrix step;
the adapters + probe are the conformance baseline it will run against.

## Model Gateway (internal, separate from Workforce)

Package `internal/harness/gateway`. `ModelRoute/RouteTarget/ProviderHealth/
QuotaState`, healthy-first deterministic routing, retry, circuit breaker
(3 failures → open + 30s cooldown), cost/latency/capability/privacy inputs.

Rule: transparent fallback ONLY before observable side effects; after side
effects the gateway refuses and the caller must do an explicit Handoff.
