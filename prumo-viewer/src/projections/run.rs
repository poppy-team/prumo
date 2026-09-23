use serde_json::Value;
use std::collections::HashMap;
use std::path::Path;

use crate::client::protocol::DaemonSnapshot;
use crate::state::{
    AgentEventItem, AgentFileStatus, AgentStatus, AppState, ChangedFile, ConnectionStatus, TreeNode,
};

#[derive(Debug, Clone, Default)]
pub struct RunProjection {
    pub active_run_id: Option<String>,
    pub status: Option<AgentStatus>,
    pub file_badges: HashMap<String, AgentFileStatus>,
    pub events: Vec<AgentEventItem>,
    pub log_lines: Vec<String>,
}

impl RunProjection {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn apply_daemon_event(&mut self, event: &crate::client::protocol::DaemonEvent) {
        if event.kind == "file.changed"
            && let Some(path) = payload_text(&event.payload, "path")
        {
            let status = match payload_text(&event.payload, "operation")
                .unwrap_or_default()
                .to_lowercase()
                .as_str()
            {
                "create" | "created" => AgentFileStatus::Created,
                "delete" | "deleted" | "remove" | "removed" => AgentFileStatus::Deleted,
                _ => AgentFileStatus::Modified,
            };
            self.file_badges.insert(path, status);
        }

        let message = event_message(event);
        self.events.push(AgentEventItem {
            id: if event.id.is_empty() {
                format!("event-{}", self.events.len() + 1)
            } else {
                event.id.clone()
            },
            time: format_time(&event.created_at),
            kind: event.kind.clone(),
            message,
        });
        if self.events.len() > 200 {
            self.events.remove(0);
        }
    }

    pub fn apply_to_app_state(&self, state: &mut AppState) {
        if let Some(status) = self.status {
            state.agent_status = status;
        }
        apply_badges_to_node(&mut state.tree, &self.file_badges, &state.workspace_root);
        state.flattened_tree = crate::services::workspace::flatten_tree(&state.tree);
        for tab in &mut state.tabs {
            tab.is_agent_modified = self
                .file_badges
                .get(&tab.rel_path)
                .is_some_and(|status| *status != AgentFileStatus::Deleted);
        }
        let mut changed_files: Vec<ChangedFile> = self
            .file_badges
            .iter()
            .map(|(path, status)| ChangedFile {
                path: path.clone(),
                status: *status,
            })
            .collect();
        changed_files.sort_by(|left, right| left.path.cmp(&right.path));
        state.changed_files = changed_files;
        state.agent_events = self.events.clone();
        if !self.log_lines.is_empty() {
            state.agent_log = self.log_lines.join("\n");
        }
    }
}

pub fn apply_daemon_snapshot(state: &mut AppState, snapshot: &DaemonSnapshot) {
    state.connection_status = ConnectionStatus::Connected;
    state.connection_message = format!("Protocol {}", snapshot.protocol_version);
    state.active_run_id = snapshot.selected_run.as_ref().map(|run| run.run_id.clone());
    state.active_run_phase = snapshot
        .selected_run
        .as_ref()
        .map(|run| run.phase.clone())
        .unwrap_or_default();
    state.pending_permissions = snapshot
        .selected_run
        .as_ref()
        .map(|run| run.pending_permissions.clone())
        .unwrap_or_default();

    let mut projection = RunProjection::new();
    projection.active_run_id = state.active_run_id.clone();
    projection.status = snapshot
        .selected_run
        .as_ref()
        .map(|run| agent_status(&run.status));
    for event in &snapshot.events {
        projection.apply_daemon_event(event);
    }
    projection.apply_to_app_state(state);

    if snapshot.events.is_empty() {
        state.agent_log = match snapshot.selected_run.as_ref() {
            Some(run) => format!("Run {} · {}", run.run_id, run.status),
            None => "Connected · no runs yet".to_string(),
        };
    }
}

pub fn apply_client_error(state: &mut AppState, error: String) {
    state.connection_status = ConnectionStatus::Disconnected;
    state.connection_message = error.clone();
    state.active_run_id = None;
    state.active_run_phase.clear();
    state.pending_permissions.clear();
    state.changed_files.clear();
    state.agent_status = AgentStatus::Disconnected;
    state.agent_events.clear();
    state.agent_log = error;
}

fn agent_status(status: &str) -> AgentStatus {
    match status {
        "running" => AgentStatus::Working,
        "awaiting_approval" => AgentStatus::AwaitingApproval,
        "complete" | "completed" => AgentStatus::Completed,
        "failed" | "cancelled" => AgentStatus::Failed,
        _ => AgentStatus::Idle,
    }
}

fn payload_text(payload: &Value, key: &str) -> Option<String> {
    let value = match payload.get(key)? {
        Value::String(value) => value.clone(),
        Value::Number(value) => value.to_string(),
        _ => return None,
    };
    (!value.is_empty()).then_some(value)
}

