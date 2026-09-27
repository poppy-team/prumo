use std::collections::HashSet;
use std::path::{Path, PathBuf};

use freya::code_editor::LineDecoration;
use freya::prelude::Color;
use similar::{ChangeTag, TextDiff};

use crate::client::protocol::DaemonEvent;
use crate::state::{AgentFileStatus, AppState, EditorReveal, NoticeTone};

pub const MAX_FOLLOWED_DECORATIONS: usize = 2_000;
const FOLLOW_HIGHLIGHT: Color = Color::from_argb(70, 60, 170, 90);

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct FollowTarget {
    pub path: PathBuf,
    pub rel_path: String,
    pub status: AgentFileStatus,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FollowAction {
    Followed { line: usize },
    Open { reveal: EditorReveal },
    Detached,
    HumanEditWins,
    Deleted,
    Skipped,
}

pub fn event_status(event: &DaemonEvent) -> Option<AgentFileStatus> {
    if event.kind != "file.changed" {
        return None;
    }
    let operation = event
        .payload
        .get("operation")
        .and_then(serde_json::Value::as_str)
        .unwrap_or_default();
    Some(AgentFileStatus::from_operation(operation))
}

pub fn target_from_event(event: &DaemonEvent, root: &Path) -> Option<FollowTarget> {
    if event.kind != "file.changed" {
        return None;
    }
    let path = event
        .payload
        .get("path")
        .and_then(serde_json::Value::as_str)
        .filter(|path| !path.is_empty())?;
    let resolved = if Path::new(path).is_absolute() {
        PathBuf::from(path)
    } else {
        root.join(path)
    };
    let status = event_status(event)?;
    resolved.starts_with(root).then(|| FollowTarget {
        rel_path: resolved
            .strip_prefix(root)
            .unwrap_or(&resolved)
            .to_string_lossy()
            .to_string(),
        path: resolved,
        status,
    })
}

pub fn newest_changed_file(
    events: &[DaemonEvent],
    known_event_ids: &HashSet<String>,
    root: &Path,
) -> Option<FollowTarget> {
    events
        .iter()
        .rev()
        .find(|event| event.kind == "file.changed" && !known_event_ids.contains(&event.id))
        .and_then(|event| target_from_event(event, root))
}

pub fn changed_line_ranges(before: &str, after: &str) -> Vec<usize> {
    if before == after {
        return Vec::new();
    }
    let mut line = 0usize;
    let mut ranges = Vec::new();
    for change in TextDiff::from_lines(before, after).iter_all_changes() {
        match change.tag() {
            ChangeTag::Delete => {}
            ChangeTag::Equal => line += 1,
            ChangeTag::Insert => {
                if ranges.len() < MAX_FOLLOWED_DECORATIONS {
                    ranges.push(line);
                }
                line += 1;
            }
        }
    }
    ranges
}

pub fn changed_line_decorations(before: &str, after: &str) -> Vec<LineDecoration> {
    changed_line_ranges(before, after)
        .into_iter()
        .map(|line| LineDecoration {
            line,
            color: FOLLOW_HIGHLIGHT,
        })
        .collect()
}

pub fn reveal_for_change(before: &str, after: &str, path: &Path) -> EditorReveal {
    let line = first_changed_line(before, after).unwrap_or(1);
    let start = byte_offset_of_line(after, line);
    EditorReveal {
        path: path.to_path_buf(),
        line,
        start,
        end: start,
    }
}

fn first_changed_line(before: &str, after: &str) -> Option<usize> {
    changed_line_ranges(before, after)
        .first()
        .map(|line| line + 1)
}

fn byte_offset_of_line(content: &str, line: usize) -> usize {
    content
        .split_inclusive('\n')
        .take(line.saturating_sub(1))
        .map(str::len)
        .sum()
}

pub fn apply_follow(
    state: &mut AppState,
    target: &FollowTarget,
    disk_content: Option<&str>,
) -> FollowAction {
    if !state.follow_agent {
        return FollowAction::Detached;
    }
    if target.status == AgentFileStatus::Deleted {
        return FollowAction::Deleted;
    }
    let Some(disk_content) = disk_content else {
        return FollowAction::Skipped;
    };
    let Some(index) = state.editor.tab_index(&target.path) else {
        let reveal = reveal_for_change("", disk_content, &target.path);
        state.pending_agent_line_decorations = None;
        state.selected_path = Some(target.path.clone());
        return FollowAction::Open { reveal };
    };
    if state.editor.tabs[index].is_dirty {
        remember_follow_conflict(state, &target.rel_path);
        return FollowAction::HumanEditWins;
    }

    let before = state.editor.tabs[index].persisted_content.clone();
    let reveal = reveal_for_change(&before, disk_content, &target.path);
    let changed_line = reveal.line;
    let tab = &mut state.editor.tabs[index];
    tab.content = disk_content.to_string();
    tab.persisted_content = disk_content.to_string();
    tab.is_dirty = false;
    tab.revision = tab.revision.saturating_add(1);
    tab.is_agent_modified = true;
    state.editor.active_tab_index = Some(index);
    state.editor.pending_editor_reveal = Some(reveal);
    state.selected_path = Some(target.path.clone());
    state.agent_line_decorations = changed_line_decorations(&before, disk_content);
    FollowAction::Followed { line: changed_line }
}

fn remember_follow_conflict(state: &mut AppState, rel_path: &str) {
    if state
        .follow_conflict_paths
        .iter()
        .any(|path| path == rel_path)
    {
        return;
    }
    state.follow_conflict_paths.push(rel_path.to_string());
    state.show_notice(
        NoticeTone::Info,
        format!("Agent changed {rel_path} while you had unsaved edits"),
    );
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::state::{DocumentTab, FileKind};
    use serde_json::json;

    fn changed_event(id: &str, path: &str, operation: &str) -> DaemonEvent {
        DaemonEvent {
            id: id.to_string(),
            run_id: "R-1".to_string(),
            kind: "file.changed".to_string(),
            payload: json!({ "path": path, "operation": operation }),
            created_at: "2026-09-25T10:00:00Z".to_string(),
        }
    }

    #[test]
    fn reports_inserted_and_replaced_lines_of_the_new_content() {
        let before = "one\ntwo\nthree\n";
        let after = "one\ntwo extra\nthree\nfour\n";

        assert_eq!(changed_line_ranges(before, after), vec![1, 3]);
        assert_eq!(
            reveal_for_change(before, after, Path::new("/w/a.txt")).line,
            2
        );
    }

    #[test]
    fn reports_nothing_when_content_is_unchanged() {
        assert!(changed_line_ranges("same\n", "same\n").is_empty());
        assert_eq!(
            reveal_for_change("same\n", "same\n", Path::new("/w/a")).line,
            1
        );
    }

    #[test]
    fn treats_a_pure_deletion_as_revealing_the_top() {
        assert!(changed_line_ranges("one\ntwo\n", "one\n").is_empty());
        assert_eq!(
            reveal_for_change("one\ntwo\n", "one\n", Path::new("/w/a")).line,
            1
        );
    }

    #[test]
    fn counts_byte_offsets_of_the_first_line() {
        assert_eq!(byte_offset_of_line("ab\ncd\n", 1), 0);
        assert_eq!(byte_offset_of_line("ab\ncd\n", 2), 3);
    }

    #[test]
    fn picks_the_newest_unseen_change_inside_the_workspace() {
        let root = Path::new("/workspace");
        let events = vec![
            changed_event("1", "a.txt", "modify"),
            changed_event("2", "../outside.txt", "modify"),
        ];
        let known: HashSet<String> = ["2".to_string()].into_iter().collect();

        let target = newest_changed_file(&events, &known, root).expect("target");

        assert_eq!(target.rel_path, "a.txt");
        assert_eq!(target.status, AgentFileStatus::Modified);
    }

    #[test]
    fn ignores_changes_already_seen_and_non_file_events() {
        let root = Path::new("/workspace");
        let events = vec![
            changed_event("1", "a.txt", "modify"),
            DaemonEvent {
                id: "3".to_string(),
                run_id: "R-1".to_string(),
                kind: "text_delta".to_string(),
                payload: json!({ "text": "hi" }),
                created_at: "2026-09-25T10:00:00Z".to_string(),
            },
        ];
        let known: HashSet<String> = ["1".to_string()].into_iter().collect();

        assert!(newest_changed_file(&events, &known, root).is_none());
    }

    #[test]
    fn classifies_operations_from_the_protocol_payload() {
        let root = Path::new("/workspace");

        assert_eq!(
            event_status(&changed_event("1", "new.txt", "create")),
            Some(AgentFileStatus::Created)
        );
        assert_eq!(
            event_status(&changed_event("1", "old.txt", "delete")),
            Some(AgentFileStatus::Deleted)
        );
        assert_eq!(
            newest_changed_file(
                &[changed_event("1", "old.txt", "delete")],
                &HashSet::new(),
                root
            )
            .map(|target| target.status),
            Some(AgentFileStatus::Deleted)
        );
    }

    fn workspace_with_open_tab(rel_path: &str, content: &str) -> (tempfile::TempDir, AppState) {
        let directory = tempfile::tempdir().expect("temp dir");
        let path = directory.path().join(rel_path);
        std::fs::write(&path, content).expect("seed file");
        let mut state = AppState::new(directory.path().to_path_buf());
        state.editor.tabs.push(DocumentTab {
            path: path.clone(),
            rel_path: rel_path.to_string(),
            title: rel_path.to_string(),
            is_dirty: false,
            content: content.to_string(),
            persisted_content: content.to_string(),
            revision: 1,
            conflict: None,
            is_agent_modified: false,
            kind: FileKind::Other,
        });
        state.editor.active_tab_index = Some(0);
        state.selected_path = Some(path);
        (directory, state)
    }

    fn target_for(state: &AppState, status: AgentFileStatus) -> FollowTarget {
        FollowTarget {
            path: state.editor.tabs[0].path.clone(),
            rel_path: state.editor.tabs[0].rel_path.clone(),
            status,
        }
    }

    #[test]
    fn reloads_a_clean_open_tab_and_highlights_the_agent_lines() {
        let (_directory, mut state) = workspace_with_open_tab("agent.txt", "one\ntwo\n");
        let target = target_for(&state, AgentFileStatus::Modified);

        let action = apply_follow(&mut state, &target, Some("one\ntwo\nthree\n"));

        assert_eq!(action, FollowAction::Followed { line: 3 });
        let tab = &state.editor.tabs[0];
        assert_eq!(tab.content, "one\ntwo\nthree\n");
        assert!(!tab.is_dirty);
        assert!(tab.is_agent_modified);
        assert_eq!(state.agent_line_decorations.len(), 1);
        assert_eq!(state.agent_line_decorations[0].line, 2);
        assert_eq!(
            state.editor.pending_editor_reveal.as_ref().map(|r| r.line),
            Some(3)
        );
    }

    #[test]
    fn never_overwrites_human_edits_and_says_so_once() {
        let (_directory, mut state) = workspace_with_open_tab("agent.txt", "one\n");
        state.editor.tabs[0].is_dirty = true;
        let target = target_for(&state, AgentFileStatus::Modified);

        let first = apply_follow(&mut state, &target, Some("one\nagent\n"));
        let second = apply_follow(&mut state, &target, Some("one\nagent\n"));

        assert_eq!(first, FollowAction::HumanEditWins);
        assert_eq!(second, FollowAction::HumanEditWins);
        assert_eq!(state.editor.tabs[0].content, "one\n");
        assert_eq!(state.follow_conflict_paths, vec!["agent.txt".to_string()]);
    }

    #[test]
    fn does_nothing_while_the_user_is_detached() {
        let (_directory, mut state) = workspace_with_open_tab("agent.txt", "one\n");
        state.follow_agent = false;
        let target = target_for(&state, AgentFileStatus::Modified);

        let action = apply_follow(&mut state, &target, Some("one\nagent\n"));

        assert_eq!(action, FollowAction::Detached);
        assert_eq!(state.editor.tabs[0].content, "one\n");
    }

    #[test]
    fn asks_the_loader_to_open_a_file_that_is_not_open_yet() {
        let directory = tempfile::tempdir().expect("temp dir");
        let mut state = AppState::new(directory.path().to_path_buf());
        let target = FollowTarget {
            path: directory.path().join("new.txt"),
            rel_path: "new.txt".to_string(),
            status: AgentFileStatus::Created,
        };

        let action = apply_follow(&mut state, &target, Some("brand new\n"));

        assert!(matches!(action, FollowAction::Open { .. }));
        assert!(state.editor.tabs.is_empty());
    }

    #[test]
    fn deleting_a_file_never_reopens_it() {
        let (_directory, mut state) = workspace_with_open_tab("gone.txt", "one\n");
        let target = target_for(&state, AgentFileStatus::Deleted);

        assert_eq!(
            apply_follow(&mut state, &target, None),
            FollowAction::Deleted
        );
    }
}
