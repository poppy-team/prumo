# Vendored freya-code-editor 0.4.3

Source: https://github.com/marc2332/freya (`freya-code-editor` 0.4.3, MIT).

Copied verbatim from the crates.io registry except for the PRUMO PATCH block
at the end of `impl CodeEditorData` in `src/editor_data.rs`, which only ADDS
public cursor/scroll accessors (`cursor_position`, `select_range`,
`place_cursor`, `scroll_offset`, `set_scroll_offset`, `reveal_line`).
No original behavior was changed.

Used through `[patch.crates-io]` in `prumo-viewer/Cargo.toml` because
upstream 0.4.3 exposes no cursor or scroll control, which the Prumo workspace
viewer needs for find-match reveal and search-result navigation.

If upstream Freya gains an equivalent public API, drop this vendor directory
and remove the patch section.
