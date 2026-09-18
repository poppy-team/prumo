# Interaction Specification — prumo-agent tui

> Authority: repository-canonical. Contracts: `ui.interaction`,
> `tui.interaction` in `docs/contracts/builtin.json`. State vocabulary:
> `docs/ui-ux/state-matrix.json`. Surface inventory:
> `docs/ui-ux/interface-map.json`. Principles and tokens:
> `docs/architecture/visual-constitution.md`.
>
> This describes the client that ships (`prumo-agent tui`, ADR 013). The archived
> from-scratch client keeps its own copy at `archive/tui/interaction.md`.
>
> The keyboard map below is **checked against the client's own bindings** by
> `TestDocumentedKeymapMatchesTheBindings`: a chord the client adds without a row
> here, or a row here for a chord the client does not bind, fails that test. A
> specification that drifts from the code it governs is a claim nobody verifies.

Navigation is **composer-first**: the transcript and the composer are where a run
is read and driven, and every dialog is one chord away from anywhere. A user
never has to know a dialog exists to act.

## Input methods

| Method | Status | Notes |
|--------|--------|-------|
| Terminal keyboard | Required | The only method that must be complete, and the only one the client implements. |
| Bracketed paste | Required | Pasted text is inserted literally; it never triggers keybindings. |
| Mouse | Not used | The client maps no mouse events at all. Every action has a keyboard route, so mouse optionality is a property of this client rather than a promise about a future one. |
| Focus events (terminal) | Not used | Focus is the client's own model — which surface holds the keyboard — and a terminal window losing focus does not change it. |

## Keyboard map

Two chords are reserved and may never be rebound by a surface: `Ctrl+C` (cancel
the active operation, never exit silently) and `Ctrl+Q` (quit). Every other
binding is listed here; a surface that adds a binding adds a row.

