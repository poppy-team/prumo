use std::path::{Path, PathBuf};
use crate::client::Event;
use crate::document::DocumentStore;

#[derive(Debug, Clone)]
pub struct AgentFollowController {
    pub enabled: bool,
    pub is_user_editing: bool,
    pub last_followed_path: Option<PathBuf>,
    pub last_followed_line: Option<usize>,
}

impl Default for AgentFollowController {
    fn default() -> Self {
        Self {
            enabled: true, // Follow Agent enabled by default
            is_user_editing: false,
            last_followed_path: None,
            last_followed_line: None,
        }
    }
}

impl AgentFollowController {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn toggle(&mut self) {
        self.enabled = !self.enabled;
    }

    pub fn notify_user_edit_started(&mut self) {
        self.is_user_editing = true;
    }

    pub fn notify_user_edit_idle(&mut self) {
        self.is_user_editing = false;
    }

    /// Process an agent event and auto-navigate if follow mode is active.
    /// Invariant: If disabled or if the user is actively typing, never steal focus!
    pub fn handle_event(&mut self, event: &Event, workspace_root: &Path, store: &mut DocumentStore) {
        if !self.enabled || self.is_user_editing {
            return;
        }

        let payload = &event.payload;

        // Try extracting target file
        let mut target_path = None;
        if let Some(p) = payload.get("path").and_then(|v| v.as_str()) {
            target_path = Some(p.to_string());
        } else if let Some(target) = payload.get("TargetFile").and_then(|v| v.as_str()) {
            target_path = Some(target.to_string());
        } else if let Some(args) = payload.get("arguments").and_then(|v| v.as_object()) {
            if let Some(p) = args.get("path").or_else(|| args.get("TargetFile")).and_then(|v| v.as_str()) {
                target_path = Some(p.to_string());
            }
        }

        let file_path = match target_path {
            Some(raw) => {
                let p = Path::new(&raw);
                if p.is_absolute() {
                    p.to_path_buf()
                } else {
                    workspace_root.join(p)
                }
            }
            None => return,
        };

        // Extract line and range if available
        let mut start_line = payload.get("line").or_else(|| payload.get("StartLine")).and_then(|v| v.as_u64()).map(|n| n as usize);
        let mut end_line = payload.get("end_line").or_else(|| payload.get("EndLine")).and_then(|v| v.as_u64()).map(|n| n as usize);

        if let Some(args) = payload.get("arguments").and_then(|v| v.as_object()) {
            if start_line.is_none() {
                start_line = args.get("line").or_else(|| args.get("StartLine")).and_then(|v| v.as_u64()).map(|n| n as usize);
            }
            if end_line.is_none() {
                end_line = args.get("end_line").or_else(|| args.get("EndLine")).and_then(|v| v.as_u64()).map(|n| n as usize);
            }
        }

        // Open the file in the document store
        if let Ok(doc) = store.open_file(&file_path) {
            self.last_followed_path = Some(file_path);

            if let Some(start) = start_line {
                let end = end_line.unwrap_or(start);
                doc.highlight_lines(start, end);
                self.last_followed_line = Some(start);
            }
        }
    }
}
