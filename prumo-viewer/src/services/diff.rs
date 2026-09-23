use crate::services::document::open_file;
use crate::state::{AppState, NoticeTone};
use similar::{ChangeTag, TextDiff};

pub const MAX_DIFF_LINES: usize = 8000;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DiffResult {
    pub lines: Vec<(String, char)>,
    pub is_truncated: bool,
    pub additions: usize,
    pub deletions: usize,
}

pub fn compute_line_diff(old_text: &str, new_text: &str) -> DiffResult {
    let diff = TextDiff::from_lines(old_text, new_text);
    let mut lines = Vec::new();
    let mut additions = 0;
    let mut deletions = 0;
    let mut is_truncated = false;

    for change in diff.iter_all_changes() {
        if lines.len() >= MAX_DIFF_LINES {
            is_truncated = true;
            lines.push((
                format!("... [Diff truncated at {MAX_DIFF_LINES} lines]"),
                ' ',
            ));
            break;
        }

        let tag = match change.tag() {
            ChangeTag::Delete => {
                deletions += 1;
                '-'
            }
            ChangeTag::Insert => {
                additions += 1;
                '+'
            }
            ChangeTag::Equal => ' ',
        };
        let mut line = change.value().to_string();
        if line.ends_with('\n') {
            line.pop();
            if line.ends_with('\r') {
                line.pop();
            }
        }
        lines.push((line, tag));
    }

    DiffResult {
        lines,
        is_truncated,
        additions,
        deletions,
    }
}

pub fn open_document_diff(state: &mut AppState) {
    let (workspace_root, path, rel_path, editor_content) = {
        let Some(tab) = state.active_tab() else {
            state.show_notice(NoticeTone::Error, "Open a file before showing a diff");
            return;
        };
        (
            state.workspace_root.clone(),
            tab.path.clone(),
            tab.rel_path.clone(),
            tab.content.clone(),
        )
    };
    let disk_tab = match open_file(&workspace_root, &path) {
        Ok(tab) => tab,
        Err(error) => {
            state.show_notice(NoticeTone::Error, error);
            return;
        }
    };
    let diff = compute_line_diff(&disk_tab.content, &editor_content);
    state.diff_title = format!("Working copy · {rel_path}");
    state.diff_lines = diff.lines;
    state.diff_open = true;
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn computes_line_diff() {
        let old = "fn main() {\n    println!(\"hello\");\n}\n";
        let new = "fn main() {\n    println!(\"hello, world!\");\n    // added\n}\n";
        let result = compute_line_diff(old, new);

        assert!(!result.is_truncated);
        assert_eq!(result.deletions, 1);
        assert_eq!(result.additions, 2);
        let tags: Vec<char> = result.lines.iter().map(|(_, tag)| *tag).collect();
        assert!(tags.contains(&'+'));
        assert!(tags.contains(&'-'));
        assert!(tags.contains(&' '));
    }

    #[test]
    fn truncates_large_diff() {
        let mut old = String::new();
        let mut new = String::new();
        for line in 0..8500 {
            old.push_str(&format!("line {line}\n"));
            new.push_str(&format!("line {line} modified\n"));
        }
        let result = compute_line_diff(&old, &new);
        assert!(result.is_truncated);
        assert!(result.lines.len() <= MAX_DIFF_LINES + 1);
    }
}
