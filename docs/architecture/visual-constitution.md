# Visual Constitution — Prumo Code TUI

> Authority: repository-canonical (W9.5). Token set:
> `docs/ui-ux/design-tokens.json`; schema: `schemas/design-token-set.schema.json`.

The TUI is a terminal client of the Shared Product Contract
(`docs/architecture/shared-product-contract.md`). It renders contract entities
and never owns domain state.

## Principles

1. **Agent-first, information-dense.** The run panel is the primary surface;
   decoration is subordinate to state legibility.
2. **Monochrome-by-default.** Color carries meaning, never brand. Every
   semantic token has a no-color fallback.
3. **Keyboard-only complete.** Every action reachable by keybinding; mouse is
   optional acceleration, never a requirement.
4. **Capability-degrading.** Missing Unicode, color or true-color terminals
   degrade gracefully; the interface must remain usable at minimal capability.
5. **State is explicit.** A user can always tell whether a run is idle,
   streaming, awaiting approval, blocked or done.

## Token discipline

- Three tiers only: `primitive → semantic → component`.
- Components consume **semantic** tokens, never primitives directly.
- Token renames are API changes: they require a migration note and impact on
  components, themes, visual evidence and user-customization docs (W9.4).
- Contrast is evidence, not intent: semantic text/focus tokens declare a
  measured WCAG ratio.

## Required semantic tokens

`surface`, `panel`, `border-subtle`, `text`, `text-muted`, `accent`,
`success`, `warning`, `error`, `focus` — shared with the Desktop track.

## TUI component tokens

`tui.panel.border`, `tui.statusline`, `tui.command-palette.highlight`,
`tui.approval.prompt`.

## Terminal capability matrix

| Capability | Degradation |
|------------|-------------|
| True color | 256-color, then 16-color, then none |
| Unicode box drawing | ASCII `+ - |` fallback |
| Mouse | keyboard-only equivalents remain complete |
| Unicode width/CJK | conservative width calculation; never corrupt alignment |

## Terminal screen-reader strategy

A terminal exposes one flat stream rather than a widget tree, so there is no
name/role/value tree to annotate. The strategy is therefore structural, and it
is what keeps the interface usable without sight:

1. **Speech-friendly output first.** Every state is expressed in text before it
   is expressed in color or glyph, so a screen reader reading the frame gets the
   same information a sighted user gets from the `accent`/`error` tokens.
2. **No color-only meaning.** A screen reader cannot report a color, so any
   distinction carried by color is also carried by a word or a glyph (see
   *Monochrome-by-default*): the state is spelled out, not tinted.
3. **Announcements are deltas, not redraws.** The timeline appends events and
   never re-renders the whole frame on each event, so a reader is not forced to
   re-read the screen to find what changed.
4. **Cursor position is never the only signal.** Focus is indicated by the
   border glyph as well as the cursor, so a reader that does not track the
   cursor still knows which pane is active.
5. **Screen-reader mode is a declared terminal capability.** When it is active
   the client suppresses transient motion and the spinner, and states progress
   in text at a bounded interval rather than on every tick.

## Screen/state evidence

Each interactive surface provides a golden terminal frame per applicable state
class (default, focus, empty, loading, error, offline/reconnecting,
permission-requested/denied, narrow width, no-color). Evidence follows the
`media-record` schema and is freshness-checked (W20).

**Visual evidence set.** One golden frame per *applicable* state in
`docs/ui-ux/state-matrix.json`, for each declared terminal width class. A state
marked not applicable carries no frame and states its reason in the matrix
instead, so the evidence set is derived from the matrix rather than maintained
beside it.

**Capture environment.** A frame is captured as the rendered frame buffer — the
exact character cells and their styling — not as a screenshot. The record names
the terminal size, the token theme (default, high-contrast, no-color), the
Unicode capability and the width calculation used, because a frame is only
reproducible in the environment that produced it. Screenshots are not accepted:
they cannot be compared byte-for-byte.

**Regression policy.** A frame is compared byte-for-byte against its record. A
difference fails until it is either fixed or the record is regenerated in the
same change that alters the surface, so a visual change and its evidence are
never split across commits. A regenerated frame whose diff was not reviewed is
the failure mode this policy exists to prevent.

**Media freshness.** Frames are linked to the surface they illustrate and go
stale by content digest (W20): changing a surface without regenerating its
frame reports `affected-by-ui-change`, and `prumo docs release` refuses to ship
while evidence is stale.
