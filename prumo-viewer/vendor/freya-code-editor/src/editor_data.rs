use std::{
    borrow::Cow,
    fmt::Display,
    ops::{
        Mul,
        Range,
    },
    time::Duration,
};

use freya_core::{
    elements::paragraph::ParagraphHolderInner,
    prelude::*,
};
use freya_edit::*;
use ropey::Rope;
use tree_sitter::InputEdit;

use crate::{
    editor_theme::EditorSyntaxTheme,
    languages::EditorLanguage,
    metrics::EditorMetrics,
    syntax::InputEditExt,
};

#[derive(Clone, Debug, PartialEq)]
pub struct LineDecoration {
    pub line: usize,
    pub color: Color,
}

pub struct CodeEditorData {
    pub(crate) history: EditorHistory,
    pub rope: Rope,
    pub(crate) selection: TextSelection,
    pub(crate) secondary_selections: Vec<TextSelection>,
    pub(crate) last_saved_history_change: usize,
    pub(crate) metrics: EditorMetrics,
    pub(crate) dragging: TextDragging,
    pub(crate) scrolls: (i32, i32),
    pub(crate) line_decorations: Vec<LineDecoration>,
    pub(crate) pending_edit: Option<InputEdit>,
    pub(crate) last_edit: Option<InputEdit>,
    pub language: Option<EditorLanguage>,
    theme: EditorSyntaxTheme,
}

impl CodeEditorData {
    pub fn new(rope: Rope, language: impl Into<Option<EditorLanguage>>) -> Self {
        let mut data = Self {
            rope,
            selection: TextSelection::new_cursor(0),
            secondary_selections: Vec::new(),
            history: EditorHistory::new(Duration::from_secs(1)),
            last_saved_history_change: 0,
            metrics: EditorMetrics::new(),
            dragging: TextDragging::default(),
            scrolls: (0, 0),
            line_decorations: Vec::new(),
            pending_edit: None,
            last_edit: None,
            language: language.into(),
            theme: EditorSyntaxTheme::default(),
        };
        data.configure_highlighter();
        data
    }

    /// Reconfigures the highlighter with the current language and theme.
    fn configure_highlighter(&mut self) {
        self.metrics
            .highlighter
            .set_language(self.language.as_ref(), &self.theme);
    }

    /// Sets the language used for syntax highlighting, or disables it with `None`.
    pub fn set_language(&mut self, language: impl Into<Option<EditorLanguage>>) {
        self.language = language.into();
        self.configure_highlighter();
    }

    pub fn is_edited(&self) -> bool {
        self.history.current_change() != self.last_saved_history_change
    }

    pub fn mark_as_saved(&mut self) {
        self.last_saved_history_change = self.history.current_change();
    }

    pub fn parse(&mut self) {
        let edit = self.pending_edit.take();
        if edit.is_some() && self.last_edit.is_some() {
            self.last_edit = None;
        } else {
            self.last_edit = edit;
        }
        self.metrics.run_parser(&self.rope, edit, &self.theme);
    }

    pub fn measure(&mut self, font_size: f32, font_family: &str) {
        self.metrics.measure_longest_line(
            font_size,
            font_family,
            &self.rope,
            self.last_edit.take(),
        );
    }

    pub fn set_theme(&mut self, theme: EditorSyntaxTheme) {
        self.theme = theme;
        self.configure_highlighter();
    }

