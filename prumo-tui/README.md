# prumo-agent tui

The Prumo terminal client. It is a **client of the harness, never part of it**:
it reaches the daemon through the public SDK
(`github.com/raillen/prumo/sdk/prumo`) and through the `prumo` binary it
supervises as a subprocess. The core module does not depend on this one.

The view layer — layout primitives, chat surface, dialogs, command palette,
themes, styling — is derived from the archived OpenCode Go implementation (MIT,
revision `73ee493265acf15fcd8caab2bc8cd3bd375b63cb`). See
`../THIRD_PARTY_NOTICES.md` and `../docs/adr/013-tui-foundation.md`, and the
upstream license preserved at `LICENSE.opencode`.

## Why it is a separate module

A client that must not know about providers, quotas or orchestration must not
live inside `internal/`, which is exactly the tree that grants that access. A
separate module makes the boundary structural instead of merely tested, and it
keeps this module's heavy terminal dependencies out of the core — the core is
standard-library only because the client moved out.

## What was imported, and what was not

Imported: the view layer only (`internal/tui/**`) plus the small support types
the view needs (message model, session/prompt/permission shapes, the event
broker, tool display types).

**Not** imported, because the harness already owns each concern:

| Upstream | Prumo owner |
|---|---|
| provider integrations and the agent loop | the harness runtime and the model adapter layer |
| SQLite persistence and migrations | the daemon's store |
| the LSP client and its vendored protocol | the harness language tooling |
| the configuration service (file + environment) | the protocol; the client keeps almost no settings |

The client therefore holds no state the daemon cannot recreate: sessions mirror
runs, and the conversation is folded from the run's event log.

## Status

**Builds and runs on the Charm v2 stack**, as a client of the harness: it
supervises (or attaches to) a daemon, drives runs, and draws what the run's own
record says.

**The conversation is live, and that took a bridge.** The services below the view
publish on brokers, and nothing carried them to the program: every event case in
the model was dead code, so text, tool calls and permission gates never reached
the screen. `internal/stream` subscribes to the five brokers and feeds the
`tea.Program`, the runner folds the run's log into the conversation, and
`internal/runtime` is the only package that knows the protocol exists.

What the integration required from the core, and got:

1. **The run timeline carries content, not just lifecycle.** It previously
   emitted `run.started`, `context.compiled` and `run.finished` and nothing in
   between, so no client could stream a conversation. It now emits
   `text_delta`, `reasoning_delta` and `tool_call_ready`, and a run that stopped
   for a decision says `run.paused` rather than `run.finished`.
2. **The timeline reports what a run cost and touched.** A tool that edits a file
   emits `file.changed` with the path and the operation, and each model request
   reports its `usage`, so the statusline reports the harness's own numbers
   instead of zeros. `ctrl+g` lists **which** files changed, not only how many.
3. **A run can be steered.** Sending while a run is in flight says something to
   that run rather than starting a second one.

Deferred deliberately, because the harness cannot answer them yet — not because
they were too hard:

- **Custom-command dialogs** — they need a decided command surface, which is a
  product decision rather than a missing operation.
- **File-history sidebar (versions and diffs)** — the harness reports which file
  changed and how, not what it looked like before, so a panel of versions needs
  events that do not exist yet.
- **Attachments** — a message in the harness carries its content as text, so
  there is nothing for an attachment to attach to yet.

Still owed, and tracked in `docs/harness/gap-register.md`:

- **The manual accessibility attestations.** The strategy and its evidence exist
  (see `docs/ui-ux/interaction.md` §Screen readers); a session with a screen
  reader actually attached has not happened, and nothing here claims it has.
- **A push channel.** The client polls; the protocol exposes replay plus status.
  The difference is documented rather than hidden.

## Where the client is described

| Question | Artifact |
|---|---|
| What keys do what | `docs/ui-ux/interaction.md`, checked against the bindings by a test |
| What is on screen, and where | `docs/ui-ux/interface-map.json`, verified by `prumo-agent ui verify` |
| What states exist | `docs/ui-ux/state-matrix.json`, with a golden frame per drawn state |
| What a user is trying to do | `docs/ui-ux/flows.md` |
| How it is themed | `docs/ui-ux/theming.md` |
| What is missing | `docs/harness/gap-register.md` |

## Boundary

`boundary_test.go` fails the build if this module ever imports
`github.com/raillen/prumo/internal/...`, transitively and including test files.
ADR 013 made the boundary structural by moving the client into its own module;
this is the check that keeps structure from being undone by a single import.

## The first time you run it

If opencode is not installed and no provider was chosen, the client asks once —
because opencode serves models for free and authenticates itself, so it is the
one path that needs no api-key, and nobody guesses that from an empty model
list. The three answers are:

- **Install opencode** (`npm install -g opencode-ai`) and run with it from then
  on. It is npm rather than a `curl | bash` line because the package and its
  binary can be checked on this machine; the command is shown before it runs.
- **Configure an api-key** — the client prints the variables to export
  (`PRUMO_MODEL_API_KEY`, and `PRUMO_MODEL_BASE_URL` for an openai-compatible
  endpoint). It does not ask you to type a secret into it: this file keeps
  preferences, not credentials.
- **Not now** — the deterministic `fake` provider keeps runs offline.

The answer is remembered, so it asks once. `--provider` always wins over it.

If opencode *is* installed, nothing is asked — but the client says so once, with
the flag that uses it, because "a real provider is one flag away" is exactly the
thing a person cannot discover by looking.

## Which models see, reason, or take tools

The picker (`ctrl+k` → models, or `--model`) shows what each model declares:
`[text reasoning vision tools audio]`, or **not declared** when nobody said —
never an invented capability. The source is the workspace's `.prumo/models.json`:

```json
{"version": 1, "models": {"some-model": {"text": true, "vision": true, "tools": true}}}
```

Format reference: `docs/harness/providers.md`; schema:
`schemas/model-declarations.schema.json`.
