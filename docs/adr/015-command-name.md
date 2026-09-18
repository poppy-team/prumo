# ADR 015 — The command is `prumo-agent`

Status: Accepted (2026-09-17)
Supersedes: the `prumo` command name used by every document written before this
ADR.

## Context

The framework and its command shared one word. `prumo` named the protocol, the
repository, the workspace directory (`.prumo/`), the environment variables
(`PRUMO_*`), the module (`github.com/raillen/prumo`) **and** the binary a person
types. That last one is the only name a user repeats daily, and it was the one
carrying no information: "prumo" says which project, never what the program does.

The consequence was visible in the tool's own surface. `prumo ui` reads as "the
framework's ui", `prumo docs verify` reads as "the framework's docs"; a reader
cannot tell the product from its parts. And the terminal client was a second,
unrelated name (`prumo-tui`) that a user had to know separately.

## Decision

**The command is `prumo-agent`.**

- The framework, the protocol, the workspace directory, the environment variables
  and the module path stay **Prumo**. Nothing in the canonical format changed.
- The terminal client is reached through the same command: **`prumo-agent tui`**.
  Its executable is `prumo-agent-tui`, which `prumo-agent tui` resolves as a
  sibling, then on `PATH`, then through `PRUMO_TUI_BIN` — the same discovery the
  client already used for the daemon, now symmetric.
- The client resolves the daemon as `prumo-agent` (sibling, `PATH`, `PRUMO_BIN`).
- Documents written before this ADR that say `prumo <command>` mean
  `prumo-agent <command>`; they are corrected on touch, not rewritten in bulk.

A single binary would have been possible — the client is a separate module by
ADR 013, so `prumo-agent tui` executes it rather than importing it. That is the
same structural boundary the client already honours in the other direction.

## Consequences

- One name to remember, and it says what the program is.
- `cmd/prumo` became `cmd/prumo-agent`; the CI builds `/tmp/prumo-agent`.
- The rename touched 75 documents and 11 source files. It did **not** touch
  `.prumo/`, `prumo.json`, `PRUMO_*`, `src/prumo/resources/**` or the module
  path — a rename that reached those would have been a format change wearing a
  product change's clothes.
- The documentation gates caught the drift the rename caused (an evidence
  artifact pointing at `cmd/prumo/help.go`), which is the behaviour they exist
  for: a claim about a file that moved is a claim that is no longer true.
