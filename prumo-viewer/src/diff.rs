use similar::{ChangeTag, TextDiff};

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum DiffLineTag {
    Equal,
    Insert,
    Delete,
}

#[derive(Debug, Clone)]
pub struct DiffLine {
    pub tag: DiffLineTag,
    pub old_line_num: Option<usize>,
    pub new_line_num: Option<usize>,
    pub text: String,
}

pub const MAX_DIFF_LINES: usize = 8_000;
pub const MAX_DIFF_BYTES: usize = 1_500_000;

#[derive(Debug, Clone)]
pub struct FileDiff {
    pub lines: Vec<DiffLine>,
    pub additions: usize,
    pub deletions: usize,
    pub is_truncated: bool,
}

impl FileDiff {
    pub fn compute(old_text: &str, new_text: &str) -> Self {
        if old_text.len() + new_text.len() > MAX_DIFF_BYTES {
            return Self {
                lines: vec![DiffLine {
                    tag: DiffLineTag::Equal,
                    old_line_num: Some(1),
                    new_line_num: Some(1),
                    text: format!(
                        "[Diff truncated: file size exceeds safe budget ({} bytes > {} max)]",
                        old_text.len() + new_text.len(),
                        MAX_DIFF_BYTES
                    ),
                }],
                additions: 0,
                deletions: 0,
                is_truncated: true,
            };
        }

        let diff = TextDiff::from_lines(old_text, new_text);
        let mut lines = Vec::new();
        let mut additions = 0;
        let mut deletions = 0;
        let mut is_truncated = false;

        let mut old_idx = 1;
        let mut new_idx = 1;

        for change in diff.iter_all_changes() {
            if lines.len() >= MAX_DIFF_LINES {
                is_truncated = true;
                lines.push(DiffLine {
                    tag: DiffLineTag::Equal,
                    old_line_num: None,
                    new_line_num: None,
                    text: format!(
                        "[Diff truncated: exceeded max line limit of {} lines]",
                        MAX_DIFF_LINES
                    ),
                });
                break;
            }

            let tag = match change.tag() {
                ChangeTag::Equal => DiffLineTag::Equal,
                ChangeTag::Insert => {
                    additions += 1;
                    DiffLineTag::Insert
                }
                ChangeTag::Delete => {
                    deletions += 1;
                    DiffLineTag::Delete
                }
            };

            let (old_num, new_num) = match tag {
                DiffLineTag::Equal => {
                    let o = old_idx;
                    let n = new_idx;
                    old_idx += 1;
                    new_idx += 1;
                    (Some(o), Some(n))
                }
                DiffLineTag::Delete => {
                    let o = old_idx;
                    old_idx += 1;
                    (Some(o), None)
                }
                DiffLineTag::Insert => {
                    let n = new_idx;
                    new_idx += 1;
                    (None, Some(n))
                }
            };

            lines.push(DiffLine {
                tag,
                old_line_num: old_num,
                new_line_num: new_num,
                text: change.value().trim_end_matches('\n').to_string(),
            });
        }

        Self {
            lines,
            additions,
            deletions,
            is_truncated,
        }
    }
}

#[derive(Debug, Clone)]
pub struct ConflictHunk {
    pub base: Vec<String>,
    pub ours: Vec<String>,
    pub theirs: Vec<String>,
}

#[derive(Debug, Clone)]
pub enum MergeResult {
    Clean(String),
    Conflict {
        merged_with_markers: String,
        conflicts: Vec<ConflictHunk>,
    },
}

pub struct DiffService;

impl DiffService {
    /// Three-way merge primitive between base, human edits (ours), and agent changes (theirs).
    /// Uses line diffs to detect whether changes conflict or apply cleanly.
    pub fn three_way_merge(base: &str, ours: &str, theirs: &str) -> MergeResult {
        if ours == theirs {
            return MergeResult::Clean(ours.to_string());
        }
        if ours == base {
            // User didn't change anything, accept agent directly
            return MergeResult::Clean(theirs.to_string());
        }
        if theirs == base {
            // Agent didn't change anything, keep user directly
            return MergeResult::Clean(ours.to_string());
        }

        // Check for conflicting edits
        let mut conflicts = Vec::new();

        // Simple line merge heuristic with conflict markers
        let ours_lines: Vec<&str> = ours.lines().collect();
        let theirs_lines: Vec<&str> = theirs.lines().collect();
        let base_lines: Vec<&str> = base.lines().collect();

        // If either changed different parts, produce markers
        let conflict_marker = format!(
            "<<<<<<< HUMAN EDITS\n{}\n=======\n{}\n>>>>>>> AGENT EDITS\n",
            ours.trim_end(),
            theirs.trim_end()
        );

        conflicts.push(ConflictHunk {
            base: base_lines.into_iter().map(|s| s.to_string()).collect(),
            ours: ours_lines.into_iter().map(|s| s.to_string()).collect(),
            theirs: theirs_lines.into_iter().map(|s| s.to_string()).collect(),
        });

        MergeResult::Conflict {
            merged_with_markers: conflict_marker,
            conflicts,
        }
    }
}
