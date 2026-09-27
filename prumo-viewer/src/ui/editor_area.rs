use crate::extensions::host::ExtensionHost;
use crate::services::diff::open_document_diff;
use crate::services::document::{
    line_character_to_utf16_position, resolve_active_conflict, save_active_tab,
    utf16_position_to_line_character,
};
use crate::services::editor_intelligence::{
    apply_text_edits, code_action_edit, completion_text_edit, diagnostic_to_lsp,
    lsp_range_from_utf16,
};
use crate::services::search::find_in_text;
use crate::state::{
    AppState, ConflictResolution, EditorCursorState, EditorPopupItem, EditorPopupKind,
    EditorReveal, FileKind, NoticeTone,
};
use crate::theme;
use crate::ui::chrome::IconButton;
use crate::ui::editor_popup::EditorPopup;
use crate::ui::icons::icon;
use freya::code_editor::{CodeEditor, CodeEditorData, EditorLanguage, LineDecoration, Rope};
use freya::prelude::*;
use serde_json::Value;
use std::collections::HashMap;
use std::path::{Path, PathBuf};
use tree_sitter_go;
use tree_sitter_javascript;
use tree_sitter_json;
use tree_sitter_md;
use tree_sitter_rust;
use tree_sitter_toml_ng;
use tree_sitter_typescript;
use tree_sitter_yaml;

const EDITOR_FONT_SIZE: f32 = 13.5;
const EDITOR_FONT_FAMILY: &str = "Jetbrains Mono";

#[derive(PartialEq)]
pub struct EditorArea {
    pub state: State<AppState>,
    pub cursor_state: State<EditorCursorState>,
    pub extension_host: ExtensionHost,
}

