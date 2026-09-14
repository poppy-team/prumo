# Harness daemon (local)

Package `internal/harness/daemon`. A local Unix-socket server hosting
headless runs: `start/status/list/events/cancel/protocol` as JSON lines.

- Run records (`daemon-run-<id>.json`) and the JSONL timeline
  (`events-<id>.jsonl`) persist under the store dir, so clients can
  disconnect, the daemon can restart, and runs stay observable
  (reconnect baseline; remote transport is future work).
- Cancellation is cooperative at state-machine safe points; cancelled runs
  record `cancelled`, permission waits record `yielded`.
- CLI: `prumo agent serve --path . [--socket ...]` blocks until SIGINT/
SIGTERM; `prumo agent ps` lists runs; `prumo agent logs --run <id>`
replays the timeline. `serve` uses the Coding ACI workspace; providers
resolve via `model.ForName` (fake default, real adapters need keys/URLs).
- Steering: `op steer` (CLI `agent steer`, SDK `Steer`, ACP `Prompt`)
  injects follow-up input into live runs (refused when terminal).
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
