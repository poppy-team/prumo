mod client;
mod extensions;
mod projections;
mod services;
mod state;
mod theme;
mod ui;

use std::collections::HashSet;
use std::env;
use std::path::PathBuf;
use std::time::Duration;

use freya::code_editor::LineDecoration;
use freya::prelude::*;
use prumo_extension_sdk::lsp::PublishDiagnosticsParams;
use prumo_extension_sdk::manifest::parse_keybinding;
use prumo_extension_sdk::rpc::{RpcError, RpcRequest};
use serde_json::{Value, json};
use tokio::runtime::Builder;
use torin::prelude::Direction;

use crate::client::protocol::PrumoClient;
use crate::extensions::host::{ExtensionHost, ExtensionRequest, LspNotification, load_grants};
use crate::extensions::registry::ExtensionStatus;
use crate::projections::run::{
    apply_client_error, apply_daemon_snapshot, apply_streaming_event,
};
use crate::services::config::ViewerConfig;
use crate::services::document::{
    activate_path, file_uri_to_path, open_file, request_close_tab, save_active_tab,
};
use crate::services::files::{create_file, create_folder, delete_path, rename_path};
use crate::services::follow;
use crate::services::git::{
    git_commit, git_discard, git_file_diff, git_pull, git_push, git_stage, git_status, git_unstage,
};
use crate::services::search::search_workspace_text;
use crate::services::terminal::{TerminalEvent, TerminalRuntime};
use crate::services::workspace::refresh_workspace;
use crate::state::{
    AgentFileStatus, AppState, Diagnostic, DiagnosticSeverity, EditorCursorState, EditorReveal,
    FilePrompt, FilePromptKind, NoticeTone, SidebarView,
};
use crate::ui::agent_panel::{AgentPanel, BottomDock};
use crate::ui::command_palette::CommandPaletteModal;
use crate::ui::diff_view::DiffModal;
use crate::ui::editor_area::EditorArea;
use crate::ui::explorer_menu::{ExplorerMenuOverlay, FilePromptModal};
use crate::ui::extension_panel::ExtensionPanelOverlay;
use crate::ui::menu::MenuOverlay;
use crate::ui::quick_open::QuickOpenModal;
use crate::ui::settings::SettingsModal;
use crate::ui::sidebar::{ActivityBar, Sidebar};
use crate::ui::status_bar::StatusBar;
use crate::ui::tab_bar::TabBar;
use crate::ui::top_bar::TopBar;

const SIDEBAR_BREAKPOINT: f32 = 820.;
const AGENT_BREAKPOINT: f32 = 900.;

fn parse_cli_args() -> (PathBuf, Option<PathBuf>) {
    let arguments: Vec<String> = env::args().collect();
    let mut workspace_path = PathBuf::from(".");
    let mut socket_path = None;
    let mut index = 1;

    while index < arguments.len() {
        match arguments[index].as_str() {
            "--workspace" | "-w" if index + 1 < arguments.len() => {
                workspace_path = PathBuf::from(&arguments[index + 1]);
                index += 1;
            }
            "--socket" if index + 1 < arguments.len() => {
                socket_path = Some(PathBuf::from(&arguments[index + 1]));
                index += 1;
            }
            argument if !argument.starts_with('-') => {
                workspace_path = PathBuf::from(argument);
            }
            _ => {}
        }
        index += 1;
    }

    let workspace_path = workspace_path.canonicalize().unwrap_or(workspace_path);
    (workspace_path, socket_path)
}

#[derive(Clone, Copy, PartialEq)]
pub struct WorkspaceCommands {
    state: State<AppState>,
}

impl WorkspaceCommands {
    pub fn refresh(self) {
        let mut state = self.state;
        let mut worker_state = state.read().clone();
        spawn(async move {
            let result = tokio::task::spawn_blocking(move || {
                let refresh_result = refresh_workspace(&mut worker_state);
                (refresh_result, worker_state)
            })
            .await;
            match result {
                Ok((Ok(()), worker_state)) => {
                    let mut state = state.write();
                    state.tree = worker_state.tree;
                    state.flattened_tree = worker_state.flattened_tree;
                    state.file_candidates = worker_state.file_candidates;
                    state.clear_notice();
                }
                Ok((Err(error), _)) => state.write().show_notice(
                    NoticeTone::Error,
                    format!("Workspace refresh failed: {error}"),
                ),
                Err(error) => state.write().show_notice(
                    NoticeTone::Error,
                    format!("Workspace refresh worker failed: {error}"),
                ),
            }
        });
    }

    pub fn load_file(self, path: PathBuf) {
        self.load_file_inner(path, None);
    }

    pub fn load_file_with_reveal(self, path: PathBuf, reveal: EditorReveal) {
        self.load_file_inner(path, Some(reveal));
    }

    fn load_file_inner(self, path: PathBuf, reveal: Option<EditorReveal>) {
        let mut state = self.state;
        let root = state.read().workspace_root.clone();
        spawn(async move {
            let worker_root = root.clone();
            let result = tokio::task::spawn_blocking(move || open_file(&worker_root, &path)).await;
            match result {
                Ok(Ok(tab)) => {
                    let mut state = state.write();
                    if let Some(index) = state.tabs.iter().position(|open| open.path == tab.path) {
                        state.active_tab_index = Some(index);
                    } else {
                        state.tabs.push(tab.clone());
                        state.active_tab_index = Some(state.tabs.len() - 1);
                    }
                    state.selected_path = Some(tab.path);
                    state.close_editor_popup();
                    state.git_line_decorations = state
                        .pending_git_line_decorations
                        .take()
                        .unwrap_or_default();
                    state.pending_editor_reveal = reveal;
                    state.clear_notice();
                }
                Ok(Err(error)) => {
                    let mut app_state = state.write();
                    app_state.pending_git_line_decorations = None;
                    app_state
                        .show_notice(NoticeTone::Error, format!("Could not open file: {error}"));
                }
                Err(error) => {
                    let mut app_state = state.write();
                    app_state.pending_git_line_decorations = None;
                    app_state
                        .show_notice(NoticeTone::Error, format!("File reader failed: {error}"));
                }
            }
        });
    }
}

pub(crate) async fn run_blocking<T, F>(operation: F) -> Result<T, String>
where
    T: Send + 'static,
    F: FnOnce() -> T + Send + 'static,
{
    let (sender, receiver) = std::sync::mpsc::sync_channel(1);
    let join = std::thread::spawn(move || {
        let _ = sender.send(operation());
    });
    loop {
        match receiver.try_recv() {
            Ok(value) => return Ok(value),
            Err(std::sync::mpsc::TryRecvError::Empty) => {
                async_io::Timer::after(Duration::from_millis(1)).await;
            }
            Err(std::sync::mpsc::TryRecvError::Disconnected) => break,
        }
    }
    match join.join() {
        Ok(()) => Err("blocking worker exited without a result".to_string()),
        Err(_) => Err("blocking worker panicked".to_string()),
    }
}

fn diagnostic_severity(value: Option<u8>) -> DiagnosticSeverity {
    match value {
        Some(1) => DiagnosticSeverity::Error,
        Some(2) => DiagnosticSeverity::Warning,
        Some(3) => DiagnosticSeverity::Information,
        Some(4) => DiagnosticSeverity::Hint,
        _ => DiagnosticSeverity::Information,
    }
}

fn parse_lsp_diagnostics(notification: &LspNotification) -> Option<(PathBuf, Vec<Diagnostic>)> {
    if notification.method != "textDocument/publishDiagnostics" {
        return None;
    }
    let params =
        serde_json::from_value::<PublishDiagnosticsParams>(notification.params.clone()).ok()?;
    let path = file_uri_to_path(&params.uri).ok()?;
    let diagnostics = params
        .diagnostics
        .into_iter()
        .map(|diagnostic| Diagnostic {
            path: path.clone(),
            line: diagnostic.range.start.line as usize + 1,
            column: diagnostic.range.start.character as usize + 1,
            end_line: Some(diagnostic.range.end.line as usize + 1),
            end_column: Some(diagnostic.range.end.character as usize + 1),
            message: diagnostic.message,
            severity: diagnostic_severity(diagnostic.severity),
            source: diagnostic
                .source
                .unwrap_or_else(|| notification.language.clone()),
            code: diagnostic.code.map(|code| match code {
                Value::String(code) => code,
                other => other.to_string(),
            }),
        })
        .collect();
    Some((path, diagnostics))
}

fn apply_lsp_notification(mut state: State<AppState>, notification: &LspNotification) {
    let Some((path, diagnostics)) = parse_lsp_diagnostics(notification) else {
        return;
    };
    state.write().replace_diagnostics(&path, diagnostics);
}

