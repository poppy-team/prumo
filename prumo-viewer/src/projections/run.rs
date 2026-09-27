use serde_json::Value;
use std::collections::HashMap;
use std::path::Path;

use crate::client::protocol::DaemonSnapshot;
use crate::state::{
    AgentEventItem, AgentFileStatus, AgentStatus, AppState, ChangedFile, ConnectionStatus,
    PendingPermissionItem, TreeNode,
};

#[derive(Debug, Clone, Default)]
pub struct RunProjection {
    pub active_run_id: Option<String>,
    pub status: Option<AgentStatus>,
    pub file_badges: HashMap<String, AgentFileStatus>,
    pub events: Vec<AgentEventItem>,
    pub log_lines: Vec<String>,
    pub pending_permissions: Vec<PendingPermissionItem>,
    pub metrics: crate::state::AgentMetrics,
}

impl RunProjection {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn apply_daemon_event(&mut self, event: &crate::client::protocol::DaemonEvent) {
        if event.kind == "file.changed"
            && let Some(path) = payload_text(&event.payload, "path")
        {
            let status = AgentFileStatus::from_operation(
                &payload_text(&event.payload, "operation").unwrap_or_default(),
            );
            self.file_badges.insert(path, status);
        }

        if event.kind == "usage" || event.kind == "budget.tick" {
            let input = payload_u64(&event.payload, "prompt_tokens")
                .or_else(|| payload_u64(&event.payload, "delta_input_tokens"))
                .unwrap_or(0);
            let output = payload_u64(&event.payload, "completion_tokens")
                .or_else(|| payload_u64(&event.payload, "delta_output_tokens"))
                .unwrap_or(0);
            let cache_read = payload_u64(&event.payload, "cache_read_tokens")
                .or_else(|| payload_u64(&event.payload, "delta_cache_read_tokens"))
                .unwrap_or(0);
            let cache_write = payload_u64(&event.payload, "cache_write_tokens")
                .or_else(|| payload_u64(&event.payload, "delta_cache_write_tokens"))
                .unwrap_or(0);
            let cost = payload_f64(&event.payload, "cost_usd")
                .or_else(|| payload_f64(&event.payload, "delta_cost_usd"))
                .or_else(|| payload_f64(&event.payload, "total_cost_usd"));

            self.metrics.input_tokens = self.metrics.input_tokens.saturating_add(input);
            self.metrics.output_tokens = self.metrics.output_tokens.saturating_add(output);
            self.metrics.cache_read_tokens =
                self.metrics.cache_read_tokens.saturating_add(cache_read);
            self.metrics.cache_write_tokens =
                self.metrics.cache_write_tokens.saturating_add(cache_write);
            if let Some(cost) = cost {
                self.metrics.total_cost_usd += cost;
            }
        }
        if event.kind == "tool_call_ready" || event.kind == "tool.call" {
            self.metrics.tool_calls += 1;
        }

        if event.kind == "permission_wait" {
            let tool = payload_text(&event.payload, "tool").unwrap_or_else(|| "tool".to_string());
            let request_id = payload_text(&event.payload, "request_id").unwrap_or_default();
            let fingerprint = payload_text(&event.payload, "fingerprint").unwrap_or_default();
            let arguments_preview = summarize_arguments(&event.payload);
            if !request_id.is_empty() {
                self.pending_permissions.push(PendingPermissionItem {
                    request_id,
                    tool,
                    fingerprint,
                    arguments_preview,
                });
            }
        }