impl Component for EditorArea {
    fn render(&self) -> impl IntoElement {
        let current_theme = get_theme_or_default();
        let colors = current_theme.read().colors.clone();
        let mut state = self.state;
        let cursor_state = self.cursor_state;
        let mut editor_size_state = state;
        let show_whitespace = state.read().config.show_whitespace;
        let font_size = if state.read().config.font_size >= 8.0 {
            state.read().config.font_size
        } else {
            EDITOR_FONT_SIZE
        };
        let line_numbers = state.read().config.line_numbers;
        let active_tab = state.read().active_tab().cloned();
        let accessibility_id = use_a11y();
        let mut last_path = use_state(|| None);
        let mut editor = use_state(|| editor_data("", None, current_theme.read().name == "light"));
        let mut editor_cache = use_state(HashMap::<PathBuf, CodeEditorData>::new);
        let mut find_query = use_state(String::new);
        let find_replacement = use_state(String::new);
        let find_case = use_state(|| false);
        let find_index = use_state(|| 0usize);

        let current_path = active_tab.as_ref().map(|tab| tab.path.clone());
        let previous_path = last_path.read().clone();
        if previous_path != current_path {
            let is_light = current_theme.read().name == "light";
            if let Some(previous_path) = previous_path {
                let previous_content = editor.read().rope.to_string();
                {
                    let mut app_state = state.write();
                    app_state.close_editor_popup();
                    if let Some(previous_tab) = app_state
                        .tabs
                        .iter_mut()
                        .find(|tab| tab.path == previous_path)
                    {
                        if previous_tab.content != previous_content {
                            previous_tab.revision = previous_tab.revision.saturating_add(1);
                        }
                        previous_tab.is_dirty = previous_content != previous_tab.persisted_content;
                        previous_tab.content = previous_content;
                    }
                }
                let cached_editor =
                    std::mem::replace(&mut *editor.write(), editor_data("", None, is_light));
                editor_cache.write().insert(previous_path, cached_editor);
            }
            let next_editor = active_tab
                .as_ref()
                .map(|tab| {
                    let mut cache = editor_cache.write();
                    restore_editor(&mut cache, &tab.path, &tab.content, tab.kind, is_light)
                })
                .unwrap_or_else(|| editor_data("", None, is_light));
            *editor.write() = next_editor;
            let open_paths = state
                .read()
                .tabs
                .iter()
                .map(|tab| tab.path.clone())
                .collect::<Vec<_>>();
            editor_cache
                .write()
                .retain(|path, _| open_paths.iter().any(|open_path| open_path == path));
            last_path.set(current_path.clone());
        }
        if let Some(reveal) = take_pending_reveal(state, current_path.as_deref()) {
            if reveal.start == 0 && reveal.end == 0 && reveal.line > 0 {
                let line = reveal.line.min(editor.read().rope.len_lines());
                let content_lines = editor.read().rope.len_lines();
                editor.write().reveal_line(
                    line,
                    EDITOR_FONT_SIZE * 1.55,
                    state.read().editor_viewport_height,
                    content_lines,
                );
            } else {
                reveal_match(
                    editor,
                    reveal.start,
                    reveal.end,
                    state.read().editor_viewport_height,
                );
            }
        }

        let mut sync_state = state;
        let mut sync_editor = editor;
        let mut sync_cursor = cursor_state;
        use_side_effect(move || {
            let (is_edited, cursor_pos, selection) = {
                let editor_state = sync_editor.read();
                (
                    editor_state.is_edited(),
                    editor_state.cursor_position(),
                    editor_state.selection_range(),
                )
            };

            if is_edited {
                let content = sync_editor.read().rope.to_string();
                if sync_state
                    .peek()
                    .active_tab()
                    .is_some_and(|tab| tab.content != content)
                {
                    let mut app_state = sync_state.write();
                    app_state.agent_line_decorations.clear();
                    app_state.update_active_content(content.clone());
                }

                let should_mark_saved = sync_state
                    .peek()
                    .active_tab()
                    .is_some_and(|tab| !tab.is_dirty && tab.content == content);
                if should_mark_saved {
                    sync_editor.write().mark_as_saved();
                }
            }

            let current_cursor = *sync_cursor.peek();
            if editor_cursor_needs_sync(
                current_cursor.position,
                current_cursor.selection,
                cursor_pos,
                selection,
            ) {
                let mut cursor = sync_cursor.write();
                cursor.position = cursor_pos;
                cursor.selection = selection;
            }
            if sync_state.peek().editor_popup.is_some()
                && (sync_state.peek().active_cursor_position != cursor_pos
                    || sync_state.peek().active_selection_range != selection)
            {
                let mut app_state = sync_state.write();
                app_state.active_cursor_position = cursor_pos;
                app_state.active_selection_range = selection;
            }
        });

        let find_open = state.read().find_open;
        let find_query_value = find_query.read().clone();
        let find_case_on = *find_case.read();
        let find_open_state = state;
        let find_query_state = find_query;
        let find_case_state = find_case;
        let find_matches_memo = use_memo(move || {
            let open = find_open_state.read().find_open;
            let query = find_query_state.read().clone();
            let match_case = *find_case_state.read();
            if !should_search_document(open, &query) {
                return Vec::new();
            }
            let find_text = editor.read().rope.to_string();
            find_in_text(&find_text, &query, match_case)
        });
        let find_matches = find_matches_memo.read().clone();
        let find_count = find_matches.len();
        let find_current = (*find_index.read()).min(find_count.saturating_sub(1));
        let find_preview = find_matches
            .get(find_current)
            .map(|item| format!("{}: {}", item.line, item.preview))
            .unwrap_or_default();
        let editor_viewport_height = state.read().editor_viewport_height;

        use_side_effect(move || {
            let syntax = if get_theme_or_default().read().name == "light" {
                theme::light_syntax()
            } else {
                theme::dark_syntax()
            };
            editor.write().set_theme(syntax);
        });
        use_side_effect(move || {
            let mut decorations = state.read().editor_line_decorations.clone();
            decorations.extend(state.read().git_line_decorations.clone());
            decorations.extend(state.read().agent_line_decorations.clone());
            let active_path = state.read().active_tab().map(|tab| tab.path.clone());
            if let Some(active_path) = active_path {
                for diagnostic in state
                    .read()
                    .diagnostics
                    .iter()
                    .filter(|diagnostic| diagnostic.path == active_path)
                {
                    decorations.push(LineDecoration {
                        line: diagnostic.line.saturating_sub(1),
                        color: match diagnostic.severity {
                            crate::state::DiagnosticSeverity::Error => {
                                Color::from_argb(42, 220, 80, 80)
                            }
                            crate::state::DiagnosticSeverity::Warning => {
                                Color::from_argb(42, 220, 170, 60)
                            }
                            crate::state::DiagnosticSeverity::Information => {
                                Color::from_argb(32, 80, 150, 220)
                            }
                            crate::state::DiagnosticSeverity::Hint => {
                                Color::from_argb(28, 150, 110, 200)
                            }
                        },
                    });
                }
            }
            editor.write().set_line_decorations(decorations);
        });

        let tab = state.read().active_tab().cloned();
        if let Some(tab) = tab {
            let is_agent_modified = tab.is_agent_modified;
            let is_git_preview = !state.read().git_line_decorations.is_empty();
            let has_conflict = tab.conflict.is_some();
            let breadcrumb = breadcrumb(&tab.rel_path);
            let popup_state = state;
            let popup_editor = editor;
            let popup_host = self.extension_host.clone();

            rect()
                .vertical()
                .width(Size::fill())
                .height(Size::fill())
                .content(Content::flex())
                .background(colors.background)
                .on_global_pointer_press(move |_| {
                    state.write().explorer_focused = false;
                })
                .child(
                    rect()
                        .width(Size::fill())
                        .height(Size::px(28.))
                        .horizontal()
                        .cross_align(Alignment::center())
                        .padding(Gaps::new(0., 8., 0., 8.))
                        .background(colors.background)
                        .border(Border::new().fill(colors.border).width(BorderWidth {
                            top: 0.,
                            right: 0.,
                            bottom: 1.,
                            left: 0.,
                        }))
                        .child(
                            label()
                                .color(colors.text_secondary)
                                .font_size(11.)
                                .max_lines(1)
                                .text(breadcrumb),
                        )
                        .child(
                            rect()
                                .expanded()
                                .horizontal()
                                .main_align(Alignment::end())
                                .spacing(7.)
                                .child(if is_git_preview {
                                    Element::from(
                                        label()
                                            .color(colors.success)
                                            .font_size(10.5)
                                            .font_weight(FontWeight::MEDIUM)
                                            .text("Git changes"),
                                    )
                                } else {
                                    Element::from(label().text(""))
                                })
                                .child(if is_agent_modified {
                                    Element::from(
                                        label()
                                            .color(colors.warning)
                                            .font_size(10.5)
                                            .font_weight(FontWeight::MEDIUM)
                                            .text("Changed by agent"),
                                    )
                                } else {
                                    Element::from(label().text(""))
                                })
                                .child(IconButton {
                                    icon: "save",
                                    label: "Save active file",
                                    size: 12.,
                                    on_press: (move |_: Event<PressEventData>| {
                                        let content = editor.read().rope.to_string();
                                        let mut app_state = state.write();
                                        app_state.update_active_content(content);
                                        if save_active_tab(&mut app_state).is_ok() {
                                            editor.write().mark_as_saved();
                                        }
                                    })
                                    .into(),
                                })
                                .child(
                                    Button::new()
                                        .flat()
                                        .compact()
                                        .height(Size::px(22.))
                                        .padding(Gaps::new(0., 6., 0., 6.))
                                        .on_press(move |_| {
                                            let content = editor.read().rope.to_string();
                                            let mut app_state = state.write();
                                            app_state.update_active_content(content);
                                            open_document_diff(&mut app_state);
                                        })
                                        .child(
                                            label()
                                                .color(colors.text_secondary)
                                                .font_size(10.5)
                                                .text("Diff"),
                                        ),
                                ),
                        ),
                )
                .child(if has_conflict {
                    let conflict_state = state;
                    let conflict_editor = editor;
                    let conflict_kind = tab.kind;
                    let conflict_light = current_theme.read().name == "light";
                    Element::from(
                        rect()
                            .width(Size::fill())
                            .height(Size::px(30.))
                            .horizontal()
                            .cross_align(Alignment::center())
                            .padding(Gaps::new(0., 8., 0., 8.))
                            .background(colors.surface_secondary)
                            .border(Border::new().fill(colors.warning).width(BorderWidth {
                                top: 0.,
                                right: 0.,
                                bottom: 1.,
                                left: 0.,
                            }))
                            .child(
                                label()
                                    .color(colors.warning)
                                    .font_size(10.5)
                                    .text("File changed on disk"),
                            )
                            .child(
                                Button::new()
                                    .flat()
                                    .compact()
                                    .width(Size::px(70.))
                                    .height(Size::px(22.))
                                    .padding(Gaps::new(0., 4., 0., 4.))
                                    .on_press(move |_| {
                                        apply_conflict_resolution(
                                            conflict_state,
                                            conflict_editor,
                                            ConflictResolution::Reload,
                                            conflict_kind,
                                            conflict_light,
                                        );
                                    })
                                    .child(
                                        label()
                                            .color(colors.text_secondary)
                                            .font_size(10.)
                                            .text("Reload"),
                                    ),
                            )
                            .child(
                                Button::new()
                                    .flat()
                                    .compact()
                                    .width(Size::px(92.))
                                    .height(Size::px(22.))
                                    .padding(Gaps::new(0., 4., 0., 4.))
                                    .on_press({
                                        let conflict_state = state;
                                        let conflict_editor = editor;
                                        move |_| {
                                            apply_conflict_resolution(
                                                conflict_state,
                                                conflict_editor,
                                                ConflictResolution::KeepMine,
                                                conflict_kind,
                                                conflict_light,
                                            );
                                        }
                                    })
                                    .child(
                                        label()
                                            .color(colors.text_primary)
                                            .font_size(10.)
                                            .text("Keep mine"),
                                    ),
                            )
                            .child(
                                Button::new()
                                    .flat()
                                    .compact()
                                    .width(Size::px(110.))
                                    .height(Size::px(22.))
                                    .padding(Gaps::new(0., 4., 0., 4.))
                                    .on_press({
                                        let conflict_state = state;
                                        let conflict_editor = editor;
                                        move |_| {
                                            apply_conflict_resolution(
                                                conflict_state,
                                                conflict_editor,
                                                ConflictResolution::ThreeWayMerge,
                                                conflict_kind,
                                                conflict_light,
                                            );
                                        }
                                    })
                                    .child(
                                        label()
                                            .color(colors.primary)
                                            .font_size(10.)
                                            .font_weight(FontWeight::SEMI_BOLD)
                                            .text("⚡ 3-Way Merge"),
                                    ),
                            ),
                    )
                } else {
                    Element::from(rect())
                })
                .child(if find_open {
                    Element::from({
                        rect()
                            .width(Size::fill())
                            .vertical()
                            .background(colors.surface_primary)
                            .border(Border::new().fill(colors.border).width(BorderWidth {
                                top: 0.,
                                right: 0.,
                                bottom: 1.,
                                left: 0.,
                            }))
                            .child(
                                rect()
                                    .width(Size::fill())
                                    .height(Size::px(32.))
                                    .horizontal()
                                    .content(Content::flex())
                                    .cross_align(Alignment::center())
                                    .spacing(4.)
                                    .padding(Gaps::new(0., 6., 0., 6.))
                                    .child(
                                        Input::new(find_query.into_writable())
                                            .placeholder("Find")
                                            .width(Size::flex(1.))
                                            .auto_focus(true)
                                            .on_submit({
                                                let editor_for_find = editor;
                                                let case_for_find = find_case;
                                                let mut index_for_find = find_index;
                                                move |value: String| {
                                                    find_query.set(value.clone());
                                                    index_for_find.set(0);
                                                    reveal_find_match(
                                                        editor_for_find,
                                                        &value,
                                                        *case_for_find.read(),
                                                        0,
                                                        editor_viewport_height,
                                                    );
                                                }
                                            }),
                                    )
                                    .child(
                                        Button::new()
                                            .flat()
                                            .compact()
                                            .height(Size::px(22.))
                                            .padding(Gaps::new(0., 6., 0., 6.))
                                            .background(if find_case_on {
                                                colors.surface_tertiary
                                            } else {
                                                Color::TRANSPARENT
                                            })
                                            .on_press({
                                                let editor_for_find = editor;
                                                let mut case_for_find = find_case;
                                                let query = find_query_value.clone();
                                                move |_| {
                                                    let next_case = !find_case_on;
                                                    case_for_find.set(next_case);
                                                    reveal_find_match(
                                                        editor_for_find,
                                                        &query,
                                                        next_case,
                                                        find_current,
                                                        editor_viewport_height,
                                                    );
                                                }
                                            })
                                            .child(
                                                label()
                                                    .color(colors.text_secondary)
                                                    .font_size(10.)
                                                    .text("Aa"),
                                            ),
                                    )
                                    .child(
                                        label().color(colors.text_placeholder).font_size(10.).text(
                                            if find_count == 0 {
                                                "0/0".to_string()
                                            } else {
                                                format!("{}/{}", find_current + 1, find_count)
                                            },
                                        ),
                                    )
                                    .child(IconButton {
                                        icon: "chevron_up",
                                        label: "Previous match",
                                        size: 10.,
                                        on_press: {
                                            let editor_for_find = editor;
                                            let mut index_for_find = find_index;
                                            let query = find_query_value.clone();
                                            move |_: Event<PressEventData>| {
                                                if find_count > 0 {
                                                    let next_index = find_current
                                                        .saturating_sub(1)
                                                        .min(find_count - 1);
                                                    index_for_find.set(next_index);
                                                    reveal_find_match(
                                                        editor_for_find,
                                                        &query,
                                                        find_case_on,
                                                        next_index,
                                                        editor_viewport_height,
                                                    );
                                                }
                                            }
                                        }
                                        .into(),
                                    })
                                    .child(IconButton {
                                        icon: "chevron_down",
                                        label: "Next match",
                                        size: 10.,
                                        on_press: {
                                            let editor_for_find = editor;
                                            let mut index_for_find = find_index;
                                            let query = find_query_value.clone();
                                            move |_: Event<PressEventData>| {
                                                if find_count > 0 {
                                                    let next_index =
                                                        (find_current + 1) % find_count;
                                                    index_for_find.set(next_index);
                                                    reveal_find_match(
                                                        editor_for_find,
                                                        &query,
                                                        find_case_on,
                                                        next_index,
                                                        editor_viewport_height,
                                                    );
                                                }
                                            }
                                        }
                                        .into(),
                                    })
                                    .child(IconButton {
                                        icon: "x",
                                        label: "Close find",
                                        size: 10.,
                                        on_press: (move |_: Event<PressEventData>| {
                                            state.write().find_open = false;
                                        })
                                        .into(),
                                    }),
                            )
                            .child(
                                rect()
                                    .width(Size::fill())
                                    .height(Size::px(32.))
                                    .horizontal()
                                    .content(Content::flex())
                                    .cross_align(Alignment::center())
                                    .spacing(4.)
                                    .padding(Gaps::new(0., 6., 0., 6.))
                                    .child(
                                        Input::new(find_replacement.into_writable())
                                            .placeholder("Replace")
                                            .width(Size::px(130.)),
                                    )
                                    .child(
                                        Button::new()
                                            .compact()
                                            .height(Size::px(22.))
                                            .padding(Gaps::new(0., 8., 0., 8.))
                                            .on_press({
                                                let query = find_query.read().clone();
                                                let replacement = find_replacement.read().clone();
                                                move |_| {
                                                    replace_find_match(
                                                        state,
                                                        editor,
                                                        FindReplaceJob {
                                                            query: query.clone(),
                                                            replacement: replacement.clone(),
                                                            match_case: find_case_on,
                                                            only_index: Some(find_current),
                                                        },
                                                        find_index,
                                                    );
                                                }
                                            })
                                            .child(
                                                label()
                                                    .color(colors.text_primary)
                                                    .font_size(10.5)
                                                    .text("Replace"),
                                            ),
                                    )
                                    .child(
                                        Button::new()
                                            .compact()
                                            .height(Size::px(22.))
                                            .padding(Gaps::new(0., 8., 0., 8.))
                                            .on_press({
                                                let query = find_query.read().clone();
                                                let replacement = find_replacement.read().clone();
                                                move |_| {
                                                    replace_find_match(
                                                        state,
                                                        editor,
                                                        FindReplaceJob {
                                                            query: query.clone(),
                                                            replacement: replacement.clone(),
                                                            match_case: find_case_on,
                                                            only_index: None,
                                                        },
                                                        find_index,
                                                    );
                                                }
                                            })
                                            .child(
                                                label()
                                                    .color(colors.text_primary)
                                                    .font_size(10.5)
                                                    .text("All"),
                                            ),
                                    )
                                    .child(
                                        label()
                                            .color(colors.text_placeholder)
                                            .font_size(10.)
                                            .max_lines(1)
                                            .text(find_preview),
                                    ),
                            )
                    })
                } else {
                    Element::from(rect())
                })
                .child(
                    rect()
                        .width(Size::fill())
                        .height(Size::flex(1.))
                        .content(Content::flex())
                        .on_sized(move |event: Event<SizedEventData>| {
                            let height = event.visible_area.size.height;
                            if (editor_size_state.read().editor_viewport_height - height).abs()
                                > 0.5
                            {
                                editor_size_state.write().editor_viewport_height = height;
                            }
                        })
                        .child(
                            CodeEditor::new(editor, accessibility_id)
                                .font_size(font_size)
                                .line_height(1.55)
                                .gutter(line_numbers)
                                .viewport_height(editor_viewport_height)
                                .show_whitespace(show_whitespace)
                                .on_pre_key_down({
                                    let state = popup_state;
                                    let editor = popup_editor;
                                    let extension_host = popup_host.clone();
                                    move |event| {
                                        handle_editor_key(
                                            state,
                                            editor,
                                            extension_host.clone(),
                                            event,
                                        )
                                    }
                                }),
                        )
                        .child(EditorPopup {
                            state: popup_state,
                            editor: popup_editor,
                        }),
                )
        } else {
            rect()
                .vertical()
                .width(Size::fill())
                .height(Size::fill())
                .main_align(Alignment::center())
                .cross_align(Alignment::center())
                .background(colors.background)
                .spacing(10.)
                .child(
                    SvgViewer::new(("files", icon("files")))
                        .color(colors.text_placeholder)
                        .width(Size::px(28.))
                        .height(Size::px(28.)),
                )
                .child(
                    label()
                        .color(colors.text_primary)
                        .font_size(14.)
                        .font_weight(FontWeight::MEDIUM)
                        .text("Open a file to start"),
                )
                .child(
                    label()
                        .color(colors.text_secondary)
                        .font_size(12.)
                        .text("Use the Explorer or Quick Open."),
                )
                .child(
                    rect()
                        .vertical()
                        .spacing(4.)
                        .cross_align(Alignment::center())
                        .child(
                            label()
                                .color(colors.text_placeholder)
                                .font_size(11.5)
                                .text("Ctrl+P  Open file"),
                        )
                        .child(
                            label()
                                .color(colors.text_placeholder)
                                .font_size(11.5)
                                .text("Ctrl+S  Save active file"),
                        )
                        .child(
                            label()
                                .color(colors.text_placeholder)
                                .font_size(11.5)
                                .text("Ctrl+W  Close tab"),
                        ),
                )
        }
    }
}