async fn handle_extension_request(
    extension_host: ExtensionHost,
    mut state: State<AppState>,
    request: ExtensionRequest,
) {
    let ExtensionRequest {
        extension_id,
        request: rpc_request,
    } = request;
    let result = match rpc_request.method.as_str() {
        "workspace/create-file"
        | "workspace/create-folder"
        | "workspace/rename"
        | "workspace/delete" => {
            handle_workspace_write_request(&extension_id, &extension_host, &mut state, &rpc_request)
                .await
        }
        "git/status" => {
            handle_git_request(
                &extension_id,
                &extension_host,
                &mut state,
                &rpc_request,
                false,
            )
            .await
        }
        "git/diff" => {
            handle_git_request(
                &extension_id,
                &extension_host,
                &mut state,
                &rpc_request,
                false,
            )
            .await
        }
        "git/stage" | "git/unstage" | "git/discard" | "git/commit" | "git/push" | "git/pull" => {
            handle_git_request(
                &extension_id,
                &extension_host,
                &mut state,
                &rpc_request,
                true,
            )
            .await
        }
        "ui/notify" => {
            if !extension_host.has_capability(&extension_id, "ui.notify") {
                Err(extension_rpc_error(
                    -32001,
                    "capability ui.notify was not granted",
                ))
            } else {
                let message = rpc_request
                    .params
                    .get("message")
                    .and_then(Value::as_str)
                    .unwrap_or_default();
                if message.is_empty() || message.len() > 2000 {
                    Err(extension_rpc_error(
                        -32602,
                        "notification message is invalid",
                    ))
                } else {
                    state
                        .write()
                        .show_notice(NoticeTone::Info, message.to_string());
                    Ok(json!({ "ok": true }))
                }
            }
        }
        "workspace/search" => {
            if !extension_host.has_capability(&extension_id, "workspace.search") {
                Err(extension_rpc_error(
                    -32001,
                    "capability workspace.search was not granted",
                ))
            } else {
                let root = state.read().workspace_root.clone();
                let query = rpc_request
                    .params
                    .get("query")
                    .and_then(Value::as_str)
                    .unwrap_or_default()
                    .to_string();
                let match_case = rpc_request
                    .params
                    .get("match_case")
                    .and_then(Value::as_bool)
                    .unwrap_or(false);
                let use_regex = rpc_request
                    .params
                    .get("use_regex")
                    .and_then(Value::as_bool)
                    .unwrap_or(false);
                let limit = rpc_request
                    .params
                    .get("limit")
                    .and_then(Value::as_u64)
                    .unwrap_or(200)
                    .clamp(1, 200) as usize;
                let result = run_blocking(move || {
                    search_workspace_text(&root, &query, match_case, use_regex, limit)
                })
                .await;
                match result {
                    Ok(Ok(matches)) => {
                        let results = matches
                            .into_iter()
                            .map(|item| {
                                json!({
                                    "path": item.rel_path,
                                    "line": item.line,
                                    "start": item.start,
                                    "end": item.end,
                                    "preview": item.preview,
                                })
                            })
                            .collect::<Vec<_>>();
                        Ok(json!({ "results": results }))
                    }
                    Ok(Err(error)) => Err(extension_rpc_error(-32002, &error)),
                    Err(error) => Err(extension_rpc_error(-32003, &error)),
                }
            }
        }
        "workspace/index" => {
            if !extension_host.has_capability(&extension_id, "workspace.index") {
                Err(extension_rpc_error(
                    -32001,
                    "capability workspace.index was not granted",
                ))
            } else {
                let query = rpc_request
                    .params
                    .get("query")
                    .and_then(Value::as_str)
                    .unwrap_or_default()
                    .to_string();
                let limit = rpc_request
                    .params
                    .get("limit")
                    .and_then(Value::as_u64)
                    .unwrap_or(200)
                    .clamp(1, 500) as usize;
                let files = state
                    .read()
                    .workspace_index
                    .query(&query, limit)
                    .into_iter()
                    .map(|file| {
                        json!({
                            "path": file.path,
                            "size": file.size,
                            "modified_ms": file.modified_ms,
                            "kind": format!("{:?}", file.kind).to_lowercase(),
                        })
                    })
                    .collect::<Vec<_>>();
                Ok(json!({ "files": files }))
            }
        }
        "workspace/read" => {
            if !extension_host.has_capability(&extension_id, "workspace.read") {
                Err(extension_rpc_error(
                    -32001,
                    "capability workspace.read was not granted",
                ))
            } else {
                let root = state.read().workspace_root.clone();
                let path = rpc_request
                    .params
                    .get("path")
                    .and_then(Value::as_str)
                    .map(str::to_string);
                let result = run_blocking(move || {
                    let path = path.ok_or_else(|| "workspace/read requires path".to_string())?;
                    let root = std::fs::canonicalize(&root)
                        .map_err(|error| format!("could not resolve workspace: {error}"))?;
                    let path = std::fs::canonicalize(root.join(path))
                        .map_err(|error| format!("could not resolve file: {error}"))?;
                    if !path.starts_with(&root) {
                        return Err("workspace/read path is outside the workspace".to_string());
                    }
                    if std::fs::metadata(&path)
                        .map_err(|error| format!("could not inspect file: {error}"))?
                        .len()
                        > 512 * 1024
                    {
                        return Err(
                            "workspace/read file exceeds the 512 KiB response limit".to_string()
                        );
                    }
                    std::fs::read_to_string(path)
                        .map_err(|error| format!("could not read file: {error}"))
                })
                .await;
                match result {
                    Ok(Ok(content)) => Ok(json!({ "content": content })),
                    Ok(Err(error)) => Err(extension_rpc_error(-32002, &error)),
                    Err(error) => Err(extension_rpc_error(-32003, &error)),
                }
            }
        }
        "editor/set-line-decorations" => {
            if !extension_host.has_capability(&extension_id, "editor.decorations") {
                Err(extension_rpc_error(
                    -32001,
                    "capability editor.decorations was not granted",
                ))
            } else {
                match parse_line_decorations(rpc_request.params.get("decorations")) {
                    Ok(decorations) => {
                        state.write().editor_line_decorations = decorations;
                        Ok(json!({ "accepted": true }))
                    }
                    Err(error) => Err(error),
                }
            }
        }
        "editor/clear-line-decorations" => {
            if !extension_host.has_capability(&extension_id, "editor.decorations") {
                Err(extension_rpc_error(
                    -32001,
                    "capability editor.decorations was not granted",
                ))
            } else {
                state.write().editor_line_decorations.clear();
                Ok(json!({ "accepted": true }))
            }
        }
        "editor/reveal" => {
            if !extension_host.has_capability(&extension_id, "editor.navigate") {
                Err(extension_rpc_error(
                    -32001,
                    "capability editor.navigate was not granted",
                ))
            } else {
                let root = state.read().workspace_root.clone();
                let path = rpc_request
                    .params
                    .get("path")
                    .and_then(Value::as_str)
                    .map(str::to_string);
                let start = rpc_request
                    .params
                    .get("start")
                    .and_then(Value::as_u64)
                    .unwrap_or(0) as usize;
                let end = rpc_request
                    .params
                    .get("end")
                    .and_then(Value::as_u64)
                    .unwrap_or(0) as usize;
                let line = rpc_request
                    .params
                    .get("line")
                    .and_then(Value::as_u64)
                    .unwrap_or(1) as usize;
                let path = path
                    .ok_or_else(|| extension_rpc_error(-32602, "editor/reveal requires path"))
                    .and_then(|path| {
                        let canonical_root = std::fs::canonicalize(&root).map_err(|error| {
                            extension_rpc_error(-32002, &format!("invalid workspace: {error}"))
                        })?;
                        let path =
                            std::fs::canonicalize(canonical_root.join(path)).map_err(|error| {
                                extension_rpc_error(
                                    -32002,
                                    &format!("invalid reveal path: {error}"),
                                )
                            })?;
                        if !path.starts_with(&canonical_root) {
                            Err(extension_rpc_error(
                                -32002,
                                "editor/reveal path is outside the workspace",
                            ))
                        } else {
                            Ok(path)
                        }
                    });
                match path {
                    Ok(path) => {
                        let workspace = WorkspaceCommands { state };
                        workspace.load_file_with_reveal(
                            path.clone(),
                            EditorReveal {
                                path,
                                line,
                                start,
                                end,
                            },
                        );
                        Ok(json!({ "accepted": true }))
                    }
                    Err(error) => Err(error),
                }
            }
        }
        _ => Err(extension_rpc_error(
            -32601,
            &format!("method {} is not implemented", rpc_request.method),
        )),
    };
    let _ = extension_host.respond(&extension_id, &rpc_request, result);
}

async fn handle_workspace_write_request(
    extension_id: &str,
    extension_host: &ExtensionHost,
    state: &mut State<AppState>,
    request: &RpcRequest,
) -> Result<Value, RpcError> {
    if !extension_host.has_capability(extension_id, "workspace.write") {
        return Err(extension_rpc_error(
            -32001,
            "capability workspace.write was not granted",
        ));
    }
    let root = state.read().workspace_root.clone();
    let method = request.method.clone();
    let params = request.params.clone();
    let result = run_blocking(move || {
        let relative_path = |value: Option<&Value>| {
            value
                .and_then(Value::as_str)
                .map(str::to_string)
                .ok_or_else(|| "workspace write requires a path".to_string())
        };
        let resolve = |relative: &str| -> Result<PathBuf, String> {
            if relative.trim().is_empty() || PathBuf::from(relative).is_absolute() {
                return Err("workspace path must be relative".to_string());
            }
            crate::services::workspace::validate_path_boundary(&root, &root.join(relative))
        };
        match method.as_str() {
            "workspace/create-file" => {
                let parent = resolve(&relative_path(params.get("parent"))?)?;
                let name = params
                    .get("name")
                    .and_then(Value::as_str)
                    .ok_or_else(|| "workspace/create-file requires name".to_string())?;
                let path = create_file(&parent, name)?;
                Ok(json!({ "path": path.strip_prefix(&root).unwrap_or(&path).display().to_string() }))
            }
            "workspace/create-folder" => {
                let parent = resolve(&relative_path(params.get("parent"))?)?;
                let name = params
                    .get("name")
                    .and_then(Value::as_str)
                    .ok_or_else(|| "workspace/create-folder requires name".to_string())?;
                let path = create_folder(&parent, name)?;
                Ok(json!({ "path": path.strip_prefix(&root).unwrap_or(&path).display().to_string() }))
            }
            "workspace/rename" => {
                let path = resolve(&relative_path(params.get("path"))?)?;
                let name = params
                    .get("name")
                    .and_then(Value::as_str)
                    .ok_or_else(|| "workspace/rename requires name".to_string())?;
                let next = rename_path(&path, name)?;
                Ok(json!({ "path": next.strip_prefix(&root).unwrap_or(&next).display().to_string() }))
            }
            "workspace/delete" => {
                let path = resolve(&relative_path(params.get("path"))?)?;
                if path == root || path.starts_with(root.join(".git")) {
                    return Err("workspace/delete cannot remove the workspace root or Git directory".to_string());
                }
                delete_path(&path)?;
                Ok(json!({ "deleted": true }))
            }
            _ => Err("unsupported workspace write operation".to_string()),
        }
    })
    .await;
    match result {
        Ok(Ok(value)) => {
            let refresh_result = {
                let mut app_state = state.write();
                refresh_workspace(&mut app_state)
            };
            if let Err(error) = refresh_result {
                return Err(extension_rpc_error(-32002, &error));
            }
            Ok(value)
        }
        Ok(Err(error)) => Err(extension_rpc_error(-32002, &error)),
        Err(error) => Err(extension_rpc_error(-32003, &error)),
    }
}

