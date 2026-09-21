use std::collections::HashMap;
use std::path::{Path, PathBuf};
use egui::Color32;
use crate::client::Event;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FileBadgeKind {
    Created,  // + (Green)
    Modified, // M (Yellow)
    Deleted,  // D (Red)
    Renamed,  // R (Purple)
    Working,  // ● (Cyan/Pulse)
    Verified, // ✓ (Green)
    Failed,   // ! (Red Alert)
}

impl FileBadgeKind {
    pub fn label(&self) -> &'static str {
        match self {
            FileBadgeKind::Created => "+",
            FileBadgeKind::Modified => "M",
            FileBadgeKind::Deleted => "D",
            FileBadgeKind::Renamed => "R",
            FileBadgeKind::Working => "●",
            FileBadgeKind::Verified => "✓",
            FileBadgeKind::Failed => "!",
        }
    }

    pub fn color(&self) -> Color32 {
        match self {
            FileBadgeKind::Created => Color32::from_rgb(70, 200, 100),
            FileBadgeKind::Modified => Color32::from_rgb(230, 180, 50),
            FileBadgeKind::Deleted => Color32::from_rgb(220, 70, 70),
            FileBadgeKind::Renamed => Color32::from_rgb(180, 100, 230),
            FileBadgeKind::Working => Color32::from_rgb(80, 210, 255),
            FileBadgeKind::Verified => Color32::from_rgb(60, 220, 120),
            FileBadgeKind::Failed => Color32::from_rgb(255, 60, 60),
        }
    }

    pub fn tooltip(&self) -> &'static str {
        match self {
            FileBadgeKind::Created => "Created in current run",
            FileBadgeKind::Modified => "Modified in current run",
            FileBadgeKind::Deleted => "Deleted in current run",
            FileBadgeKind::Renamed => "Renamed in current run",
            FileBadgeKind::Working => "Agent is currently editing this file",
            FileBadgeKind::Verified => "Verified by tests / quality gate",
            FileBadgeKind::Failed => "Verification failed on this file",
        }
    }
}

#[derive(Debug, Default, Clone)]
pub struct RunFileProjection {
    pub run_id: String,
    pub badges: HashMap<PathBuf, FileBadgeKind>,
    pub currently_working_file: Option<PathBuf>,
    pub recent_files: Vec<PathBuf>,
}

impl RunFileProjection {
    pub fn new(run_id: impl Into<String>) -> Self {
        Self {
            run_id: run_id.into(),
            badges: HashMap::new(),
            currently_working_file: None,
            recent_files: Vec::new(),
        }
    }

    pub fn reset(&mut self, new_run_id: &str) {
        self.run_id = new_run_id.to_string();
        self.badges.clear();
        self.currently_working_file = None;
        self.recent_files.clear();
    }

    pub fn get_badge(&self, path: &Path) -> Option<FileBadgeKind> {
        // If it's the currently working file, show Working badge
        if let Some(working) = &self.currently_working_file {
            if working == path {
                return Some(FileBadgeKind::Working);
            }
        }
        self.badges.get(path).copied()
    }

    pub fn record_mutation(&mut self, path: PathBuf, kind: FileBadgeKind) {
        if !self.recent_files.contains(&path) {
            self.recent_files.push(path.clone());
        }
        self.badges.insert(path, kind);
    }

    pub fn process_event(&mut self, event: &Event, workspace_root: &Path) {
        let payload = &event.payload;

        // Check if event refers to a file path
        let mut extracted_path = None;
        if let Some(p) = payload.get("path").and_then(|v| v.as_str()) {
            extracted_path = Some(p.to_string());
        } else if let Some(target) = payload.get("TargetFile").and_then(|v| v.as_str()) {
            extracted_path = Some(target.to_string());
        } else if let Some(args) = payload.get("arguments").and_then(|v| v.as_object()) {
            if let Some(p) = args.get("path").or_else(|| args.get("TargetFile")).and_then(|v| v.as_str()) {
                extracted_path = Some(p.to_string());
            }
        }

        // Events that clear active working file
        match event.kind.as_str() {
            "tool_result" | "run.finished" | "run.completed" | "run.failed" | "run.cancelled" => {
                self.currently_working_file = None;
            }
            _ => {}
        }

        let file_path = if let Some(raw) = extracted_path {
            let p = Path::new(&raw);
            if p.is_absolute() {
                p.to_path_buf()
            } else {
                workspace_root.join(p)
            }
        } else {
            return;
        };

        // Determine badge based on event kind
        match event.kind.as_str() {
            "tool_call_delta" | "tool_call_ready" => {
                let name = payload.get("name").and_then(|v| v.as_str()).unwrap_or("");
                if name.contains("write") || name.contains("create") {
                    self.currently_working_file = Some(file_path.clone());
                    self.record_mutation(file_path, FileBadgeKind::Created);
                } else if name.contains("edit") || name.contains("replace") || name.contains("modify") {
                    self.currently_working_file = Some(file_path.clone());
                    self.record_mutation(file_path, FileBadgeKind::Modified);
                }
            }
            "gate.passed" | "verification.passed" => {
                self.badges.insert(file_path, FileBadgeKind::Verified);
            }
            "gate.failed" | "verification.failed" => {
                self.badges.insert(file_path, FileBadgeKind::Failed);
            }
            _ => {}
        }
    }
}