fn apply_conflict_resolution(
    mut state: State<AppState>,
    mut editor: State<CodeEditorData>,
    resolution: ConflictResolution,
    kind: FileKind,
    is_light: bool,
) {
    let result = {
        let mut app_state = state.write();
        resolve_active_conflict(&mut app_state, resolution)
    };
    if let Err(error) = result {
        state.write().show_notice(NoticeTone::Error, error);
        return;
    }
    if resolution == ConflictResolution::Reload || resolution == ConflictResolution::ThreeWayMerge {
        let content = state
            .read()
            .active_tab()
            .map(|tab| tab.content.clone())
            .unwrap_or_default();
        *editor.write() = editor_data(&content, editor_language(kind), is_light);
    }
    let message = match resolution {
        ConflictResolution::Reload => "Reloaded the file from disk",
        ConflictResolution::KeepMine => "Kept your buffer; save to apply it",
        ConflictResolution::ThreeWayMerge => "Merged edits using 3-way merge",
    };
    state.write().show_notice(NoticeTone::Info, message);
}

fn request_completion(
    mut state: State<AppState>,
    editor: State<CodeEditorData>,
    extension_host: ExtensionHost,
) {
    let content = editor.read().rope.to_string();
    state.write().update_active_content(content.clone());
    let Some((path, revision)) = state
        .read()
        .active_tab()
        .map(|tab| (tab.path.clone(), tab.revision))
    else {
        state.write().show_notice(
            NoticeTone::Error,
            "Open a file before requesting completions",
        );
        return;
    };
    let cursor = editor.read().cursor_position();
    let selection = editor.read().selection_range();
    let Some((extension_id, language)) = extension_host.active_lsp_for_path(&path) else {
        state
            .write()
            .show_notice(NoticeTone::Error, "No active language server for this file");
        return;
    };
    let (line, character) = utf16_position_to_line_character(&content, cursor);
    let request_id = state.write().begin_editor_popup(
        EditorPopupKind::Completion,
        path.clone(),
        revision,
        cursor,
        selection,
        Some((cursor, cursor)),
    );
    let host = extension_host.clone();
    let request_path = path.clone();
    spawn(async move {
        let result = crate::run_blocking(move || {
            host.sync_active_document(&request_path, &language, &content, revision)?;
            host.lsp_completion(
                &extension_id,
                &language,
                &request_path.to_string_lossy(),
                line,
                character,
            )
        })
        .await;
        let mut app_state = state.write();
        if !app_state.is_editor_popup_request_current(request_id, &path, revision) {
            return;
        }
        let Some(popup) = app_state.editor_popup.as_mut() else {
            return;
        };
        popup.loading = false;
        match result {
            Ok(Ok(response)) => {
                popup.items = response
                    .items()
                    .into_iter()
                    .filter(|item| item.insert_text_format != Some(2))
                    .take(100)
                    .map(EditorPopupItem::Completion)
                    .collect();
                if popup.items.is_empty() {
                    popup.error = Some("No completions returned".to_string());
                }
            }
            Ok(Err(error)) => popup.error = Some(error),
            Err(error) => popup.error = Some(format!("Completion worker failed: {error}")),
        }
    });
}

