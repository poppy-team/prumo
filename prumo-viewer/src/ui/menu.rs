use crate::state::{AppState, DockView, NoticeTone};
use crate::theme;
use freya::prelude::*;

#[derive(Clone, Copy, PartialEq, Eq)]
enum MenuAction {
    QuickOpen,
    CommandPalette,
    ToggleSidebar,
    ToggleAgent,
    ToggleDock,
    OpenTerminal,
    OpenSettings,
    ToggleTheme,
}

const MENU_ITEMS: [(MenuAction, &str, &str); 8] = [
    (MenuAction::QuickOpen, "Quick Open", "Ctrl+P"),
    (
        MenuAction::CommandPalette,
        "Command Palette",
        "Ctrl+Shift+P",
    ),
    (MenuAction::ToggleSidebar, "Toggle Sidebar", "Ctrl+B"),
    (MenuAction::ToggleAgent, "Toggle Agent Panel", ""),
    (MenuAction::ToggleDock, "Toggle Dock", "Ctrl+J"),
    (MenuAction::OpenTerminal, "Open Terminal", ""),
    (MenuAction::OpenSettings, "Settings", ""),
    (MenuAction::ToggleTheme, "Toggle Theme", ""),
];

#[derive(PartialEq)]
pub struct MenuOverlay {
    pub state: State<AppState>,
}

const MENU_WIDTH: f32 = 230.;
const MENU_RIGHT_MARGIN: f32 = 8.;
const MENU_TOP: f32 = 34.;

fn menu_left(window_width: f32) -> f32 {
    (window_width - MENU_WIDTH - MENU_RIGHT_MARGIN).max(MENU_RIGHT_MARGIN)
}

impl Component for MenuOverlay {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let window = Platform::get().root_size.read();
        let left = menu_left(window.width);

        rect()
            .width(Size::px(window.width))
            .height(Size::px(window.height))
            .position(Position::new_global().top(0.).left(0.))
            .layer(Layer::Overlay)
            .background(Color::TRANSPARENT)
            .on_press(move |_| state.write().menu_open = false)
            .child(
                rect()
                    .width(Size::px(MENU_WIDTH))
                    .position(Position::new_global().top(MENU_TOP).left(left))
                    .layer(Layer::Overlay)
                    .vertical()
                    .spacing(2.)
                    .padding(Gaps::new_all(6.))
                    .background(colors.surface_primary)
                    .border(Border::new().fill(colors.border).width(1.))
                    .corner_radius(CornerRadius::new_all(6.))
                    .on_mouse_up(|event: Event<MouseEventData>| {
                        event.stop_propagation();
                    })
                    .children(MENU_ITEMS.into_iter().map(|(action, text, hint)| {
                        MenuRow {
                            state,
                            action,
                            text,
                            hint,
                        }
                        .into()
                    })),
            )
    }
}

#[derive(PartialEq)]
struct MenuRow {
    state: State<AppState>,
    action: MenuAction,
    text: &'static str,
    hint: &'static str,
}

impl Component for MenuRow {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let mut current_theme = use_theme();
        let action = self.action;

        Button::new()
            .flat()
            .expanded()
            .height(Size::px(28.))
            .padding(Gaps::new(0., 8., 0., 8.))
            .on_press(move |_| {
                let mut app_state = state.write();
                match action {
                    MenuAction::QuickOpen => {
                        app_state.quick_open_open = true;
                        app_state.diff_open = false;
                    }
                    MenuAction::CommandPalette => {
                        app_state.palette_open = true;
                    }
                    MenuAction::ToggleSidebar => {
                        app_state.sidebar_visible = !app_state.sidebar_visible;
                    }
                    MenuAction::ToggleAgent => {
                        app_state.agent_panel_visible = !app_state.agent_panel_visible;
                    }
                    MenuAction::ToggleDock => {
                        app_state.dock_open = !app_state.dock_open;
                    }
                    MenuAction::OpenTerminal => {
                        app_state.dock_view = DockView::Terminal;
                        app_state.dock_open = true;
                    }
                    MenuAction::OpenSettings => {
                        app_state.config_open = true;
                    }
                    MenuAction::ToggleTheme => {
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
                        app_state.config.theme = next_name.to_string();
                        if let Err(error) = app_state.config.save() {
                            app_state.show_notice(
                                NoticeTone::Error,
                                format!("Theme preference was not saved: {error}"),
                            );
                        }
                    }
                }
                app_state.menu_open = false;
            })
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::fill())
                    .horizontal()
                    .main_align(Alignment::space_between())
                    .cross_align(Alignment::center())
                    .child(
                        label()
                            .color(colors.text_primary)
                            .font_size(11.5)
                            .text(self.text),
                    )
                    .child(
                        label()
                            .color(colors.text_placeholder)
                            .font_size(10.)
                            .text(self.hint),
                    ),
            )
    }
}

#[cfg(test)]
mod tests {
    use super::menu_left;

    #[test]
    fn menu_is_anchored_below_the_header_button() {
        assert_eq!(menu_left(1440.), 1202.);
        assert_eq!(menu_left(200.), 8.);
    }
}