    pub fn process(
        &mut self,
        font_size: f32,
        font_family: &str,
        edit_event: EditableEvent,
    ) -> bool {
        let mut processed = false;
        match edit_event {
            EditableEvent::Down {
                location,
                editor_line,
                holder,
            } => {
                self.secondary_selections.clear();
                let holder = holder.0.borrow();
                let ParagraphHolderInner {
                    paragraph,
                    scale_factor,
                } = holder.as_ref().unwrap();

                let current_selection = self.selection().clone();

                if self.dragging.shift || self.dragging.clicked {
                    self.selection_mut().set_as_range();
                } else {
                    self.clear_selection();
                }

                if &current_selection != self.selection() {
                    processed = true;
                }

                self.dragging.clicked = true;

                let char_position = paragraph.get_glyph_position_at_coordinate(
                    location.mul(*scale_factor).to_i32().to_tuple(),
                );
                let press_selection =
                    self.measure_selection(char_position.position as usize, editor_line);

                let new_selection = match EventsCombos::pressed(location) {
                    PressEventType::Quadruple => {
                        TextSelection::new_range((0, self.rope.len_utf16_cu()))
                    }
                    PressEventType::Triple => {
                        let line = self.char_to_line(press_selection.pos());
                        let line_char = self.line_to_char(line);
                        let line_len = self.line(line).unwrap().utf16_len();
                        TextSelection::new_range((line_char, line_char + line_len))
                    }
                    PressEventType::Double => {
                        let range = self.find_word_boundaries(press_selection.pos());
                        TextSelection::new_range(range)
                    }
                    PressEventType::Single => press_selection,
                };

                if *self.selection() != new_selection {
                    *self.selection_mut() = new_selection;
                    processed = true;
                }
            }
            EditableEvent::Move {
                location,
                editor_line,
                holder,
            } => {
                if self.dragging.clicked {
                    let paragraph = holder.0.borrow();
                    let ParagraphHolderInner {
                        paragraph,
                        scale_factor,
                    } = paragraph.as_ref().unwrap();

                    let dist_position = location.mul(*scale_factor);

                    // Calculate the end of the highlighting
                    let dist_char = paragraph
                        .get_glyph_position_at_coordinate(dist_position.to_i32().to_tuple());
                    let to = dist_char.position as usize;

                    if self.get_selection().is_none() {
                        self.selection_mut().set_as_range();
                        processed = true;
                    }

                    let current_selection = self.selection().clone();

                    let new_selection = self.measure_selection(to, editor_line);

                    // Update the cursor if it has changed
                    if current_selection != new_selection {
                        *self.selection_mut() = new_selection;
                        processed = true;
                    }
                }
            }
            EditableEvent::Release => {
                self.dragging.clicked = false;
            }
            EditableEvent::KeyDown { key, modifiers } => {
                self.secondary_selections.clear();
                match key {
                    // Handle dragging
                    Key::Named(NamedKey::Shift) => {
                        self.dragging.shift = true;
                    }
                    // Handle editing
                    _ => {
                        let event = self.process_key(key, &modifiers, true, true, true, true);
                        if event.contains(TextEvent::TEXT_CHANGED) {
                            self.parse();
                            self.measure(font_size, font_family);
                            self.dragging = TextDragging::default();
                        }
                        if !event.is_empty() {
                            processed = true;
                        }
                    }
                }
            }
            EditableEvent::KeyUp { key, .. } => {
                if *key == Key::Named(NamedKey::Shift) {
                    self.dragging.shift = false;
                }
            }
        };
        processed
    }

    pub fn cursor_position(&self) -> usize {
        self.selection.pos()
    }

    pub fn selection_range(&self) -> Option<(usize, usize)> {
        let (from, to) = match self.selection {
            TextSelection::Cursor(_) => return None,
            TextSelection::Range { from, to } => (from, to),
        };
        if from <= to {
            Some((from, to))
        } else {
            Some((to, from))
        }
    }

    pub fn selection_ranges(&self) -> Vec<(usize, usize)> {
        std::iter::once(&self.selection)
            .chain(self.secondary_selections.iter())
            .filter_map(|selection| match selection {
                TextSelection::Cursor(_) => None,
                TextSelection::Range { from, to } => {
                    Some(if from <= to { (*from, *to) } else { (*to, *from) })
                }
            })
            .collect()
    }