async fn handle_git_request(
    extension_id: &str,
    extension_host: &ExtensionHost,
    state: &mut State<AppState>,
    request: &RpcRequest,
    write: bool,
) -> Result<Value, RpcError> {
    let capability = if write { "git.write" } else { "git.read" };
    if !extension_host.has_capability(extension_id, capability) {
        return Err(extension_rpc_error(
            -32001,
            &format!("capability {capability} was not granted"),
        ));
    }
    let root = state.read().workspace_root.clone();
    let method = request.method.clone();
    let params = request.params.clone();
    let result = run_blocking(move || match method.as_str() {
        "git/status" => Ok(json!({
            "status": git_status_json(&git_status(&root)),
        })),
        "git/diff" => {
            let path = params
                .get("path")
                .and_then(Value::as_str)
                .ok_or_else(|| "git/diff requires path".to_string())?;
            let staged = params
                .get("staged")
                .and_then(Value::as_bool)
                .unwrap_or(false);
            let lines = git_file_diff(&root, path, staged)?;
            Ok(json!({
                "lines": lines.into_iter().take(2000).map(|(line, tag)| json!({
                    "text": line,
                    "tag": tag.to_string(),
                })).collect::<Vec<_>>(),
            }))
        }
        "git/stage" => {
            let path = params
                .get("path")
                .and_then(Value::as_str)
                .ok_or_else(|| "git/stage requires path".to_string())?;
            git_stage(&root, path).map(|()| json!({ "ok": true }))
        }
        "git/unstage" => {
            let path = params
                .get("path")
                .and_then(Value::as_str)
                .ok_or_else(|| "git/unstage requires path".to_string())?;
            git_unstage(&root, path).map(|()| json!({ "ok": true }))
        }
        "git/discard" => {
            let path = params
                .get("path")
                .and_then(Value::as_str)
                .ok_or_else(|| "git/discard requires path".to_string())?;
            let tracked = params
                .get("tracked")
                .and_then(Value::as_bool)
                .unwrap_or(false);
            git_discard(&root, path, tracked).map(|()| json!({ "ok": true }))
        }
        "git/commit" => {
            let message = params
                .get("message")
                .and_then(Value::as_str)
                .ok_or_else(|| "git/commit requires message".to_string())?;
            if message.len() > 4000 {
                return Err("commit message is too long".to_string());
            }
            git_commit(&root, message).map(|message| json!({ "message": message }))
        }
        "git/push" => git_push(&root).map(|message| json!({ "message": message })),
        "git/pull" => git_pull(&root).map(|message| json!({ "message": message })),
        _ => Err("unsupported Git operation".to_string()),
    })
    .await;
    match result {
        Ok(Ok(value)) => {
            if write {
                let current_root = state.read().workspace_root.clone();
                state.write().git = git_status(&current_root);
            }
            Ok(value)
        }
        Ok(Err(error)) => Err(extension_rpc_error(-32002, &error)),
        Err(error) => Err(extension_rpc_error(-32003, &error)),
    }
}

fn git_status_json(status: &crate::services::git::GitStatus) -> Value {
    json!({
        "is_repo": status.is_repo,
        "branch": status.branch,
        "ahead": status.ahead,
        "behind": status.behind,
        "files": status.files.iter().map(|file| json!({
            "path": file.path,
            "staged_kind": file.staged_kind.map(|kind| kind.to_string()),
            "unstaged_kind": file.unstaged_kind.map(|kind| kind.to_string()),
        })).collect::<Vec<_>>(),
    })
}

fn extension_rpc_error(code: i64, message: &str) -> RpcError {
    RpcError {
        code,
        message: message.to_string(),
        data: None,
    }
}

fn parse_line_decorations(value: Option<&Value>) -> Result<Vec<LineDecoration>, RpcError> {
    let decorations = value
        .and_then(Value::as_array)
        .ok_or_else(|| extension_rpc_error(-32602, "decorations must be an array"))?;
    if decorations.len() > 500 {
        return Err(extension_rpc_error(-32602, "too many decorations"));
    }
    decorations
        .iter()
        .map(|decoration| {
            let line = decoration
                .get("line")
                .and_then(Value::as_u64)
                .and_then(|line| usize::try_from(line).ok())
                .filter(|line| *line > 0)
                .ok_or_else(|| extension_rpc_error(-32602, "decoration line is invalid"))?;
            let color = decoration
                .get("color")
                .and_then(Value::as_str)
                .and_then(parse_extension_color)
                .ok_or_else(|| extension_rpc_error(-32602, "decoration color is invalid"))?;
            Ok(LineDecoration {
                line: line - 1,
                color,
            })
        })
        .collect()
}

fn parse_extension_color(value: &str) -> Option<Color> {
    let value = value.strip_prefix('#').unwrap_or(value);
    if value.len() != 6 && value.len() != 8 {
        return None;
    }
    let channel = |start: usize| u8::from_str_radix(value.get(start..start + 2)?, 16).ok();
    if value.len() == 6 {
        Some(Color::from_rgb(channel(0)?, channel(2)?, channel(4)?))
    } else {
        Some(Color::from_argb(
            channel(0)?,
            channel(2)?,
            channel(4)?,
            channel(6)?,
        ))
    }
}

fn key_name(key: &Key) -> String {
    match key {
        Key::Character(character) => character.to_lowercase(),
        Key::Named(named) => format!("{named:?}").to_lowercase(),
    }
}

fn extension_keybinding(
    state: &State<AppState>,
    key: &Key,
    modifiers: Modifiers,
) -> Option<(String, String)> {
    let key_name = key_name(key);
    let chord = parse_keybinding(&format!(
        "{}{}{}{}",
        if modifiers.contains(Modifiers::CONTROL) {
            "Ctrl+"
        } else {
            ""
        },
        if modifiers.contains(Modifiers::SHIFT) {
            "Shift+"
        } else {
            ""
        },
        if modifiers.contains(Modifiers::ALT) {
            "Alt+"
        } else {
            ""
        },
        key_name,
    ))
    .ok()?;
    state.read().extensions.iter().find_map(|extension| {
        if extension.status != ExtensionStatus::Ready {
            return None;
        }
        extension
            .manifest
            .contributions
            .keybindings
            .iter()
            .find(|keybinding| {
                parse_keybinding(&keybinding.key).is_ok_and(|candidate| {
                    candidate.key == key_name
                        && candidate.control == chord.control
                        && candidate.shift == chord.shift
                        && candidate.alt == chord.alt
                })
            })
            .map(|keybinding| (extension.manifest.id.clone(), keybinding.command.clone()))
    })
}

#[derive(Clone, Copy)]
enum ExplorerShortcutAction {
    NewFile,
    NewFolder,
    Rename,
    Delete,
}

fn explorer_shortcut(
    state: &AppState,
    key: &Key,
    modifiers: Modifiers,
) -> Option<ExplorerShortcutAction> {
    if !state.sidebar_visible
        || state.sidebar_view != crate::state::SidebarView::Explorer
        || state.file_prompt.is_some()
        || state.palette_open
        || state.quick_open_open
        || state.config_open
        || state.agent_model_picker_open
    {
        return None;
    }
    let control = modifiers.contains(Modifiers::CONTROL);
    let shift = modifiers.contains(Modifiers::SHIFT);
    let alt = modifiers.contains(Modifiers::ALT);
    match key {
        Key::Character(character)
            if alt && control && !shift && character.eq_ignore_ascii_case("n") =>
        {
            Some(ExplorerShortcutAction::NewFolder)
        }
        Key::Character(character) if control && shift && character.eq_ignore_ascii_case("n") => {
            Some(ExplorerShortcutAction::NewFolder)
        }
        Key::Character(character) if control && character.eq_ignore_ascii_case("n") => {
            Some(ExplorerShortcutAction::NewFile)
        }
        Key::Named(NamedKey::F2) if state.explorer_focused => Some(ExplorerShortcutAction::Rename),
        Key::Named(NamedKey::Backspace | NamedKey::Delete) if state.explorer_focused => {
            Some(ExplorerShortcutAction::Delete)
        }
        _ => None,
    }
}

fn open_explorer_shortcut(state: &mut AppState, action: ExplorerShortcutAction) {
    let root = state.workspace_root.clone();
    let selected = state.selected_path.clone();
    match action {
        ExplorerShortcutAction::NewFile | ExplorerShortcutAction::NewFolder => {
            let parent = selected
                .as_ref()
                .map(|path| {
                    if path.is_dir() {
                        path.clone()
                    } else {
                        path.parent().unwrap_or(&root).to_path_buf()
                    }
                })
                .unwrap_or(root);
            state.file_prompt = Some(FilePrompt {
                kind: if matches!(action, ExplorerShortcutAction::NewFile) {
                    FilePromptKind::NewFile(parent)
                } else {
                    FilePromptKind::NewFolder(parent)
                },
                input: String::new(),
            });
        }
        ExplorerShortcutAction::Rename => {
            let Some(path) = selected else {
                return;
            };
            state.file_prompt = Some(FilePrompt {
                kind: FilePromptKind::Rename(path.clone()),
                input: path
                    .file_name()
                    .and_then(|name| name.to_str())
                    .unwrap_or("")
                    .to_string(),
            });
        }
        ExplorerShortcutAction::Delete => {
            let Some(path) = selected.filter(|path| path != &root) else {
                return;
            };
            state.file_prompt = Some(FilePrompt {
                kind: FilePromptKind::ConfirmTrash(path),
                input: String::new(),
            });
        }
    }
    state.explorer_focused = true;
}

pub(crate) fn dispatch_global_key(
    mut state: State<AppState>,
    key: &Key,
    modifiers: Modifiers,
) -> bool {
    let extension_binding = extension_keybinding(&state, key, modifiers);
    if let Some((extension_id, command_id)) = extension_binding {
        let mut app_state = state.write();
        app_state.pending_extension_command = Some((extension_id, command_id));
        app_state.palette_open = true;
        return true;
    }
    let shortcut_action = explorer_shortcut(&state.read(), key, modifiers);
    if let Some(action) = shortcut_action {
        let mut app_state = state.write();
        open_explorer_shortcut(&mut app_state, action);
        return true;
    }
    let control = modifiers.contains(Modifiers::CONTROL);
    match key {
        Key::Character(character) if control && character.eq_ignore_ascii_case("p") => {
            let mut app_state = state.write();
            if modifiers.contains(Modifiers::SHIFT) {
                app_state.palette_open = !app_state.palette_open;
            } else {
                app_state.quick_open_open = !app_state.quick_open_open;
            }
            app_state.close_editor_popup();
            app_state.diff_open = false;
            app_state.clear_notice();
            true
        }
        Key::Character(character) if control && character.eq_ignore_ascii_case("f") => {
            let mut app_state = state.write();
            if modifiers.contains(Modifiers::SHIFT) {
                app_state.dock_view = crate::state::DockView::Search;
                app_state.dock_open = true;
            } else {
                app_state.find_open = !app_state.find_open;
            }
            app_state.close_editor_popup();
            true
        }
        Key::Character(character)
            if control
                && modifiers.contains(Modifiers::SHIFT)
                && character.eq_ignore_ascii_case("m") =>
        {
            let mut app_state = state.write();
            app_state.close_editor_popup();
            app_state.dock_view = crate::state::DockView::Problems;
            app_state.dock_open = true;
            true
        }
        Key::Character(character) if control && character.eq_ignore_ascii_case("s") => {
            let mut app_state = state.write();
            let _ = save_active_tab(&mut app_state);
            true
        }
        Key::Character(character) if control && character.eq_ignore_ascii_case("w") => {
            let mut app_state = state.write();
            if let Some(index) = app_state.active_tab_index {
                request_close_tab(&mut app_state, index);
            }
            true
        }
        Key::Character(character) if control && character.eq_ignore_ascii_case("b") => {
            let mut app_state = state.write();
            app_state.sidebar_visible = !app_state.sidebar_visible;
            true
        }
        Key::Character(character)
            if control
                && modifiers.contains(Modifiers::SHIFT)
                && character.eq_ignore_ascii_case("j") =>
        {
            let mut app_state = state.write();
            app_state.agent_panel_visible = !app_state.agent_panel_visible;
            true
        }
        Key::Character(character) if control && character.eq_ignore_ascii_case("j") => {
            let mut app_state = state.write();
            app_state.dock_open = !app_state.dock_open;
            true
        }
        Key::Named(NamedKey::Escape) => {
            let mut app_state = state.write();
            app_state.quick_open_open = false;
            app_state.palette_open = false;
            app_state.find_open = false;
            app_state.menu_open = false;
            app_state.explorer_menu = None;
            app_state.file_prompt = None;
            app_state.diff_open = false;
            app_state.pending_close_tab = None;
            app_state.config_open = false;
            app_state.pending_extension_command = None;
            app_state.extension_panel = None;
            app_state.close_editor_popup();
            true
        }
        _ => false,
    }
}