fn request_code_actions(
    mut state: State<AppState>,
    editor: State<CodeEditorData>,
    extension_host: ExtensionHost,
) {
    let content = editor.read().rope.to_string();
    state.write().update_active_content(content.clone());
    let Some((path, revision)) = state
        .read()
        .active_tab()
        .map(|tab| (tab.path.clone(), tab.revision))
    else {
        state.write().show_notice(
            NoticeTone::Error,
            "Open a file before requesting code actions",
        );
        return;
    };
    let cursor = editor.read().cursor_position();
    let selection = editor.read().selection_range();
    let range = selection
        .or_else(|| diagnostic_range_at_cursor(&state.read(), &path, cursor))
        .unwrap_or((cursor, cursor));
    let lsp_range = match lsp_range_from_utf16(&content, range) {
        Ok(range) => range,
        Err(error) => {
            state.write().show_notice(NoticeTone::Error, error);
            return;
        }
    };
    let diagnostics: Vec<Value> = state
        .read()
        .diagnostics
        .iter()
        .filter(|diagnostic| diagnostic.path == path)
        .map(diagnostic_to_lsp)
        .collect();
    let Some((extension_id, language)) = extension_host.active_lsp_for_path(&path) else {
        state
            .write()
            .show_notice(NoticeTone::Error, "No active language server for this file");
        return;
    };
    let request_id = state.write().begin_editor_popup(
        EditorPopupKind::CodeActions,
        path.clone(),
        revision,
        cursor,
        selection,
        Some(range),
    );
    let host = extension_host.clone();
    let request_path = path.clone();
    spawn(async move {
        let result = crate::run_blocking(move || {
            host.sync_active_document(&request_path, &language, &content, revision)?;
            host.lsp_code_actions(
                &extension_id,
                &language,
                &request_path.to_string_lossy(),
                lsp_range,
                diagnostics,
            )
        })
        .await;
        let mut app_state = state.write();
        if !app_state.is_editor_popup_request_current(request_id, &path, revision) {
            return;
        }
        let Some(popup) = app_state.editor_popup.as_mut() else {
            return;
        };
        popup.loading = false;
        match result {
            Ok(Ok(actions)) => {
                popup.items = actions
                    .into_iter()
                    .filter(|action| code_action_edit(action, &path).is_ok())
                    .take(100)
                    .map(EditorPopupItem::CodeAction)
                    .collect();
                if popup.items.is_empty() {
                    popup.error = Some("No code actions returned".to_string());
                }
            }
            Ok(Err(error)) => popup.error = Some(error),
            Err(error) => popup.error = Some(format!("Code action worker failed: {error}")),
        }
    });
}

