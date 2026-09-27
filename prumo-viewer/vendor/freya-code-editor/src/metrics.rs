use freya_core::prelude::consume_root_context;
use freya_engine::prelude::*;
use ropey::Rope;
use tree_sitter::InputEdit;

use crate::{editor_theme::EditorSyntaxTheme, syntax::*};

pub struct EditorMetrics {
    pub(crate) syntax_blocks: SyntaxBlocks,
    pub(crate) longest_width: f32,
    pub(crate) longest_line_chars: usize,
    pub(crate) longest_line_row: usize,
    pub(crate) measured: bool,
    pub(crate) highlighter: SyntaxHighlighter,
}

impl Default for EditorMetrics {
    fn default() -> Self {
        Self::new()
    }
}

impl EditorMetrics {
    pub fn new() -> Self {
        Self {
            syntax_blocks: SyntaxBlocks::default(),
            longest_width: 0.0,
            longest_line_chars: 0,
            longest_line_row: 0,
            measured: false,
            highlighter: SyntaxHighlighter::new(),
        }
    }

    pub fn measure_longest_line(
        &mut self,
        font_size: f32,
        font_family: &str,
        rope: &Rope,
        edit: Option<InputEdit>,
    ) {
        let font_collection = consume_root_context::<FontCollection>();
        let mut paragraph_style = ParagraphStyle::default();
        let mut text_style = TextStyle::default();
        text_style.set_font_size(font_size);
        text_style.set_font_families(&[font_family]);
        paragraph_style.set_text_style(&text_style);
        let mut paragraph_builder = ParagraphBuilder::new(&paragraph_style, font_collection);
        paragraph_builder.add_text("W");
        let mut paragraph = paragraph_builder.build();
        paragraph.layout(f32::MAX);
        let char_width = paragraph.longest_line();

        if let Some(edit) = edit.filter(|_| self.measured) {
            self.update_longest_line(rope, edit);
        } else {
            self.recompute_longest_line(rope);
        }
        self.longest_width = self.longest_line_chars as f32 * char_width;
    }

    pub fn run_parser(&mut self, rope: &Rope, edit: Option<InputEdit>, theme: &EditorSyntaxTheme) {
        self.highlighter
            .parse(rope, &mut self.syntax_blocks, edit, theme);
    }

    fn recompute_longest_line(&mut self, rope: &Rope) {
        self.longest_line_chars = 0;
        self.longest_line_row = 0;
        for (row, line) in rope.lines().enumerate() {
            let length = line.len_chars();
            if length > self.longest_line_chars {
                self.longest_line_chars = length;
                self.longest_line_row = row;
            }
        }
        self.measured = true;
    }

    fn update_longest_line(&mut self, rope: &Rope, edit: InputEdit) {
        let start_row = edit.start_position.row;
        let old_end_row = edit.old_end_position.row;
        let new_end_row = edit.new_end_position.row;
        let row_delta = new_end_row as isize - old_end_row as isize;

        if row_delta == 0 {
            if start_row < rope.len_lines() {
                let length = rope.line(start_row).len_chars();
                if length >= self.longest_line_chars {
                    self.longest_line_chars = length;
                    self.longest_line_row = start_row;
                }
            }
            return;
        }

        if self.longest_line_row >= start_row && self.longest_line_row <= old_end_row {
            self.recompute_longest_line(rope);
            return;
        }
        if self.longest_line_row > old_end_row {
            self.longest_line_row = (self.longest_line_row as isize + row_delta).max(0) as usize;
        }

        let last_row = new_end_row.min(rope.len_lines().saturating_sub(1));
        if start_row > last_row {
            return;
        }
        for row in start_row..=last_row {
            let length = rope.line(row).len_chars();
            if length > self.longest_line_chars {
                self.longest_line_chars = length;
                self.longest_line_row = row;
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tree_sitter::Point;

    fn edit(start_row: usize, old_end_row: usize, new_end_row: usize) -> InputEdit {
        InputEdit {
            start_byte: 0,
            old_end_byte: 0,
            new_end_byte: 0,
            start_position: Point::new(start_row, 0),
            old_end_position: Point::new(old_end_row, 0),
            new_end_position: Point::new(new_end_row, 0),
        }
    }

    #[test]
    fn updates_longest_line_for_a_single_line_edit() {
        let mut metrics = EditorMetrics::new();
        let rope = Rope::from_str("a\nlongest\nc");
        metrics.recompute_longest_line(&rope);
        assert_eq!(metrics.longest_line_chars, 8);
        assert_eq!(metrics.longest_line_row, 1);

        let rope = Rope::from_str("a\nlongest\nc");
        metrics.update_longest_line(&rope, edit(0, 0, 0));
        assert_eq!(metrics.longest_line_chars, 8);
    }

    #[test]
    fn recomputes_when_a_multi_line_edit_removes_the_longest_line() {
        let mut metrics = EditorMetrics::new();
        let rope = Rope::from_str("a\nlongest\nc\nd");
        metrics.recompute_longest_line(&rope);
        assert_eq!(metrics.longest_line_row, 1);

        let rope = Rope::from_str("a\nc\nd");
        metrics.update_longest_line(&rope, edit(1, 2, 1));
        assert_eq!(metrics.longest_line_chars, 2);
        assert_eq!(metrics.longest_line_row, 0);
    }

    #[test]
    fn handles_large_rope_with_incremental_metric_updates() {
        let source = "line\n".repeat(20_000);
        let rope = Rope::from_str(&source);
        let mut metrics = EditorMetrics::new();
        metrics.recompute_longest_line(&rope);
        assert_eq!(metrics.longest_line_chars, 5);

        let edited_source = format!("longest\n{}", "line\n".repeat(19_999));
        let edited_rope = Rope::from_str(&edited_source);
        metrics.update_longest_line(&edited_rope, edit(0, 0, 0));
        assert_eq!(metrics.longest_line_chars, 8);
        assert_eq!(metrics.longest_line_row, 0);
    }
}