fn main() {
    let (workspace_root, socket_path) = parse_cli_args();
    let mut initial_state = AppState::new(workspace_root.clone());
    match ViewerConfig::load() {
        Ok(config) => initial_state.config = config,
        Err(error) => {
            initial_state.show_notice(NoticeTone::Error, format!("Viewer config ignored: {error}"))
        }
    }
    if let Err(error) = refresh_workspace(&mut initial_state) {
        initial_state.show_notice(NoticeTone::Error, error);
    }
    initial_state.git_ready = crate::services::git::git_available();
    if initial_state.git_ready {
        initial_state.git = crate::services::git::git_status(&workspace_root);
    }
    let development_root = option_env!("CARGO_MANIFEST_DIR")
        .map(std::path::Path::new)
        .map(|path| path.join("extensions"));
    let mut extension_registry =
        extensions::discover_extensions(&workspace_root, development_root.as_deref());
    extension_registry.load_lockfile(&workspace_root.join(".prumo/extensions.lock.json"));
    initial_state.extensions = extension_registry.records;
    initial_state.extension_errors = extension_registry.errors;
    let extension_host = ExtensionHost::new(workspace_root.clone());
    let grants = match env::var_os("PRUMO_VIEWER_EXTENSION_GRANTS") {
        Some(path) => {
            let path = PathBuf::from(path);
            match load_grants(&path) {
                Ok(grants) => grants,
                Err(error) => {
                    initial_state.extension_errors.push(
                        crate::extensions::registry::ExtensionLoadError {
                            root: path,
                            message: error,
                        },
                    );
                    Vec::new()
                }
            }
        }
        None => Vec::new(),
    };
    for error in extension_host.start(&initial_state.extensions, &grants) {
        initial_state
            .extension_errors
            .push(crate::extensions::registry::ExtensionLoadError {
                root: PathBuf::new(),
                message: error,
            });
    }
    for extension in &mut initial_state.extensions {
        if extension_host.is_active(&extension.manifest.id) {
            extension.status = ExtensionStatus::Ready;
        }
    }

    for candidate in ["README.md", "src/main.rs", "main.go", "Cargo.toml"] {
        let path = workspace_root.join(candidate);
        if path.is_file() && activate_path(&mut initial_state, &path).is_ok() {
            break;
        }
    }

    let client = PrumoClient::new(
        socket_path.or_else(|| initial_state.config.socket()),
        &workspace_root,
    );
    match client.snapshot() {
        Ok(snapshot) => apply_daemon_snapshot(&mut initial_state, &snapshot),
        Err(error) => apply_client_error(&mut initial_state, error.to_string()),
    }

    let terminal_runtime = match TerminalRuntime::new(&workspace_root, initial_state.config.shell())
    {
        Ok(terminal) => {
            initial_state.terminal_running = true;
            Some(terminal)
        }
        Err(error) => {
            initial_state.show_notice(
                NoticeTone::Error,
                format!("Integrated terminal unavailable: {error}"),
            );
            None
        }
    };

    let runtime = match Builder::new_multi_thread()
        .worker_threads(2)
        .enable_all()
        .build()
    {
        Ok(runtime) => runtime,
        Err(error) => {
            eprintln!("Could not start the Prumo Native runtime: {error}");
            return;
        }
    };
    let _runtime_guard = runtime.enter();

    let title = format!(
        "Prumo - {}",
        workspace_root
            .file_name()
            .and_then(|name| name.to_str())
            .unwrap_or("Workspace")
    );
    let leaked_title: &'static str = Box::leak(title.into_boxed_str());
    let app_client = client.clone();
    let app_terminal = terminal_runtime.clone();
    let app_extension_host = extension_host.clone();
    let app_watcher = std::sync::Mutex::new(crate::services::watcher::start_workspace_watcher(
        &workspace_root,
    ));
    launch(
        LaunchConfig::new().with_window(
            WindowConfig::new(move || {
                app_with_extensions(
                    initial_state.clone(),
                    app_client.clone(),
                    app_terminal.clone(),
                    app_watcher.lock().unwrap().take(),
                    app_extension_host.clone(),
                )
            })
            .with_title(leaked_title)
            .with_size(1440., 900.)
            .with_min_size(720., 540.),
        ),
    );
}

#[cfg(test)]
fn app(
    initial_state: AppState,
    client: PrumoClient,
    terminal: Option<TerminalRuntime>,
    watcher: Option<std::sync::mpsc::Receiver<()>>,
) -> impl IntoElement {
    app_with_extensions(
        initial_state,
        client,
        terminal,
        watcher,
        ExtensionHost::new(PathBuf::new()),
    )
}