    pub fn set_selection_ranges(&mut self, ranges: Vec<(usize, usize)>) {
        let length = self.rope.len_utf16_cu();
        let mut ranges = ranges
            .into_iter()
            .map(|(from, to)| (from.min(length), to.min(length)))
            .collect::<Vec<_>>();
        if ranges.is_empty() {
            self.selection = TextSelection::new_cursor(0);
            self.secondary_selections.clear();
            return;
        }
        let primary = ranges.remove(0);
        self.selection = TextSelection::new_range(primary);
        self.secondary_selections = ranges
            .into_iter()
            .map(|(from, to)| TextSelection::new_range((from, to)))
            .collect();
    }

    pub fn set_selection_ranges_bytes(&mut self, ranges: Vec<(usize, usize)>) {
        let ranges = ranges
            .into_iter()
            .map(|(from, to)| {
                let from = self.rope.byte_to_char(from.min(self.rope.len_bytes()));
                let to = self.rope.byte_to_char(to.min(self.rope.len_bytes()));
                (
                    self.rope.char_to_utf16_cu(from),
                    self.rope.char_to_utf16_cu(to),
                )
            })
            .collect();
        self.set_selection_ranges(ranges);
    }

    pub fn visible_selection_ranges(&self, line: usize) -> Vec<(usize, usize)> {
        self.selection_ranges()
            .into_iter()
            .filter_map(|(from, to)| self.visible_range(line, from, to))
            .collect()
    }

    pub fn clear_secondary_selections(&mut self) {
        self.secondary_selections.clear();
    }

    pub fn select_range(&mut self, from: usize, to: usize) {
        let length = self.rope.len_utf16_cu();
        self.selection = TextSelection::new_range((from.min(length), to.min(length)));
    }

    pub fn select_range_bytes(&mut self, from: usize, to: usize) {
        let from = self.rope.byte_to_char(from.min(self.rope.len_bytes()));
        let to = self.rope.byte_to_char(to.min(self.rope.len_bytes()));
        let from = self.rope.char_to_utf16_cu(from);
        let to = self.rope.char_to_utf16_cu(to);
        self.select_range(from, to);
    }

    pub fn place_cursor(&mut self, position: usize) {
        self.selection = TextSelection::new_cursor(position.min(self.rope.len_utf16_cu()));
    }

    pub fn place_cursor_bytes(&mut self, position: usize) {
        let position = self.rope.byte_to_char(position.min(self.rope.len_bytes()));
        let position = self.rope.char_to_utf16_cu(position);
        self.place_cursor(position);
    }

    pub fn scroll_offset(&self) -> (i32, i32) {
        self.scrolls
    }

    pub fn set_scroll_offset(&mut self, scroll: (i32, i32)) {
        self.scrolls = scroll;
    }

    pub fn set_line_decorations(&mut self, decorations: Vec<LineDecoration>) {
        self.line_decorations = decorations;
    }

    pub fn line_decoration(&self, line: usize) -> Option<Color> {
        self.line_decorations
            .iter()
            .rev()
            .find(|decoration| decoration.line == line)
            .map(|decoration| decoration.color)
    }

    pub fn ensure_cursor_visible(
        &mut self,
        line_height_px: f32,
        viewport_height_px: f32,
        content_lines: usize,
    ) -> bool {
        let line_height_px = line_height_px.max(1.);
        let viewport_height_px = viewport_height_px.max(1.);
        let cursor_position = self.cursor_position().min(self.rope.len_utf16_cu());
        let cursor_char = self.utf16_cu_to_char(cursor_position);
        let cursor_line = self.char_to_line(cursor_char);
        let line_top = cursor_line as f32 * line_height_px;
        let line_bottom = line_top + line_height_px;
        let current_top = (-self.scrolls.1).max(0) as f32;
        let margin = line_height_px;
        let next_top = if line_top < current_top {
            (line_top - margin).max(0.)
        } else if line_bottom > current_top + viewport_height_px {
            line_bottom - viewport_height_px + margin
        } else {
            current_top
        };
        let content_lines = content_lines.max(self.rope.len_lines());
        let max_top = (content_lines as f32 * line_height_px - viewport_height_px).max(0.);
        let next_scroll = -(next_top.min(max_top).round() as i32);
        let changed = self.scrolls.1 != next_scroll;
        self.scrolls.1 = next_scroll;
        changed
    }