        if event.kind == "text_delta" || event.kind == "reasoning_delta" {
            return;
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
        if self.events.len() > 250 {
            self.events.drain(..50);
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
        state.agent_metrics = self.metrics.clone();
        if !self.log_lines.is_empty() {
            state.agent_log = self.log_lines.join("\n");
        }
        let status = state.agent_status;
        let phase = state.active_run_phase.clone();
        let changed = state.changed_files.clone();
        let perms = state.pending_permissions.clone();
        let details = state.pending_permission_details.clone();
        let log = state.agent_log.clone();
        if let Some(session) = state.active_session_mut() {
            session.events = self.events.clone();
            session.status = status;
            session.phase = phase;
            session.metrics = self.metrics.clone();
            session.changed_files = changed;
            session.pending_permissions = perms;
            session.pending_permission_details = details;
            if !log.is_empty() {
                session.agent_log = log;
            }
        }
    }
}

pub fn apply_daemon_snapshot(state: &mut AppState, snapshot: &DaemonSnapshot) {
    state.connection_status = ConnectionStatus::Connected;
    state.connection_message = format!("Protocol {}", snapshot.protocol_version);
    state.track_run(snapshot.selected_run.as_ref().map(|run| run.run_id.clone()));
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

    let mut details = Vec::new();
    if let Some(run) = snapshot.selected_run.as_ref() {
        for req_id in &run.pending_permissions {
            let fp = run
                .permission_fingerprints
                .get(req_id)
                .cloned()
                .unwrap_or_default();
            if let Some(existing) = projection
                .pending_permissions
                .iter()
                .find(|p| &p.request_id == req_id)
            {
                let mut item = existing.clone();
                if item.fingerprint.is_empty() && !fp.is_empty() {
                    item.fingerprint = fp;
                }
                details.push(item);
            } else {
                details.push(PendingPermissionItem {
                    request_id: req_id.clone(),
                    tool: "tool".to_string(),
                    fingerprint: fp,
                    arguments_preview: String::new(),
                });
            }
        }
    }
    state.pending_permission_details = details;

    if snapshot.events.is_empty() {
        state.agent_log = match snapshot.selected_run.as_ref() {
            Some(run) => format!("Run {} · {}", run.run_id, run.status),
            None => "Connected · no runs yet".to_string(),
        };
    }
}

pub fn apply_streaming_event(state: &mut AppState, event: &crate::client::protocol::DaemonEvent) {
    if event.kind == "file.changed"
        && let Some(path) = payload_text(&event.payload, "path")
    {
        let status = AgentFileStatus::from_operation(
            &payload_text(&event.payload, "operation").unwrap_or_default(),
        );
        let mut badges = HashMap::new();
        badges.insert(path.clone(), status);
        apply_badges_to_node(&mut state.tree, &badges, &state.workspace_root);
        state.flattened_tree = crate::services::workspace::flatten_tree(&state.tree);
        for tab in &mut state.tabs {
            if tab.rel_path == path {
                tab.is_agent_modified = status != AgentFileStatus::Deleted;
            }
        }
        if let Some(cf) = state.changed_files.iter_mut().find(|c| c.path == path) {
            cf.status = status;
        } else {
            state.changed_files.push(ChangedFile {
                path: path.clone(),
                status,
            });
            state.changed_files.sort_by(|left, right| left.path.cmp(&right.path));
        }
    }

    if event.kind == "permission_wait" {
        let tool = payload_text(&event.payload, "tool").unwrap_or_else(|| "tool".to_string());
        let request_id = payload_text(&event.payload, "request_id").unwrap_or_default();
        let fingerprint = payload_text(&event.payload, "fingerprint").unwrap_or_default();
        let arguments_preview = summarize_arguments(&event.payload);
        if !request_id.is_empty() {
            if !state.pending_permissions.contains(&request_id) {
                state.pending_permissions.push(request_id.clone());
            }
            if let Some(item) = state
                .pending_permission_details
                .iter_mut()
                .find(|p| p.request_id == request_id)
            {
                if item.fingerprint.is_empty() && !fingerprint.is_empty() {
                    item.fingerprint = fingerprint;
                }
            } else {
                state.pending_permission_details.push(PendingPermissionItem {
                    request_id,
                    tool,
                    fingerprint,
                    arguments_preview,
                });
            }
            state.agent_status = AgentStatus::AwaitingApproval;
        }
    }

    if event.kind == "usage" || event.kind == "budget.tick" {
        let input = payload_u64(&event.payload, "prompt_tokens")
            .or_else(|| payload_u64(&event.payload, "delta_input_tokens"))
            .unwrap_or(0);
        let output = payload_u64(&event.payload, "completion_tokens")
            .or_else(|| payload_u64(&event.payload, "delta_output_tokens"))
            .unwrap_or(0);
        let cache_read = payload_u64(&event.payload, "cache_read_tokens")
            .or_else(|| payload_u64(&event.payload, "delta_cache_read_tokens"))
            .unwrap_or(0);
        let cache_write = payload_u64(&event.payload, "cache_write_tokens")
            .or_else(|| payload_u64(&event.payload, "delta_cache_write_tokens"))
            .unwrap_or(0);
        let cost = payload_f64(&event.payload, "cost_usd")
            .or_else(|| payload_f64(&event.payload, "delta_cost_usd"))
            .or_else(|| payload_f64(&event.payload, "total_cost_usd"));

        state.agent_metrics.input_tokens = state.agent_metrics.input_tokens.saturating_add(input);
        state.agent_metrics.output_tokens = state.agent_metrics.output_tokens.saturating_add(output);
        state.agent_metrics.cache_read_tokens =
            state.agent_metrics.cache_read_tokens.saturating_add(cache_read);
        state.agent_metrics.cache_write_tokens =
            state.agent_metrics.cache_write_tokens.saturating_add(cache_write);
        if let Some(cost) = cost {
            state.agent_metrics.total_cost_usd += cost;
        }
        let metrics = state.agent_metrics.clone();
        if let Some(session) = state.active_session_mut() {
            session.metrics = metrics;
        }
    }
    if event.kind == "tool_call_ready" || event.kind == "tool.call" {
        state.agent_metrics.tool_calls += 1;
        let tool_calls = state.agent_metrics.tool_calls;
        if let Some(session) = state.active_session_mut() {
            session.metrics.tool_calls = tool_calls;
        }
    }

    if event.kind == "text_delta" {
        if let Some(text) = payload_text(&event.payload, "text") {
            state.append_assistant_chunk(&text);
        }
        return;
    }

    if event.kind == "reasoning_delta" {
        if let Some(text) = payload_text(&event.payload, "text") {
            state.append_reasoning_chunk(&text);
        }
        return;
    }

    if event.kind == "run.started" {
        state.agent_status = AgentStatus::Working;
        if let Some(goal) = payload_text(&event.payload, "goal") {
            if !state.agent_messages.iter().any(|m| m.content == goal) {
                state.add_user_message(&goal);
                state.start_assistant_streaming_message();
            }
            if let Some(session) = state.active_session_mut()
                && session.title.starts_with("Chat ") {
                    session.title = truncate(&goal, 20);
                }
        }
    } else if event.kind == "run.paused" {
        state.agent_status = if state.pending_permissions.is_empty() {
            AgentStatus::Idle
        } else {
            AgentStatus::AwaitingApproval
        };
    } else if event.kind == "run.finished" {
        let status = payload_text(&event.payload, "status").unwrap_or_default();
        state.agent_status = match status.as_str() {
            "complete" | "completed" => AgentStatus::Completed,
            _ => AgentStatus::Failed,
        };
        state.pending_permissions.clear();
        state.pending_permission_details.clear();
        state.finish_assistant_streaming();
    }

    let message = event_message(event);
    let event_id = if event.id.is_empty() {
        format!("event-{}", state.agent_events.len() + 1)
    } else {
        event.id.clone()
    };
    if !state.agent_events.iter().any(|item| item.id == event_id) {
        state.agent_events.push(AgentEventItem {
            id: event_id,
            time: format_time(&event.created_at),
            kind: event.kind.clone(),
            message: message.clone(),
        });
        if state.agent_events.len() > 250 {
            state.agent_events.drain(..50);
        }
    }
    let events = state.agent_events.clone();
    let status = state.agent_status;
    let phase = state.active_run_phase.clone();
    let changed = state.changed_files.clone();
    let perms = state.pending_permissions.clone();
    let details = state.pending_permission_details.clone();
    let log = state.agent_log.clone();
    if let Some(session) = state.active_session_mut() {
        session.events = events;
        session.status = status;
        session.phase = phase;
        session.changed_files = changed;
        session.pending_permissions = perms;
        session.pending_permission_details = details;
        session.agent_log = log;
    }
}

pub fn apply_client_error(state: &mut AppState, error: String) {
    state.connection_status = ConnectionStatus::Disconnected;
    state.connection_message = error.clone();
    state.active_run_id = None;
    state.active_run_phase.clear();
    state.pending_permissions.clear();
    state.pending_permission_details.clear();
    state.changed_files.clear();
    state.agent_status = AgentStatus::Disconnected;
    state.agent_events.clear();
    state.finish_assistant_streaming();
    state.add_system_message(&format!("⚠️ Connection error: {}", error));
    if let Some(session) = state.active_session_mut() {
        session.status = AgentStatus::Disconnected;
    }
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

fn payload_u64(payload: &Value, key: &str) -> Option<u64> {
    match payload.get(key)? {
        Value::Number(num) => num.as_u64().or_else(|| num.as_f64().map(|f| f as u64)),
        Value::String(s) => s.parse().ok(),
        _ => None,
    }
}

fn payload_f64(payload: &Value, key: &str) -> Option<f64> {
    match payload.get(key)? {
        Value::Number(num) => num.as_f64(),
        Value::String(s) => s.parse().ok(),
        _ => None,
    }
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

fn summarize_arguments(payload: &Value) -> String {
    if let Some(arguments) = payload.get("arguments") {
        if let Some(cmd) = arguments.get("command").and_then(Value::as_str) {
            return cmd.to_string();
        }
        if let Some(path) = arguments.get("path").and_then(Value::as_str) {
            return format!("path: {path}");
        }
        if let Some(s) = arguments.as_str() {
            return s.to_string();
        }
        return arguments.to_string();
    }
    String::new()
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
                permission_fingerprints: HashMap::from([("permission-1".to_string(), "fp-123".to_string())]),
            }),
            runs: vec![DaemonRun {
                run_id: "R-1".to_string(),
                status: "running".to_string(),
                phase: "execute_tool".to_string(),
                pending_permissions: vec!["permission-1".to_string()],
                permission_fingerprints: HashMap::from([("permission-1".to_string(), "fp-123".to_string())]),
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
        assert_eq!(state.pending_permission_details.len(), 1);
        assert_eq!(state.pending_permission_details[0].fingerprint, "fp-123");
        assert_eq!(state.agent_events.len(), 2);
        assert_eq!(state.agent_events[1].message, "src/main.rs · update");
    }

    #[test]
    fn applies_streaming_event_incrementally() {
        let mut state = AppState::new(PathBuf::from("/workspace"));
        state.active_run_id = Some("R-STREAM".to_string());

        // 1. run.started
        apply_streaming_event(
            &mut state,
            &DaemonEvent {
                id: "e-1".to_string(),
                run_id: "R-STREAM".to_string(),
                kind: "run.started".to_string(),
                payload: json!({"goal": "refactor stream"}),
                created_at: "2026-09-26T12:00:00Z".to_string(),
            },
        );
        assert_eq!(state.agent_status, AgentStatus::Working);
        assert_eq!(state.agent_events.len(), 1);

        // 2. permission_wait
        apply_streaming_event(
            &mut state,
            &DaemonEvent {
                id: "e-2".to_string(),
                run_id: "R-STREAM".to_string(),
                kind: "permission_wait".to_string(),
                payload: json!({
                    "tool": "bash",
                    "request_id": "req-99",
                    "fingerprint": "sha256-abc",
                    "arguments": {"command": "cargo test"}
                }),
                created_at: "2026-09-26T12:00:01Z".to_string(),
            },
        );
        assert_eq!(state.agent_status, AgentStatus::AwaitingApproval);
        assert_eq!(state.pending_permissions, vec!["req-99"]);
        assert_eq!(state.pending_permission_details.len(), 1);
        assert_eq!(state.pending_permission_details[0].tool, "bash");
        assert_eq!(state.pending_permission_details[0].fingerprint, "sha256-abc");
        assert_eq!(state.pending_permission_details[0].arguments_preview, "cargo test");

        // 3. file.changed
        apply_streaming_event(
            &mut state,
            &DaemonEvent {
                id: "e-3".to_string(),
                run_id: "R-STREAM".to_string(),
                kind: "file.changed".to_string(),
                payload: json!({"path": "src/lib.rs", "operation": "create"}),
                created_at: "2026-09-26T12:00:02Z".to_string(),
            },
        );
        assert_eq!(state.changed_files.len(), 1);
        assert_eq!(state.changed_files[0].path, "src/lib.rs");
        assert_eq!(state.changed_files[0].status, AgentFileStatus::Created);

        // 4. run.finished
        apply_streaming_event(
            &mut state,
            &DaemonEvent {
                id: "e-4".to_string(),
                run_id: "R-STREAM".to_string(),
                kind: "run.finished".to_string(),
                payload: json!({"status": "complete"}),
                created_at: "2026-09-26T12:00:03Z".to_string(),
            },
        );
        assert_eq!(state.agent_status, AgentStatus::Completed);
        assert!(state.pending_permissions.is_empty());
        assert!(state.pending_permission_details.is_empty());
        assert_eq!(state.agent_events.len(), 4);
    }

    #[test]
    fn test_streaming_usage_and_session_sync() {
        let mut state = AppState::new(PathBuf::from("/workspace"));
        state.create_new_session();
        assert_eq!(state.agent_sessions.len(), 2);
        assert_eq!(state.active_session_index, 1);

        // 1. run.started
        apply_streaming_event(
            &mut state,
            &DaemonEvent {
                id: "ev-1".to_string(),
                run_id: "R-MULTI".to_string(),
                kind: "run.started".to_string(),
                payload: json!({"goal": "Refactor token counter"}),
                created_at: "2026-09-26T14:00:00Z".to_string(),
            },
        );
        assert_eq!(state.agent_status, AgentStatus::Working);
        assert_eq!(state.active_session().unwrap().title, "Refactor token coun…");

        // 2. usage event
        apply_streaming_event(
            &mut state,
            &DaemonEvent {
                id: "ev-2".to_string(),
                run_id: "R-MULTI".to_string(),
                kind: "usage".to_string(),
                payload: json!({
                    "prompt_tokens": 1250,
                    "completion_tokens": 340,
                    "cache_read_tokens": 800,
                    "cache_write_tokens": 150,
                    "cost_usd": 0.0042
                }),
                created_at: "2026-09-26T14:00:01Z".to_string(),
            },
        );
        assert_eq!(state.agent_metrics.input_tokens, 1250);
        assert_eq!(state.agent_metrics.output_tokens, 340);
        assert_eq!(state.agent_metrics.cache_read_tokens, 800);
        assert_eq!(state.agent_metrics.cache_write_tokens, 150);
        assert!((state.agent_metrics.total_cost_usd - 0.0042).abs() < 1e-6);

        // Check active session received metrics
        let session = state.active_session().unwrap();
        assert_eq!(session.metrics.input_tokens, 1250);
        assert_eq!(session.metrics.output_tokens, 340);
        assert_eq!(session.metrics.cache_read_tokens, 800);

        // 3. tool_call_ready event
        apply_streaming_event(
            &mut state,
            &DaemonEvent {
                id: "ev-3".to_string(),
                run_id: "R-MULTI".to_string(),
                kind: "tool_call_ready".to_string(),
                payload: json!({"tool": "read_file"}),
                created_at: "2026-09-26T14:00:02Z".to_string(),
            },
        );
        assert_eq!(state.agent_metrics.tool_calls, 1);
        assert_eq!(state.active_session().unwrap().metrics.tool_calls, 1);
    }
}
