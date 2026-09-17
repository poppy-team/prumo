# Harness daemon (local + remote)

Package `internal/harness/daemon`. A daemon hosting headless runs:
`start/status/list/events/cancel/steer/approve/deny/schedule/unschedule/jobs/models/protocol`
as JSON lines over a Unix socket, or over TCP+TLS+token when `--listen` is set
(`remote.go`; the same dispatch serves both transports).

- Run records (`daemon-run-<id>.json`) and the JSONL timeline
  (`events-<id>.jsonl`) persist under the store dir, so clients can
  disconnect, the daemon can restart, and runs stay observable
  (reconnect baseline).
- Cancellation is cooperative at state-machine safe points; cancelled runs
  record `cancelled`. A run that parks records `yielded`; a run waiting for a
  client decision records **`awaiting_approval`** and names the request ids in
  `pending_permissions`. The two are deliberately distinct: conflating them
  would make a client either wait forever on a parked run or leave the panel on
  one that needs an answer.
- CLI: `prumo agent serve --path . [--socket ...] [--permission allow|ask|deny]
  [--ask-kind <kinds>]` blocks until SIGINT/SIGTERM; `prumo agent ps` lists runs
  (and the request ids a run is waiting on); `prumo agent logs --run <id>`
  replays the timeline. `serve` uses the Coding ACI workspace; providers
  resolve via `model.ForName` (fake/fake-tools default, real adapters need
  keys/URLs).
- Steering: `op steer` (CLI `agent steer`, SDK `Steer`, ACP `Prompt`)
  injects follow-up input into live runs (refused when terminal).
- Model discovery: `op models` (CLI `agent models`, SDK `Models`) asks a provider
  what it can serve, through the harness. The client does not carry a
  catalogue: which models exist is a property of the provider, and only the
  process talking to it can answer. A provider that cannot enumerate (Anthropic
  exposes no listing) returns what it was configured with rather than an empty
  list.
- Approvals: `op approve` / `op deny` (CLI `agent approve|deny`, SDK
  `Approve`/`Deny`, TUI `a`/`d`) answer the permission request a run stopped
  on. The decision is recorded in the permission trail and the turn continues;
  a denial fails the run without executing the tool. A run waiting for
  approval stays in the daemon's live map — a run a client cannot reach is a
  run a client cannot answer — and is dropped once it reaches a terminal state.
- Scheduling: `schedule/unschedule/jobs` ops (CLI + SDK) persist cron-like
  jobs; the serve loop fires due jobs once each (no catch-up storms).
  `prumo agent schedule --goal ... --every 3600`. Start failures back off
  linearly and dead-letter after MaxRetries (default 3), keeping last status.
- Tests: lifecycle (start→complete→events→list→protocol), cancel of a
  blocking run, and reconnect (new server, same store).
- IDL: `schemas/protocol-manifest.json` (version, ops, args, schemas,
  errors) is served by the `protocol` op and `prumo agent protocol
  --manifest`. `protocol.Manifest()` is the code truth; the checked-in file
  must match it (manifest_test.go) and every listed op must have a dispatch
  branch (daemon dispatch test).
- Public SDK: `sdk/prumo` (stdlib only, typed Start/Status/List/Events/
  Cancel/Protocol/Wait) is the client surface for prumo-code and third
  parties. `TestBoundaryNoInternalImports` fails the build if the SDK ever
  imports `prumo/internal`; `TestSDKRoundtrip` pins it against a live daemon.
