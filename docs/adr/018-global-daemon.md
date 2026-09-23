# ADR 018 — One global Prumo daemon supervises every registered project

Status: Accepted (2026-09-23)
Relates to: GAP-104, GAP-105, GAP-122, GAP-136, GAP-137, GAP-140, GAP-164

## Context

Prumo is installed as a per-user binary (`scripts/install.sh:5-7`,
`scripts/install.ps1:5-9`) with a global state area under `PRUMO_HOME`
(`internal/install/install.go:27-44`), but everything that does work is
project-scoped:

- the daemon is a foreground process per project, with its socket, PID lock,
  schedule and event store under `<project>/.prumo/runtime/harness`
  (`cmd/prumo/agent_commands.go:44-46`, `511-568`);
- skills are cached globally then **copied into every project**
  (`internal/workforcesync/sync.go:174-215`);
- connector artifacts are project-local, and the install manifest records one
  connector status per ID with no project root
  (`internal/install/install.go:11-17`, `41-44`) — so a second project's install
  overwrites the first's cleanup record;
- credentials come from an environment variable or a CLI flag, per process;
- cost and telemetry are per run, with no cross-project rollup;
- there is no project identity, no user identity and no installation identity.

The practical consequence: a user with 50 projects gets up to 50 daemons,
50 PID locks, 50 copies of the same skills, 50 inherited copies of every API key
in their environment, and a cleanup record that only remembers the last project.

`agent serve` also cannot safely host more than one project today. It builds one
tool executor rooted at the daemon's `--path` (`cmd/prumo/agent_commands.go:521-536`),
and a start request's own `workspace` field never rebuilds or re-roots that
executor (`internal/harness/daemon/daemon.go:360-386`).

The comparable tools converge on a global supervisor:

- **multica** installs globally, runs `multica daemon start` in the background
  with `daemon.log`/`daemon.pid`/`daemon.err.log` in a state directory, detects
  installed agent CLIs, and registers a runtime per watched workspace.
- **Agent Orchestrator** keeps a durable project registry in SQLite
  (`ao project add --path --id`, `ao project ls`) with typed, independent
  per-project configuration, and reports fleet health and cost across projects.
- **Caret** exposes `caret daemon start|stop|status`, `--foreground`,
  `supervisor install --global`, and cross-session `health`, `cost --by agent`
  and `diff`.

## Decision

1. **One daemon per user, installed as a user service.**
   `prumo daemon install` writes a systemd user unit (Linux), a launchd
   LaunchAgent (macOS) or a Windows Service. `prumo daemon autostart` enables
   start-on-login. The daemon listens on a single socket inside
   `PRUMO_HOME`, never on a network interface unless explicitly configured.

2. **Projects are registered, not discovered.**
   `prumo project add --path <dir> [--id <id>]` writes a row to a SQLite
   registry under `~/.local/share/prumo/registry.db`. `prumo project ls` lists
   them. A project is opted in; Prumo never scans a filesystem looking for work.

3. **A run is bound to exactly one project, and tools are rooted per run.**
   The daemon creates the tool executor from the run's resolved workspace, not
   from a single daemon-wide executor. This is the precondition for hosting many
   projects safely and is also the fix for the current bug where `SetWorkspace`
   is applied before the request's workspace is parsed
   (`daemon.go:342-347` vs `360-365`).

4. **State follows XDG.**
   Config in `~/.config/prumo`, mutable state and the registry in
   `~/.local/share/prumo`, content-addressed cache in
   `~/.local/share/prumo/cache`, logs in `~/.local/share/prumo/logs`.
   `PRUMO_HOME` remains the single override for all four.

5. **Credentials are references, resolved at use.**
   `PRUMO_MODEL_API_KEY` and `--api-key` become credential *references* into an
   OS keychain (or an operator-provided file when no keychain exists). A
   reference travels with a run; the secret is injected only into the provider
   that needs it. Child processes receive an explicit allowlisted environment
   rather than inheriting the parent wholesale.

6. **Identity is explicit.**
   Each installation has a stable `installation_id`; each registered project a
   `project_uuid`; the user a `user_id`. Default run IDs derive from these, so
   `R-daemon-1` restarting at 1 and colliding with a prior run
   (`daemon.go:84-88`, `134-141`) cannot happen.

7. **Cost and telemetry roll up.**
   `prumo cost --by project --by agent --by model` aggregates across projects.
   Project-level budget policy still applies per project; the rollup is
   reporting, not a shared pool, so one project cannot spend another's budget.

8. **The per-project daemon is removed, not deprecated.**
   `prumo agent serve` becomes a thin client of the global daemon. Two daemons on
   one machine is a bug, not a feature, and the non-atomic PID lock
   (`daemon/supervise.go:18-35`) exists only because more than one was possible.

## Consequences

- Installing Prumo once gives every project the same skills, the same workforce
  and the same daemon, instead of duplicating them per project. The global
  skill cache stops being a copy source and becomes the install target.
- Because the socket is no longer a per-project boundary, it must be
  authenticated. The local socket gains peer-credential checks and, for
  non-local access, a token, before this ADR is considered done.
- The registry is the thing that makes uninstall correct: `RemoveManagedPaths`
  becomes per project, so uninstalling a connector in project B no longer
  deletes project A's files.
- Cross-platform parity requires replacing the Unix-socket default and Unix
  signals with a platform-appropriate transport and service control, which is
  already inconsistent today (`daemon.go:194-203`, `supervise.go:20-35`).
- Global cost visibility is a real capability that a per-project daemon cannot
  provide, and it is what makes provider choice at the fleet level meaningful.