    pub fn reveal_line(
        &mut self,
        line: usize,
        line_height_px: f32,
        viewport_height_px: f32,
        content_lines: usize,
    ) {
        let line_height_px = line_height_px.max(1.);
        let viewport_height_px = viewport_height_px.max(1.);
        let line_top = line.saturating_sub(1) as f32 * line_height_px;
        let padding = viewport_height_px * 0.25;
        let desired_top = (line_top - padding).max(0.0);
        let max_top = (content_lines as f32 * line_height_px - viewport_height_px).max(0.0);
        self.scrolls.1 = -(desired_top.min(max_top) as i32);
    }

    pub fn reveal_byte_range(
        &mut self,
        start: usize,
        end: usize,
        line_height_px: f32,
        viewport_height_px: f32,
    ) {
        let start = start.min(self.rope.len_bytes());
        let line = self.rope.byte_to_line(self.rope.byte_to_char(start)) + 1;
        let content_lines = self.rope.len_lines();
        self.select_range_bytes(start, end);
        self.reveal_line(
            line,
            line_height_px,
            viewport_height_px,
            content_lines,
        );
    }

    pub fn replace_byte_range(&mut self, start_byte: usize, end_byte: usize, replacement: &str) {
        let start_char = self
            .rope
            .byte_to_char(start_byte.min(self.rope.len_bytes()));
        let end_char = self
            .rope
            .byte_to_char(end_byte.min(self.rope.len_bytes()));
        let start = self.rope.char_to_utf16_cu(start_char);
        let end = self.rope.char_to_utf16_cu(end_char);
        if start > end {
            return;
        }
        if start < end {
            self.remove(start..end);
            self.parse();
        }
        self.place_cursor(start);
        if replacement.is_empty() {
            self.parse();
            return;
        }
        let inserted = self.insert(replacement, start);
        self.parse();
        self.place_cursor(start + inserted);
    }

    pub fn undo(&mut self) -> bool {
        TextEditor::undo(self).is_some()
    }

    pub fn redo(&mut self) -> bool {
        TextEditor::redo(self).is_some()
    }

    pub fn can_undo(&self) -> bool {
        self.history.can_undo()
    }

    pub fn can_redo(&self) -> bool {
        self.history.can_redo()
    }

    fn visible_range(&self, line: usize, from: usize, to: usize) -> Option<(usize, usize)> {
        if from == to || line >= self.rope.len_lines() {
            return None;
        }
        let from_char = self.utf16_cu_to_char(from.min(self.rope.len_utf16_cu()));
        let to_char = self.utf16_cu_to_char(to.min(self.rope.len_utf16_cu()));
        let from_row = self.char_to_line(from_char);
        let to_row = self.char_to_line(to_char);
        let line_start = self.char_to_utf16_cu(self.line_to_char(line));
        let line_length = self.line(line)?.utf16_len();
        let range = if from_row == to_row {
            if line == from_row {
                (from.saturating_sub(line_start), to.min(line_start + line_length).saturating_sub(line_start))
            } else {
                return None;
            }
        } else if line > from_row && line < to_row {
            (0, line_length)
        } else if line == from_row {
            (from.saturating_sub(line_start), line_length)
        } else if line == to_row {
            (0, to.saturating_sub(line_start).min(line_length))
        } else {
            return None;
        };
        (range.0 < range.1).then_some(range)
    }
}

impl Display for CodeEditorData {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.rope.to_string())
    }
}

impl TextEditor for CodeEditorData {
    type LinesIterator<'a>
        = LinesIterator<'a>
    where
        Self: 'a;

