use crate::state::{AppState, EditorPopupItem, EditorPopupKind};
use crate::ui::editor_area::apply_editor_popup_item;
use freya::code_editor::CodeEditorData;
use freya::prelude::*;
use freya::text_edit::TextEditor;
use torin::prelude::Direction;

const POPUP_WIDTH: f32 = 330.;
const POPUP_MAX_HEIGHT: f32 = 280.;
const LINE_HEIGHT: f32 = 20.925;
const CHARACTER_WIDTH: f32 = 8.2;
const GUTTER_WIDTH: f32 = 68.;

#[derive(PartialEq)]
pub struct EditorPopup {
    pub state: State<AppState>,
    pub editor: State<CodeEditorData>,
}

impl Component for EditorPopup {
    fn render(&self) -> impl IntoElement {
        let Some(popup) = self.state.read().editor_popup.clone() else {
            return rect();
        };
        if !self.state.read().is_editor_popup_request_current(
            popup.request_id,
            &popup.path,
            popup.revision,
        ) {
            return rect();
        }
        let colors = get_theme_or_default().read().colors.clone();
        let (left, top) = popup_position(&self.editor);
        let title = match popup.kind {
            EditorPopupKind::Completion => "Completions",
            EditorPopupKind::CodeActions => "Code actions",
        };
        let selected_index = popup
            .selected_index
            .min(popup.items.len().saturating_sub(1));
        let popup_height = (64. + popup.items.len().min(6) as f32 * 30.).clamp(96., 280.);
        let state = self.state;
        let editor = self.editor;
        let items = popup.items.clone();

        rect()
            .width(Size::px(POPUP_WIDTH))
            .height(Size::px(popup_height.min(POPUP_MAX_HEIGHT)))
            .position(Position::new_absolute().top(top).left(left))
            .layer(Layer::Overlay)
            .vertical()
            .content(Content::flex())
            .spacing(4.)
            .padding(Gaps::new_all(6.))
            .background(colors.surface_primary)
            .border(Border::new().fill(colors.border_focus).width(1.))
            .corner_radius(CornerRadius::new_all(6.))
            .on_mouse_up(|event: Event<MouseEventData>| {
                event.stop_propagation();
            })
            .child(
                label()
                    .color(colors.text_secondary)
                    .font_size(10.)
                    .text(title),
            )
            .maybe(popup.loading, |element| {
                element.child(
                    label()
                        .color(colors.text_placeholder)
                        .font_size(11.)
                        .text("Loading…"),
                )
            })
            .maybe(popup.error.is_some(), |element| {
                element.child(
                    label()
                        .color(colors.text_placeholder)
                        .font_size(11.)
                        .max_lines(2)
                        .text(popup.error.clone().unwrap_or_default()),
                )
            })
            .child(
                ScrollView::new()
                    .direction(Direction::Vertical)
                    .height(Size::px((popup_height - 54.).max(30.)))
                    .children(items.into_iter().enumerate().map(|(index, item)| {
                        let is_selected = index == selected_index;
                        let row_state = state;
                        let row_editor = editor;
                        let row_item = item.clone();
                        let (title, detail) = popup_item_text(&item);
                        let a11y_id = use_a11y();
                        rect()
                            .width(Size::fill())
                            .min_height(Size::px(30.))
                            .horizontal()
                            .spacing(8.)
                            .cross_align(Alignment::center())
                            .padding(Gaps::new(0., 7., 0., 7.))
                            .a11y_id(a11y_id)
                            .a11y_focusable(true)
                            .a11y_role(AccessibilityRole::Button)
                            .a11y_alt(title.clone())
                            .corner_radius(CornerRadius::new_all(4.))
                            .background(if is_selected {
                                colors.surface_tertiary
                            } else {
                                Color::TRANSPARENT
                            })
                            .on_all_press(move |_| {
                                apply_editor_popup_item(row_state, row_editor, row_item.clone());
                            })
                            .child(
                                label()
                                    .color(colors.text_primary)
                                    .font_size(11.5)
                                    .max_lines(1)
                                    .text(title),
                            )
                            .child(
                                label()
                                    .color(colors.text_placeholder)
                                    .font_size(10.)
                                    .max_lines(1)
                                    .text(detail),
                            )
                            .into()
                    })),
            )
            .child(
                label()
                    .color(colors.text_placeholder)
                    .font_size(9.5)
                    .text("↑↓ navigate · Enter apply · Esc close"),
            )
    }
}

fn popup_item_text(item: &EditorPopupItem) -> (String, String) {
    match item {
        EditorPopupItem::Completion(item) => {
            (item.label.clone(), item.detail.clone().unwrap_or_default())
        }
        EditorPopupItem::CodeAction(item) => {
            (item.title.clone(), item.kind.clone().unwrap_or_default())
        }
    }
}

fn popup_position(editor: &State<CodeEditorData>) -> (f32, f32) {
    let editor = editor.read();
    let cursor_char = editor.utf16_cu_to_char(editor.cursor_position());
    let line = editor.char_to_line(cursor_char);
    let line_start = editor.line_to_char(line);
    let column = cursor_char.saturating_sub(line_start) as f32;
    let scroll = -editor.scroll_offset().1 as f32;
    let top = (line as f32 + 1.) * LINE_HEIGHT + scroll + 4.;
    let left = GUTTER_WIDTH + column * CHARACTER_WIDTH + 8.;
    (left.clamp(8., 380.), top.clamp(44., 520.))
}
