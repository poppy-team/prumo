use crate::state::{AppState, ConnectionStatus};
use crate::ui::icons::icon;
use freya::prelude::*;

#[derive(PartialEq)]
pub struct StatusBar {
    pub state: State<AppState>,
}

impl Component for StatusBar {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let state = self.state;
        let tabs_count = state.read().tabs.len();
        let files_count = state.read().file_candidates.len();
        let changed_count = state.read().changed_files.len();
        let active_document = state
            .read()
            .active_tab()
            .map(|tab| tab.rel_path.clone())
            .unwrap_or_else(|| "No file".to_string());
        let connection = state.read().connection_status;
        let active_run = state.read().active_run_id.clone();
        let connection_color = match connection {
            ConnectionStatus::Connecting => colors.warning,
            ConnectionStatus::Connected => colors.success,
            ConnectionStatus::Disconnected => colors.error,
        };

        rect()
            .width(Size::fill())
            .height(Size::px(26.))
            .horizontal()
            .cross_align(Alignment::center())
            .padding(Gaps::new(0., 10., 0., 10.))
            .spacing(16.)
            .background(colors.surface_inverse)
            .border(Border::new().fill(colors.border).width(BorderWidth {
                top: 1.,
                right: 0.,
                bottom: 0.,
                left: 0.,
            }))
            .child(StatusItem {
                icon_name: "file_text",
                text: active_document,
                color: colors.text_secondary,
            })
            .child(StatusItem {
                icon_name: "files",
                text: format!("{files_count} files  ·  {tabs_count} open"),
                color: colors.text_secondary,
            })
            .child(
                rect()
                    .expanded()
                    .horizontal()
                    .main_align(Alignment::end())
                    .spacing(16.)
                    .child(StatusItem {
                        icon_name: "list_checks",
                        text: format!("{changed_count} changed"),
                        color: colors.text_secondary,
                    })
                    .child(StatusItem {
                        icon_name: "bot",
                        text: active_run.unwrap_or_else(|| "No active run".to_string()),
                        color: colors.text_secondary,
                    })
                    .child(StatusItem {
                        icon_name: "circle_alert",
                        text: format!("Core {}", connection.label()),
                        color: connection_color,
                    }),
            )
    }
}

#[derive(PartialEq)]
struct StatusItem {
    icon_name: &'static str,
    text: String,
    color: Color,
}

impl Component for StatusItem {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        rect()
            .horizontal()
            .spacing(5.)
            .cross_align(Alignment::center())
            .child(
                SvgViewer::new((self.icon_name, icon(self.icon_name)))
                    .color(self.color)
                    .width(Size::px(12.))
                    .height(Size::px(12.)),
            )
            .child(
                label()
                    .color(colors.text_secondary)
                    .font_size(11.)
                    .text(self.text.clone()),
            )
    }
}
