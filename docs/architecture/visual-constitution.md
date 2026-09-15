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

## Screen/state evidence

Each interactive surface provides a golden terminal frame per applicable state
class (default, focus, empty, loading, error, offline/reconnecting,
permission-requested/denied, narrow width, no-color). Evidence follows the
`media-record` schema and is freshness-checked (W20).