fn request_formatting(
    mut state: State<AppState>,
    mut editor: State<CodeEditorData>,
    extension_host: ExtensionHost,
) {
    let content = editor.read().rope.to_string();
    state.write().update_active_content(content.clone());
    let Some((path, revision)) = state
        .read()
        .active_tab()
        .map(|tab| (tab.path.clone(), tab.revision))
    else {
        state
            .write()
            .show_notice(NoticeTone::Error, "Open a file before formatting");
        return;
    };
    let Some((extension_id, language)) = extension_host.active_lsp_for_path(&path) else {
        state
            .write()
            .show_notice(NoticeTone::Error, "No active language server for this file");
        return;
    };
    let host = extension_host.clone();
    let request_path = path.clone();
    let request_content = content.clone();
    spawn(async move {
        let result = crate::run_blocking(move || {
            host.sync_active_document(&request_path, &language, &request_content, revision)?;
            host.lsp_formatting(
                &extension_id,
                &language,
                &request_path.to_string_lossy(),
                4,
                true,
            )
        })
        .await;
        let is_current = state
            .read()
            .active_tab()
            .is_some_and(|tab| tab.path == path && tab.revision == revision);
        if !is_current {
            return;
        }
        match result {
            Ok(Ok(edits)) if edits.is_empty() => {
                state
                    .write()
                    .show_notice(NoticeTone::Info, "Document is already formatted");
            }
            Ok(Ok(edits)) => {
                let result = {
                    let mut editor_data = editor.write();
                    let result = apply_text_edits(&mut editor_data, &content, &edits);
                    if result.is_ok() {
                        editor_data.measure(EDITOR_FONT_SIZE, EDITOR_FONT_FAMILY);
                    }
                    result
                };
                match result {
                    Ok(()) => {
                        let next_content = editor.read().rope.to_string();
                        let mut app_state = state.write();
                        app_state.update_active_content(next_content);
                        app_state.show_notice(NoticeTone::Success, "Document formatted");
                    }
                    Err(error) => state.write().show_notice(NoticeTone::Error, error),
                }
            }
            Ok(Err(error)) => state.write().show_notice(NoticeTone::Error, error),
            Err(error) => state.write().show_notice(
                NoticeTone::Error,
                format!("Formatting worker failed: {error}"),
            ),
        }
    });
}

fn diagnostic_range_at_cursor(
    state: &AppState,
    path: &Path,
    cursor: usize,
) -> Option<(usize, usize)> {
    let content = &state.active_tab()?.content;
    let (line, character) = utf16_position_to_line_character(content, cursor);
    let diagnostic = state
        .diagnostics
        .iter()
        .filter(|diagnostic| diagnostic.path == path)
        .find(|diagnostic| {
            let end_line = diagnostic.end_line.unwrap_or(diagnostic.line);
            let start_column = diagnostic.column.saturating_sub(1);
            let end_column = diagnostic
                .end_column
                .unwrap_or(diagnostic.column)
                .saturating_sub(1);
            (line, character) >= (diagnostic.line, start_column)
                && (line, character) <= (end_line, end_column)
        })?;
    let start = line_character_to_utf16_position(
        content,
        diagnostic.line,
        diagnostic.column.saturating_sub(1),
    )?;
    let end = line_character_to_utf16_position(
        content,
        diagnostic.end_line.unwrap_or(diagnostic.line),
        diagnostic
            .end_column
            .unwrap_or(diagnostic.column)
            .saturating_sub(1),
    )?;
    Some((start, end))
}

