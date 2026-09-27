# ADR 014 — Reading what a run changed, and being told instead of asking

**Status:** Accepted; implemented (v0.4.0 protocol) — GAP-084, GAP-085.
**Date:** 2026-09-17
**Supersedes:** nothing. **Related:** ADR 013 (the client polls, it is not pushed
to), GAP-084, GAP-085.

## Context

Two needs came from the same place — the client can only see what the daemon
chose to publish — and they resolve in opposite directions.

**Reading a diff after the fact.** The harness reports `file.changed` with a path
and an operation. A person who wants to know *what* changed in a file has no
route to it: the patch exists while the tool runs (it is in the call's arguments)
and is gone once the run ends, unless a policy gated that call — in which case the
approval dialog shows it. The obvious shortcut, putting the patch on the event,
was rejected: the timeline is replayed whole on reconnect, so every patch of every
run would travel again on every reconnect to serve a question almost nobody asks.

**Being told instead of asking.** The client asks the daemon for status and events
four times a second while a run is in flight. That works, and it is visible in the
code as `PollInterval` with a comment explaining why it exists. It is the last
place where the client's design is shaped by the protocol's absence rather than by
the interface's needs.

## Decision

Both are protocol additions, and both belong to version **0.4.0** (from 0.3.0,
which the manifest declares). They are additive: an older client keeps polling and
keeps working, and a daemon that does not know an operation answers with its usual
failure.

### 1. A read, on demand: `diff`

```
request:  {"op": "diff", "run_id": "<id>", "path": "<workspace-relative path>"}
response: {"ok": true, "path": "...", "kind": "patch", "content": "<unified diff>"}
```

- The **daemon** keeps what it needs to answer: when a tool call changes a file,
  the args that produced the change are recorded beside the change in the run's
  own record. This is the opposite trade from the one rejected above — the patch
  lives in the daemon's store, where it is read once when asked, instead of on the
  timeline, where it would be replayed forever.
- The answer is a **unified diff** or an explicit `kind`: a file whose change
  cannot be expressed as a patch (a deletion, a binary) says so rather than
  returning an empty document.
- A path the run did not touch is an **error**, not an empty diff: "no diff" and
  "not changed" are different answers, and only one of them is true.

### 2. A push channel: `subscribe`

```
request:  {"op": "subscribe", "run_id": "<id>", "from": <cursor>}
stream:   {"op": "event", "run_id": "...", "event": {…}}\n   (one JSON object per line, until cancelled)
          {"op": "subscribed", "run_id": "...", "from": <cursor>}   (first line)
```

- **Framing.** The transport is JSONL with one response per request, so a stream
  needs a shape that cannot be confused with a response: the first line is an
  acknowledgement carrying the cursor the stream starts at, every following line
  is one event, and the stream ends when the run reaches a terminal status or the
  connection closes. A request while a subscription is open is answered on the
  same connection: the protocol gains one concurrent stream per connection, not a
  second channel.
- **Replay stays the contract.** The subscription is a faster path to the same
  events, not a different record: `from` is a cursor, the first line says which
  one was honoured, and a client that reconnects replays from its last cursor
  exactly as it does today (W12). A subscription that cannot honour the cursor
  says so instead of starting from zero in silence.
- **The client keeps a fallback.** If the daemon does not advertise `subscribe`
  in `protocol`, the poll loop stands. That is what makes the version bump
  additive rather than breaking, and it is why `PollInterval` does not disappear
  from the code — it becomes the fallback with a reason instead of the only path.

## Consequences

- **The harness gains a store for what a run changed**, keyed by run and path.
  Its retention follows the run's own retention (GAP-020); a pruned run answers
  "not available" rather than a wrong diff.
- **The SDK gains two methods** — `Diff(ctx, runID, path)` and
  `Subscribe(ctx, runID, from)` returning a channel of events — and the client
  gains one panel that reads on demand and one loop that reads a stream.
- **`PollInterval` becomes conditional**, and the comment explaining it changes
  meaning: it is no longer "the protocol has no subscription" but "this daemon
  does not offer one".
- **Both additions are testable without a terminal**: the daemon side by asking
  for a diff of a run that changed a file and of one that did not, and the stream
  side by subscribing, cancelling mid-run, and reconnecting from the cursor.
- **Nothing here changes what a run records.** The timeline keeps its payloads
  minimal, `file.changed` keeps carrying a path and an operation, and the diff is
  a question asked of the store rather than a fact replayed forever.
