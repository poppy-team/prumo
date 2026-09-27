# Interaction Specification — the from-scratch H10 client

> ARCHIVED. Interaction specification of the from-scratch terminal client, kept
> beside the artifact it describes: the client was archived to `archive/tui/`
> and its strategy replaced by ADR 013. The canonical specification for the
> shipping client is `docs/ui-ux/interaction.md`.

> Authority: repository-canonical (H10). Contracts: `ui.interaction`,
> `tui.interaction` in `docs/contracts/builtin.json`. State vocabulary:
> `docs/ui-ux/state-matrix.json`. Principles and tokens:
> `docs/architecture/visual-constitution.md`.

Navigation is **palette-first**: the command palette is the entry point and the
primary way to reach every action. Panes are a view of state, not a menu, so a
user never has to know a pane exists to act.

## Input methods

| Method | Status | Notes |
|--------|--------|-------|
| Terminal keyboard | Required | The only method that must be complete. |
| Bracketed paste | Required | Pasted text is inserted literally; it never triggers keybindings. |
| Mouse | Optional acceleration | Never the sole route to an action; may be entirely absent. |
| Focus events (terminal) | Optional | When the terminal reports focus loss, the client stops consuming redraw budget; it never changes run state. |

## Keyboard map

Two chords are reserved and may never be rebound by a surface: `Ctrl+C`
(cancel the active operation, never exit silently) and `Ctrl+Q` (quit). Every
other binding is listed here; a surface that adds a binding adds a row.

| Chord | Action | Scope |
|-------|--------|-------|
| `Ctrl+P` | Open the command palette | App shell |
| `Esc` | Dismiss the topmost layer; return focus one level toward the palette | Any |
| `Enter` | Accept the selected palette action or run the prompt | Palette, run panel |
| `Tab` / `Shift+Tab` | Move focus to the next / previous pane | App shell |
| `Up` / `Down` | Move the selection within the focused list | Palette, lists |
| `PgUp` / `PgDn` | Scroll the timeline by a page without moving focus | Timeline |
| `Ctrl+T` | Toggle the file tree (read-only) | App shell |
| `Ctrl+E` | Open the focused file in the external editor | File tree |
| `Ctrl+R` | Re-run the last goal with the same route | Run panel |
| `a` / `d` | Approve / deny the pending permission (default is deny) | Approval |
| `?` | Show the keymap for the current focus | Any |

**Mouse optionality is a test, not a claim.** Every action above is reachable
with the keyboard alone, and the spike's acceptance run uses no mouse input. A
new action that can only be clicked is a defect, not a limitation.

## Focus semantics

- Exactly one pane holds focus. The focused pane draws its border with the
  `focus` token — contrast over boxes, never a filled background.
- Focus is **never hidden**: the indicator survives a no-color terminal because
  it also changes the border glyph (`┃` focused, `│` unfocused).
- Focus is **never trapped**: `Esc` always yields the topmost layer, and the
  palette is always reachable in one chord from any focus position.
- Focus returns to the invoking surface when a layer closes, so a dismissed
  prompt never strands the selection in a dead pane.
- After an error or a closed modal, focus returns to the element the user was
  operating on, not to the start of the pane.

## Selection semantics

- Selection is a property of a **list**, not of the screen: at most one row is
  selected per list, and moving focus between lists preserves each list's
  selection independently.
- Moving the selection never has a side effect. Acts happen on `Enter` or on the
  action's own chord, never on cursor movement.
- The timeline is a stream, not a list: it scrolls without a selection, and
  copying is explicit rather than selection-based.
- Selection is cleared when its owning list is refilled, so a stale index can
  never point at a different row.

## Terminal size classes

| Class | Columns | Behaviour |
|-------|---------|-----------|
| Compact | `< 80` | File tree collapses to a toggle; the run timeline is the only pane; long lines truncate with an explicit marker. |
| Standard | `80–119` | Two panes: palette/run plus the timeline; the file tree is a toggled overlay. |
| Wide | `≥ 120` | Three panes: file tree, run, timeline. |

**No horizontal scrolling.** Content reflows or truncates with a visible
truncation marker; a row that silently loses its tail is a defect. Width is
computed conservatively for Unicode and CJK, so alignment never corrupts; the
client re-lays out on terminal resize rather than assuming a fixed size.

## Mouse policy

Mouse events, when the terminal reports them, are mapped only to actions that
already have a keyboard route. Scrolling the timeline with the wheel and
clicking a palette row to select it are the complete set. No drag interaction
exists in the spike, so no action depends on a gesture a keyboard cannot
express.