    fn lines(&self) -> Self::LinesIterator<'_> {
        unimplemented!("Unused.")
    }

    fn insert_char(&mut self, ch: char, idx: usize) -> usize {
        let idx_utf8 = self.utf16_cu_to_char(idx);
        let selection = self.selection.clone();

        // Capture byte offset and position before mutation for InputEdit.
        let start_byte = self.rope.char_to_byte(idx_utf8);
        let start_line = self.rope.char_to_line(idx_utf8);
        let start_line_byte = self.rope.line_to_byte(start_line);
        let start_col = start_byte - start_line_byte;

        let len_before_insert = self.rope.len_utf16_cu();
        self.rope.insert_char(idx_utf8, ch);
        let len_after_insert = self.rope.len_utf16_cu();

        let inserted_text_len = len_after_insert - len_before_insert;

        // Compute new end position after insertion.
        let new_end_char = idx_utf8 + 1; // one char inserted
        let new_end_byte = self.rope.char_to_byte(new_end_char);
        let new_end_line = self.rope.char_to_line(new_end_char);
        let new_end_line_byte = self.rope.line_to_byte(new_end_line);
        let new_end_col = new_end_byte - new_end_line_byte;

        self.pending_edit = Some(InputEdit::new_edit(
            start_byte,
            start_byte,
            new_end_byte,
            (start_line, start_col),
            (start_line, start_col),
            (new_end_line, new_end_col),
        ));

        self.history.push_change(HistoryChange::InsertChar {
            idx,
            ch,
            len: inserted_text_len,
            selection,
        });

        inserted_text_len
    }

    fn insert(&mut self, text: &str, idx: usize) -> usize {
        let idx_utf8 = self.utf16_cu_to_char(idx);
        let selection = self.selection.clone();

        // Capture byte offset and position before mutation for InputEdit.
        let start_byte = self.rope.char_to_byte(idx_utf8);
        let start_line = self.rope.char_to_line(idx_utf8);
        let start_line_byte = self.rope.line_to_byte(start_line);
        let start_col = start_byte - start_line_byte;

        let len_before_insert = self.rope.len_utf16_cu();
        self.rope.insert(idx_utf8, text);
        let len_after_insert = self.rope.len_utf16_cu();

        let inserted_text_len = len_after_insert - len_before_insert;

        // Compute new end position after insertion.
        let inserted_chars = text.chars().count();
        let new_end_char = idx_utf8 + inserted_chars;
        let new_end_byte = self.rope.char_to_byte(new_end_char);
        let new_end_line = self.rope.char_to_line(new_end_char);
        let new_end_line_byte = self.rope.line_to_byte(new_end_line);
        let new_end_col = new_end_byte - new_end_line_byte;

        self.pending_edit = Some(InputEdit::new_edit(
            start_byte,
            start_byte,
            new_end_byte,
            (start_line, start_col),
            (start_line, start_col),
            (new_end_line, new_end_col),
        ));

        self.history.push_change(HistoryChange::InsertText {
            idx,
            text: text.to_owned(),
            len: inserted_text_len,
            selection,
        });

        inserted_text_len
    }

    fn remove(&mut self, range_utf16: Range<usize>) -> usize {
        let range =
            self.utf16_cu_to_char(range_utf16.start)..self.utf16_cu_to_char(range_utf16.end);
        let text = self.rope.slice(range.clone()).to_string();
        let selection = self.selection.clone();

        // Capture byte offsets and positions before mutation for InputEdit.
        let start_byte = self.rope.char_to_byte(range.start);
        let old_end_byte = self.rope.char_to_byte(range.end);
        let start_line = self.rope.char_to_line(range.start);
        let start_line_byte = self.rope.line_to_byte(start_line);
        let start_col = start_byte - start_line_byte;
        let old_end_line = self.rope.char_to_line(range.end);
        let old_end_line_byte = self.rope.line_to_byte(old_end_line);
        let old_end_col = old_end_byte - old_end_line_byte;

        let len_before_remove = self.rope.len_utf16_cu();
        self.rope.remove(range);
        let len_after_remove = self.rope.len_utf16_cu();

        let removed_text_len = len_before_remove - len_after_remove;

        // After removal, new_end == start (the removed range collapses to a point).
        self.pending_edit = Some(InputEdit::new_edit(
            start_byte,
            old_end_byte,
            start_byte,
            (start_line, start_col),
            (old_end_line, old_end_col),
            (start_line, start_col),
        ));

        self.history.push_change(HistoryChange::Remove {
            idx: range_utf16.end - removed_text_len,
            text,
            len: removed_text_len,
            selection,
        });

        removed_text_len
    }

    fn char_to_line(&self, char_idx: usize) -> usize {
        self.rope.char_to_line(char_idx)
    }

    fn line_to_char(&self, line_idx: usize) -> usize {
        self.rope.line_to_char(line_idx)
    }

    fn utf16_cu_to_char(&self, utf16_cu_idx: usize) -> usize {
        self.rope.utf16_cu_to_char(utf16_cu_idx)
    }

    fn char_to_utf16_cu(&self, idx: usize) -> usize {
        self.rope.char_to_utf16_cu(idx)
    }

    fn line(&self, line_idx: usize) -> Option<Line<'_>> {
        let line = self.rope.get_line(line_idx);

        line.map(|line| Line {
            text: Cow::Owned(line.to_string()),
            utf16_len: line.len_utf16_cu(),
        })
    }

    fn len_lines(&self) -> usize {
        self.rope.len_lines()
    }

    fn len_chars(&self) -> usize {
        self.rope.len_chars()
    }

    fn len_utf16_cu(&self) -> usize {
        self.rope.len_utf16_cu()
    }

    fn has_any_selection(&self) -> bool {
        self.selection.is_range()
    }

    fn get_selection(&self) -> Option<(usize, usize)> {
        match self.selection {
            TextSelection::Cursor(_) => None,
            TextSelection::Range { from, to } => Some((from, to)),
        }
    }

    fn set(&mut self, text: &str) {
        self.rope.remove(0..);
        self.rope.insert(0, text);
    }

    fn clear_selection(&mut self) {
        let end = self.selection().end();
        self.selection_mut().set_as_cursor();
        self.selection_mut().move_to(end);
    }

    fn set_selection(&mut self, (from, to): (usize, usize)) {
        self.selection = TextSelection::Range { from, to };
    }

    fn get_selected_text(&self) -> Option<String> {
        let (start, end) = self.get_selection_range()?;

        Some(self.rope.get_slice(start..end)?.to_string())
    }

    fn get_selection_range(&self) -> Option<(usize, usize)> {
        let (start, end) = match self.selection {
            TextSelection::Cursor(_) => return None,
            TextSelection::Range { from, to } => (from, to),
        };

        // Use left-to-right selection
        let (start, end) = if start < end {
            (start, end)
        } else {
            (end, start)
        };

        Some((start, end))
    }

    fn undo(&mut self) -> Option<TextSelection> {
        // Undo can make arbitrary changes — invalidate the tree for a full re-parse.
        self.pending_edit = None;
        self.last_edit = None;
        self.metrics.highlighter.invalidate_tree();
        self.history.undo(&mut self.rope)
    }

    fn redo(&mut self) -> Option<TextSelection> {
        // Redo can make arbitrary changes — invalidate the tree for a full re-parse.
        self.pending_edit = None;
        self.last_edit = None;
        self.metrics.highlighter.invalidate_tree();
        self.history.redo(&mut self.rope)
    }

    fn editor_history(&mut self) -> &mut EditorHistory {
        &mut self.history
    }

    fn selection(&self) -> &TextSelection {
        &self.selection
    }

    fn selection_mut(&mut self) -> &mut TextSelection {
        &mut self.selection
    }

    fn get_indentation(&self) -> u8 {
        4
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use freya_edit::TextEditor;

    #[test]
    fn consecutive_parses_force_a_full_metric_refresh() {
        let mut editor = CodeEditorData::new(Rope::from_str("one"), None);
        editor.insert_char('!', 0);
        editor.parse();
        assert!(editor.last_edit.is_some());
        editor.parse();
        assert!(editor.last_edit.is_none());
        editor.undo();
        assert!(editor.last_edit.is_none());
    }
}
