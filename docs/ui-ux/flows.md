# Flows — prumo-agent tui

> Authority: repository-canonical. Contracts: `ui.product-ux`,
> `ui.screen-inventory`. Surfaces: `docs/ui-ux/interface-map.json`. Interaction
> and keymap: `docs/ui-ux/interaction.md`. States:
> `docs/ui-ux/state-matrix.json`.

A **flow** is what a person is trying to get done, from where they start to the
point they are done. It is not a screen listing: the same screen serves several
flows, and a flow that exists only as somebody's habit is a flow nobody can
review. Each one below names its goal, the route through the client, and what
happens when it fails — a flow without a failure path is a flow that strands
people.

## The goals

| Goal | The person wants |
|---|---|
| G1 | to have work done in this workspace |
| G2 | to know what the run is doing while it does it |
| G3 | to answer a run that stopped for a decision |
| G4 | to come back to a run that is already going |
| G5 | to know what a run cost and what it touched |
| G6 | to take the account of a run somewhere else |

## The flows

### F1 — Start a run (G1)

`prumo-agent tui` → the composer holds the keyboard → the goal is typed → `Enter`.

The daemon is supervised on launch, so the client is attached before the first
keyword is typed; the goal becomes a run and the transcript starts growing.
**Failure:** the run cannot start (no daemon, a refused goal) → the statusline
names the operation and the next action (`-` `util.ReportFailure`), and the
composer keeps what was typed.

### F2 — Follow a run (G2)

The transcript draws the answer as it arrives, with a verb beside a marker naming
what the run is doing. `Ctrl+L` shows the client's own log when something is not
where it should be.
**Failure:** the link to the daemon is lost → the statusline says `reconnecting
k/8`, and only after the attempts run out does it say `offline` and stop the run.

### F3 — Answer a decision (G3)

A run yields on a permission gate: the approval dialog names the tool and its
target, and the statusline says the same thing in words. `a` allows, `s` allows
for the session, `d` denies.
**Failure:** the answer does not reach the daemon → the failure is named and the
gate can be answered again from the run's own state.

### F4 — Re-attach to a run (G4)

`Ctrl+S` lists the runs the daemon knows; selecting one re-reads that run's log —
the transcript and the spend are rebuilt from the record rather than remembered,
and a run still in flight keeps streaming from where it stopped.
**Failure:** the run is not known to the daemon → an error, never an empty
conversation pretending the session produced nothing.

### F5 — Read what a run cost and touched (G5)

The statusline reports what the session spent and how many files changed; `Ctrl+G`
opens the panel naming **which** files, in the order the harness reported them.
**Failure:** nothing was reported → the panel says "no file has changed yet"
rather than showing an empty box, and the accounting shows what was reported
rather than a ratio the client cannot verify.

### F6 — Take the account elsewhere (G6)

Two routes: the palette's *Export the session's timeline* writes the run's own
record under `.prumo/runtime/exports/` as plain text, and `--prompt "<goal>"`
runs headless, printing prose (`--plain`) or one JSON object per fact (`--json`).
**Failure:** the export cannot be written → the failure names the operation and
the path is not claimed.

### F7 — Add to a run in flight (G1)

Typing while a run is going does not start a second one: it **steers** the run
that is running, which is the only way to add to a conversation the daemon is
already having. What was said joins the transcript.
**Failure:** there is no run in flight → the text starts a new run instead, and
the failure of *that* is reported in the same shape as F1.

## The command and interaction map

- **Keys** are the table in `docs/ui-ux/interaction.md`, checked against the
  client's own bindings in both directions.
- **Commands** are the palette (`Ctrl+K`), which lists only actions that can run
  and says what each one does.
- **Surfaces** are `docs/ui-ux/interface-map.json`: one shell, the pages, the
  statusline and the overlays, each with the inputs that operate it.

A new action enters through one of those three, and a flow that needs an action
in none of them is a flow the client cannot express.
