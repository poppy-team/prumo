use crate::WorkspaceCommands;
use crate::services::search::fuzzy_search;
use crate::state::AppState;
use crate::ui::icons::icon;
use freya::prelude::*;
use std::path::PathBuf;
use torin::prelude::Direction;

#[derive(PartialEq)]
pub struct QuickOpenModal {
    pub state: State<AppState>,
    pub workspace: WorkspaceCommands,
}

impl Component for QuickOpenModal {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let workspace = self.workspace;
        let query_state = use_state(String::new);
        let mut selected_state = use_state(|| 0usize);
        let query = query_state.read().clone();
        let root = state.read().workspace_root.clone();
        let candidates = state.read().file_candidates.clone();
        let results = fuzzy_search(&root, &query, &candidates, 30);
        let result_count = results.len();
        let selected_index = (*selected_state.read()).min(result_count.saturating_sub(1));
        let submit_results = results.clone();

        rect()
            .width(Size::fill())
            .height(Size::fill())
            .position(Position::new_global().top(0.).left(0.))
            .layer(Layer::Overlay)
            .background(Color::from_argb(150, 0, 0, 0))
            .horizontal()
            .main_align(Alignment::center())
            .cross_align(Alignment::start())
            .padding(Gaps::new(72., 0., 0., 0.))
            .on_mouse_up(move |_| {
                state.write().quick_open_open = false;
            })
            .child(
                rect()
                    .width(Size::px(620.))
                    .max_height(Size::px(460.))
                    .background(colors.surface_primary)
                    .corner_radius(CornerRadius::new_all(8.))
                    .border(Border::new().fill(colors.border_focus).width(1.))
                    .padding(Gaps::new_all(10.))
                    .vertical()
                    .content(Content::flex())
                    .spacing(8.)
                    .on_mouse_up(|event: Event<MouseEventData>| {
                        event.stop_propagation();
                    })
                    .child(
                        rect()
                            .width(Size::fill())
                            .horizontal()
                            .cross_align(Alignment::center())
                            .spacing(8.)
                            .child(
                                SvgViewer::new(("search", icon("search")))
                                    .color(colors.text_secondary)
                                    .width(Size::px(16.))
                                    .height(Size::px(16.)),
                            )
                            .child(
                                Input::new(query_state.into_writable())
                                    .placeholder("Type a file name…")
                                    .width(Size::fill())
                                    .auto_focus(true)
                                    .filled()
                                    .on_pre_key_down(move |event: Event<KeyboardEventData>| {
                                        match event.key {
                                            Key::Named(NamedKey::ArrowDown) => {
                                                if result_count > 0 {
                                                    let next = (*selected_state.read() + 1)
                                                        .min(result_count - 1);
                                                    selected_state.set(next);
                                                }
                                                event.stop_propagation();
                                                event.prevent_default();
                                                false
                                            }
                                            Key::Named(NamedKey::ArrowUp) => {
                                                if result_count > 0 {
                                                    let previous =
                                                        selected_state.read().saturating_sub(1);
                                                    selected_state.set(previous);
                                                }
                                                event.stop_propagation();
                                                event.prevent_default();
                                                false
                                            }
                                            Key::Named(NamedKey::Escape) => {
                                                state.write().quick_open_open = false;
                                                event.stop_propagation();
                                                event.prevent_default();
                                                false
                                            }
                                            _ => true,
                                        }
                                    })
                                    .on_submit(move |_| {
                                        if let Some(result) = submit_results.get(selected_index) {
                                            open_quick_open_result(
                                                state,
                                                workspace,
                                                result.path.clone(),
                                            );
                                        }
                                    }),
                            )
                            .child(
                                Button::new()
                                    .on_press(move |_| {
                                        state.write().quick_open_open = false;
                                    })
                                    .child(
                                        SvgViewer::new(("x", icon("x")))
                                            .color(colors.text_placeholder)
                                            .width(Size::px(14.))
                                            .height(Size::px(14.)),
                                    ),
                            ),
                    )
                    .child(
                        ScrollView::new()
                            .direction(Direction::Vertical)
                            .height(Size::flex(1.))
                            .children(results.into_iter().enumerate().map(|(index, item)| {
                                let is_selected = index == selected_index;
                                let item_path = item.path.clone();
                                QuickOpenRow {
                                    state,
                                    workspace,
                                    path: item_path,
                                    relative_path: item.rel_path.clone(),
                                    is_selected,
                                }
                                .into()
                            })),
                    )
                    .child(
                        label()
                            .color(colors.text_placeholder)
                            .font_size(10.5)
                            .text("↑↓ navigate · Enter open · Esc close"),
                    ),
            )
    }
}

#[derive(PartialEq)]
struct QuickOpenRow {
    state: State<AppState>,
    workspace: WorkspaceCommands,
    path: PathBuf,
    relative_path: String,
    is_selected: bool,
}

impl Component for QuickOpenRow {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let a11y_id = use_a11y();
        let focus = use_focus(a11y_id);
        let state = self.state;
        let workspace = self.workspace;
        let path = self.path.clone();
        let relative_path = self.relative_path.clone();
        let is_selected = self.is_selected;

        rect()
            .width(Size::fill())
            .min_height(Size::px(30.))
            .horizontal()
            .cross_align(Alignment::center())
            .padding(Gaps::new(0., 8., 0., 8.))
            .a11y_id(a11y_id)
            .a11y_focusable(true)
            .a11y_role(AccessibilityRole::Button)
            .a11y_alt(format!("Open {relative_path}"))
            .corner_radius(CornerRadius::new_all(4.))
            .border(
                Border::new()
                    .fill(colors.primary)
                    .width(if focus() == Focus::Keyboard { 1. } else { 0. }),
            )
            .background(if is_selected {
                colors.surface_tertiary
            } else {
                Color::TRANSPARENT
            })
            .on_all_press(move |_| {
                open_quick_open_result(state, workspace, path.clone());
            })
            .child(
                SvgViewer::new(("file", icon("file")))
                    .color(colors.text_secondary)
                    .width(Size::px(14.))
                    .height(Size::px(14.)),
            )
            .child(
                label()
                    .color(colors.text_primary)
                    .font_size(12.5)
                    .max_lines(1)
                    .text(relative_path),
            )
    }
}

fn open_quick_open_result(mut state: State<AppState>, workspace: WorkspaceCommands, path: PathBuf) {
    let already_open = {
        let mut app_state = state.write();
        app_state.quick_open_open = false;
        app_state.activate_path(&path)
    };
    if !already_open {
        workspace.load_file(path);
    }
}