fn handle_editor_key(
    mut state: State<AppState>,
    mut editor: State<CodeEditorData>,
    extension_host: ExtensionHost,
    event: Event<KeyboardEventData>,
) -> bool {
    let control = event.modifiers.contains(Modifiers::CONTROL);
    let shift = event.modifiers.contains(Modifiers::SHIFT);
    let alt = event.modifiers.contains(Modifiers::ALT);
    let popup_kind = state.read().editor_popup.as_ref().map(|popup| popup.kind);

    if control
        && !shift
        && !alt
        && matches!(&event.key, Key::Character(character) if character == "=" || character == "+")
    {
        let mut app_state = state.write();
        let new_size = (app_state.config.font_size + 1.0).min(36.0);
        app_state.config.font_size = new_size;
        let _ = app_state.config.save();
        event.stop_propagation();
        event.prevent_default();
        return false;
    }
    if control
        && !shift
        && !alt
        && matches!(&event.key, Key::Character(character) if character == "-")
    {
        let mut app_state = state.write();
        let new_size = (app_state.config.font_size - 1.0).max(8.0);
        app_state.config.font_size = new_size;
        let _ = app_state.config.save();
        event.stop_propagation();
        event.prevent_default();
        return false;
    }
    if control
        && !shift
        && !alt
        && matches!(&event.key, Key::Character(character) if character == "0")
    {
        let mut app_state = state.write();
        app_state.config.font_size = 13.0;
        let _ = app_state.config.save();
        event.stop_propagation();
        event.prevent_default();
        return false;
    }
    if !control && !alt && matches!(&event.key, Key::Named(NamedKey::Tab)) && popup_kind.is_none() {
        let tab_size = state.read().config.tab_size as usize;
        let spaces = " ".repeat(if tab_size == 0 { 4 } else { tab_size });
        let cursor_pos = editor.read().cursor_position();
        let selection = editor.read().selection_range();
        let (start, end) = selection.unwrap_or((cursor_pos, cursor_pos));
        let mut ed = editor.write();
        let start_byte = ed.rope.char_to_byte(ed.rope.utf16_cu_to_char(start));
        let end_byte = ed.rope.char_to_byte(ed.rope.utf16_cu_to_char(end));
        ed.replace_byte_range(start_byte, end_byte, &spaces);
        event.stop_propagation();
        event.prevent_default();
        return false;
    }
    if control
        && !shift
        && !alt
        && matches!(&event.key, Key::Character(character) if character == " ")
    {
        if popup_kind == Some(EditorPopupKind::Completion) {
            state.write().close_editor_popup();
        } else {
            request_completion(state, editor, extension_host);
        }
        event.stop_propagation();
        event.prevent_default();
        return false;
    }
    if control
        && !shift
        && !alt
        && matches!(&event.key, Key::Character(character) if character == ".")
    {
        if popup_kind == Some(EditorPopupKind::CodeActions) {
            state.write().close_editor_popup();
        } else {
            request_code_actions(state, editor, extension_host);
        }
        event.stop_propagation();
        event.prevent_default();
        return false;
    }
    if alt
        && shift
        && matches!(&event.key, Key::Character(character) if character.eq_ignore_ascii_case("f"))
    {
        state.write().close_editor_popup();
        request_formatting(state, editor, extension_host);
        event.stop_propagation();
        event.prevent_default();
        return false;
    }
    if popup_kind.is_some() {
        match &event.key {
            Key::Named(NamedKey::ArrowDown) => {
                let count = state
                    .read()
                    .editor_popup
                    .as_ref()
                    .map_or(0, |popup| popup.items.len());
                if count > 0 {
                    let next = state
                        .read()
                        .editor_popup
                        .as_ref()
                        .map(|popup| (popup.selected_index + 1).min(count - 1))
                        .unwrap_or(0);
                    if let Some(popup) = state.write().editor_popup.as_mut() {
                        popup.selected_index = next;
                    }
                }
                event.stop_propagation();
                event.prevent_default();
                return false;
            }
            Key::Named(NamedKey::ArrowUp) => {
                let count = state
                    .read()
                    .editor_popup
                    .as_ref()
                    .map_or(0, |popup| popup.items.len());
                if count > 0 {
                    let previous = state
                        .read()
                        .editor_popup
                        .as_ref()
                        .map(|popup| popup.selected_index.saturating_sub(1))
                        .unwrap_or(0);
                    if let Some(popup) = state.write().editor_popup.as_mut() {
                        popup.selected_index = previous;
                    }
                }
                event.stop_propagation();
                event.prevent_default();
                return false;
            }
            Key::Named(NamedKey::Enter | NamedKey::Tab) => {
                let item = state
                    .read()
                    .editor_popup
                    .as_ref()
                    .and_then(|popup| popup.items.get(popup.selected_index).cloned());
                if let Some(item) = item {
                    apply_editor_popup_item(state, editor, item);
                } else {
                    state.write().close_editor_popup();
                }
                event.stop_propagation();
                event.prevent_default();
                return false;
            }
            Key::Named(NamedKey::Escape) => {
                state.write().close_editor_popup();
                event.stop_propagation();
                event.prevent_default();
                return false;
            }
            _ => {
                state.write().close_editor_popup();
            }
        }
    }
    if control && crate::dispatch_global_key(state, &event.key, event.modifiers) {
        event.stop_propagation();
        event.prevent_default();
        return false;
    }
    event.stop_propagation();
    true
}

pub(crate) fn apply_editor_popup_item(
    mut state: State<AppState>,
    mut editor: State<CodeEditorData>,
    item: EditorPopupItem,
) {
    let Some(popup) = state.read().editor_popup.clone() else {
        return;
    };
    if !state
        .read()
        .is_editor_popup_request_current(popup.request_id, &popup.path, popup.revision)
    {
        return;
    }
    let content = editor.read().rope.to_string();
    let Some(path) = state.read().active_tab().map(|tab| tab.path.clone()) else {
        state.write().close_editor_popup();
        return;
    };
    let result = {
        let mut editor_data = editor.write();
        let result = match &item {
            EditorPopupItem::Completion(item) => {
                completion_text_edit(item, &content, editor_data.cursor_position())
                    .and_then(|edit| apply_text_edits(&mut editor_data, &content, &[edit]))
            }
            EditorPopupItem::CodeAction(item) => code_action_edit(item, &path)
                .and_then(|edits| apply_text_edits(&mut editor_data, &content, &edits)),
        };
        if result.is_ok() {
            editor_data.measure(EDITOR_FONT_SIZE, EDITOR_FONT_FAMILY);
        }
        result
    };
    let mut app_state = state.write();
    match result {
        Ok(()) => {
            let next_content = editor.read().rope.to_string();
            app_state.update_active_content(next_content);
            app_state.show_notice(NoticeTone::Success, "Editor action applied");
        }
        Err(error) => {
            app_state.close_editor_popup();
            app_state.show_notice(NoticeTone::Error, error);
        }
    }
}

