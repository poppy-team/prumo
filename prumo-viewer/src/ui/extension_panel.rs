use crate::state::{AppState, ExtensionPanelKind};
use freya::prelude::*;

#[derive(PartialEq)]
pub struct ExtensionPanelOverlay {
    pub state: State<AppState>,
}

impl Component for ExtensionPanelOverlay {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let panel = self.state.read().extension_panel.clone();
        let Some(panel) = panel else {
            return rect();
        };
        let kind = match panel.kind {
            ExtensionPanelKind::View => "View",
            ExtensionPanelKind::Panel => "Panel",
        };
        let mut state = self.state;
        let window = Platform::get().root_size.read();
        rect()
            .width(Size::px(window.width))
            .height(Size::px(window.height))
            .position(Position::new_global().top(0.).left(0.))
            .layer(Layer::Overlay)
            .background(Color::TRANSPARENT)
            .horizontal()
            .main_align(Alignment::center())
            .cross_align(Alignment::center())
            .on_press(move |_| state.write().extension_panel = None)
            .child(
                rect()
                    .width(Size::px(720.))
                    .max_height(Size::px(560.))
                    .background(colors.surface_primary)
                    .border(Border::new().fill(colors.border_focus).width(1.))
                    .corner_radius(CornerRadius::new_all(8.))
                    .padding(Gaps::new_all(16.))
                    .vertical()
                    .spacing(12.)
                    .on_mouse_up(|event: Event<MouseEventData>| event.stop_propagation())
                    .child(
                        rect()
                            .width(Size::fill())
                            .horizontal()
                            .main_align(Alignment::space_between())
                            .child(
                                label()
                                    .color(colors.text_primary)
                                    .font_size(16.)
                                    .text(format!("{kind}: {}", panel.title)),
                            )
                            .child(
                                rect()
                                    .padding(Gaps::new(4., 8., 4., 8.))
                                    .background(colors.surface_tertiary)
                                    .corner_radius(CornerRadius::new_all(4.))
                                    .on_all_press(move |_| state.write().extension_panel = None)
                                    .child(
                                        label()
                                            .color(colors.text_secondary)
                                            .font_size(12.)
                                            .text("Close"),
                                    ),
                            ),
                    )
                    .child(
                        ScrollView::new()
                            .direction(torin::prelude::Direction::Vertical)
                            .width(Size::fill())
                            .height(Size::flex(1.))
                            .child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(13.)
                                    .text(panel.content),
                            ),
                    ),
            )
    }
}
