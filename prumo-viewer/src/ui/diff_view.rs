use freya::prelude::*;
use torin::prelude::Direction;

use crate::state::AppState;
use crate::ui::chrome::IconButton;

#[derive(PartialEq)]
pub struct DiffModal {
    pub state: State<AppState>,
}

impl Component for DiffModal {
    fn render(&self) -> impl IntoElement {
        let theme = get_theme_or_default();
        let c = theme.read().colors.clone();
        let mut state = self.state;

        let title = state.read().diff_title.clone();
        let lines = state.read().diff_lines.clone();

        let additions = lines.iter().filter(|(_, tag)| *tag == '+').count();
        let deletions = lines.iter().filter(|(_, tag)| *tag == '-').count();
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
            .padding(Gaps::new_all(24.))
            .on_mouse_up(move |_| {
                state.write().diff_open = false;
            })
            .child(
                rect()
                    .width(Size::fill())
                    .max_width(Size::px(760.))
                    .height(Size::fill())
                    .max_height(Size::px(560.))
                    .background(c.background)
                    .corner_radius(CornerRadius::new_all(8.))
                    .border(Border::new().fill(c.border).width(BorderWidth {
                        top: 1.,
                        right: 1.,
                        bottom: 1.,
                        left: 1.,
                    }))
                    .vertical()
                    .content(Content::flex())
                    .on_mouse_up(move |e: Event<MouseEventData>| {
                        e.stop_propagation();
                    })
                    .child(
                        rect()
                            .width(Size::fill())
                            .height(Size::px(40.))
                            .horizontal()
                            .cross_align(Alignment::center())
                            .padding(Gaps::new(0., 14., 0., 14.))
                            .background(c.surface_primary)
                            .border(Border::new().fill(c.border).width(BorderWidth {
                                top: 0.,
                                right: 0.,
                                bottom: 1.,
                                left: 0.,
                            }))
                            .child(
                                label()
                                    .color(c.text_primary)
                                    .font_size(12.5)
                                    .font_weight(FontWeight::MEDIUM)
                                    .max_lines(1)
                                    .text(title),
                            )
                            .child(
                                rect()
                                    .horizontal()
                                    .spacing(8.)
                                    .padding(Gaps::new(0., 0., 0., 12.))
                                    .child(
                                        label()
                                            .color(c.success)
                                            .font_size(11.)
                                            .text(format!("+{additions}")),
                                    )
                                    .child(
                                        label()
                                            .color(c.error)
                                            .font_size(11.)
                                            .text(format!("-{deletions}")),
                                    ),
                            )
                            .child(
                                rect()
                                    .expanded()
                                    .horizontal()
                                    .main_align(Alignment::end())
                                    .child(IconButton {
                                        icon: "x",
                                        label: "Close diff",
                                        size: 12.,
                                        on_press: (move |_: Event<PressEventData>| {
                                            state.write().diff_open = false;
                                        })
                                        .into(),
                                    }),
                            ),
                    )
                    .child(
                        ScrollView::new()
                            .direction(Direction::Vertical)
                            .height(Size::flex(1.))
                            .child(
                                rect()
                                    .width(Size::fill())
                                    .padding(Gaps::new_all(8.))
                                    .vertical()
                                    .children(lines.into_iter().enumerate().map(
                                        |(idx, (line, tag))| {
                                            let (bg_color, text_color, prefix) = match tag {
                                                '+' => (
                                                    Color::from_argb(35, 46, 160, 67),
                                                    c.success,
                                                    "+ ",
                                                ),
                                                '-' => (
                                                    Color::from_argb(35, 248, 81, 73),
                                                    c.error,
                                                    "- ",
                                                ),
                                                _ => (Color::TRANSPARENT, c.text_secondary, "  "),
                                            };

                                            rect()
                                                .width(Size::fill())
                                                .height(Size::px(18.))
                                                .horizontal()
                                                .cross_align(Alignment::center())
                                                .background(bg_color)
                                                .padding(Gaps::new(0., 8., 0., 8.))
                                                .spacing(8.)
                                                .child(
                                                    label()
                                                        .color(c.text_placeholder)
                                                        .font_size(11.)
                                                        .text(format!("{:4}", idx + 1)),
                                                )
                                                .child(
                                                    label()
                                                        .color(text_color)
                                                        .font_size(11.5)
                                                        .font_family("Jetbrains Mono")
                                                        .text(format!("{prefix}{line}")),
                                                )
                                                .into()
                                        },
                                    )),
                            ),
                    ),
            )
    }
}