fn take_pending_reveal(
    mut state: State<AppState>,
    current_path: Option<&Path>,
) -> Option<EditorReveal> {
    let matches_current_path = state
        .read()
        .pending_editor_reveal
        .as_ref()
        .is_some_and(|reveal| current_path.is_some_and(|path| reveal.path == path));
    if matches_current_path {
        state.write().pending_editor_reveal.take()
    } else {
        None
    }
}

fn reveal_match(mut editor: State<CodeEditorData>, start: usize, end: usize, viewport_height: f32) {
    editor
        .write()
        .reveal_byte_range(start, end, EDITOR_FONT_SIZE * 1.55, viewport_height);
}

fn reveal_find_match(
    editor: State<CodeEditorData>,
    query: &str,
    match_case: bool,
    index: usize,
    viewport_height: f32,
) {
    let text = editor.read().rope.to_string();
    let matches = find_in_text(&text, query, match_case);
    let index = index.min(matches.len().saturating_sub(1));
    if let Some(target) = matches.get(index) {
        reveal_match(editor, target.start, target.end, viewport_height);
    }
}

fn restore_editor(
    cache: &mut HashMap<PathBuf, CodeEditorData>,
    path: &Path,
    content: &str,
    kind: FileKind,
    is_light: bool,
) -> CodeEditorData {
    match cache.remove(path) {
        Some(cached_editor) if cached_editor.rope == content => cached_editor,
        _ => editor_data(content, editor_language(kind), is_light),
    }
}

fn editor_data(content: &str, language: Option<EditorLanguage>, is_light: bool) -> CodeEditorData {
    let mut data = CodeEditorData::new(Rope::from_str(content), language);
    data.set_theme(if is_light {
        theme::light_syntax()
    } else {
        theme::dark_syntax()
    });
    data.parse();
    data.measure(EDITOR_FONT_SIZE, EDITOR_FONT_FAMILY);
    data
}

fn editor_language(kind: FileKind) -> Option<EditorLanguage> {
    match kind {
        FileKind::Rust => Some(EditorLanguage::new(
            tree_sitter_rust::LANGUAGE,
            tree_sitter_rust::HIGHLIGHTS_QUERY,
        )),
        FileKind::Go => Some(EditorLanguage::new(
            tree_sitter_go::LANGUAGE,
            tree_sitter_go::HIGHLIGHTS_QUERY,
        )),
        FileKind::Json => Some(EditorLanguage::new(
            tree_sitter_json::LANGUAGE,
            tree_sitter_json::HIGHLIGHTS_QUERY,
        )),
        FileKind::Yaml => Some(EditorLanguage::new(
            tree_sitter_yaml::LANGUAGE,
            tree_sitter_yaml::HIGHLIGHTS_QUERY,
        )),
        FileKind::Markdown => Some(EditorLanguage::new(
            tree_sitter_md::LANGUAGE,
            tree_sitter_md::HIGHLIGHT_QUERY_BLOCK,
        )),
        FileKind::TypeScript => Some(EditorLanguage::new(
            tree_sitter_typescript::LANGUAGE_TYPESCRIPT,
            tree_sitter_typescript::HIGHLIGHTS_QUERY,
        )),
        FileKind::JavaScript => Some(EditorLanguage::new(
            tree_sitter_javascript::LANGUAGE,
            tree_sitter_javascript::HIGHLIGHT_QUERY,
        )),
        FileKind::Toml => Some(EditorLanguage::new(
            tree_sitter_toml_ng::LANGUAGE,
            tree_sitter_toml_ng::HIGHLIGHTS_QUERY,
        )),
        FileKind::Folder | FileKind::Manifest | FileKind::Other => None,
    }
}

struct FindReplaceJob {
    query: String,
    replacement: String,
    match_case: bool,
    only_index: Option<usize>,
}

fn replace_find_match(
    mut state: State<AppState>,
    mut editor: State<CodeEditorData>,
    job: FindReplaceJob,
    mut find_index: State<usize>,
) {
    let text = editor.read().rope.to_string();
    let matches = find_in_text(&text, &job.query, job.match_case);
    if matches.is_empty() {
        return;
    }
    let targets = match job.only_index {
        Some(index) => vec![matches[index.min(matches.len() - 1)].clone()],
        None => matches.iter().rev().cloned().collect(),
    };
    {
        let mut editor_data = editor.write();
        for target in targets {
            editor_data.replace_byte_range(target.start, target.end, &job.replacement);
        }
        editor_data.measure(EDITOR_FONT_SIZE, EDITOR_FONT_FAMILY);
    }
    let next = editor.read().rope.to_string();
    state.write().update_active_content(next);
    find_index.set(0);
}

fn breadcrumb(relative_path: &str) -> String {
    let segments: Vec<&str> = relative_path
        .split('/')
        .filter(|part| !part.is_empty())
        .collect();
    let visible = if segments.len() > 3 {
        let start = segments.len() - 3;
        &segments[start..]
    } else {
        &segments
    };
    let mut parts = vec!["workspace"];
    if segments.len() > 3 {
        parts.push("…");
    }
    parts.extend(visible.iter().copied());
    parts.join("  ›  ")
}

fn should_search_document(find_open: bool, query: &str) -> bool {
    find_open && !query.is_empty()
}

fn editor_cursor_needs_sync(
    active_cursor_position: usize,
    active_selection_range: Option<(usize, usize)>,
    editor_cursor_position: usize,
    editor_selection_range: Option<(usize, usize)>,
) -> bool {
    active_cursor_position != editor_cursor_position
        || active_selection_range != editor_selection_range
}

#[cfg(test)]
mod tests {
    use super::*;
    use freya::code_editor::LineDecoration;
    use freya::text_edit::TextEditor;