| Chord | Action | Scope |
|-------|--------|-------|
| `Ctrl+K` | Open the command menu | App shell |
| `Ctrl+S` | List the runs the daemon knows, to re-attach to one | App shell |
| `Ctrl+O` | Ask the harness which models the provider serves | App shell |
| `Ctrl+T` | Switch the palette | App shell |
| `Ctrl+F` | Browse the workspace's files; picking one writes its path into the goal | App shell |
| `Ctrl+L` | Show the client's own log | App shell |
| `Ctrl+G` | Show the files this run changed | App shell |
| `Ctrl+B` | Toggle the sidebar panel | App shell |
| `Ctrl+N` | Start a new session | Composer |
| `Ctrl+C` | Cancel the run in flight; with nothing running, ask before quitting | Any (reserved) |
| `Ctrl+Q` | Ask before quitting | Any (reserved) |
| `Esc` | Dismiss the topmost layer; cancel the run when the page has it | Any |
| `Enter` | Send the goal — with a trailing `\` it inserts a newline instead — or accept the selected row | Composer, lists |
| `Space` | Accept the selected action without `Enter` | Approval, quit prompt |
| `Up` / `Down` or `j` / `k` | Move the selection within the focused list | Lists |
| `Left` / `Right` / `Tab` / `Shift+Tab` | Move the selection between the actions of a dialog | Approval, quit prompt, first-run offer |
| `@` | Complete a file or folder path into the goal | Composer |
| `/` | Complete a slash command or run a shortcut | Composer |
| `Ctrl+E` | Open the goal in the external editor | Composer |
| `PgUp` / `b` | Scroll the transcript up a page | Transcript |
| `PgDn` / `f` | Scroll the transcript down a page | Transcript |
| `Ctrl+U` / `Ctrl+D` | Scroll the transcript half a page | Transcript |
| `h` / `Backspace` | Go up a directory | Files |
| `l` | Enter the selected directory | Files |
| `i` | Show or hide hidden files | Files |
| `Ctrl+_` or `Ctrl+H` | Show the keymap for the current surface | Any |
| `?` | Toggle the keymap | Any |
| `q` | Leave the log page | Logs |
| `a` / `s` / `d` | Approve / approve for the session / deny the pending permission | Approval |
| `y` / `n` | Confirm or decline leaving | Quit prompt |

`Ctrl+C` never ends a session silently: with a run in flight it asks the daemon
to stop it, and with nothing running it opens the confirmation the quit chord
opens. A chord that answered with silence would read as a broken key.

## Focus semantics

- Exactly one surface holds the keyboard: the composer, or the dialog drawn over
  it. A dialog takes it away from the page and gives it back when it closes.
- Focus is **never hidden**. The composer's border is heavy (`━`) while it holds
  the keyboard and light (`─`) while a layer is over it, and it also changes
  colour with the `focus` token. The glyph is what carries the distinction,
  because a terminal without colour is a supported terminal.
- Focus is **never trapped**: `Esc` yields the topmost layer, and every dialog
  closes back to the page that opened it.
- The shell decides focus, not the surface: a dialog opening is what moves it,
  and the composer is told rather than left to guess.

## Selection semantics

- Selection is a property of a **list**, not of the screen: at most one row is
  selected per list — sessions, commands, models, themes, files, completions —
  and moving between lists preserves each list's selection.
- Moving the selection never has a side effect. Acts happen on `Enter` or on the
  action's own chord, never on cursor movement.
- The transcript is a stream, not a list: it scrolls without a selection, and its
  rows are never editable.
- The changed-files panel is a list over the run's own record: it is filled when
  it opens, from what the harness reported, so it cannot disagree with the
  statusline count.

## Terminal size classes

| Class | Columns | Behaviour |
|-------|---------|-----------|
| Compact | `< 80` | The transcript and the composer are the whole page; the statusline drops its optional segments — the accounting first, then the change count — rather than running past the screen; a row that loses its tail ends with a truncation marker. |
| Standard | `80–119` | The same page, with room for the statusline's accounting. |
| Wide | `≥ 120` | The same page, with room for the accounting, the change count and the message beside the model. |

**No horizontal scrolling, at any width.** The frame is fitted to the terminal it
was drawn for: content reflows or truncates with a visible marker, and a row that
silently loses its tail is a defect. Width is computed conservatively for Unicode
and CJK, so alignment never corrupts, and the client re-lays out on resize rather
than assuming a fixed size.

## Screen readers

A terminal screen reader reads what the terminal receives. That one fact decides
the strategy: an interface that repaints a whole frame per keystroke hands a
reader a wall of repeated text, so the client offers a mode that never repaints —
and the interactive mode keeps every fact it shows reachable as words.

**The linear mode.** `--prompt "<goal>"` runs the goal and prints what happened
instead of drawing it, in one of two encodings:

| Invocation | What comes out |
|---|---|
| `--prompt … --plain` | prose, in order: the answer as it grows, then what the run spent, then how it ended |
| `--prompt … --json` | the same facts, one JSON object per line, for a program |

Announcements are **deltas, not repaints**: an answer is printed once as it
grows, never re-read because it arrived in pieces, and every fact carries a
`kind` so a reader can filter it. A permission gate in this mode is refused out
loud with its reason — there is nobody to answer it, and a run that waits forever
is worse than a run that says why it stopped.

**The interactive mode** keeps the same guarantees for a reader who is watching a
terminal: nothing is announced only by colour, motion is never the only signal
(`--reduced-motion`), the statusline states the gate, the failure and the spend
in words, and the frame is never the only place a fact exists — everything on it
comes from the same record a `--plain` run prints.

**Name, role, value.** A terminal has no accessibility tree, so the client
declares the one it can: `docs/ui-ux/interface-map.json` gives every element a
label (the name a reader hears), a kind (its role) and the states it can be in
(its value), and `prumo-agent ui verify` refuses an element without them. That map is
the landmark structure as well: a header, a body, a footer, and the overlays that
take the keyboard.

**What is owed.** The strategy above is implemented and tested; what it has not
had is a session with a screen reader actually attached. That attestation is a
human act, it is tracked in `docs/harness/gap-register.md`, and nothing here
claims it has happened.

## Error copy

Every failure the client shows a user has the same shape, and the shape is the
policy:

> **what failed** — the error itself — **what to do next**

Three rules follow from it.

- **The operation is named, not implied.** "Starting the run", "Re-attaching to
  the run", "Listing the runs" — a reader who sees only the statusline still
  knows what the client was doing.
- **The action is available.** The next step named is one the reader can take
  from where they are (press the key again, read the log with `Ctrl+L`, pick
  another session). "Contact support" is not a next action.
- **The error text is carried unchanged.** The client does not rewrite the
  harness's account of what went wrong: it surrounds it. A client that
  paraphrased a failure would be editing the record of it.

The failures the client raises itself go through `util.ReportFailure`, which
enforces the shape; the harness's own message arrives on the run timeline and is
shown verbatim, wrapped by the same three rules. Long messages are truncated by
the statusline with the marker described above, never silently.

## Motion policy

Animation is used for exactly two things, and neither of them carries
information on its own:

| What moves | What it means | Under reduced motion |
|---|---|---|
| The marker beside the run's verb (`Thinking…`, `Generating…`, …) | that a run is in flight | the marker holds still; the verb is unchanged |
| The composer's caret | where the next character lands | the caret keeps its shape and colour and stops blinking |

Reduced motion is the user's setting, taken from `--reduced-motion` or
`PRUMO_REDUCED_MOTION` (unset, `0`, `false` and `no` all mean off). It is not a
terminal capability: a terminal reports what it can draw, not what its user
wants to watch.

The rule the two rows share: **motion is never the only signal.** Every state
that animates also says what it is in words, so a user who asked for no
animation — or whose terminal draws a still frame — loses the movement and
nothing else. `TestReducedMotionStatesProgressWithoutAnimatingIt` asserts that
the frames with and without animation differ in that one line and in no other.

## Mouse policy

Mouse events are not mapped. The client's input methods are the keyboard and
bracketed paste, and neither the interface map nor the keymap above declares a
pointer input — so no action depends on a gesture a keyboard cannot express.
Adding one would be a defect rather than a limitation, because it would break the
only completeness claim this specification makes.
