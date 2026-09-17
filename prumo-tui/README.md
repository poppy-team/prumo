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

**In progress — the module does not compile yet.** The structural import is
done (dependencies resolve; `go mod tidy` succeeds) and the remaining work is a
bounded, listable set:

1. **Permission dialog previews.** `internal/tui/components/dialog/permission.go`
   renders per-tool previews using the upstream tool names and argument structs
   (`tools.BashToolName`, `EditPermissionsParams`, …). Prumo's tool names come
   from the ACI catalog (`fs.read`, `edit.patch`, `process.exec`, …) and its
   arguments are open maps, so these previews must be rewritten against our
   catalog. This is the approval surface, so it is the first thing to finish.
2. **Model picker and custom-command dialogs** — removed. Both need protocol
   surface Prumo does not expose yet: a models operation, and a source for
   commands. They will come back when the harness can answer them, not before.
3. **File-history sidebar** — removed with the same reasoning: the timeline
   carries no file-change or diff events yet. That is a real gap in the event
   vocabulary, not a client bug.
4. **`tui.go` wiring.** The root model still calls the upstream `app` shape in a
   few places; it now receives `internal/app`, which is backed by
   `internal/runtime.Runner`.
5. **`cmd/prumo-tui`** — the entrypoint is not written yet.
6. **Streaming assistant text needs a harness change.** The run timeline
   currently carries run lifecycle and permission events, not model deltas, so
   `internal/runtime.Runner` folds `text_delta` / `reasoning_delta` /
   `tool_call_ready` defensively — the harness must emit them for the client to
   stream a real conversation. This is the one change the integration requires
   on the core side.

## Boundary

`TestBoundaryNoInternalImports`-style enforcement belongs here too: this module
must never import `github.com/raillen/prumo/internal/...`. It is not written yet,
and it is the first test to add when the module compiles.