fn app_with_extensions(
    initial_state: AppState,
    client: PrumoClient,
    terminal: Option<TerminalRuntime>,
    watcher: Option<std::sync::mpsc::Receiver<()>>,
    extension_host: ExtensionHost,
) -> impl IntoElement {
    let configured_theme = initial_state.config.theme.clone();
    let custom_themes = initial_state.config.custom_themes.clone();
    use_init_theme(move || {
        let preferred = *Platform::get().preferred_theme.read();
        theme::resolve_theme(&configured_theme, &custom_themes, Some(preferred))
    });

    let state = use_state(move || initial_state);
    let cursor_state = use_state(EditorCursorState::default);
    let workspace = WorkspaceCommands { state };
    let terminal_for_hook = terminal.clone();
    use_hook(move || {
        let Some(terminal) = terminal_for_hook.clone() else {
            return;
        };
        let mut terminal_state = state;
        spawn(async move {
            let mut parser = vt100::Parser::new(24, 120, 0);
            loop {
                async_io::Timer::after(Duration::from_millis(30)).await;
                let mut changed = false;
                let mut exited = false;
                for event in terminal.drain() {
                    match event {
                        TerminalEvent::Output(bytes) => {
                            parser.process(&bytes);
                            changed = true;
                        }
                        TerminalEvent::Reset => {
                            parser = vt100::Parser::new(24, 120, 0);
                            changed = true;
                        }
                        TerminalEvent::Exited => exited = true,
                    }
                }
                if changed {
                    let output = parser.screen().contents();
                    let mut state = terminal_state.write();
                    if state.terminal_output != output {
                        state.terminal_output = output;
                    }
                }
                if exited {
                    terminal_state.write().terminal_running = false;
                }
            }
        });
    });
    use_hook(|| {
        let mut extension_state = state;
        let extension_event_host = extension_host.clone();
        spawn_forever(async move {
            let mut last_synced_document: Option<(PathBuf, u64)> = None;
            let mut known_open_paths = HashSet::<PathBuf>::new();
            loop {
                async_io::Timer::after(Duration::from_millis(100)).await;
                for request in extension_event_host.poll_requests() {
                    handle_extension_request(
                        extension_event_host.clone(),
                        extension_state,
                        request,
                    )
                    .await;
                }
                for notification in extension_event_host.drain_lsp_notifications() {
                    apply_lsp_notification(extension_state, &notification);
                }
                let current_open_paths = extension_state
                    .read()
                    .tabs
                    .iter()
                    .map(|tab| tab.path.clone())
                    .collect::<HashSet<_>>();
                for path in known_open_paths.difference(&current_open_paths) {
                    if let Some((_, language)) = extension_event_host.active_lsp_for_path(path) {
                        let _ = extension_event_host.close_lsp_document(path, &language);
                    }
                }
                known_open_paths = current_open_paths;
                let active_document = extension_state
                    .read()
                    .active_tab()
                    .map(|tab| (tab.path.clone(), tab.revision));
                if let Some((path, revision)) = active_document {
                    let needs_sync =
                        last_synced_document
                            .as_ref()
                            .is_none_or(|(known_path, known_revision)| {
                                known_path != &path || *known_revision != revision
                            });
                    if needs_sync
                        && let Some((_, language)) = extension_event_host.active_lsp_for_path(&path)
                    {
                        let content = extension_state
                            .read()
                            .active_tab()
                            .map(|tab| tab.content.clone());
                        if let Some(content) = content
                            && extension_event_host
                                .sync_active_document(&path, &language, &content, revision)
                                .is_ok()
                        {
                            last_synced_document = Some((path, revision));
                        }
                    }
                } else {
                    last_synced_document = None;
                }
                let pending_save = extension_state.read().pending_lsp_save.clone();
                if let Some(path) = pending_save
                    && let Some((_, language)) = extension_event_host.active_lsp_for_path(&path)
                    && extension_event_host
                        .notify_lsp_save(&path, &language)
                        .is_ok()
                {
                    extension_state.write().pending_lsp_save = None;
                }
            }
        });
    });
    use_hook(|| {
        let mut polling_state = state;
        let polling_client = client.clone();
        let configured_poll_interval = state.read().config.poll_interval_seconds;
        let workspace_watcher = watcher;
        let workspace_root = state.read().workspace_root.clone();
        let follow_loader = workspace;
        spawn_forever(async move {
            let mut poll_delay = Duration::from_secs(configured_poll_interval);
            let mut last_snapshot = None;
            let mut last_client_error: Option<String> = None;
            let mut active_subscription: Option<(
                String,
                std::sync::mpsc::Receiver<crate::client::protocol::DaemonEvent>,
            )> = None;
            let mut last_snapshot_time = std::time::Instant::now();

            loop {
                async_io::Timer::after(poll_delay).await;

                // 1. Drain streaming events if subscription is active
                let mut streamed_events = Vec::new();
                let mut stream_disconnected = false;
                if let Some((_, ref rx)) = active_subscription {
                    loop {
                        match rx.try_recv() {
                            Ok(event) => streamed_events.push(event),
                            Err(std::sync::mpsc::TryRecvError::Empty) => break,
                            Err(std::sync::mpsc::TryRecvError::Disconnected) => {
                                stream_disconnected = true;
                                break;
                            }
                        }
                    }
                }
                if stream_disconnected {
                    active_subscription = None;
                }

                if !streamed_events.is_empty() {
                    let mut files_changed = Vec::new();
                    {
                        let mut app_state = polling_state.write();
                        for event in &streamed_events {
                            apply_streaming_event(&mut app_state, event);
                            if event.kind == "file.changed" {
                                files_changed.push(event.clone());
                            }
                        }
                    }
                    for event in &files_changed {
                        if let Some(target) = follow::target_from_event(event, &workspace_root) {
                            let content = std::fs::read_to_string(&target.path)
                                .ok()
                                .filter(|_| target.status != AgentFileStatus::Deleted);
                            let action = {
                                let mut app_state = polling_state.write();
                                follow::apply_follow(&mut app_state, &target, content.as_deref())
                            };
                            if let follow::FollowAction::Open { reveal } = action {
                                follow_loader.load_file_with_reveal(target.path.clone(), reveal);
                            }
                        }
                    }
                    if !files_changed.is_empty() {
                        let mut worker_state = polling_state.read().clone();
                        let refresh_result = run_blocking(move || {
                            let _ = refresh_workspace(&mut worker_state);
                            worker_state
                        })
                        .await;
                        if let Ok(worker_state) = refresh_result {
                            let mut app_state = polling_state.write();
                            app_state.tree = worker_state.tree;
                            app_state.flattened_tree = worker_state.flattened_tree;
                            app_state.file_candidates = worker_state.file_candidates;
                        }
                    }
                    poll_delay = Duration::from_millis(50);
                    continue;
                }

                // If active run exists and we don't have a subscription yet, attempt subscribe
                let (active_run_id, is_active_status, event_count) = {
                    let s = polling_state.read();
                    let is_active = s.agent_status == crate::state::AgentStatus::Working
                        || s.agent_status == crate::state::AgentStatus::AwaitingApproval;
                    (s.active_run_id.clone(), is_active, s.agent_events.len())
                };

                let should_subscribe = active_run_id.is_some()
                    && is_active_status
                    && (active_subscription.as_ref().map(|(id, _)| id) != active_run_id.as_ref());

                if should_subscribe
                    && let Some(run_id) = active_run_id {
                        let sub_client = polling_client.clone();
                        let sub_run_id = run_id.clone();
                        let sub_result = run_blocking(move || {
                            sub_client.subscribe(&sub_run_id, event_count)
                        })
                        .await;
                        if let Ok(Ok(rx)) = sub_result {
                            active_subscription = Some((run_id, rx));
                            poll_delay = Duration::from_millis(50);
                            continue;
                        }
                    }

                // If currently subscribed, wait for stream events with fast tick, but periodically reconcile snapshot
                if active_subscription.is_some() && last_snapshot_time.elapsed() < Duration::from_secs(3) {
                    poll_delay = Duration::from_millis(50);
                    continue;
                }

                last_snapshot_time = std::time::Instant::now();

                let mut workspace_changed = false;
                if let Some(watcher) = workspace_watcher.as_ref() {
                    while watcher.try_recv().is_ok() {
                        workspace_changed = true;
                    }
                }
                let (known_event_ids, known_event_count, refresh_git): (
                    HashSet<String>,
                    usize,
                    bool,
                ) = {
                    let app_state = polling_state.read();
                    let events: Vec<String> = app_state
                        .agent_events
                        .iter()
                        .map(|event| event.id.clone())
                        .collect();
                    let refresh_git = app_state.git_ready
                        && app_state.sidebar_visible
                        && app_state.sidebar_view == SidebarView::Git;
                    let event_count = events.len();
                    (events.into_iter().collect(), event_count, refresh_git)
                };
                let worker_client = polling_client.clone();
                let worker_root = workspace_root.clone();
                let result = run_blocking(move || {
                    let snapshot_result = worker_client.snapshot();
                    let should_refresh = snapshot_result.as_ref().is_ok_and(|snapshot| {
                        snapshot.events.iter().any(|event| {
                            event.kind == "file.changed" && !known_event_ids.contains(&event.id)
                        })
                    });
                    let follow_target = snapshot_result.as_ref().ok().and_then(|snapshot| {
                        follow::newest_changed_file(
                            &snapshot.events,
                            &known_event_ids,
                            &worker_root,
                        )
                    });
                    let follow_content = follow_target.as_ref().and_then(|target| {
                        std::fs::read_to_string(&target.path)
                            .ok()
                            .filter(|_| target.status != AgentFileStatus::Deleted)
                    });
                    (
                        snapshot_result,
                        should_refresh,
                        follow_target,
                        follow_content,
                    )
                })
                .await;

                match result {
                    Ok((snapshot_result, should_refresh, follow_target, follow_content)) => {
                        let refresh_workspace_requested = should_refresh || workspace_changed;
                        let worker_state = if refresh_workspace_requested || refresh_git {
                            let mut worker_state = polling_state.read().clone();
                            let result = run_blocking(move || {
                                if refresh_workspace_requested {
                                    let _ = refresh_workspace(&mut worker_state);
                                }
                                if refresh_git {
                                    worker_state.git = git_status(&worker_state.workspace_root);
                                }
                                worker_state
                            })
                            .await;
                            match result {
                                Ok(worker_state) => Some(worker_state),
                                Err(error) => {
                                    polling_state.write().show_notice(
                                        NoticeTone::Error,
                                        format!("Workspace refresh worker failed: {error}"),
                                    );
                                    None
                                }
                            }
                        } else {
                            None
                        };
                        let snapshot_event_count = snapshot_result
                            .as_ref()
                            .map(|snapshot| snapshot.events.len())
                            .unwrap_or_default();
                        let snapshot_changed = match &snapshot_result {
                            Ok(snapshot) => {
                                last_snapshot.as_ref() != Some(snapshot)
                                    || last_client_error.is_some()
                            }
                            Err(error) => {
                                let message = error.to_string();
                                last_client_error.as_deref() != Some(message.as_str())
                            }
                        };
                        if snapshot_changed || refresh_workspace_requested || refresh_git {
                            let mut app_state = polling_state.write();
                            if let Some(worker_state) = worker_state {
                                if refresh_workspace_requested {
                                    app_state.tree = worker_state.tree;
                                    app_state.flattened_tree = worker_state.flattened_tree;
                                    app_state.file_candidates = worker_state.file_candidates;
                                }
                                if refresh_git {
                                    app_state.git = worker_state.git.clone();
                                }
                            }
                            match snapshot_result {
                                Ok(snapshot) => {
                                    apply_daemon_snapshot(&mut app_state, &snapshot);
                                    if let Some(target) = follow_target {
                                        let action = follow::apply_follow(
                                            &mut app_state,
                                            &target,
                                            follow_content.as_deref(),
                                        );
                                        if let follow::FollowAction::Open { reveal } = action {
                                            follow_loader
                                                .load_file_with_reveal(target.path.clone(), reveal);
                                        }
                                    }
                                    last_snapshot = Some(snapshot);
                                    last_client_error = None;
                                    poll_delay = if active_subscription.is_some() {
                                        Duration::from_millis(50)
                                    } else if snapshot_event_count > known_event_count {
                                        Duration::from_secs(configured_poll_interval)
                                    } else {
                                        poll_delay.saturating_mul(2).min(Duration::from_secs(10))
                                    };
                                }
                                Err(error) => {
                                    let message = error.to_string();
                                    apply_client_error(&mut app_state, message.clone());
                                    last_client_error = Some(message);
                                    poll_delay = Duration::from_secs(5);
                                }
                            }
                        } else if snapshot_result.is_ok() {
                            poll_delay = if active_subscription.is_some() {
                                Duration::from_millis(50)
                            } else if snapshot_event_count > known_event_count {
                                Duration::from_secs(configured_poll_interval)
                            } else {
                                poll_delay.saturating_mul(2).min(Duration::from_secs(10))
                            };
                        } else {
                            poll_delay = Duration::from_secs(5);
                        }
                    }
                    Err(error) => {
                        polling_state.write().show_notice(
                            NoticeTone::Error,
                            format!("Protocol worker failed: {error}"),
                        );
                        poll_delay = Duration::from_secs(5);
                    }
                }
            }
        });
    });

    let colors = get_theme_or_default().read().colors.clone();
    let is_quick_open = state.read().quick_open_open;
    let is_palette_open = state.read().palette_open;
    let is_menu_open = state.read().menu_open;
    let has_explorer_menu = state.read().explorer_menu.is_some();
    let has_file_prompt = state.read().file_prompt.is_some();
    let is_diff_open = state.read().diff_open;
    let has_notice = state.read().notice.is_some();
    let has_close_dialog = state.read().pending_close_tab.is_some();
    let has_config = state.read().config_open;
    let has_extension_panel = state.read().extension_panel.is_some();
    let state_for_keys = state;
    let mut state_for_size = state;

    rect()
        .width(Size::fill())
        .height(Size::fill())
        .vertical()
        .content(Content::flex())
        .background(colors.background)
        .on_sized(move |event: Event<SizedEventData>| {
            let width = event.visible_area.size.width;
            if (state_for_size.read().viewport_width - width).abs() > 0.5 {
                state_for_size.write().viewport_width = width;
            }
        })
        .on_global_key_down(move |event: Event<KeyboardEventData>| {
            if dispatch_global_key(state_for_keys, &event.key, event.modifiers) {
                event.stop_propagation();
                event.prevent_default();
            }
        })
        .child(TopBar { state })
        .child(if has_notice {
            Element::from(NoticeBar { state })
        } else {
            Element::from(rect().height(Size::px(0.)))
        })
        .child(
            rect()
                .width(Size::fill())
                .height(Size::flex(1.))
                .child(Body {
                    state,
                    cursor_state,
                    client: client.clone(),
                    workspace,
                    terminal: terminal.clone(),
                    extension_host: extension_host.clone(),
                }),
        )
        .child(StatusBar { state })
        .child(if is_quick_open {
            Element::from(QuickOpenModal { state, workspace })
        } else {
            Element::from(rect())
        })
        .child(if is_palette_open {
            Element::from(CommandPaletteModal {
                state,
                cursor_state,
                workspace,
                terminal: terminal.clone(),
                extension_host: extension_host.clone(),
            })
        } else {
            Element::from(rect())
        })
        .child(if is_menu_open {
            Element::from(MenuOverlay { state })
        } else {
            Element::from(rect())
        })
        .child(if is_diff_open {
            Element::from(DiffModal { state })
        } else {
            Element::from(rect())
        })
        .child(if has_extension_panel {
            Element::from(ExtensionPanelOverlay { state })
        } else {
            Element::from(rect())
        })
        .child(if has_config {
            Element::from(SettingsModal {
                state,
                client: client.clone(),
            })
        } else {
            Element::from(rect())
        })
        .child(if has_close_dialog {
            Element::from(CloseDialog { state })
        } else {
            Element::from(rect())
        })
        .child(if has_explorer_menu {
            Element::from(ExplorerMenuOverlay { state })
        } else {
            Element::from(rect())
        })
        .child(if has_file_prompt {
            Element::from(FilePromptModal { state, workspace })
        } else {
            Element::from(rect())
        })
}

#[derive(PartialEq)]
struct Body {
    state: State<AppState>,
    cursor_state: State<EditorCursorState>,
    client: PrumoClient,
    workspace: WorkspaceCommands,
    terminal: Option<TerminalRuntime>,
    extension_host: ExtensionHost,
}

