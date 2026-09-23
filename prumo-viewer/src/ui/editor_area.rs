use crate::services::diff::open_document_diff;
use crate::services::document::save_active_tab;
use crate::state::{AppState, FileKind};
use crate::theme;
use crate::ui::icons::icon;
use freya::code_editor::{CodeEditor, CodeEditorData, EditorLanguage, Rope};
use freya::prelude::*;
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
}

impl Component for EditorArea {
    fn render(&self) -> impl IntoElement {
        let current_theme = get_theme_or_default();
        let colors = current_theme.read().colors.clone();
        let mut state = self.state;
        let show_whitespace = state.read().config.show_whitespace;
        let active_tab = state.read().active_tab().cloned();
        let accessibility_id = use_a11y();
        let mut last_path = use_state(|| None);
        let mut editor = use_state(|| editor_data("", None, current_theme.read().name == "light"));

        let current_path = active_tab.as_ref().map(|tab| tab.path.clone());
        let previous_path = last_path.read().clone();
        if previous_path != current_path {
            if let Some(previous_path) = previous_path {
                let previous_content = editor.read().rope.to_string();
                let mut app_state = state.write();
                if let Some(previous_tab) = app_state
                    .tabs
                    .iter_mut()
                    .find(|tab| tab.path == previous_path)
                {
                    previous_tab.is_dirty = previous_content != previous_tab.persisted_content;
                    previous_tab.content = previous_content;
                }
            }
            let next_editor = active_tab
                .as_ref()
                .map(|tab| {
                    editor_data(
                        &tab.content,
                        editor_language(tab.kind),
                        current_theme.read().name == "light",
                    )
                })
                .unwrap_or_else(|| editor_data("", None, current_theme.read().name == "light"));
            *editor.write() = next_editor;
            last_path.set(current_path);
        }

        let editor_content = {
            let editor_state = editor.read();
            let content = editor_state.rope.to_string();
            (editor_state.is_edited(), content)
        };
        if editor_content.0 {
            let mut app_state = state.write();
            if app_state
                .active_tab()
                .is_some_and(|tab| tab.content != editor_content.1)
            {
                app_state.update_active_content(editor_content.1);
            }
        }

        use_side_effect(move || {
            let syntax = if get_theme_or_default().read().name == "light" {
                theme::light_syntax()
            } else {
                theme::dark_syntax()
            };
            editor.write().set_theme(syntax);
        });

        let tab = state.read().active_tab().cloned();
        if let Some(tab) = tab {
            let line_count = editor.read().rope.len_lines();
            let kind_name = language_label(tab.kind);
            let is_dirty = tab.is_dirty;
            let is_agent_modified = tab.is_agent_modified;
            let breadcrumb = breadcrumb(&tab.rel_path);

            rect()
                .vertical()
                .width(Size::fill())
                .height(Size::fill())
                .content(Content::flex())
                .background(colors.background)
                .child(
                    rect()
                        .width(Size::fill())
                        .height(Size::px(34.))
                        .horizontal()
                        .cross_align(Alignment::center())
                        .padding(Gaps::new(0., 10., 0., 10.))
                        .background(colors.surface_primary)
                        .border(Border::new().fill(colors.border).width(BorderWidth {
                            top: 0.,
                            right: 0.,
                            bottom: 1.,
                            left: 0.,
                        }))
                        .child(
                            label()
                                .color(colors.text_secondary)
                                .font_size(11.5)
                                .max_lines(1)
                                .text(breadcrumb),
                        )
                        .child(
                            rect()
                                .expanded()
                                .horizontal()
                                .main_align(Alignment::end())
                                .spacing(7.)
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
                                .child(
                                    Button::new()
                                        .enabled(is_dirty)
                                        .on_press(move |_| {
                                            let content = editor.read().rope.to_string();
                                            let mut app_state = state.write();
                                            app_state.update_active_content(content);
                                            let _ = save_active_tab(&mut app_state);
                                        })
                                        .child(
                                            rect()
                                                .horizontal()
                                                .spacing(5.)
                                                .cross_align(Alignment::center())
                                                .child(
                                                    SvgViewer::new(("save", icon("save")))
                                                        .color(if is_dirty {
                                                            colors.primary
                                                        } else {
                                                            colors.text_placeholder
                                                        })
                                                        .width(Size::px(13.))
                                                        .height(Size::px(13.)),
                                                )
                                                .child(
                                                    label()
                                                        .color(if is_dirty {
                                                            colors.text_primary
                                                        } else {
                                                            colors.text_placeholder
                                                        })
                                                        .font_size(11.)
                                                        .text(if is_dirty {
                                                            "Save  Ctrl+S"
                                                        } else {
                                                            "Saved"
                                                        }),
                                                ),
                                        ),
                                )
                                .child(
                                    Button::new()
                                        .on_press(move |_| {
                                            let content = editor.read().rope.to_string();
                                            let mut app_state = state.write();
                                            app_state.update_active_content(content);
                                            open_document_diff(&mut app_state);
                                        })
                                        .child(
                                            label()
                                                .color(colors.text_secondary)
                                                .font_size(11.)
                                                .text("Diff"),
                                        ),
                                ),
                        ),
                )
                .child(
                    rect()
                        .width(Size::fill())
                        .height(Size::flex(1.))
                        .content(Content::flex())
                        .child(
                            CodeEditor::new(editor, accessibility_id)
                                .font_size(EDITOR_FONT_SIZE)
                                .line_height(1.55)
                                .show_whitespace(show_whitespace),
                        ),
                )
                .child(
                    rect()
                        .width(Size::fill())
                        .height(Size::px(27.))
                        .horizontal()
                        .cross_align(Alignment::center())
                        .padding(Gaps::new(0., 10., 0., 10.))
                        .background(colors.surface_primary)
                        .border(Border::new().fill(colors.border).width(BorderWidth {
                            top: 1.,
                            right: 0.,
                            bottom: 0.,
                            left: 0.,
                        }))
                        .child(
                            label()
                                .color(colors.text_placeholder)
                                .font_size(11.)
                                .text(format!("{} · {line_count} lines", tab.rel_path)),
                        )
                        .child(
                            rect()
                                .expanded()
                                .horizontal()
                                .main_align(Alignment::end())
                                .spacing(14.)
                                .child(
                                    label()
                                        .color(colors.text_placeholder)
                                        .font_size(11.)
                                        .text(kind_name),
                                )
                                .child(
                                    label()
                                        .color(colors.text_placeholder)
                                        .font_size(11.)
                                        .text("UTF-8"),
                                ),
                        ),
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
                        .width(Size::px(52.))
                        .height(Size::px(52.)),
                )
                .child(
                    label()
                        .color(colors.text_primary)
                        .font_size(17.)
                        .font_weight(FontWeight::BOLD)
                        .text("Open a file to start"),
                )
                .child(
                    label()
                        .color(colors.text_secondary)
                        .font_size(13.)
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

fn language_label(kind: FileKind) -> &'static str {
    match kind {
        FileKind::Rust => "Rust",
        FileKind::Go => "Go",
        FileKind::Markdown => "Markdown",
        FileKind::Json => "JSON",
        FileKind::Yaml => "YAML",
        FileKind::Toml => "TOML",
        FileKind::TypeScript => "TypeScript",
        FileKind::JavaScript => "JavaScript",
        FileKind::Manifest => "Config",
        _ => "Plain Text",
    }
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

#[cfg(test)]
mod tests {
    use super::*;

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
}
