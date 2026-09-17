# prumo-tui

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

**Builds and runs on the Charm v2 stack.** The migration from the upstream
Bubble Tea v1 stack is done, the harness integration is in place, and the client
starts, supervises a daemon and drives a run.

What the integration required from the core, and got:

1. **The run timeline now carries content, not just lifecycle.** It previously
   emitted `run.started`, `context.compiled` and `run.finished` and nothing in
   between, so no client could stream a conversation. It now emits
   `text_delta`, `reasoning_delta` and `tool_call_ready`, and a run that stopped
   for a decision says `run.paused` rather than `run.finished`.

Deferred deliberately, because the harness cannot answer them yet — not because
they were too hard:

2. **Model picker and custom-command dialogs** — removed. They need a models
   operation on the protocol, and a decided command surface. A client with its
   own list would be asserting what it cannot verify.
3. **File-history sidebar** — removed. The timeline carries no file-change or
   diff events; that is a gap in the event vocabulary, not a client bug.
4. **Tool-call previews** are driven by the ACI catalog (`fs.read`,
   `edit.patch`, `process.exec`, …) with open argument maps, so a renamed
   parameter renders rather than breaks.

Still owed:

5. **This client's own interface map.** `docs/ui-ux/interface-map.json` verifies
   the archived spike; the shipping client has no map yet.

## Boundary

`boundary_test.go` fails the build if this module ever imports
`github.com/raillen/prumo/internal/...`, transitively and including test files.
ADR 013 made the boundary structural by moving the client into its own module;
this is the check that keeps structure from being undone by a single import.