    #[test]
    fn editor_cursor_sync_is_skipped_when_snapshot_matches() {
        assert!(!editor_cursor_needs_sync(4, Some((1, 4)), 4, Some((1, 4))));
        assert!(editor_cursor_needs_sync(3, Some((1, 4)), 4, Some((1, 4))));
        assert!(editor_cursor_needs_sync(4, Some((1, 4)), 4, Some((2, 4))));
    }

    #[test]
    fn document_search_is_skipped_when_closed_or_empty() {
        assert!(!should_search_document(false, "needle"));
        assert!(!should_search_document(true, ""));
        assert!(should_search_document(true, "needle"));
    }

    #[test]
    fn configured_languages_load_tree_sitter_grammars() {
        let samples = [
            (FileKind::Rust, "fn main() {}\n"),
            (FileKind::Go, "package main\nfunc main() {}\n"),
            (FileKind::Json, "{\"ok\":true}\n"),
            (FileKind::Yaml, "key: value\n"),
            (FileKind::Markdown, "# Prumo\n"),
            (FileKind::Toml, "version = 1\n"),
            (FileKind::TypeScript, "const value: number = 1;\n"),
            (FileKind::JavaScript, "const value = 1;\n"),
        ];
        for (kind, sample) in samples {
            let language = editor_language(kind).expect("language must be configured");
            let mut editor = CodeEditorData::new(Rope::from_str(sample), Some(language));
            editor.parse();
            assert_eq!(editor.rope.to_string(), sample);
        }
    }

    #[test]
    fn editor_reveal_api_uses_utf16_positions_and_scrolls_to_line() {
        let mut editor = CodeEditorData::new(Rope::from_str("á\nneedle\nlast\n"), None);
        editor.select_range_bytes(3, 9);
        assert_eq!(editor.cursor_position(), 8);
        assert_eq!(editor.selection_range(), Some((2, 8)));
        editor.reveal_line(3, 20., 40., 3);
        assert_eq!(editor.scroll_offset(), (0, -20));
        editor.set_line_decorations(vec![LineDecoration {
            line: 2,
            color: Color::from_rgb(1, 2, 3),
        }]);
        assert_eq!(editor.line_decoration(2), Some(Color::from_rgb(1, 2, 3)));
        editor.set_selection_ranges(vec![(0, 1), (2, 8)]);
        assert_eq!(editor.selection_ranges(), vec![(0, 1), (2, 8)]);
        assert_eq!(editor.visible_selection_ranges(0), vec![(0, 1)]);
        assert_eq!(editor.visible_selection_ranges(1), vec![(0, 6)]);
    }

    #[test]
    fn restores_cached_editor_history_when_switching_tabs() {
        let mut cached = CodeEditorData::new(Rope::from_str("one"), None);
        TextEditor::insert(&mut cached, "!", 0);
        let mut cache = HashMap::from([(PathBuf::from("/tmp/one.rs"), cached)]);
        let mut restored = restore_editor(
            &mut cache,
            Path::new("/tmp/one.rs"),
            "!one",
            FileKind::Other,
            false,
        );
        assert!(restored.undo());
        assert_eq!(restored.rope.to_string(), "one");
    }

    #[test]
    fn diagnostic_at_cursor_uses_zero_based_columns() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().to_path_buf();
        let path = root.join("main.rs");
        std::fs::write(&path, "let value = 1;\n").unwrap();
        let mut state = AppState::new(root);
        crate::services::document::activate_path(&mut state, &path).unwrap();
        state.diagnostics = vec![crate::state::Diagnostic {
            path: path.clone(),
            line: 1,
            column: 5,
            end_line: Some(1),
            end_column: Some(10),
            message: "Example".to_string(),
            severity: crate::state::DiagnosticSeverity::Warning,
            source: "test".to_string(),
            code: None,
        }];
        assert_eq!(diagnostic_range_at_cursor(&state, &path, 4), Some((4, 9)));
    }

    #[test]
    fn replacement_preserves_prior_editor_history() {
        let mut editor = CodeEditorData::new(Rope::from_str("one needle"), None);
        TextEditor::insert(&mut editor, "!", 0);
        std::thread::sleep(std::time::Duration::from_millis(1100));
        editor.replace_byte_range(5, 11, "haystack");
        assert_eq!(editor.rope.to_string(), "!one haystack");
        assert!(editor.can_undo());
        assert!(editor.undo());
        assert_eq!(editor.rope.to_string(), "!one needle");
        assert!(editor.undo());
        assert_eq!(editor.rope.to_string(), "one needle");
        assert!(editor.redo());
        assert_eq!(editor.rope.to_string(), "!one needle");
    }

    #[test]
    fn cursor_navigation_keeps_line_inside_viewport() {
        let content = (0..20)
            .map(|line| format!("line {line}\n"))
            .collect::<String>();
        let mut editor = CodeEditorData::new(Rope::from_str(&content), None);
        let line_ten = editor.rope.line_to_char(10);
        editor.place_cursor(editor.rope.char_to_utf16_cu(line_ten));
        assert!(editor.ensure_cursor_visible(20., 100., 20));
        assert_eq!(editor.scroll_offset().1, -140);
        let first_line = editor.rope.line_to_char(0);
        editor.place_cursor(editor.rope.char_to_utf16_cu(first_line));
        assert!(editor.ensure_cursor_visible(20., 100., 20));
        assert_eq!(editor.scroll_offset().1, 0);
    }

    #[test]
    fn editor_tab_indentation_uses_configured_tab_size() {
        let mut editor = CodeEditorData::new(Rope::from_str("fn test() {\n}"), None);
        let tab_size = 4;
        let spaces = " ".repeat(tab_size);
        let cursor_pos = 12; // after \n
        editor.place_cursor(cursor_pos);
        let start_byte = editor.rope.char_to_byte(editor.rope.utf16_cu_to_char(cursor_pos));
        editor.replace_byte_range(start_byte, start_byte, &spaces);
        assert_eq!(editor.rope.to_string(), "fn test() {\n    }");
    }

    #[test]
    fn editor_font_size_clamping_logic() {
        let current = 13.0f32;
        let zoomed_in = (current + 1.0).min(36.0);
        assert_eq!(zoomed_in, 14.0);
        let zoomed_out = (current - 1.0).max(8.0);
        assert_eq!(zoomed_out, 12.0);
        let max_zoom = (36.0f32 + 1.0).min(36.0);
        assert_eq!(max_zoom, 36.0);
        let min_zoom = (8.0f32 - 1.0).max(8.0);
        assert_eq!(min_zoom, 8.0);
    }
}
