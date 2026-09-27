use crate::services::document::request_close_tab;
use crate::state::{AppState, FileKind};
use crate::ui::chrome::IconButton;
use crate::ui::icons::icon;
use freya::prelude::*;
use torin::prelude::Direction;

#[derive(PartialEq)]
pub struct TabBar {
    pub state: State<AppState>,
}

impl Component for TabBar {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let state = self.state;
        let active_index = state.read().active_tab_index;
        let tabs = state
            .read()
            .tabs
            .iter()
            .enumerate()
            .map(|(index, tab)| TabItem {
                state,
                title: tab.title.clone(),
                kind: tab.kind,
                is_agent_modified: tab.is_agent_modified,
                is_dirty: tab.is_dirty,
                index,
                is_active: active_index == Some(index),
            })
            .collect::<Vec<_>>();

        rect()
            .width(Size::fill())
            .height(Size::px(30.))
            .horizontal()
            .cross_align(Alignment::end())
            .background(colors.background)
            .border(Border::new().fill(colors.border).width(BorderWidth {
                top: 0.,
                right: 0.,
                bottom: 1.,
                left: 0.,
            }))
            .child(
                ScrollView::new()
                    .direction(Direction::Horizontal)
                    .children(tabs.into_iter().map(|tab| tab.into())),
            )
    }
}

#[derive(PartialEq)]
struct TabItem {
    state: State<AppState>,
    title: String,
    kind: FileKind,
    is_agent_modified: bool,
    is_dirty: bool,
    index: usize,
    is_active: bool,
}

impl Component for TabItem {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let a11y_id = use_a11y();
        let focus = use_focus(a11y_id);
        let mut hovered = use_state(|| false);
        let mut state = self.state;
        let index = self.index;
        let is_active = self.is_active;
        let icon_name = match self.kind {
            FileKind::Rust => "file_code_2",
            FileKind::Go => "file_code",
            FileKind::Manifest => "package",
            _ => "file_text",
        };

        rect()
            .min_width(Size::px(110.))
            .max_width(Size::px(220.))
            .height(Size::px(29.))
            .horizontal()
            .cross_align(Alignment::center())
            .spacing(6.)
            .padding(Gaps::new(0., 8., 0., 8.))
            .a11y_id(a11y_id)
            .a11y_focusable(true)
            .a11y_role(AccessibilityRole::Tab)
            .a11y_alt(format!("{} tab", self.title))
            .background(if is_active {
                colors.surface_primary
            } else {
                colors.background
            })
            .border(
                Border::new()
                    .fill(if focus() == Focus::Keyboard {
                        colors.border_focus
                    } else {
                        colors.border
                    })
                    .width(BorderWidth {
                        top: 0.,
                        right: if is_active { 0. } else { 1. },
                        bottom: 0.,
                        left: 0.,
                    }),
            )
            .on_all_press(move |_| {
                state.write().active_tab_index = Some(index);
            })
            .on_pointer_enter(move |_| hovered.set(true))
            .on_pointer_leave(move |_| hovered.set(false))
            .child(
                SvgViewer::new((icon_name, icon(icon_name)))
                    .color(colors.text_secondary)
                    .width(Size::px(11.))
                    .height(Size::px(11.)),
            )
            .child(
                label()
                    .color(if is_active {
                        colors.text_primary
                    } else {
                        colors.text_secondary
                    })
                    .font_size(12.)
                    .max_lines(1)
                    .text(self.title.clone()),
            )
            .child(if self.is_agent_modified {
                Element::from(
                    rect()
                        .width(Size::px(5.))
                        .height(Size::px(5.))
                        .corner_radius(CornerRadius::new_all(3.))
                        .background(colors.warning),
                )
            } else {
                Element::from(label().text(""))
            })
            .child(if *hovered.read() || is_active {
                Element::from(IconButton {
                    icon: "x",
                    label: "Close tab",
                    size: 10.,
                    on_press: (move |_: Event<PressEventData>| {
                        let mut app_state = state.write();
                        request_close_tab(&mut app_state, index);
                    })
                    .into(),
                })
            } else if self.is_dirty {
                Element::from(
                    rect()
                        .width(Size::px(22.))
                        .horizontal()
                        .main_align(Alignment::center())
                        .child(
                            rect()
                                .width(Size::px(6.))
                                .height(Size::px(6.))
                                .corner_radius(CornerRadius::new_all(3.))
                                .background(colors.warning),
                        ),
                )
            } else {
                Element::from(rect().width(Size::px(22.)))
            })
    }
}
