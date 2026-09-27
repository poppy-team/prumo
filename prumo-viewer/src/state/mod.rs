use crate::extensions::registry::{ExtensionLoadError, ExtensionRecord};
use crate::services::config::ViewerConfig;
use crate::services::git::GitStatus;
use crate::services::workspace::WorkspaceIndex;
use freya::code_editor::LineDecoration;
use prumo_extension_sdk::lsp::{LspCodeAction, LspCompletionItem};
use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};

mod editor;
pub use editor::{EditorCursorState, EditorState};

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

    pub fn from_operation(operation: &str) -> Self {
        match operation.to_lowercase().as_str() {
            "create" | "created" => Self::Created,
            "delete" | "deleted" | "remove" | "removed" => Self::Deleted,
            _ => Self::Modified,
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
    Git,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DockView {
    Activity,
    Changes,
    Terminal,
    Search,
    Problems,
    Evidence,
    Quality,
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
pub enum DocumentConflict {
    ExternalChange { disk_content: String },
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ConflictResolution {
    Reload,
    KeepMine,
    ThreeWayMerge,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DiagnosticSeverity {
    Error,
    Warning,
    Information,
    Hint,
}

impl DiagnosticSeverity {
    pub fn label(self) -> &'static str {
        match self {
            Self::Error => "Error",
            Self::Warning => "Warning",
            Self::Information => "Info",
            Self::Hint => "Hint",
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Diagnostic {
    pub path: PathBuf,
    pub line: usize,
    pub column: usize,
    pub end_line: Option<usize>,
    pub end_column: Option<usize>,
    pub message: String,
    pub severity: DiagnosticSeverity,
    pub source: String,
    pub code: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DocumentTab {
    pub path: PathBuf,
    pub rel_path: String,
    pub title: String,
    pub is_dirty: bool,
    pub content: String,
    pub persisted_content: String,
    pub revision: u64,
    pub conflict: Option<DocumentConflict>,
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

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct AgentMetrics {
    pub input_tokens: u64,
    pub cache_read_tokens: u64,
    pub cache_write_tokens: u64,
    pub output_tokens: u64,
    pub total_cost_usd: f64,
    pub tool_calls: u32,
}

impl AgentMetrics {
    #[allow(dead_code)]
    pub fn total_tokens(&self) -> u64 {
        self.input_tokens + self.output_tokens
    }

    pub fn format_tokens(count: u64) -> String {
        if count >= 1_000_000 {
            format!("{:.1}M", count as f64 / 1_000_000.0)
        } else if count >= 1_000 {
            format!("{:.1}k", count as f64 / 1_000.0)
        } else {
            count.to_string()
        }
    }

    pub fn cache_hit_percent(&self) -> Option<u32> {
        let total_input = self.input_tokens + self.cache_read_tokens;
        if total_input > 0 && self.cache_read_tokens > 0 {
            Some(((self.cache_read_tokens as f64 / total_input as f64) * 100.0) as u32)
        } else {
            None
        }
    }

    pub fn format_cost(&self) -> String {
        if self.total_cost_usd >= 1.0 {
            format!("${:.2}", self.total_cost_usd)
        } else if self.total_cost_usd > 0.0 {
            format!("${:.4}", self.total_cost_usd)
        } else {
            "$0.00".to_string()
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ChatMessageRole {
    User,
    Assistant,
    System,
}

#[derive(Debug, Clone, PartialEq)]
pub struct ChatMessage {
    pub id: String,
    pub role: ChatMessageRole,
    pub content: String,
    pub timestamp: String,
    pub is_streaming: bool,
    pub reasoning: Option<String>,
}

#[derive(Debug, Clone, PartialEq)]
pub struct AgentSession {
    pub id: String,
    pub title: String,
    pub run_id: Option<String>,
    pub status: AgentStatus,
    pub phase: String,
    pub provider: String,
    pub model: String,
    pub events: Vec<AgentEventItem>,
    pub messages: Vec<ChatMessage>,
    pub metrics: AgentMetrics,
    pub changed_files: Vec<ChangedFile>,
    pub pending_permissions: Vec<String>,
    pub pending_permission_details: Vec<PendingPermissionItem>,
    pub agent_log: String,
}

impl AgentSession {
    pub fn new(index: usize, provider: &str, model: &str) -> Self {
        Self {
            id: format!("session-{}", index + 1),
            title: format!("Chat {}", index + 1),
            run_id: None,
            status: AgentStatus::Idle,
            phase: String::new(),
            provider: provider.to_string(),
            model: model.to_string(),
            events: Vec::new(),
            messages: Vec::new(),
            metrics: AgentMetrics::default(),
            changed_files: Vec::new(),
            pending_permissions: Vec::new(),
            pending_permission_details: Vec::new(),
            agent_log: String::new(),
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AgentEventItem {
    pub id: String,
    pub time: String,
    pub kind: String,
    pub message: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PendingPermissionItem {
    pub request_id: String,
    pub tool: String,
    pub fingerprint: String,
    pub arguments_preview: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct QuickOpenMatch {
    pub path: PathBuf,
    pub rel_path: String,
    pub score: i32,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ExtensionPanelKind {
    View,
    Panel,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ExtensionPanel {
    pub kind: ExtensionPanelKind,
    pub title: String,
    pub content: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct EditorReveal {
    pub path: PathBuf,
    pub line: usize,
    pub start: usize,
    pub end: usize,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum EditorPopupKind {
    Completion,
    CodeActions,
}

#[derive(Debug, Clone, PartialEq)]
pub enum EditorPopupItem {
    Completion(LspCompletionItem),
    CodeAction(LspCodeAction),
}

#[derive(Debug, Clone, PartialEq)]
pub struct EditorPopupState {
    pub kind: EditorPopupKind,
    pub path: PathBuf,
    pub revision: u64,
    pub cursor: usize,
    pub selection: Option<(usize, usize)>,
    pub range: Option<(usize, usize)>,
    pub request_id: u64,
    pub items: Vec<EditorPopupItem>,
    pub selected_index: usize,
    pub loading: bool,
    pub error: Option<String>,
}

#[derive(Debug, Clone, PartialEq)]
pub struct ExplorerMenu {
    pub path: PathBuf,
    pub is_dir: bool,
    pub is_empty: bool,
    pub x: f32,
    pub y: f32,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FilePromptKind {
    NewFile(PathBuf),
    NewFolder(PathBuf),
    Rename(PathBuf),
    ConfirmDelete(PathBuf),
    ConfirmTrash(PathBuf),
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct FilePrompt {
    pub kind: FilePromptKind,
    pub input: String,
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
    pub workspace_index: WorkspaceIndex,
    pub changed_files: Vec<ChangedFile>,
    pub selected_path: Option<PathBuf>,
    pub editor: EditorState,
    pub connection_status: ConnectionStatus,
    pub connection_message: String,
    pub active_run_id: Option<String>,
    pub active_run_phase: String,
    pub pending_permissions: Vec<String>,
    pub pending_permission_details: Vec<PendingPermissionItem>,
    pub agent_status: AgentStatus,
    pub agent_events: Vec<AgentEventItem>,
    pub agent_messages: Vec<ChatMessage>,
    pub prompt_history: Vec<String>,
    pub agent_log: String,
    pub agent_sessions: Vec<AgentSession>,
    pub active_session_index: usize,
    pub agent_model_picker_open: bool,
    pub agent_metrics: AgentMetrics,
    pub sidebar_visible: bool,
    pub sidebar_view: SidebarView,
    pub agent_panel_visible: bool,
    pub agent_panel_width: f32,
    pub dock_open: bool,
    pub dock_view: DockView,
    pub quick_open_open: bool,
    pub palette_open: bool,
    pub menu_open: bool,
    pub find_open: bool,
    pub search_query: String,
    pub search_replacement: String,
    pub search_match_case: bool,
    pub search_use_regex: bool,
    pub git: GitStatus,
    pub git_ready: bool,
    pub extensions: Vec<ExtensionRecord>,
    pub extension_errors: Vec<ExtensionLoadError>,
    pub explorer_menu: Option<ExplorerMenu>,
    pub explorer_focused: bool,
    pub file_prompt: Option<FilePrompt>,
    pub notice: Option<UiNotice>,
    pub diff_open: bool,
    pub diff_title: String,
    pub diff_lines: Vec<(String, char)>,
    pub editor_line_decorations: Vec<LineDecoration>,
    pub git_line_decorations: Vec<LineDecoration>,
    pub pending_git_line_decorations: Option<Vec<LineDecoration>>,
    pub agent_line_decorations: Vec<LineDecoration>,
    pub pending_agent_line_decorations: Option<Vec<LineDecoration>>,
    pub follow_agent: bool,
    pub follow_conflict_paths: Vec<String>,
    pub extension_panel: Option<ExtensionPanel>,
    pub pending_extension_command: Option<(String, String)>,
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
            workspace_index: WorkspaceIndex::default(),
            changed_files: Vec::new(),
            selected_path: None,
            editor: EditorState::default(),
            connection_status: ConnectionStatus::Connecting,
            connection_message: "Connecting to Prumo Core...".to_string(),
            active_run_id: None,
            active_run_phase: String::new(),
            pending_permissions: Vec::new(),
            pending_permission_details: Vec::new(),
            agent_status: AgentStatus::Disconnected,
            agent_events: Vec::new(),
            agent_messages: Vec::new(),
            prompt_history: Vec::new(),
            agent_log: "Waiting for Prumo Core...".to_string(),
            agent_sessions: vec![AgentSession::new(0, "fake", "")],
            active_session_index: 0,
            agent_model_picker_open: false,
            agent_metrics: AgentMetrics::default(),
            sidebar_visible: true,
            sidebar_view: SidebarView::Explorer,
            agent_panel_visible: false,
            agent_panel_width: 360.0,
            dock_open: false,
            dock_view: DockView::Activity,
            quick_open_open: false,
            palette_open: false,
            menu_open: false,
            find_open: false,
            search_query: String::new(),
            search_replacement: String::new(),
            search_match_case: false,
            search_use_regex: false,
            git: GitStatus::default(),
            git_ready: false,
            extensions: Vec::new(),
            extension_errors: Vec::new(),
            explorer_menu: None,
            explorer_focused: false,
            file_prompt: None,
            notice: None,
            diff_open: false,
            diff_title: String::new(),
            diff_lines: Vec::new(),
            editor_line_decorations: Vec::new(),
            git_line_decorations: Vec::new(),
            pending_git_line_decorations: None,
            agent_line_decorations: Vec::new(),
            pending_agent_line_decorations: None,
            follow_agent: true,
            follow_conflict_paths: Vec::new(),
            extension_panel: None,
            pending_extension_command: None,
        }
    }

    #[allow(dead_code)]
    pub fn active_session(&self) -> Option<&AgentSession> {
        self.agent_sessions.get(self.active_session_index)
    }

    pub fn active_session_mut(&mut self) -> Option<&mut AgentSession> {
        self.agent_sessions.get_mut(self.active_session_index)
    }

    pub fn create_new_session(&mut self) {
        let index = self.agent_sessions.len();
        let provider = self.config.provider.clone();
        let model = self.config.model.clone();
        self.agent_sessions
            .push(AgentSession::new(index, &provider, &model));
        self.select_session(index);
    }

    pub fn close_session(&mut self, index: usize) {
        if self.agent_sessions.len() <= 1 {
            let provider = self.config.provider.clone();
            let model = self.config.model.clone();
            self.agent_sessions = vec![AgentSession::new(0, &provider, &model)];
            self.select_session(0);
            return;
        }
        self.agent_sessions.remove(index);
        let new_idx = if self.active_session_index >= self.agent_sessions.len() {
            self.agent_sessions.len() - 1
        } else if self.active_session_index > index {
            self.active_session_index - 1
        } else {
            self.active_session_index
        };
        self.select_session(new_idx);
    }

    pub fn select_session(&mut self, index: usize) {
        if index < self.agent_sessions.len() {
            self.active_session_index = index;
            let session = &self.agent_sessions[index];
            self.active_run_id = session.run_id.clone();
            self.active_run_phase = session.phase.clone();
            self.agent_status = session.status;
            self.agent_events = session.events.clone();
            self.agent_messages = session.messages.clone();
            self.agent_metrics = session.metrics.clone();
            self.changed_files = session.changed_files.clone();
            self.pending_permissions = session.pending_permissions.clone();
            self.pending_permission_details = session.pending_permission_details.clone();
            if !session.agent_log.is_empty() {
                self.agent_log = session.agent_log.clone();
            }
            if !session.provider.is_empty() {
                self.config.provider = session.provider.clone();
            }
            if !session.model.is_empty() {
                self.config.model = session.model.clone();
            }
        }
    }

    pub fn add_user_message(&mut self, content: &str) {
        let timestamp = current_time_str();
        let message = ChatMessage {
            id: format!("msg-{}", self.agent_messages.len() + 1),
            role: ChatMessageRole::User,
            content: content.to_string(),
            timestamp,
            is_streaming: false,
            reasoning: None,
        };
        self.agent_messages.push(message.clone());
        if !self.prompt_history.contains(&content.to_string()) {
            self.prompt_history.push(content.to_string());
        }
        if let Some(session) = self.active_session_mut() {
            session.messages.push(message);
        }
    }

    pub fn add_system_message(&mut self, content: &str) {
        let timestamp = current_time_str();
        let message = ChatMessage {
            id: format!("msg-{}", self.agent_messages.len() + 1),
            role: ChatMessageRole::System,
            content: content.to_string(),
            timestamp,
            is_streaming: false,
            reasoning: None,
        };
        self.agent_messages.push(message.clone());
        if let Some(session) = self.active_session_mut() {
            session.messages.push(message);
        }
    }

    pub fn start_assistant_streaming_message(&mut self) {
        let timestamp = current_time_str();
        let message = ChatMessage {
            id: format!("msg-{}", self.agent_messages.len() + 1),
            role: ChatMessageRole::Assistant,
            content: String::new(),
            timestamp,
            is_streaming: true,
            reasoning: None,
        };
        self.agent_messages.push(message.clone());
        if let Some(session) = self.active_session_mut() {
            session.messages.push(message);
        }
    }

    pub fn append_assistant_chunk(&mut self, text: &str) {
        if let Some(msg) = self
            .agent_messages
            .iter_mut()
            .rev()
            .find(|m| m.role == ChatMessageRole::Assistant && m.is_streaming)
        {
            msg.content.push_str(text);
            let msg_id = msg.id.clone();
            if let Some(session) = self.active_session_mut()
                && let Some(s_msg) = session.messages.iter_mut().rev().find(|m| m.id == msg_id)
            {
                s_msg.content.push_str(text);
            }
        } else {
            self.start_assistant_streaming_message();
            if let Some(msg) = self.agent_messages.last_mut() {
                msg.content.push_str(text);
            }
            if let Some(session) = self.active_session_mut()
                && let Some(s_msg) = session.messages.last_mut()
            {
                s_msg.content.push_str(text);
            }
        }
    }

    pub fn append_reasoning_chunk(&mut self, text: &str) {
        if let Some(msg) = self
            .agent_messages
            .iter_mut()
            .rev()
            .find(|m| m.role == ChatMessageRole::Assistant && m.is_streaming)
        {
            if let Some(reasoning) = &mut msg.reasoning {
                reasoning.push_str(text);
            } else {
                msg.reasoning = Some(text.to_string());
            }
            let msg_id = msg.id.clone();
            if let Some(session) = self.active_session_mut()
                && let Some(s_msg) = session.messages.iter_mut().rev().find(|m| m.id == msg_id)
            {
                if let Some(reasoning) = &mut s_msg.reasoning {
                    reasoning.push_str(text);
                } else {
                    s_msg.reasoning = Some(text.to_string());
                }
            }
        }
    }

    pub fn finish_assistant_streaming(&mut self) {
        for msg in &mut self.agent_messages {
            if msg.role == ChatMessageRole::Assistant && msg.is_streaming {
                msg.is_streaming = false;
            }
        }
        if let Some(session) = self.active_session_mut() {
            for msg in &mut session.messages {
                if msg.role == ChatMessageRole::Assistant && msg.is_streaming {
                    msg.is_streaming = false;
                }
            }
        }
    }

    #[allow(dead_code)]
    pub fn set_agent_panel_width(&mut self, width: f32) {
        self.agent_panel_width = width.clamp(260.0, 750.0);
    }

    pub fn toggle_follow_agent(&mut self) -> bool {
        self.follow_agent = !self.follow_agent;
        if !self.follow_agent {
            self.follow_conflict_paths.clear();
            self.agent_line_decorations.clear();
            self.pending_agent_line_decorations = None;
        }
        self.follow_agent
    }

    pub fn track_run(&mut self, run_id: Option<String>) {
        if self.active_run_id != run_id {
            self.follow_conflict_paths.clear();
            self.agent_line_decorations.clear();
        }
        self.active_run_id = run_id.clone();
        if let Some(ref r_id) = run_id {
            if let Some(session) = self
                .agent_sessions
                .iter_mut()
                .find(|s| s.run_id.as_deref() == Some(r_id))
            {
                session.status = self.agent_status;
            } else if let Some(session) = self.active_session_mut()
                && session.run_id.is_none()
            {
                session.run_id = Some(r_id.clone());
            }
        }
    }

    pub fn activate_path(&mut self, path: &Path) -> bool {
        if let Some(index) = self.editor.tabs.iter().position(|tab| tab.path == path) {
            self.editor.active_tab_index = Some(index);
            self.selected_path = Some(path.to_path_buf());
            if self
                .editor
                .editor_popup
                .as_ref()
                .is_some_and(|popup| popup.path.as_path() != path)
            {
                self.editor.close_editor_popup();
            }
            return true;
        }
        false
    }

    pub fn remove_tab(&mut self, index: usize) -> bool {
        let Some(_) = self.editor.remove_tab_at(index) else {
            return false;
        };
        if self.editor.tabs.is_empty() {
            self.selected_path = None;
        }
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

impl std::ops::Deref for AppState {
    type Target = EditorState;

    fn deref(&self) -> &Self::Target {
        &self.editor
    }
}

impl std::ops::DerefMut for AppState {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.editor
    }
}

fn current_time_str() -> String {
    let now = std::time::SystemTime::now();
    if let Ok(duration) = now.duration_since(std::time::UNIX_EPOCH) {
        let secs = duration.as_secs();
        let hours = (secs / 3600) % 24;
        let mins = (secs / 60) % 60;
        let s = secs % 60;
        format!("{hours:02}:{mins:02}:{s:02}")
    } else {
        "00:00:00".to_string()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn invalidates_editor_popup_on_edit_and_tab_close() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().to_path_buf();
        let path = root.join("main.rs");
        std::fs::write(&path, "fn main() {}").unwrap();
        let mut state = AppState::new(root);
        crate::services::document::activate_path(&mut state, &path).unwrap();
        let (revision, cursor) = state
            .active_tab()
            .map(|tab| (tab.revision, tab.content.len()))
            .unwrap();
        state.active_cursor_position = cursor;
        state.active_selection_range = None;
        let request_id = state.begin_editor_popup(
            EditorPopupKind::Completion,
            path.clone(),
            revision,
            cursor,
            None,
            Some((cursor, cursor)),
        );
        assert!(state.is_editor_popup_request_current(request_id, &path, revision));
        state.active_selection_range = Some((0, 1));
        assert!(!state.is_editor_popup_request_current(request_id, &path, revision));
        state.active_selection_range = None;
        state.update_active_content("fn main() { /* changed */ }".to_string());
        assert!(state.editor_popup.is_none());

        let revision = state.active_tab().unwrap().revision;
        state.begin_editor_popup(
            EditorPopupKind::CodeActions,
            path.clone(),
            revision,
            0,
            None,
            None,
        );
        state.remove_tab(0);
        assert!(state.editor_popup.is_none());
    }

    #[test]
    fn tracks_diagnostics_per_file() {
        let mut state = AppState::new(PathBuf::from("/workspace"));
        let path = PathBuf::from("/workspace/main.rs");
        state.replace_diagnostics(
            &path,
            vec![Diagnostic {
                path: path.clone(),
                line: 3,
                column: 5,
                end_line: Some(3),
                end_column: Some(8),
                message: "Example".to_string(),
                severity: DiagnosticSeverity::Error,
                source: "rust-analyzer".to_string(),
                code: Some("E0001".to_string()),
            }],
        );
        assert_eq!(state.diagnostic_counts(), (1, 0));
        state.replace_diagnostics(&path, Vec::new());
        assert!(state.diagnostics.is_empty());
    }

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

    #[test]
    fn test_agent_metrics_formatting_and_hit_rate() {
        let metrics = AgentMetrics {
            input_tokens: 15_400,
            cache_read_tokens: 12_320,
            cache_write_tokens: 3_080,
            output_tokens: 2_600,
            total_cost_usd: 0.05214,
            tool_calls: 5,
        };

        assert_eq!(metrics.total_tokens(), 18_000);
        assert_eq!(AgentMetrics::format_tokens(500), "500");
        assert_eq!(AgentMetrics::format_tokens(15_400), "15.4k");
        assert_eq!(AgentMetrics::format_tokens(1_500_000), "1.5M");

        // Cache hit percent = 12320 / (15400 + 12320) = 12320 / 27720 = 44%
        assert_eq!(metrics.cache_hit_percent(), Some(44));

        // Format cost
        assert_eq!(metrics.format_cost(), "$0.0521");
    }

    #[test]
    fn test_agent_multi_sessions_workflow() {
        let mut state = AppState::new(PathBuf::from("/workspace"));
        state.config.provider = "anthropic".to_string();
        state.config.model = "claude-3-7-sonnet".to_string();

        // 1. Initial state has 1 default session
        assert_eq!(state.agent_sessions.len(), 1);
        assert_eq!(state.active_session_index, 0);
        assert_eq!(state.agent_sessions[0].title, "Chat 1");

        // 2. Track run on session 0
        state.track_run(Some("run-001".to_string()));
        assert_eq!(state.active_run_id, Some("run-001".to_string()));
        assert_eq!(state.agent_sessions[0].run_id, Some("run-001".to_string()));

        // 3. Create new session -> switches to session 1
        state.create_new_session();
        assert_eq!(state.agent_sessions.len(), 2);
        assert_eq!(state.active_session_index, 1);
        assert_eq!(state.agent_sessions[1].title, "Chat 2");
        assert_eq!(state.active_run_id, None);
        assert_eq!(state.agent_sessions[1].provider, "anthropic");
        assert_eq!(state.agent_sessions[1].model, "claude-3-7-sonnet");

        // 4. Switch back to session 0
        state.select_session(0);
        assert_eq!(state.active_session_index, 0);
        assert_eq!(state.active_run_id, Some("run-001".to_string()));

        // 5. Close session 1
        state.close_session(1);
        assert_eq!(state.agent_sessions.len(), 1);
        assert_eq!(state.active_session_index, 0);

        // 6. Close remaining session -> creates a fresh empty session
        state.close_session(0);
        assert_eq!(state.agent_sessions.len(), 1);
        assert_eq!(state.active_session_index, 0);
        assert_eq!(state.agent_sessions[0].title, "Chat 1");
    }

    #[test]
    fn test_agent_panel_width_persistence_and_clamping() {
        let mut state = AppState::new(PathBuf::from("/workspace"));
        assert_eq!(state.agent_panel_width, 360.0);

        state.set_agent_panel_width(500.0);
        assert_eq!(state.agent_panel_width, 500.0);

        // Clamping below minimum (260.0)
        state.set_agent_panel_width(100.0);
        assert_eq!(state.agent_panel_width, 260.0);

        // Clamping above maximum (750.0)
        state.set_agent_panel_width(1200.0);
        assert_eq!(state.agent_panel_width, 750.0);
    }
}