impl Component for Body {
    fn render(&self) -> impl IntoElement {
        let state = self.state;
        let client = self.client.clone();
        let viewport_width = state.read().viewport_width;
        let requested_sidebar = state.read().sidebar_visible;
        let requested_agent = state.read().agent_panel_visible;
        let both_panels_fit = viewport_width >= 1180.;
        let sidebar_visible = requested_sidebar
            && viewport_width >= SIDEBAR_BREAKPOINT
            && (!requested_agent || both_panels_fit);
        let agent_visible = requested_agent && viewport_width >= AGENT_BREAKPOINT;

        let editor_view = rect()
            .width(Size::flex(1.))
            .height(Size::fill())
            .vertical()
            .content(Content::flex())
            .child(TabBar { state })
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::flex(1.))
                    .content(Content::flex())
                    .child(EditorArea {
                        state,
                        cursor_state: self.cursor_state,
                        extension_host: self.extension_host.clone(),
                    }),
            );

        let upper_content = if agent_visible {
            let panel_width = state.read().agent_panel_width.clamp(260., 750.);
            rect()
                .width(Size::fill())
                .height(Size::flex(1.))
                .horizontal()
                .content(Content::flex())
                .child(editor_view)
                .child(AgentPanelResizeHandle { state })
                .child(
                    rect()
                        .width(Size::px(panel_width))
                        .height(Size::fill())
                        .child(AgentPanel {
                            state,
                            client: client.clone(),
                        }),
                )
        } else {
            rect()
                .width(Size::fill())
                .height(Size::flex(1.))
                .horizontal()
                .content(Content::flex())
                .child(editor_view)
        };

        let center_panel = ResizablePanel::new(PanelSize::percent(1.))
            .min_size(0.25)
            .key("workspace-center")
            .order(1usize)
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::fill())
                    .vertical()
                    .content(Content::flex())
                    .child(upper_content)
                    .child(BottomDock {
                        state,
                        workspace: self.workspace,
                        terminal: self.terminal.clone(),
                    }),
            );
        let workspace_container = ResizableContainer::new().direction(Direction::Horizontal);
        let workspace_container = if sidebar_visible {
            workspace_container
                .panel(
                    ResizablePanel::new(PanelSize::px(264.))
                        .min_size(220.)
                        .key("workspace-sidebar")
                        .order(0usize)
                        .child(Sidebar {
                            state,
                            workspace: self.workspace,
                        }),
                )
                .panel(center_panel)
        } else {
            workspace_container.panel(center_panel)
        };

        rect()
            .width(Size::fill())
            .height(Size::fill())
            .horizontal()
            .content(Content::flex())
            .child(ActivityBar { state })
            .child(workspace_container)
    }
}

#[derive(PartialEq)]
struct AgentPanelResizeHandle {
    state: State<AppState>,
}

impl Component for AgentPanelResizeHandle {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let mut clicking = use_state(|| false);
        let mut hovering = use_state(|| false);
        let mut start_x = use_state(|| 0.0f32);
        let mut start_width = use_state(|| 360.0f32);

        use_drop(move || {
            if *hovering.peek() {
                Cursor::set(CursorIcon::default());
            }
        });

        let on_pointer_enter = move |_| {
            hovering.set(true);
            Cursor::set(CursorIcon::ColResize);
        };

        let on_pointer_leave = move |_| {
            hovering.set(false);
            if !*clicking.read() {
                Cursor::set(CursorIcon::default());
            }
        };

        let on_pointer_down = move |e: Event<PointerEventData>| {
            if !e.data().is_primary() {
                return;
            }
            e.stop_propagation();
            e.prevent_default();
            let coords = e.global_location();
            start_x.set(coords.x as f32);
            start_width.set(state.read().agent_panel_width);
            clicking.set(true);
        };

        let on_capture_global_pointer_move = move |e: Event<PointerEventData>| {
            if *clicking.read() {
                e.prevent_default();
                let coords = e.global_location();
                let delta = (coords.x as f32) - *start_x.read();
                let new_width = (*start_width.read() - delta).clamp(260.0, 750.0);
                if (state.read().agent_panel_width - new_width).abs() >= 1.0 {
                    state.write().agent_panel_width = new_width;
                }
            }
        };

        let on_global_pointer_press = move |_: Event<PointerEventData>| {
            if *clicking.read() {
                if !*hovering.peek() {
                    Cursor::set(CursorIcon::default());
                }
                clicking.set(false);
            }
        };

        let is_active = *clicking.read() || *hovering.read();
        let handle_color = if is_active {
            colors.primary
        } else {
            colors.border
        };

        rect()
            .width(Size::px(4.))
            .height(Size::fill())
            .background(handle_color)
            .on_pointer_enter(on_pointer_enter)
            .on_pointer_leave(on_pointer_leave)
            .on_pointer_down(on_pointer_down)
            .on_capture_global_pointer_move(on_capture_global_pointer_move)
            .on_global_pointer_press(on_global_pointer_press)
    }
}

#[derive(PartialEq)]
struct NoticeBar {
    state: State<AppState>,
}

impl Component for NoticeBar {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let notice = state.read().notice.clone();
        let Some(notice) = notice else {
            return rect().height(Size::px(0.));
        };
        let color = match notice.tone {
            NoticeTone::Info => colors.primary,
            NoticeTone::Success => colors.success,
            NoticeTone::Error => colors.error,
        };

        rect()
            .width(Size::fill())
            .min_height(Size::px(26.))
            .horizontal()
            .cross_align(Alignment::center())
            .padding(Gaps::new(0., 8., 0., 8.))
            .spacing(8.)
            .background(colors.surface_secondary)
            .border(Border::new().fill(color).width(BorderWidth {
                top: 0.,
                right: 0.,
                bottom: 1.,
                left: 3.,
            }))
            .child(
                label()
                    .color(colors.text_primary)
                    .font_size(10.5)
                    .text(notice.message),
            )
            .child(
                Button::new()
                    .flat()
                    .compact()
                    .height(Size::px(22.))
                    .padding(Gaps::new(8., 0., 8., 0.))
                    .on_press(move |_| state.write().clear_notice())
                    .child(
                        label()
                            .color(colors.text_secondary)
                            .font_size(10.5)
                            .text("Dismiss"),
                    ),
            )
    }
}

#[derive(PartialEq)]
struct CloseDialog {
    state: State<AppState>,
}

impl Component for CloseDialog {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let pending_index = state.read().pending_close_tab;
        let title = pending_index
            .and_then(|index| state.read().tabs.get(index).cloned())
            .map(|tab| tab.title)
            .unwrap_or_else(|| "File".to_string());
        let window = Platform::get().root_size.read();

        rect()
            .width(Size::px(window.width))
            .height(Size::px(window.height))
            .position(Position::new_global().top(0.).left(0.))
            .layer(Layer::Overlay)
            .background(Color::TRANSPARENT)
            .horizontal()
            .main_align(Alignment::center())
            .cross_align(Alignment::center())
            .on_mouse_up(move |_| {
                state.write().pending_close_tab = None;
            })
            .child(
                rect()
                    .width(Size::px(400.))
                    .vertical()
                    .padding(Gaps::new_all(12.))
                    .spacing(10.)
                    .background(colors.surface_primary)
                    .corner_radius(CornerRadius::new_all(8.))
                    .border(Border::new().fill(colors.border_focus).width(1.))
                    .on_mouse_up(|event: Event<MouseEventData>| {
                        event.stop_propagation();
                    })
                    .child(
                        label()
                            .color(colors.text_primary)
                            .font_size(13.)
                            .font_weight(FontWeight::MEDIUM)
                            .text("Unsaved changes"),
                    )
                    .child(
                        label()
                            .color(colors.text_secondary)
                            .font_size(11.)
                            .text(format!("Save changes to {title} before closing?")),
                    )
                    .child(
                        rect()
                            .width(Size::fill())
                            .horizontal()
                            .main_align(Alignment::end())
                            .spacing(8.)
                            .child(
                                Button::new()
                                    .flat()
                                    .compact()
                                    .height(Size::px(26.))
                                    .padding(Gaps::new(0., 10., 0., 10.))
                                    .on_press(move |_| {
                                        state.write().pending_close_tab = None;
                                    })
                                    .child(
                                        label()
                                            .color(colors.text_secondary)
                                            .font_size(11.)
                                            .text("Cancel"),
                                    ),
                            )
                            .child(
                                Button::new()
                                    .flat()
                                    .compact()
                                    .height(Size::px(26.))
                                    .padding(Gaps::new(0., 10., 0., 10.))
                                    .on_press(move |_| {
                                        let mut app_state = state.write();
                                        crate::services::document::discard_pending_tab(
                                            &mut app_state,
                                        );
                                    })
                                    .child(
                                        label().color(colors.error).font_size(11.).text("Discard"),
                                    ),
                            )
                            .child(
                                Button::new()
                                    .compact()
                                    .height(Size::px(26.))
                                    .padding(Gaps::new(0., 12., 0., 12.))
                                    .on_press(move |_| {
                                        let mut app_state = state.write();
                                        crate::services::document::save_and_close_pending_tab(
                                            &mut app_state,
                                        );
                                    })
                                    .child(
                                        label()
                                            .color(colors.primary)
                                            .font_size(11.)
                                            .text("Save and close"),
                                    ),
                            ),
                    ),
            )
    }
}

#[cfg(test)]
mod ui_tests {
    use super::*;
    use freya_testing::prelude::TestingRunner;
    use std::{env, fs, path::PathBuf};

    #[test]
    fn narrow_explorer_click_replaces_agent_without_taking_the_editor() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().to_path_buf();
        let mut app_state = AppState::new(root.clone());
        refresh_workspace(&mut app_state).unwrap();
        app_state.viewport_width = 1024.;
        let client = PrumoClient::new(Some(root.join(".prumo/agent.sock")), &root);
        let extension_host = ExtensionHost::new(root.clone());
        let (mut runner, state) = TestingRunner::new(
            move || {
                use_init_theme(theme::dark);
                let state = use_consume::<State<AppState>>();
                Body {
                    state,
                    cursor_state: State::create(EditorCursorState::default()),
                    client: client.clone(),
                    workspace: WorkspaceCommands { state },
                    terminal: None,
                    extension_host: extension_host.clone(),
                }
            },
            (1024., 720.).into(),
            move |runner| runner.provide_root_context(|| State::create(app_state)),
            1.,
        );

