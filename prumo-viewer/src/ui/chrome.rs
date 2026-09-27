use crate::ui::icons::icon;
use freya::prelude::*;

pub fn hairline() -> impl IntoElement {
    let border = get_theme_or_default().read().colors.border;
    rect()
        .width(Size::fill())
        .height(Size::px(1.))
        .background(border)
}

pub fn section_label(text: &'static str) -> impl IntoElement {
    let colors = get_theme_or_default().read().colors.clone();
    label()
        .color(colors.text_secondary)
        .font_size(11.)
        .font_weight(FontWeight::MEDIUM)
        .text(text)
}

#[derive(PartialEq)]
pub struct IconButton {
    pub icon: &'static str,
    pub label: &'static str,
    pub size: f32,
    pub on_press: EventHandler<Event<PressEventData>>,
}

impl Component for IconButton {
    fn render(&self) -> impl IntoElement {
        let a11y_id = use_a11y();
        let focus = use_focus(a11y_id);
        let colors = get_theme_or_default().read().colors.clone();
        let on_press = self.on_press.clone();
        let ring = if focus() == Focus::Keyboard { 1. } else { 0. };

        rect()
            .width(Size::px(self.size + 12.))
            .height(Size::px(self.size + 12.))
            .a11y_id(a11y_id)
            .a11y_focusable(true)
            .a11y_role(AccessibilityRole::Button)
            .a11y_alt(self.label)
            .background(Color::TRANSPARENT)
            .corner_radius(CornerRadius::new_all(4.))
            .border(Border::new().fill(colors.border_focus).width(ring))
            .horizontal()
            .main_align(Alignment::center())
            .cross_align(Alignment::center())
            .on_press(move |event| on_press.call(event))
            .child(
                SvgViewer::new((self.icon, icon(self.icon)))
                    .color(colors.text_secondary)
                    .width(Size::px(self.size))
                    .height(Size::px(self.size)),
            )
    }
}
