use crate::state::{AppState, ConnectionStatus};
use freya::prelude::*;

#[derive(PartialEq)]
pub struct StatusBar {
    pub state: State<AppState>,
}

impl Component for StatusBar {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let state = self.state;
        let active_document = state
            .read()
            .active_tab()
            .map(|tab| tab.rel_path.clone())
            .unwrap_or_else(|| "No file".to_string());
        let connection = state.read().connection_status;
        let active_run = state.read().active_run_id.clone();
        let (error_count, warning_count) = state.read().diagnostic_counts();
        let connection_color = match connection {
            ConnectionStatus::Connecting => colors.warning,
            ConnectionStatus::Connected => colors.success,
            ConnectionStatus::Disconnected => colors.error,
        };
        let extension_count = state.read().extensions.len();
        let extension_error_count = state.read().extension_errors.len();
        let extension_label = if extension_error_count > 0 {
            format!("Extensions: {extension_count} ({extension_error_count} errors)")
        } else {
            format!("Extensions: {extension_count}")
        };

        rect()
            .width(Size::fill())
            .height(Size::px(24.))
            .horizontal()
            .cross_align(Alignment::center())
            .padding(Gaps::new(0., 8., 0., 8.))
            .spacing(12.)
            .background(colors.background)
            .border(Border::new().fill(colors.border).width(BorderWidth {
                top: 1.,
                right: 0.,
                bottom: 0.,
                left: 0.,
            }))
            .child(
                label()
                    .color(colors.text_secondary)
                    .font_size(10.5)
                    .max_lines(1)
                    .text(active_document),
            )
            .maybe(error_count > 0 || warning_count > 0, |bar| {
                bar.child(
                    label()
                        .color(colors.text_secondary)
                        .font_size(10.5)
                        .text(format!("{error_count} errors · {warning_count} warnings")),
                )
            })
            .child(
                rect()
                    .expanded()
                    .horizontal()
                    .main_align(Alignment::end())
                    .cross_align(Alignment::center())
                    .spacing(10.)
                    .child(
                        label()
                            .color(colors.text_placeholder)
                            .font_size(10.5)
                            .max_lines(1)
                            .text(
                                active_run
                                    .clone()
                                    .unwrap_or_else(|| "No active run".to_string()),
                            ),
                    )
                    .maybe(extension_count > 0 || extension_error_count > 0, |bar| {
                        bar.child(
                            label()
                                .color(if extension_error_count > 0 {
                                    colors.error
                                } else {
                                    colors.text_secondary
                                })
                                .font_size(10.5)
                                .text(extension_label.clone()),
                        )
                    })
                    .child(
                        rect()
                            .horizontal()
                            .cross_align(Alignment::center())
                            .spacing(5.)
                            .child(
                                rect()
                                    .width(Size::px(6.))
                                    .height(Size::px(6.))
                                    .corner_radius(CornerRadius::new_all(3.))
                                    .background(connection_color),
                            )
                            .child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(10.5)
                                    .text(connection.label()),
                            ),
                    ),
            )
    }
}
