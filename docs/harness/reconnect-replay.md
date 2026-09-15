# Reconnect & Replay (client contract)

> Authority: repository-canonical (W12.4). Conformance tests:
> `internal/harness/daemon/reconnect_client_test.go`.

Reconnect and replay are a **public client contract**: the TUI, desktop client,
CLI and SDKs all depend on being able to re-attach to a run and rebuild their
view without owning canonical state.

## Contract

1. **Replay is stable.** Reading a run timeline twice — including from a
   different client process — returns the same events in the same order, with
   unique event IDs. Replay never duplicates or reorders.
2. **Replay is durable.** The timeline is append-only JSONL on the daemon store.
   Restarting the daemon over the same store preserves exactly what clients saw.
3. **State is serialisable.** Status and events are plain JSON, so a client can
   persist and restore its own view without server assistance.
4. **Clients own presentation, not truth.** A client that reconnects reconciles
   against the replayed timeline; it never invents events or merges divergent
   state.
5. **Bounded replay.** The server caps a replay response (currently the last 500
   events per run). Clients must not assume an unbounded timeline; durable
   history beyond the cap lives in the run record and evidence artifacts.

## Client reconnect sequence

```text
1. connect (local socket or remote TLS + token)
2. protocol handshake        → verify compatible version
3. list / status run         → recover authoritative run state
4. events(run_id)            → rebuild the timeline
5. resume streaming          → continue from the last observed event
```

## Failure and recovery behaviour

| Situation | Required behaviour |
|-----------|--------------------|
| Daemon restarted | Reconnect and replay from the store; no data loss |
| Socket closed mid-stream | Reconnect, re-read `status`, replay, then continue |
| Unknown run ID | Explicit `ok:false` result — never a silent empty timeline |
| Replay beyond the cap | Client shows the bounded window and points at durable evidence |
| Version mismatch | Fail closed; do not guess the protocol |

## Operator diagnostics

- `prumo agent logs <run>` — run record and event tail.
- `prumo agent ps` — live runs and their state.
- `prumo agent events <run> --json` — machine-readable replay for incident bundles.

## TUI state recovery

The TUI treats reconnect as a state transition, not a restart: it keeps
scrollback, marks the connection as `reconnecting`, replays the timeline,
reconciles the run panel, then returns to `current`. If the replay cannot be
obtained, the panel shows an explicit error state with a retry action — never a
frozen or silently stale view.

## Agent/client integration guidance

- Prefer `status` + `events` over custom side channels.
- Treat `id` as the idempotency key when folding replayed events into local state.
- Do not persist derived UI aggregates as canonical; they are reconstructible.

## Evidence

```bash
go test ./internal/harness/daemon/ -run 'Replay|Reconnect'
```

Covers stable replay across clients, durability across a server restart,
idempotent repeated reads and state serialisability.
