use crate::services::config::ViewerConfig;
use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum FileKind {
    Folder,
    Rust,
    Go,
    Markdown,
    Json,
    Yaml,
    Toml,
    TypeScript,
    JavaScript,
    Manifest,
    Other,
}

impl FileKind {
    pub fn from_path(path: &Path) -> Self {
        if path.is_dir() {
            return FileKind::Folder;
        }

        let file_name = path
            .file_name()
            .and_then(|name| name.to_str())
            .unwrap_or("")
            .to_lowercase();

        if matches!(
            file_name.as_str(),
            "go.mod" | "go.sum" | "makefile" | "dockerfile"
        ) {
            return FileKind::Manifest;
        }

        match path.extension().and_then(|extension| extension.to_str()) {
            Some("rs") => FileKind::Rust,
            Some("go") => FileKind::Go,
            Some("md" | "markdown") => FileKind::Markdown,
            Some("json") => FileKind::Json,
            Some("yaml" | "yml") => FileKind::Yaml,
            Some("toml") => FileKind::Toml,
            Some("ts" | "tsx" | "mts" | "cts") => FileKind::TypeScript,
            Some("js" | "jsx" | "mjs" | "cjs") => FileKind::JavaScript,
            _ => FileKind::Other,
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum AgentFileStatus {
    Created,
    Modified,
    Deleted,
}

impl AgentFileStatus {
    pub fn marker(self) -> &'static str {
        match self {
            Self::Created => "+",
            Self::Modified => "M",
            Self::Deleted => "D",
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ChangedFile {
    pub path: String,
    pub status: AgentFileStatus,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SidebarView {
    Explorer,
    Changes,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DockView {
    Activity,
    Changes,
    Terminal,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TreeNode {
    pub name: String,
    pub path: PathBuf,
    pub rel_path: String,
    pub is_dir: bool,
    pub kind: FileKind,
    pub children: Vec<TreeNode>,
    pub expanded: bool,
    pub agent_status: Option<AgentFileStatus>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct FlattenedTreeItem {
    pub path: PathBuf,
    pub rel_path: String,
    pub name: String,
    pub is_dir: bool,
    pub depth: usize,
    pub expanded: bool,
    pub kind: FileKind,
    pub agent_status: Option<AgentFileStatus>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DocumentTab {
    pub path: PathBuf,
    pub rel_path: String,
    pub title: String,
    pub is_dirty: bool,
    pub content: String,
    pub persisted_content: String,
    pub is_agent_modified: bool,
    pub kind: FileKind,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum AgentStatus {
    Disconnected,
    Idle,
    Working,
    AwaitingApproval,
    Completed,
    Failed,
}

impl AgentStatus {
    pub fn label(self) -> &'static str {
        match self {
            Self::Disconnected => "offline",
            Self::Idle => "idle",
            Self::Working => "working",
            Self::AwaitingApproval => "approval required",
            Self::Completed => "completed",
            Self::Failed => "failed",
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ConnectionStatus {
    Connecting,
    Connected,
    Disconnected,
}

impl ConnectionStatus {
    pub fn label(self) -> &'static str {
        match self {
            Self::Connecting => "connecting",
            Self::Connected => "connected",
            Self::Disconnected => "offline",
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum NoticeTone {
    Info,
    Success,
    Error,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct UiNotice {
    pub tone: NoticeTone,
    pub message: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AgentEventItem {
    pub id: String,
    pub time: String,
    pub kind: String,
    pub message: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct QuickOpenMatch {
    pub path: PathBuf,
    pub rel_path: String,
    pub score: i32,
}

#[derive(Debug, Clone, PartialEq)]
pub struct AppState {
    pub viewport_width: f32,
    pub config: ViewerConfig,
    pub config_open: bool,
    pub available_models: Vec<String>,
    pub terminal_output: String,
    pub terminal_running: bool,
    pub workspace_root: PathBuf,
    pub workspace_name: String,
    pub tree: TreeNode,
    pub flattened_tree: Vec<FlattenedTreeItem>,
    pub file_candidates: Vec<String>,
    pub changed_files: Vec<ChangedFile>,
    pub selected_path: Option<PathBuf>,
    pub tabs: Vec<DocumentTab>,
    pub active_tab_index: Option<usize>,
    pub connection_status: ConnectionStatus,
    pub connection_message: String,
    pub active_run_id: Option<String>,
    pub active_run_phase: String,
    pub pending_permissions: Vec<String>,
    pub agent_status: AgentStatus,
    pub agent_events: Vec<AgentEventItem>,
    pub agent_log: String,
    pub sidebar_visible: bool,
    pub sidebar_view: SidebarView,
    pub agent_panel_visible: bool,
    pub dock_open: bool,
    pub dock_view: DockView,
    pub quick_open_open: bool,
    pub pending_close_tab: Option<usize>,
    pub notice: Option<UiNotice>,
    pub diff_open: bool,
    pub diff_title: String,
    pub diff_lines: Vec<(String, char)>,
}

impl AppState {
    pub fn new(root: PathBuf) -> Self {
        let name = root
            .file_name()
            .and_then(|name| name.to_str())
            .unwrap_or("workspace")
            .to_string();

        let empty_tree = TreeNode {
            name: name.clone(),
            path: root.clone(),
            rel_path: String::new(),
            is_dir: true,
            kind: FileKind::Folder,
            children: Vec::new(),
            expanded: true,
            agent_status: None,
        };

        Self {
            viewport_width: 1440.,
            config: ViewerConfig::default(),
            config_open: false,
            available_models: Vec::new(),
            terminal_output: String::new(),
            terminal_running: false,
            workspace_root: root,
            workspace_name: name,
            tree: empty_tree,
            flattened_tree: Vec::new(),
            file_candidates: Vec::new(),
            changed_files: Vec::new(),
            selected_path: None,
            tabs: Vec::new(),
            active_tab_index: None,
            connection_status: ConnectionStatus::Connecting,
            connection_message: "Connecting to Prumo Core...".to_string(),
            active_run_id: None,
            active_run_phase: String::new(),
            pending_permissions: Vec::new(),
            agent_status: AgentStatus::Disconnected,
            agent_events: Vec::new(),
            agent_log: "Waiting for Prumo Core...".to_string(),
            sidebar_visible: true,
            sidebar_view: SidebarView::Explorer,
            agent_panel_visible: true,
            dock_open: false,
            dock_view: DockView::Activity,
            quick_open_open: false,
            pending_close_tab: None,
            notice: None,
            diff_open: false,
            diff_title: String::new(),
            diff_lines: Vec::new(),
        }
    }

    pub fn active_tab(&self) -> Option<&DocumentTab> {
        self.active_tab_index.and_then(|index| self.tabs.get(index))
    }

    pub fn active_tab_mut(&mut self) -> Option<&mut DocumentTab> {
        self.active_tab_index
            .and_then(|index| self.tabs.get_mut(index))
    }

    pub fn activate_path(&mut self, path: &Path) -> bool {
        if let Some(index) = self.tabs.iter().position(|tab| tab.path == path) {
            self.active_tab_index = Some(index);
            self.selected_path = Some(path.to_path_buf());
            return true;
        }
        false
    }

    pub fn update_active_content(&mut self, content: String) {
        if let Some(tab) = self.active_tab_mut() {
            tab.is_dirty = content != tab.persisted_content;
            tab.content = content;
        }
    }

    pub fn mark_active_saved(&mut self) {
        if let Some(tab) = self.active_tab_mut() {
            tab.persisted_content = tab.content.clone();
            tab.is_dirty = false;
        }
    }

    pub fn remove_tab(&mut self, index: usize) -> bool {
        if index >= self.tabs.len() {
            return false;
        }
        self.tabs.remove(index);
        if self.tabs.is_empty() {
            self.active_tab_index = None;
            self.selected_path = None;
        } else if let Some(active_index) = self.active_tab_index {
            self.active_tab_index = Some(match active_index.cmp(&index) {
                std::cmp::Ordering::Less => active_index,
                std::cmp::Ordering::Equal => index.min(self.tabs.len() - 1),
                std::cmp::Ordering::Greater => active_index - 1,
            });
        } else {
            self.active_tab_index = Some(index.min(self.tabs.len() - 1));
        }
        self.pending_close_tab = None;
        true
    }

    pub fn show_notice(&mut self, tone: NoticeTone, message: impl Into<String>) {
        self.notice = Some(UiNotice {
            tone,
            message: message.into(),
        });
    }

    pub fn clear_notice(&mut self) {
        self.notice = None;
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn classifies_editor_languages_from_extensions() {
        let cases = [
            ("main.rs", FileKind::Rust),
            ("main.go", FileKind::Go),
            ("README.md", FileKind::Markdown),
            ("data.json", FileKind::Json),
            ("config.yaml", FileKind::Yaml),
            ("Cargo.toml", FileKind::Toml),
            ("app.ts", FileKind::TypeScript),
            ("app.js", FileKind::JavaScript),
            ("go.mod", FileKind::Manifest),
        ];
        for (path, expected) in cases {
            assert_eq!(FileKind::from_path(Path::new(path)), expected);
        }
    }
}