        runner.sync_and_update();
        runner.click_cursor((20., 24.));
        assert!(state.peek().sidebar_visible);
        assert!(!state.peek().agent_panel_visible);
    }

    #[test]
    fn settings_modal_renders_all_real_controls() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().to_path_buf();
        let mut app_state = AppState::new(root.clone());
        app_state.config_open = true;
        let client = PrumoClient::new(Some(root.join(".prumo/agent.sock")), &root);
        let (mut runner, state) = TestingRunner::new(
            move || {
                use_init_theme(theme::dark);
                SettingsModal {
                    state: use_consume(),
                    client: client.clone(),
                }
            },
            (1440., 900.).into(),
            move |runner| runner.provide_root_context(|| State::create(app_state)),
            1.,
        );
        let _ = state;
        runner.sync_and_update();
        let snapshot = env::var_os("PRUMO_VIEWER_SETTINGS_SNAPSHOT")
            .map(PathBuf::from)
            .unwrap_or_else(|| directory.path().join("viewer-settings.png"));
        runner.render_to_file(&snapshot);
        assert!(snapshot.is_file());
    }

    #[test]
    fn settings_modal_renders_inside_full_app_shell() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().to_path_buf();
        let mut app_state = AppState::new(root.clone());
        app_state.config_open = true;
        let client = PrumoClient::new(Some(root.join(".prumo/agent.sock")), &root);
        let app_client = client.clone();
        let (mut runner, ()) = TestingRunner::new(
            move || {
                use_init_theme(theme::dark);
                app(app_state.clone(), app_client.clone(), None, None)
            },
            (1440., 900.).into(),
            |_| {},
            1.,
        );

        runner.sync_and_update();
        let snapshot = env::var_os("PRUMO_VIEWER_SETTINGS_IN_APP_SNAPSHOT")
            .map(PathBuf::from)
            .unwrap_or_else(|| directory.path().join("viewer-settings-in-app.png"));
        runner.render_to_file(&snapshot);
        assert!(snapshot.is_file());
    }

    #[test]
    fn menu_and_explorer_overlays_render_inside_full_app_shell() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().to_path_buf();
        fs::create_dir_all(root.join("src")).unwrap();
        fs::write(root.join("src/main.rs"), "fn main() {}\n").unwrap();
        let mut app_state = AppState::new(root.clone());
        refresh_workspace(&mut app_state).unwrap();
        app_state.menu_open = true;
        app_state.explorer_menu = Some(crate::state::ExplorerMenu {
            path: root.join("src/main.rs"),
            is_dir: false,
            is_empty: false,
            x: 120.,
            y: 120.,
        });
        app_state.file_prompt = Some(crate::state::FilePrompt {
            kind: crate::state::FilePromptKind::Rename(root.join("src/main.rs")),
            input: "main.rs".to_string(),
        });
        let client = PrumoClient::new(Some(root.join(".prumo/agent.sock")), &root);
        let app_client = client.clone();
        let (mut runner, ()) = TestingRunner::new(
            move || {
                use_init_theme(theme::dark);
                app(app_state.clone(), app_client.clone(), None, None)
            },
            (1440., 900.).into(),
            |_| {},
            1.,
        );

        runner.sync_and_update();
        let snapshot = env::var_os("PRUMO_VIEWER_OVERLAYS_SNAPSHOT")
            .map(PathBuf::from)
            .unwrap_or_else(|| directory.path().join("viewer-overlays.png"));
        runner.render_to_file(&snapshot);
        assert!(snapshot.is_file());
    }

    #[test]
    fn command_palette_lists_real_commands() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().to_path_buf();
        let mut app_state = AppState::new(root.clone());
        let extension_root = PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("extensions");
        let extension_registry = extensions::discover_extensions(&root, Some(&extension_root));
        app_state.extensions = extension_registry.records;
        app_state.palette_open = true;
        let client = PrumoClient::new(Some(root.join(".prumo/agent.sock")), &root);
        let app_client = client.clone();
        let (mut runner, ()) = TestingRunner::new(
            move || {
                use_init_theme(theme::dark);
                app(app_state.clone(), app_client.clone(), None, None)
            },
            (1440., 900.).into(),
            |_| {},
            1.,
        );

        runner.sync_and_update();
        let snapshot = env::var_os("PRUMO_VIEWER_PALETTE_SNAPSHOT")
            .map(PathBuf::from)
            .unwrap_or_else(|| directory.path().join("viewer-palette.png"));
        runner.render_to_file(&snapshot);
        assert!(snapshot.is_file());
    }

    #[test]
    fn find_bar_and_search_dock_render() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().to_path_buf();
        fs::create_dir_all(root.join("src")).unwrap();
        fs::write(
            root.join("src/main.rs"),
            "fn main() {\n    println!(\"needle\");\n}\n",
        )
        .unwrap();
        let mut app_state = AppState::new(root.clone());
        refresh_workspace(&mut app_state).unwrap();
        activate_path(&mut app_state, &root.join("src/main.rs")).unwrap();
        app_state.find_open = true;
        app_state.dock_open = true;
        app_state.dock_view = crate::state::DockView::Search;
        app_state.search_query = "needle".to_string();
        let client = PrumoClient::new(Some(root.join(".prumo/agent.sock")), &root);
        let app_client = client.clone();
        let (mut runner, ()) = TestingRunner::new(
            move || {
                use_init_theme(theme::dark);
                app(app_state.clone(), app_client.clone(), None, None)
            },
            (1440., 900.).into(),
            |_| {},
            1.,
        );

        runner.sync_and_update();
        let snapshot = env::var_os("PRUMO_VIEWER_FIND_SNAPSHOT")
            .map(PathBuf::from)
            .unwrap_or_else(|| directory.path().join("viewer-find.png"));
        runner.render_to_file(&snapshot);
        assert!(snapshot.is_file());
    }

    #[test]
    fn find_input_reveals_selected_match() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().to_path_buf();
        fs::create_dir_all(root.join("src")).unwrap();
        let mut content = String::new();
        for line in 1..=80 {
            content.push_str(&format!("line {line}\n"));
        }
        content.push_str("needle\n");
        fs::write(root.join("src/main.rs"), content).unwrap();
        let mut app_state = AppState::new(root.clone());
        refresh_workspace(&mut app_state).unwrap();
        activate_path(&mut app_state, &root.join("src/main.rs")).unwrap();
        app_state.find_open = true;
        let client = PrumoClient::new(Some(root.join(".prumo/agent.sock")), &root);
        let app_client = client.clone();
        let (mut runner, ()) = TestingRunner::new(
            move || {
                use_init_theme(theme::dark);
                app(app_state.clone(), app_client.clone(), None, None)
            },
            (1440., 900.).into(),
            |_| {},
            1.,
        );

        runner.sync_and_update();
        runner.click_cursor((500., 105.));
        for character in "needle".chars() {
            runner.write_text(character);
        }
        runner.press_key(freya::prelude::Key::Named(freya::prelude::NamedKey::Enter));
        runner.sync_and_update();
        let snapshot = env::var_os("PRUMO_VIEWER_FIND_REVEAL_SNAPSHOT")
            .map(PathBuf::from)
            .unwrap_or_else(|| directory.path().join("viewer-find-reveal.png"));
        runner.render_to_file(&snapshot);
        assert!(snapshot.is_file());
    }

    #[test]
    fn conflict_banner_renders_resolution_actions() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().to_path_buf();
        fs::write(root.join("main.rs"), "fn main() {}\n").unwrap();
        let mut app_state = AppState::new(root.clone());
        refresh_workspace(&mut app_state).unwrap();
        activate_path(&mut app_state, &root.join("main.rs")).unwrap();
        app_state.set_active_conflict("fn main() { /* agent */ }\n".to_string());
        let client = PrumoClient::new(Some(root.join(".prumo/agent.sock")), &root);
        let app_client = client.clone();
        let (mut runner, ()) = TestingRunner::new(
            move || {
                use_init_theme(theme::dark);
                app(app_state.clone(), app_client.clone(), None, None)
            },
            (1440., 900.).into(),
            |_| {},
            1.,
        );

        runner.sync_and_update();
        let snapshot = env::var_os("PRUMO_VIEWER_CONFLICT_SNAPSHOT")
            .map(PathBuf::from)
            .unwrap_or_else(|| directory.path().join("viewer-conflict.png"));
        runner.render_to_file(&snapshot);
        assert!(snapshot.is_file());
    }

    #[test]
    fn problems_dock_renders_diagnostics() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().to_path_buf();
        fs::write(root.join("main.rs"), "fn main() {}\n").unwrap();
        let mut app_state = AppState::new(root.clone());
        refresh_workspace(&mut app_state).unwrap();
        activate_path(&mut app_state, &root.join("main.rs")).unwrap();
        app_state.dock_open = true;
        app_state.dock_view = crate::state::DockView::Problems;
        app_state.diagnostics = vec![crate::state::Diagnostic {
            path: root.join("main.rs"),
            line: 1,
            column: 4,
            end_line: Some(1),
            end_column: Some(8),
            message: "Example diagnostic".to_string(),
            severity: crate::state::DiagnosticSeverity::Warning,
            source: "test".to_string(),
            code: Some("W001".to_string()),
        }];
        let client = PrumoClient::new(Some(root.join(".prumo/agent.sock")), &root);
        let app_client = client.clone();
        let (mut runner, ()) = TestingRunner::new(
            move || {
                use_init_theme(theme::dark);
                app(app_state.clone(), app_client.clone(), None, None)
            },
            (1440., 900.).into(),
            |_| {},
            1.,
        );

        runner.sync_and_update();
        let snapshot = env::var_os("PRUMO_VIEWER_PROBLEMS_SNAPSHOT")
            .map(PathBuf::from)
            .unwrap_or_else(|| directory.path().join("viewer-problems.png"));
        runner.render_to_file(&snapshot);
        assert!(snapshot.is_file());
    }

    #[test]
    fn editor_popup_renders_completion_rows() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().to_path_buf();
        fs::write(root.join("main.rs"), "fn main() {}\n").unwrap();
        let mut app_state = AppState::new(root.clone());
        refresh_workspace(&mut app_state).unwrap();
        activate_path(&mut app_state, &root.join("main.rs")).unwrap();
        let (path, revision) = app_state
            .active_tab()
            .map(|tab| (tab.path.clone(), tab.revision))
            .unwrap();
        app_state.active_cursor_position = 5;
        app_state.active_selection_range = None;
        let request_id = app_state.begin_editor_popup(
            crate::state::EditorPopupKind::Completion,
            path,
            revision,
            5,
            None,
            Some((5, 5)),
        );
        let popup = app_state.editor_popup.as_mut().unwrap();
        popup.request_id = request_id;
        popup.loading = false;
        popup.items = vec![crate::state::EditorPopupItem::Completion(
            prumo_extension_sdk::lsp::LspCompletionItem {
                label: "println".to_string(),
                kind: Some(3),
                detail: Some("macro".to_string()),
                insert_text: None,
                insert_text_format: None,
                text_edit: None,
            },
        )];
        let client = PrumoClient::new(Some(root.join(".prumo/agent.sock")), &root);
        let app_client = client.clone();
        let (mut runner, ()) = TestingRunner::new(
            move || {
                use_init_theme(theme::dark);
                app(app_state.clone(), app_client.clone(), None, None)
            },
            (1440., 900.).into(),
            |_| {},
            1.,
        );

        runner.sync_and_update();
        let snapshot = env::var_os("PRUMO_VIEWER_COMPLETION_SNAPSHOT")
            .map(PathBuf::from)
            .unwrap_or_else(|| directory.path().join("viewer-completion.png"));
        runner.render_to_file(&snapshot);
        assert!(snapshot.is_file());
    }

    #[test]
    fn follow_agent_renders_its_toggle_with_a_dirty_conflict() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().to_path_buf();
        let mut app_state = AppState::new(root.clone());
        app_state.agent_panel_visible = true;
        app_state.connection_status = crate::state::ConnectionStatus::Connected;
        app_state.active_run_id = Some("R-1".to_string());
        app_state.agent_status = crate::state::AgentStatus::Working;
        app_state.follow_conflict_paths = vec!["src/main.rs".to_string()];
        let client = PrumoClient::new(Some(root.join(".prumo/agent.sock")), &root);
        let app_client = client.clone();
        let (mut runner, ()) = TestingRunner::new(
            move || {
                use_init_theme(theme::dark);
                app(app_state.clone(), app_client.clone(), None, None)
            },
            (1440., 900.).into(),
            |_| {},
            1.,
        );

        runner.sync_and_update();
        let snapshot = env::var_os("PRUMO_VIEWER_FOLLOW_SNAPSHOT")
            .map(PathBuf::from)
            .unwrap_or_else(|| directory.path().join("viewer-follow.png"));
        runner.render_to_file(&snapshot);
        assert!(snapshot.is_file());
    }

    #[test]
    fn quality_and_evidence_docks_render_run_facts() {
        for (view, snapshot_name) in [
            (crate::state::DockView::Quality, "viewer-quality.png"),
            (crate::state::DockView::Evidence, "viewer-evidence.png"),
        ] {
            let directory = tempfile::tempdir().unwrap();
            let root = directory.path().to_path_buf();
            fs::write(root.join("main.rs"), "fn main() {}\n").unwrap();
            let mut app_state = AppState::new(root.clone());
            refresh_workspace(&mut app_state).unwrap();
            app_state.connection_status = crate::state::ConnectionStatus::Connected;
            app_state.active_run_id = Some("R-1".to_string());
            app_state.agent_status = crate::state::AgentStatus::Failed;
            app_state.changed_files = vec![crate::state::ChangedFile {
                path: "main.rs".to_string(),
                status: crate::state::AgentFileStatus::Modified,
            }];
            app_state.agent_events = vec![
                crate::state::AgentEventItem {
                    id: "e1".to_string(),
                    time: "10:00:00".to_string(),
                    kind: "tool.failed".to_string(),
                    message: "Tool · cargo test exited with 1".to_string(),
                },
                crate::state::AgentEventItem {
                    id: "e2".to_string(),
                    time: "10:00:02".to_string(),
                    kind: "run.finished".to_string(),
                    message: "Run finished · failed".to_string(),
                },
            ];
            app_state.dock_open = true;
            app_state.dock_view = view;
            let client = PrumoClient::new(Some(root.join(".prumo/agent.sock")), &root);
            let app_client = client.clone();
            let (mut runner, ()) = TestingRunner::new(
                move || {
                    use_init_theme(theme::dark);
                    app(app_state.clone(), app_client.clone(), None, None)
                },
                (1440., 900.).into(),
                |_| {},
                1.,
            );

            runner.sync_and_update();
            let snapshot = directory.path().join(snapshot_name);
            runner.render_to_file(&snapshot);
            assert!(snapshot.is_file());
        }
    }

    #[test]
    fn git_view_renders_status() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().to_path_buf();
        fs::create_dir_all(root.join("src")).unwrap();
        fs::write(root.join("src/main.rs"), "fn main() {}\n").unwrap();
        let mut app_state = AppState::new(root.clone());
        refresh_workspace(&mut app_state).unwrap();
        app_state.sidebar_view = crate::state::SidebarView::Git;
        app_state.git_ready = true;
        app_state.git = crate::services::git::GitStatus {
            is_repo: true,
            branch: "main".to_string(),
            ahead: 1,
            behind: 0,
            files: vec![
                crate::services::git::GitFile {
                    path: "src/main.rs".to_string(),
                    staged_kind: Some('M'),
                    unstaged_kind: None,
                },
                crate::services::git::GitFile {
                    path: "new.txt".to_string(),
                    staged_kind: None,
                    unstaged_kind: Some('?'),
                },
            ],
        };
        let client = PrumoClient::new(Some(root.join(".prumo/agent.sock")), &root);
        let app_client = client.clone();
        let (mut runner, ()) = TestingRunner::new(
            move || {
                use_init_theme(theme::dark);
                app(app_state.clone(), app_client.clone(), None, None)
            },
            (1440., 900.).into(),
            |_| {},
            1.,
        );

        runner.sync_and_update();
        let snapshot = env::var_os("PRUMO_VIEWER_GIT_SNAPSHOT")
            .map(PathBuf::from)
            .unwrap_or_else(|| directory.path().join("viewer-git.png"));
        runner.render_to_file(&snapshot);
        assert!(snapshot.is_file());
    }

    #[test]
    fn integrated_terminal_dock_renders_with_live_pty() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().to_path_buf();
        fs::create_dir_all(root.join("src")).unwrap();
        fs::write(root.join("src/main.rs"), "fn main() {}\n").unwrap();
        let mut app_state = AppState::new(root.clone());
        refresh_workspace(&mut app_state).unwrap();
        activate_path(&mut app_state, &root.join("src/main.rs")).unwrap();
        app_state.dock_open = true;
        app_state.dock_view = crate::state::DockView::Terminal;
        app_state.terminal_running = true;
        let client = PrumoClient::new(Some(root.join(".prumo/agent.sock")), &root);
        let terminal = TerminalRuntime::new(&root, Some("/bin/sh".to_string())).unwrap();
        terminal
            .write_bytes(b"printf '__prumo_terminal_visible__\\n'\r")
            .unwrap();
        let app_client = client.clone();
        let app_terminal = terminal.clone();
        let tokio_runtime = tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .build()
            .unwrap();
        let _runtime_guard = tokio_runtime.enter();
        let (mut runner, ()) = TestingRunner::new(
            move || {
                app(
                    app_state.clone(),
                    app_client.clone(),
                    Some(app_terminal.clone()),
                    None,
                )
            },
            (1440., 900.).into(),
            |_| {},
            1.,
        );

        runner.poll(
            std::time::Duration::from_millis(20),
            std::time::Duration::from_millis(500),
        );
        let snapshot = env::var_os("PRUMO_VIEWER_TERMINAL_SNAPSHOT")
            .map(PathBuf::from)
            .unwrap_or_else(|| directory.path().join("viewer-terminal.png"));
        runner.render_to_file(&snapshot);
        assert!(snapshot.is_file());
    }

    #[test]
    fn native_code_editor_fills_the_center_region() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().to_path_buf();
        fs::create_dir_all(root.join("src")).unwrap();
        fs::write(
            root.join("src/main.rs"),
            "fn main() {\n    println!(\"Prumo\");\n}\n",
        )
        .unwrap();
        let mut app_state = AppState::new(root.clone());
        refresh_workspace(&mut app_state).unwrap();
        activate_path(&mut app_state, &root.join("src/main.rs")).unwrap();
        app_state.viewport_width = 1440.;
        let client = PrumoClient::new(Some(root.join(".prumo/agent.sock")), &root);
        let extension_host = ExtensionHost::new(root.clone());
        let (mut runner, _) = TestingRunner::new(
            move || {
                use_init_theme(theme::dark);
                let state = use_consume::<State<AppState>>();
                Body {
                    state,
                    cursor_state: State::create(EditorCursorState::default()),
                    client: client.clone(),
                    workspace: WorkspaceCommands { state },
                    terminal: None,
                    extension_host: extension_host.clone(),
                }
            },
            (1440., 900.).into(),
            move |runner| runner.provide_root_context(|| State::create(app_state)),
            1.,
        );

        runner.sync_and_update();
        let editor_area = runner
            .find(|node, element| {
                if element.accessibility().builder.role() != AccessibilityRole::TextInput {
                    return None;
                }
                let area = node.layout().area;
                (area.size.height > 300.).then_some(area)
            })
            .expect("native Freya CodeEditor must be present");
        assert!(editor_area.size.width > 500.);
        assert!(editor_area.size.height > 500.);

        let snapshot = env::var_os("PRUMO_VIEWER_TEST_SNAPSHOT")
            .map(PathBuf::from)
            .unwrap_or_else(|| directory.path().join("viewer-layout.png"));
        runner.render_to_file(&snapshot);
        assert!(snapshot.is_file());
    }

    #[test]
    fn explorer_shortcuts_create_and_edit_selection() {
        let directory = tempfile::tempdir().unwrap();
        let mut state = AppState::new(directory.path().to_path_buf());
        let key = Key::Character("n".to_string());
        assert!(matches!(
            explorer_shortcut(&state, &key, Modifiers::CONTROL),
            Some(ExplorerShortcutAction::NewFile)
        ));
        assert!(matches!(
            explorer_shortcut(&state, &key, Modifiers::CONTROL | Modifiers::SHIFT),
            Some(ExplorerShortcutAction::NewFolder)
        ));
        assert!(matches!(
            explorer_shortcut(&state, &key, Modifiers::CONTROL | Modifiers::ALT),
            Some(ExplorerShortcutAction::NewFolder)
        ));
        state.explorer_focused = true;
        state.selected_path = Some(directory.path().join("file.txt"));
        assert!(matches!(
            explorer_shortcut(&state, &Key::Named(NamedKey::F2), Modifiers::empty()),
            Some(ExplorerShortcutAction::Rename)
        ));
        open_explorer_shortcut(&mut state, ExplorerShortcutAction::Rename);
        assert!(matches!(
            state.file_prompt.as_ref().map(|prompt| &prompt.kind),
            Some(FilePromptKind::Rename(_))
        ));
        open_explorer_shortcut(&mut state, ExplorerShortcutAction::Delete);
        assert!(matches!(
            state.file_prompt.as_ref().map(|prompt| &prompt.kind),
            Some(FilePromptKind::ConfirmTrash(_))
        ));
    }

    #[test]
    fn parses_lsp_diagnostic_notifications() {
        let notification = LspNotification {
            extension_id: "example".to_string(),
            language: "rust".to_string(),
            method: "textDocument/publishDiagnostics".to_string(),
            params: json!({
                "uri": "file:///tmp/main.rs",
                "diagnostics": [{
                    "range": {
                        "start": { "line": 0, "character": 2 },
                        "end": { "line": 0, "character": 5 }
                    },
                    "severity": 1,
                    "source": "rust-analyzer",
                    "message": "Example"
                }]
            }),
        };
        let (path, diagnostics) = parse_lsp_diagnostics(&notification).unwrap();
        assert_eq!(path, PathBuf::from("/tmp/main.rs"));
        assert_eq!(diagnostics.len(), 1);
        assert_eq!(diagnostics[0].line, 1);
        assert_eq!(diagnostics[0].severity, DiagnosticSeverity::Error);
    }
}
