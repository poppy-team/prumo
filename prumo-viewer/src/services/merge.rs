use similar::{ChangeTag, TextDiff};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[allow(dead_code)]
pub enum MergeSide {
    Mine,
    Theirs,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ConflictRegion {
    pub base_line: usize,
    pub mine: Vec<String>,
    pub theirs: Vec<String>,
    pub choice: MergeSide,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MergeOutcome {
    pub content: String,
    pub conflicts: Vec<ConflictRegion>,
}

impl MergeOutcome {
    pub fn is_clean(&self) -> bool {
        self.conflicts.is_empty()
    }

    #[allow(dead_code)]
    pub fn resolved_with(&self, choices: &[MergeSide]) -> String {
        let mut content = self.content.clone();
        for (region, choice) in self.conflicts.iter().zip(choices) {
            content = content.replace(
                &render_conflict(region),
                &render_resolution(region, *choice),
            );
        }
        content
    }
}

struct Change {
    base_start: usize,
    base_end: usize,
    replacement: Vec<String>,
}

use std::collections::VecDeque;

pub fn three_way_merge(base: &str, mine: &str, theirs: &str) -> MergeOutcome {
    if mine == theirs {
        return MergeOutcome {
            content: mine.to_string(),
            conflicts: Vec::new(),
        };
    }
    if mine == base {
        return MergeOutcome {
            content: theirs.to_string(),
            conflicts: Vec::new(),
        };
    }
    if theirs == base {
        return MergeOutcome {
            content: mine.to_string(),
            conflicts: Vec::new(),
        };
    }

    let base_lines = split_lines(base);
    let mut mine_queue: VecDeque<Change> = changes_from_base(&base_lines, mine).into();
    let mut theirs_queue: VecDeque<Change> = changes_from_base(&base_lines, theirs).into();

    let mut content = String::new();
    let mut conflicts = Vec::new();
    let mut line = 0usize;

    while line < base_lines.len() || !mine_queue.is_empty() || !theirs_queue.is_empty() {
        let mine_starts = mine_queue.front().map(|c| c.base_start == line).unwrap_or(false);
        let theirs_starts = theirs_queue.front().map(|c| c.base_start == line).unwrap_or(false);

        match (mine_starts, theirs_starts) {
            (true, true) => {
                let m = mine_queue.pop_front().unwrap();
                let t = theirs_queue.pop_front().unwrap();
                if m.replacement == t.replacement && m.base_end == t.base_end {
                    append_all(&mut content, &m.replacement);
                    if m.base_end > line {
                        line = m.base_end;
                    }
                } else {
                    let region = ConflictRegion {
                        base_line: line,
                        mine: m.replacement.clone(),
                        theirs: t.replacement.clone(),
                        choice: MergeSide::Mine,
                    };
                    content.push_str(&render_conflict(&region));
                    conflicts.push(region);
                    if m.base_end.max(t.base_end) > line {
                        line = m.base_end.max(t.base_end);
                    }
                }
            }
            (true, false) => {
                let m = mine_queue.pop_front().unwrap();
                if let Some(t) = theirs_queue.pop_front_if(|t_front| t_front.base_start < m.base_end) {
                    let region = ConflictRegion {
                        base_line: line,
                        mine: m.replacement.clone(),
                        theirs: t.replacement.clone(),
                        choice: MergeSide::Mine,
                    };
                    content.push_str(&render_conflict(&region));
                    conflicts.push(region);
                    if m.base_end.max(t.base_end) > line {
                        line = m.base_end.max(t.base_end);
                    }
                    continue;
                }
                append_all(&mut content, &m.replacement);
                if m.base_end > line {
                    line = m.base_end;
                }
            }
            (false, true) => {
                let t = theirs_queue.pop_front().unwrap();
                if let Some(m) = mine_queue.pop_front_if(|m_front| m_front.base_start < t.base_end) {
                    let region = ConflictRegion {
                        base_line: line,
                        mine: m.replacement.clone(),
                        theirs: t.replacement.clone(),
                        choice: MergeSide::Mine,
                    };
                    content.push_str(&render_conflict(&region));
                    conflicts.push(region);
                    if m.base_end.max(t.base_end) > line {
                        line = m.base_end.max(t.base_end);
                    }
                    continue;
                }
                append_all(&mut content, &t.replacement);
                if t.base_end > line {
                    line = t.base_end;
                }
            }
            (false, false) => {
                if line < base_lines.len() {
                    content.push_str(&base_lines[line]);
                    line += 1;
                } else {
                    if let Some(m) = mine_queue.front() {
                        line = m.base_start;
                    } else if let Some(t) = theirs_queue.front() {
                        line = t.base_start;
                    } else {
                        break;
                    }
                }
            }
        }
    }

    MergeOutcome { content, conflicts }
}

pub fn render_conflict(region: &ConflictRegion) -> String {
    let mut rendered = String::new();
    rendered.push_str("<<<<<<< yours\n");
    append_all(&mut rendered, &region.mine);
    rendered.push_str("=======\n");
    append_all(&mut rendered, &region.theirs);
    rendered.push_str(">>>>>>> agent\n");
    rendered
}

#[allow(dead_code)]
pub fn render_resolution(region: &ConflictRegion, choice: MergeSide) -> String {
    let lines = match choice {
        MergeSide::Mine => &region.mine,
        MergeSide::Theirs => &region.theirs,
    };
    let mut rendered = String::new();
    append_all(&mut rendered, lines);
    rendered
}

fn append_all(target: &mut String, lines: &[String]) {
    for line in lines {
        target.push_str(line);
    }
}

fn split_lines(text: &str) -> Vec<String> {
    text.split_inclusive('\n').map(str::to_string).collect()
}

fn changes_from_base(base_lines: &[String], other: &str) -> Vec<Change> {
    let other_lines = split_lines(other);
    let base_refs: Vec<&str> = base_lines.iter().map(String::as_str).collect();
    let other_refs: Vec<&str> = other_lines.iter().map(String::as_str).collect();
    let mut changes = Vec::new();
    let mut base_idx = 0usize;
    let mut change_start = 0usize;
    let mut replacement = Vec::new();
    let mut in_change = false;

    for change in TextDiff::from_slices(&base_refs, &other_refs).iter_all_changes() {
        match change.tag() {
            ChangeTag::Equal => {
                if in_change {
                    changes.push(Change {
                        base_start: change_start,
                        base_end: base_idx,
                        replacement: std::mem::take(&mut replacement),
                    });
                    in_change = false;
                }
                base_idx += 1;
            }
            ChangeTag::Delete => {
                if !in_change {
                    in_change = true;
                    change_start = base_idx;
                }
                base_idx += 1;
            }
            ChangeTag::Insert => {
                if !in_change {
                    in_change = true;
                    change_start = base_idx;
                }
                replacement.push(change.value().to_string());
            }
        }
    }

    if in_change {
        changes.push(Change {
            base_start: change_start,
            base_end: base_idx,
            replacement,
        });
    }

    changes
}

#[cfg(test)]
mod tests {
    use super::*;

    fn lines(text: &str) -> Vec<String> {
        text.split_inclusive('\n').map(str::to_string).collect()
    }

    #[test]
    fn takes_the_agent_side_when_only_the_human_edited() {
        let outcome = three_way_merge("a\nb\n", "a\nmine\n", "a\nb\n");

        assert!(outcome.is_clean());
        assert_eq!(outcome.content, "a\nmine\n");
    }

    #[test]
    fn takes_the_human_side_when_only_the_agent_edited() {
        let outcome = three_way_merge("a\nb\n", "a\nb\n", "a\nagent\n");

        assert!(outcome.is_clean());
        assert_eq!(outcome.content, "a\nagent\n");
    }

    #[test]
    fn merges_edits_in_different_lines_without_asking() {
        let outcome = three_way_merge("a\nb\nc\n", "a\nmine\nc\n", "a\nb\nagent\n");

        assert!(outcome.is_clean());
        assert_eq!(outcome.content, "a\nmine\nagent\n");
    }

    #[test]
    fn reports_a_conflict_when_both_touch_the_same_line() {
        let outcome = three_way_merge("a\nb\n", "a\nmine\n", "a\nagent\n");

        assert_eq!(outcome.conflicts.len(), 1);
        assert_eq!(outcome.conflicts[0].mine, lines("mine\n"));
        assert_eq!(outcome.conflicts[0].theirs, lines("agent\n"));
        assert!(outcome.content.contains("<<<<<<< yours"));
    }

    #[test]
    fn identical_edits_on_both_sides_are_not_a_conflict() {
        let outcome = three_way_merge("a\nb\n", "a\nsame\n", "a\nsame\n");

        assert!(outcome.is_clean());
        assert_eq!(outcome.content, "a\nsame\n");
    }

    #[test]
    fn keeps_insertions_from_both_sides() {
        let outcome = three_way_merge("a\nc\n", "a\nb\nc\n", "a\nc\nd\n");

        assert!(outcome.is_clean());
        assert_eq!(outcome.content, "a\nb\nc\nd\n");
    }

    #[test]
    fn deleting_on_one_side_and_editing_on_the_other_is_a_conflict() {
        let outcome = three_way_merge("a\nb\nc\n", "a\nc\n", "a\nedited\nc\n");

        assert_eq!(outcome.conflicts.len(), 1);
    }

    #[test]
    fn resolving_every_conflict_yields_one_sided_content() {
        let outcome = three_way_merge("a\nb\nc\n", "a\nmine\nc\n", "a\nagent\nc\n");

        let mine = outcome.resolved_with(&[MergeSide::Mine]);
        let theirs = outcome.resolved_with(&[MergeSide::Theirs]);

        assert_eq!(mine, "a\nmine\nc\n");
        assert_eq!(theirs, "a\nagent\nc\n");
    }

    #[test]
    fn a_whole_file_rewrite_by_the_agent_merges_clean_when_human_did_nothing() {
        let outcome = three_way_merge("old\ncontent\n", "old\ncontent\n", "new\nfile\n");

        assert!(outcome.is_clean());
        assert_eq!(outcome.content, "new\nfile\n");
    }
}
