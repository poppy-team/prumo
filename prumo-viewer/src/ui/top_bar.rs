use crate::state::{AppState, ConnectionStatus, NoticeTone};
use crate::theme;
use crate::ui::chrome::IconButton;
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
        let active_run = state.read().active_run_id.clone();
        let theme_icon = if current_theme.read().name == "light" {
            "moon"
        } else {
            "sun"
        };
        let (connection_color, connection_label) = match connection {
            ConnectionStatus::Connecting => (colors.warning, connection.label()),
            ConnectionStatus::Connected => (colors.success, connection.label()),
            ConnectionStatus::Disconnected => (colors.error, connection.label()),
        };

        rect()
            .width(Size::fill())
            .height(Size::px(32.))
            .horizontal()
            .content(Content::flex())
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
                            .font_size(12.)
                            .font_weight(FontWeight::MEDIUM)
                            .text("Prumo"),
                    )
                    .child(
                        label()
                            .color(colors.text_secondary)
                            .font_size(11.)
                            .max_lines(1)
                            .text(workspace_name),
                    ),
            )
            .child(
                rect()
                    .width(Size::flex(1.))
                    .horizontal()
                    .main_align(Alignment::center())
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .height(Size::px(22.))
                            .padding(0.)
                            .on_press(move |_| {
                                let mut app_state = state.write();
                                app_state.quick_open_open = !app_state.quick_open_open;
                                app_state.menu_open = false;
                                app_state.diff_open = false;
                                app_state.clear_notice();
                            })
                            .child(
                                rect()
                                    .width(Size::px(220.))
                                    .height(Size::px(22.))
                                    .horizontal()
                                    .main_align(Alignment::center())
                                    .cross_align(Alignment::center())
                                    .spacing(6.)
                                    .background(Color::TRANSPARENT)
                                    .border(Border::new().fill(colors.border).width(1.))
                                    .corner_radius(CornerRadius::new_all(4.))
                                    .child(
                                        SvgViewer::new(("search", icon("search")))
                                            .color(colors.text_placeholder)
                                            .width(Size::px(10.))
                                            .height(Size::px(10.)),
                                    )
                                    .child(
                                        label()
                                            .color(colors.text_placeholder)
                                            .font_size(10.5)
                                            .text("Quick Open"),
                                    )
                                    .child(
                                        label()
                                            .color(colors.text_placeholder)
                                            .font_size(10.)
                                            .text("Ctrl+P"),
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
                                .height(Size::px(22.))
                                .max_width(Size::px(160.))
                                .horizontal()
                                .cross_align(Alignment::center())
                                .spacing(5.)
                                .padding(Gaps::new(0., 6., 0., 6.))
                                .background(colors.surface_primary)
                                .border(Border::new().fill(colors.border).width(1.))
                                .corner_radius(CornerRadius::new_all(4.))
                                .child(
                                    rect()
                                        .width(Size::px(6.))
                                        .height(Size::px(6.))
                                        .corner_radius(CornerRadius::new_all(3.))
                                        .background(colors.primary),
                                )
                                .child(
                                    label()
                                        .color(colors.text_secondary)
                                        .font_size(10.)
                                        .max_lines(1)
                                        .text(active_run.clone().unwrap_or_default()),
                                ),
                        )
                    })
                    .child(
                        rect()
                            .horizontal()
                            .spacing(5.)
                            .cross_align(Alignment::center())
                            .child(
                                rect()
                                    .width(Size::px(7.))
                                    .height(Size::px(7.))
                                    .corner_radius(CornerRadius::new_all(4.))
                                    .background(connection_color),
                            )
                            .child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(10.)
                                    .text(connection_label),
                            ),
                    )
                    .child(IconButton {
                        icon: "settings",
                        label: "Settings",
                        size: 13.,
                        on_press: (move |_: Event<PressEventData>| {
                            state.write().config_open = true;
                        })
                        .into(),
                    })
                    .child(IconButton {
                        icon: theme_icon,
                        label: "Toggle theme",
                        size: 13.,
                        on_press: (move |_: Event<PressEventData>| {
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
                                Ok(_) => state.write().clear_notice(),
                                Err(error) => state.write().show_notice(
                                    NoticeTone::Error,
                                    format!("Theme preference was not saved: {error}"),
                                ),
                            }
                        })
                        .into(),
                    })
                    .child(IconButton {
                        icon: "menu",
                        label: "Menu",
                        size: 14.,
                        on_press: (move |_: Event<PressEventData>| {
                            let open = state.read().menu_open;
                            state.write().menu_open = !open;
                        })
                        .into(),
                    }),
            )
    }
}
