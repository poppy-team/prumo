use crate::services::document::request_close_tab;
use crate::state::{AppState, DocumentTab, FileKind};
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
        let tabs = state.read().tabs.clone();
        let active_index = state.read().active_tab_index;

        rect()
            .width(Size::fill())
            .height(Size::px(36.))
            .horizontal()
            .cross_align(Alignment::end())
            .background(colors.surface_primary)
            .border(Border::new().fill(colors.border).width(BorderWidth {
                top: 0.,
                right: 0.,
                bottom: 1.,
                left: 0.,
            }))
            .child(ScrollView::new().direction(Direction::Horizontal).children(
                tabs.into_iter().enumerate().map(|(index, tab)| {
                    TabItem {
                        state,
                        tab,
                        index,
                        is_active: active_index == Some(index),
                    }
                    .into()
                }),
            ))
    }
}

#[derive(PartialEq)]
struct TabItem {
    state: State<AppState>,
    tab: DocumentTab,
    index: usize,
    is_active: bool,
}

impl Component for TabItem {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let a11y_id = use_a11y();
        let focus = use_focus(a11y_id);
        let mut state = self.state;
        let tab = self.tab.clone();
        let index = self.index;
        let is_active = self.is_active;
        let (icon_name, icon_color) = match tab.kind {
            FileKind::Rust => ("file_code_2", Color::from_rgb(0xe0, 0x8f, 0x62)),
            FileKind::Go => ("file_code", Color::from_rgb(0x5c, 0xa8, 0xe0)),
            FileKind::Manifest => ("package", colors.text_secondary),
            FileKind::Markdown => ("file_text", Color::from_rgb(0x89, 0xb4, 0xfa)),
            _ => ("file_text", colors.text_secondary),
        };

        rect()
            .min_width(Size::px(118.))
            .max_width(Size::px(240.))
            .height(Size::px(35.))
            .horizontal()
            .cross_align(Alignment::center())
            .spacing(6.)
            .padding(Gaps::new(0., 9., 0., 10.))
            .a11y_id(a11y_id)
            .a11y_focusable(true)
            .a11y_role(AccessibilityRole::Tab)
            .a11y_alt(format!("{} tab", self.tab.title))
            .background(if is_active {
                colors.background
            } else {
                colors.surface_primary
            })
            .border(
                Border::new()
                    .fill(if focus() == Focus::Keyboard {
                        colors.primary
                    } else {
                        colors.border
                    })
                    .width(BorderWidth {
                        top: if is_active || focus() == Focus::Keyboard {
                            2.
                        } else {
                            0.
                        },
                        right: if is_active { 0. } else { 1. },
                        bottom: 0.,
                        left: if is_active { 0. } else { 1. },
                    }),
            )
            .on_all_press(move |_| {
                state.write().active_tab_index = Some(index);
            })
            .child(
                SvgViewer::new((icon_name, icon(icon_name)))
                    .color(if is_active {
                        icon_color
                    } else {
                        colors.text_secondary
                    })
                    .width(Size::px(13.))
                    .height(Size::px(13.)),
            )
            .child(
                label()
                    .color(if is_active {
                        colors.text_primary
                    } else {
                        colors.text_secondary
                    })
                    .font_size(12.5)
                    .max_lines(1)
                    .text(tab.title.clone()),
            )
            .child(if tab.is_agent_modified {
                Element::from(
                    label()
                        .color(colors.warning)
                        .font_size(10.)
                        .font_weight(FontWeight::BOLD)
                        .text("A"),
                )
            } else {
                Element::from(label().text(""))
            })
            .child(if tab.is_dirty {
                Element::from(
                    rect()
                        .width(Size::px(6.))
                        .height(Size::px(6.))
                        .corner_radius(CornerRadius::new_all(3.))
                        .background(colors.warning),
                )
            } else {
                Element::from(label().text(""))
            })
            .child(
                Button::new()
                    .flat()
                    .compact()
                    .width(Size::px(20.))
                    .height(Size::px(20.))
                    .padding(0.)
                    .on_press(move |_| {
                        let mut app_state = state.write();
                        request_close_tab(&mut app_state, index);
                    })
                    .child(
                        SvgViewer::new(("close tab", icon("x")))
                            .color(colors.text_placeholder)
                            .width(Size::px(11.))
                            .height(Size::px(11.)),
                    ),
            )
    }
}
