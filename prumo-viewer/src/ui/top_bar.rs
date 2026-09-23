use crate::state::{AppState, ConnectionStatus, NoticeTone};
use crate::theme;
use crate::ui::icons::icon;
use freya::prelude::*;

#[derive(PartialEq)]
pub struct TopBar {
    pub state: State<AppState>,
}

impl Component for TopBar {
    fn render(&self) -> impl IntoElement {
        let mut current_theme = use_theme();
        let mut state = self.state;
        let colors = current_theme.read().colors.clone();
        let workspace_name = state.read().workspace_name.clone();
        let connection = state.read().connection_status;
        let connection_message = state.read().connection_message.clone();
        let active_run = state.read().active_run_id.clone();
        let (connection_color, connection_label) = match connection {
            ConnectionStatus::Connecting => (colors.warning, connection.label()),
            ConnectionStatus::Connected => (colors.success, connection.label()),
            ConnectionStatus::Disconnected => (colors.error, connection.label()),
        };

        rect()
            .width(Size::fill())
            .height(Size::px(40.))
            .horizontal()
            .cross_align(Alignment::center())
            .padding(Gaps::new(0., 10., 0., 10.))
            .background(colors.surface_inverse)
            .border(Border::new().fill(colors.border).width(BorderWidth {
                top: 0.,
                right: 0.,
                bottom: 1.,
                left: 0.,
            }))
            .child(
                rect()
                    .horizontal()
                    .spacing(8.)
                    .cross_align(Alignment::center())
                    .child(
                        rect()
                            .width(Size::px(9.))
                            .height(Size::px(9.))
                            .corner_radius(CornerRadius::new_all(5.))
                            .background(connection_color),
                    )
                    .child(
                        label()
                            .color(colors.text_primary)
                            .font_size(13.)
                            .font_weight(FontWeight::BOLD)
                            .text("Prumo"),
                    )
                    .child(
                        label()
                            .color(colors.text_secondary)
                            .font_size(12.)
                            .text(format!("{workspace_name} · Workspace")),
                    ),
            )
            .child(
                rect()
                    .expanded()
                    .horizontal()
                    .main_align(Alignment::center())
                    .child(
                        Button::new()
                            .on_press(move |_| {
                                let mut app_state = state.write();
                                app_state.quick_open_open = !app_state.quick_open_open;
                                app_state.diff_open = false;
                                app_state.clear_notice();
                            })
                            .child(
                                rect()
                                    .width(Size::px(320.))
                                    .height(Size::px(26.))
                                    .horizontal()
                                    .main_align(Alignment::center())
                                    .cross_align(Alignment::center())
                                    .spacing(7.)
                                    .background(colors.background)
                                    .border(Border::new().fill(colors.border).width(1.))
                                    .corner_radius(CornerRadius::new_all(4.))
                                    .child(
                                        SvgViewer::new(("search", icon("search")))
                                            .color(colors.text_secondary)
                                            .width(Size::px(13.))
                                            .height(Size::px(13.)),
                                    )
                                    .child(
                                        label()
                                            .color(colors.text_secondary)
                                            .font_size(11.5)
                                            .text("Quick Open  Ctrl+P"),
                                    ),
                            ),
                    ),
            )
            .child(
                rect()
                    .horizontal()
                    .spacing(10.)
                    .cross_align(Alignment::center())
                    .maybe(active_run.is_some(), |parent| {
                        parent.child(
                            rect()
                                .height(Size::px(24.))
                                .horizontal()
                                .cross_align(Alignment::center())
                                .spacing(5.)
                                .padding(Gaps::new(7., 0., 7., 0.))
                                .background(colors.surface_secondary)
                                .corner_radius(CornerRadius::new_all(4.))
                                .child(
                                    SvgViewer::new(("bot", icon("bot")))
                                        .color(colors.primary)
                                        .width(Size::px(12.))
                                        .height(Size::px(12.)),
                                )
                                .child(
                                    label()
                                        .color(colors.text_primary)
                                        .font_size(10.)
                                        .max_lines(1)
                                        .text(format!(
                                            "RUN {}",
                                            active_run.clone().unwrap_or_default()
                                        )),
                                ),
                        )
                    })
                    .child(
                        rect()
                            .horizontal()
                            .spacing(6.)
                            .cross_align(Alignment::center())
                            .child(
                                label()
                                    .color(connection_color)
                                    .font_size(11.)
                                    .font_weight(FontWeight::MEDIUM)
                                    .text(connection_label),
                            )
                            .child(
                                label()
                                    .color(colors.text_placeholder)
                                    .font_size(10.5)
                                    .max_lines(1)
                                    .text(connection_message),
                            ),
                    )
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .height(Size::px(28.))
                            .padding(Gaps::new(8., 0., 8., 0.))
                            .on_press(move |_| state.write().config_open = true)
                            .child(
                                rect()
                                    .horizontal()
                                    .cross_align(Alignment::center())
                                    .spacing(5.)
                                    .child(
                                        SvgViewer::new(("settings", icon("settings")))
                                            .color(colors.text_secondary)
                                            .width(Size::px(13.))
                                            .height(Size::px(13.)),
                                    )
                                    .child(
                                        label()
                                            .color(colors.text_secondary)
                                            .font_size(11.)
                                            .text("Settings"),
                                    ),
                            ),
                    )
                    .child(
                        Button::new()
                            .on_press(move |_| {
                                let next_name = if current_theme.read().name == "light" {
                                    "dark"
                                } else {
                                    "light"
                                };
                                let next = if next_name == "light" {
                                    theme::light()
                                } else {
                                    theme::dark()
                                };
                                current_theme.set(next);
                                let result = {
                                    let mut app_state = state.write();
                                    app_state.config.theme = next_name.to_string();
                                    app_state.config.save()
                                };
                                match result {
                                    Ok(_) => state.write().show_notice(
                                        NoticeTone::Success,
                                        format!("Theme saved: {next_name}"),
                                    ),
                                    Err(error) => state.write().show_notice(
                                        NoticeTone::Error,
                                        format!("Theme preference was not saved: {error}"),
                                    ),
                                }
                            })
                            .child(label().color(colors.text_secondary).font_size(11.).text(
                                if current_theme.read().name == "light" {
                                    "Dark"
                                } else {
                                    "Light"
                                },
                            )),
                    ),
            )
    }
}