fn event_message(event: &crate::client::protocol::DaemonEvent) -> String {
    match event.kind.as_str() {
        "run.started" => format!(
            "Run started · {}",
            payload_text(&event.payload, "goal").unwrap_or_else(|| "Goal received".to_string())
        ),
        "text_delta" => truncate(
            &payload_text(&event.payload, "text").unwrap_or_default(),
            140,
        ),
        "reasoning_delta" => format!(
            "Thinking · {}",
            truncate(
                &payload_text(&event.payload, "text").unwrap_or_default(),
                120
            )
        ),
        "tool_call_ready" => format!(
            "Tool · {}",
            payload_text(&event.payload, "name").unwrap_or_else(|| "tool".to_string())
        ),
        "file.changed" => {
            let path = payload_text(&event.payload, "path").unwrap_or_else(|| "file".to_string());
            let operation =
                payload_text(&event.payload, "operation").unwrap_or_else(|| "modified".to_string());
            format!("{path} · {operation}")
        }
        "permission_wait" => format!(
            "Approval required · {}",
            payload_text(&event.payload, "tool").unwrap_or_else(|| "tool".to_string())
        ),
        "run.paused" => format!(
            "Run paused · {}",
            payload_text(&event.payload, "status").unwrap_or_else(|| "waiting".to_string())
        ),
        "run.finished" => format!(
            "Run finished · {}",
            payload_text(&event.payload, "status").unwrap_or_else(|| "complete".to_string())
        ),
        "usage" => {
            let input =
                payload_text(&event.payload, "prompt_tokens").unwrap_or_else(|| "0".to_string());
            let output = payload_text(&event.payload, "completion_tokens")
                .unwrap_or_else(|| "0".to_string());
            let cost = payload_text(&event.payload, "cost_usd");
            match cost {
                Some(cost) => format!("Usage · in {input} · out {output} · ${cost}"),
                None => format!("Usage · in {input} · out {output}"),
            }
        }
        _ => event.kind.replace(['.', '_'], " "),
    }
}

fn truncate(value: &str, max_chars: usize) -> String {
    let value = value.trim();
    if value.chars().count() <= max_chars {
        return value.to_string();
    }
    let mut output: String = value.chars().take(max_chars.saturating_sub(1)).collect();
    output.push('…');
    output
}

fn format_time(value: &str) -> String {
    if let Some(time) = value.split_once('T').map(|(_, value)| value) {
        return time.trim_end_matches('Z').chars().take(8).collect();
    }
    value.to_string()
}

fn apply_badges_to_node(
    node: &mut TreeNode,
    badges: &HashMap<String, AgentFileStatus>,
    root: &Path,
) {
    if !node.is_dir
        && let Ok(relative_path) = node.path.strip_prefix(root)
    {
        node.agent_status = badges
            .get(&relative_path.to_string_lossy().to_string())
            .copied();
    }
    for child in &mut node.children {
        apply_badges_to_node(child, badges, root);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::client::protocol::{DaemonEvent, DaemonRun, DaemonSnapshot};
    use serde_json::json;
    use std::path::PathBuf;

    #[test]
    fn formats_runtime_usage_fields() {
        let event = DaemonEvent {
            id: "usage-1".to_string(),
            run_id: "R-1".to_string(),
            kind: "usage".to_string(),
            payload: json!({
                "prompt_tokens": 120,
                "completion_tokens": 30,
                "cost_usd": 0.0042
            }),
            created_at: "2026-09-23T12:00:00Z".to_string(),
        };
        assert_eq!(event_message(&event), "Usage · in 120 · out 30 · $0.0042");
    }

    #[test]
    fn applies_real_daemon_snapshot() {
        let mut state = AppState::new(PathBuf::from("/workspace"));
        let snapshot = DaemonSnapshot {
            protocol_version: "0.4.0".to_string(),
            selected_run: Some(DaemonRun {
                run_id: "R-1".to_string(),
                status: "running".to_string(),
                phase: "execute_tool".to_string(),
                pending_permissions: vec!["permission-1".to_string()],
            }),
            runs: vec![DaemonRun {
                run_id: "R-1".to_string(),
                status: "running".to_string(),
                phase: "execute_tool".to_string(),
                pending_permissions: vec!["permission-1".to_string()],
            }],
            events: vec![
                DaemonEvent {
                    id: "event-1".to_string(),
                    run_id: "R-1".to_string(),
                    kind: "run.started".to_string(),
                    payload: json!({"goal": "repair viewer"}),
                    created_at: "2026-09-23T12:00:00Z".to_string(),
                },
                DaemonEvent {
                    id: "event-2".to_string(),
                    run_id: "R-1".to_string(),
                    kind: "file.changed".to_string(),
                    payload: json!({"path": "src/main.rs", "operation": "update"}),
                    created_at: "2026-09-23T12:00:01Z".to_string(),
                },
            ],
        };

        apply_daemon_snapshot(&mut state, &snapshot);
        assert_eq!(state.connection_status, ConnectionStatus::Connected);
        assert_eq!(state.active_run_id.as_deref(), Some("R-1"));
        assert_eq!(state.active_run_phase, "execute_tool");
        assert_eq!(state.agent_status, AgentStatus::Working);
        assert_eq!(state.changed_files.len(), 1);
        assert_eq!(state.changed_files[0].path, "src/main.rs");
        assert_eq!(state.changed_files[0].status, AgentFileStatus::Modified);
        assert_eq!(state.pending_permissions, vec!["permission-1"]);
        assert_eq!(state.agent_events.len(), 2);
        assert_eq!(state.agent_events[1].message, "src/main.rs · update");
    }
}
