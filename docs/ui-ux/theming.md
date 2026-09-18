# Theming — prumo-agent tui

> Authority: repository-canonical. Contracts: `ui.theming`,
> `accessibility.contrast`. Token set: `docs/ui-ux/design-tokens.json`.
> Surfaces and states: `docs/ui-ux/interface-map.json`,
> `docs/ui-ux/state-matrix.json`. Interaction, including motion:
> `docs/ui-ux/interaction.md`.

A theme is a **palette bound to a fixed contract**. It may choose colours; it may
not choose which things have colours. That is what makes a theme switchable
without reviewing the interface again, and what makes the contrast evidence mean
something: the same pairs are measured in every theme.

## What a theme must define

The interface reads its colours through `theme.Theme`. A theme supplies every
group below — the compiler refuses one that does not — and inherits the rest from
`BaseTheme`:

| Group | Covers | Why it is in the contract |
|---|---|---|
| Base | primary, secondary, accent | The three colours that carry emphasis and selection |
| Status | error, warning, success, info | A state is never only a colour, but it is always at least a colour |
| Text | text, text-muted, text-emphasized | The three weights of prose the interface draws |
| Background | background, background-secondary, background-darker | The surfaces everything else is measured against |
| Border | border-normal, border-focused, border-dim | Focus is a border glyph **and** a colour |
| Diff | added, removed, context, hunk header, highlights, line numbers | A patch is read row by row, so each row kind needs its own pair |
| Markdown | prose, headings, links, code, quotes, lists, images | The transcript renders markdown, and a code block is not prose |
| Syntax | comment, keyword, function, variable, string, number, type, operator, punctuation | Code inside a run's output is read, not merely displayed |

## Token inheritance

- **Light and dark are one token, resolved once.** Every colour is an
  `AdaptiveColor` holding a value per background mode; the client asks its
  terminal which mode it is in and resolves at read time. A theme that only
  supplies one mode is a theme that drew for the wrong terminal.
- **Semantic names, never primitives.** Components ask for `text-muted`, not for
  `gray.800`. Renaming a semantic token is a change to this contract; adding a
  primitive is not.
- **The client keeps no colour of its own.** Every pixel comes from the active
  theme, which is why the no-color and high-contrast rows below are settings of
  the same system rather than special cases in the renderer.

## Which themes ship, and how one is chosen

Nine palettes ship: `prumo` (the default), `catppuccin`, `dracula`, `flexoki`,
`gruvbox`, `monokai`, `onedark`, `tokyonight` and `tron`. `ctrl+t` lists them and
switches live; the choice is applied where the interface is built
(`tui.New`) and written to the client's own configuration
(`$XDG_CONFIG_HOME/prumo-agent tui/config.json`), so a restart keeps it. `--theme`
overrides it for one run. A name the client does not know is reported and the
theme in use is kept: a typo must not leave the terminal unstyled.

## Contrast

Every theme is **measured**, not trusted. `TestEveryThemeMeasuresItsContrast`
computes the WCAG contrast ratio of each token against the surface it is drawn
on and fails below the floor — 4.5:1 for text, 3:1 for the lines and shapes that
carry information — and `TestContrastEvidenceIsReported` prints every ratio for
every theme, which is the evidence this contract asks for.

Where a theme's upstream palette fell below the floor, the token was carried
toward **that theme's own foreground** until it reached it: the same colour,
dimmer, rather than a new one. The rule is written next to each adjusted token,
and the tests fail if it drifts back.

**One pair is exempt, with its reason**: the *unfocused* border measures below
3:1 in every palette, and it does not need to reach it — it separates rather than
identifies. The focused/unfocused distinction is the border **glyph** (heavy
against light, asserted by `TestTheComposerShowsWhetherItHoldsTheKeyboard`), and
the state it marks is carried by the focused border, which reaches the floor in
every theme. The exemption is declared in the test, not implied by silence.

## High contrast and no colour

- **No colour.** A terminal without colour support draws the same interface:
  every distinction is carried by text or glyph, the statusline drops segments
  rather than overflowing, and the ASCII glyph set replaces the Unicode one when
  the locale cannot render it (`styles.ResolveIcons`).
- **High contrast** is the `focus` token's border plus the floors above, applied
  in every theme rather than offered as a separate palette: a palette that needs
  a second palette to be readable is not readable.
- **Reduced motion** is a separate setting and is documented in
  `docs/ui-ux/interaction.md` §Motion policy: it removes the movement, never the
  information.

## Personalization

One preference exists, and it is the palette.

| Question | Answer |
|---|---|
| Customizable surface | The active theme, and `--reduced-motion` (documented in `docs/ui-ux/interaction.md` §Motion policy). Nothing else is on that surface. |
| Defaults | `prumo`, and motion on. A client that has never been configured behaves the same as one configured to the defaults. |
| Persistence location | `$XDG_CONFIG_HOME/prumo-agent tui/config.json`, the client's own file. `PRUMO_TUI_CONFIG_DIR` moves it, which is what lets a test point at a directory it owns instead of the developer's home. |
| Import and export | Not offered. The file holds one short value, its format is the client's own, and a client that imported another's preferences would be accepting settings it cannot verify. |
| Invalid value behavior | Reported, and the theme in use is kept: a typo in `--theme` must not leave the terminal unstyled. A settings file that cannot be parsed is reported rather than silently ignored, because a preference the user set and never had applied is worse than an error. |
| Token overrides | Not offered by the client. The token set (`docs/ui-ux/design-tokens.json`) is the framework's, and a client that overrode tokens would be editing the contract it is measured against. |
| What is **not** on the surface | Where the workspace is, which provider to ask, which model to run, how to reach the daemon. Those are decisions of the invocation, and everything about a run belongs to the harness. A client-side preference that changed any of them would make the client a second source of truth about a run. |
| Platform limitations | A terminal without colour support draws the same interface, and a locale that cannot render the Unicode set gets the ASCII one. Nothing on the surface is platform-conditional: the client adapts to the terminal, it does not ask the user to. |
