use crate::WorkspaceCommands;
use crate::extensions::host::ExtensionHost;
use crate::extensions::registry::ExtensionStatus;
use crate::services::document::{
    file_uri_to_path, request_close_tab, save_active_tab, utf16_position_to_line_character,
};
use crate::services::terminal::TerminalRuntime;
use crate::state::{
    AppState, DockView, EditorCursorState, EditorReveal, ExtensionPanel, ExtensionPanelKind,
    FilePrompt, FilePromptKind, NoticeTone,
};
use crate::theme;
use crate::ui::icons::icon;
use freya::prelude::*;
use serde_json::json;
use std::time::Duration;

#[derive(Clone, Copy)]
struct PaletteEditorContext {
    state: State<AppState>,
    cursor_state: State<EditorCursorState>,
}

#[derive(Clone, Copy, PartialEq, Eq)]
enum PaletteAction {
    GoToFile,
    FindInFile,
    FindInProject,
    ShowProblems,
    ToggleSidebar,
    ToggleAgent,
    ToggleDock,
    OpenTerminal,
    RestartTerminal,
    OpenSettings,
    ToggleTheme,
    ToggleFollowAgent,
    SaveFile,
    CloseTab,
    RefreshWorkspace,
    NewFile,
    NewFolder,
    Extension(usize, usize),
    ExtensionTheme(usize, usize),
    ExtensionPanel(usize, usize, ExtensionPanelKind),
    ExtensionTask(usize, usize),
    ExtensionDebugger(usize, usize),
    LspWorkspaceSymbols,
    LspHover,
    LspDefinition,
}

#[derive(Clone, PartialEq, Eq)]
struct PaletteEntry {
    action: PaletteAction,
    title: String,
    hint: String,
}

const PALETTE_COMMANDS: [(PaletteAction, &str, &str); 20] = [
    (PaletteAction::GoToFile, "Go to File", "Ctrl+P"),
    (PaletteAction::FindInFile, "Find in File", "Ctrl+F"),
    (
        PaletteAction::FindInProject,
        "Find in Project",
        "Ctrl+Shift+F",
    ),
    (PaletteAction::ShowProblems, "Show Problems", "Ctrl+Shift+M"),
    (PaletteAction::ToggleSidebar, "Toggle Sidebar", "Ctrl+B"),
    (PaletteAction::ToggleAgent, "Toggle Agent Panel", ""),
    (PaletteAction::ToggleDock, "Toggle Dock", "Ctrl+J"),
    (PaletteAction::OpenTerminal, "Open Terminal", ""),
    (PaletteAction::RestartTerminal, "Restart Terminal", ""),
    (PaletteAction::OpenSettings, "Open Settings", ""),
    (PaletteAction::ToggleTheme, "Toggle Theme", ""),
    (PaletteAction::ToggleFollowAgent, "Toggle Follow Agent", ""),
    (PaletteAction::SaveFile, "Save Active File", "Ctrl+S"),
    (PaletteAction::CloseTab, "Close Active Tab", "Ctrl+W"),
    (PaletteAction::RefreshWorkspace, "Refresh Workspace", ""),
    (PaletteAction::NewFile, "New File in Workspace", ""),
    (PaletteAction::NewFolder, "New Folder in Workspace", ""),
    (
        PaletteAction::LspWorkspaceSymbols,
        "Language Server Symbols",
        "",
    ),
    (PaletteAction::LspHover, "Language Server Hover", ""),
    (PaletteAction::LspDefinition, "Go to Definition", ""),
];

#[derive(PartialEq)]
pub struct CommandPaletteModal {
    pub state: State<AppState>,
    pub cursor_state: State<EditorCursorState>,
    pub workspace: WorkspaceCommands,
    pub terminal: Option<TerminalRuntime>,
    pub extension_host: ExtensionHost,
}

impl Component for CommandPaletteModal {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let cursor_state = self.cursor_state;
        let editor_context = PaletteEditorContext {
            state,
            cursor_state,
        };
        let workspace = self.workspace;
        let terminal = self.terminal.clone();
        let extension_host = self.extension_host.clone();
        let current_theme = use_theme();
        let pending_command = if state.peek().pending_extension_command.is_some() {
            state.write().pending_extension_command.take()
        } else {
            None
        };
        if let Some((extension_id, command_id)) = pending_command {
            let target = state.read().extensions.iter().enumerate().find_map(
                |(extension_index, extension)| {
                    extension
                        .manifest
                        .id
                        .eq(&extension_id)
                        .then(|| {
                            extension
                                .manifest
                                .contributions
                                .commands
                                .iter()
                                .position(|command| command.id == command_id)
                                .map(|command_index| (extension_index, command_index))
                        })
                        .flatten()
                },
            );
            if let Some((extension_index, command_index)) = target {
                run_extension_command(
                    editor_context,
                    workspace,
                    terminal.clone(),
                    current_theme,
                    extension_host.clone(),
                    extension_index,
                    command_index,
                );
            } else {
                state
                    .write()
                    .show_notice(NoticeTone::Error, "Keybinding command is unavailable");
            }
        }
        let query_state = use_state(String::new);
        let mut selected_state = use_state(|| 0usize);
        let query = query_state.read().to_lowercase();
        let extension_records = state.read().extensions.clone();
        let mut matches: Vec<PaletteEntry> = PALETTE_COMMANDS
            .iter()
            .filter(|(_, title, _)| title.to_lowercase().contains(&query))
            .map(|(action, title, hint)| PaletteEntry {
                action: *action,
                title: (*title).to_string(),
                hint: (*hint).to_string(),
            })
            .collect();
        for (extension_index, extension) in extension_records.iter().enumerate() {
            if extension.status != ExtensionStatus::Ready {
                continue;
            }
            for (command_index, command) in
                extension.manifest.contributions.commands.iter().enumerate()
            {
                if command.title.to_lowercase().contains(&query) {
                    matches.push(PaletteEntry {
                        action: PaletteAction::Extension(extension_index, command_index),
                        title: command.title.clone(),
                        hint: command
                            .keybinding
                            .clone()
                            .unwrap_or_else(|| extension.manifest.name.clone()),
                    });
                }
            }
            for (theme_index, theme) in extension.manifest.contributions.themes.iter().enumerate() {
                if theme.name.to_lowercase().contains(&query) {
                    matches.push(PaletteEntry {
                        action: PaletteAction::ExtensionTheme(extension_index, theme_index),
                        title: format!("Theme: {}", theme.name),
                        hint: extension.manifest.name.clone(),
                    });
                }
            }
            for (view_index, view) in extension.manifest.contributions.views.iter().enumerate() {
                if view.title.to_lowercase().contains(&query) {
                    matches.push(PaletteEntry {
                        action: PaletteAction::ExtensionPanel(
                            extension_index,
                            view_index,
                            ExtensionPanelKind::View,
                        ),
                        title: format!("View: {}", view.title),
                        hint: extension.manifest.name.clone(),
                    });
                }
            }
            for (panel_index, panel) in extension.manifest.contributions.panels.iter().enumerate() {
                if panel.title.to_lowercase().contains(&query) {
                    matches.push(PaletteEntry {
                        action: PaletteAction::ExtensionPanel(
                            extension_index,
                            panel_index,
                            ExtensionPanelKind::Panel,
                        ),
                        title: format!("Panel: {}", panel.title),
                        hint: extension.manifest.name.clone(),
                    });
                }
            }
            for (task_index, task) in extension.manifest.contributions.tasks.iter().enumerate() {
                if task.title.to_lowercase().contains(&query) {
                    matches.push(PaletteEntry {
                        action: PaletteAction::ExtensionTask(extension_index, task_index),
                        title: format!("Task: {}", task.title),
                        hint: extension.manifest.name.clone(),
                    });
                }
            }
            for (debugger_index, debugger) in extension
                .manifest
                .contributions
                .debuggers
                .iter()
                .enumerate()
            {
                if debugger.title.to_lowercase().contains(&query) {
                    matches.push(PaletteEntry {
                        action: PaletteAction::ExtensionDebugger(extension_index, debugger_index),
                        title: format!("Debug: {}", debugger.title),
                        hint: extension.manifest.name.clone(),
                    });
                }
            }
        }
        let result_count = matches.len();
        let selected_index = (*selected_state.read()).min(result_count.saturating_sub(1));
        let submit_matches = matches.clone();
        let window = Platform::get().root_size.read();

        rect()
            .width(Size::px(window.width))
            .height(Size::px(window.height))
            .position(Position::new_global().top(0.).left(0.))
            .layer(Layer::Overlay)
            .background(Color::TRANSPARENT)
            .horizontal()
            .main_align(Alignment::center())
            .cross_align(Alignment::start())
            .padding(Gaps::new(72., 0., 0., 0.))
            .on_press(move |_| state.write().palette_open = false)
            .child(
                rect()
                    .width(Size::px(520.))
                    .max_height(Size::px(400.))
                    .background(colors.surface_primary)
                    .corner_radius(CornerRadius::new_all(8.))
                    .border(Border::new().fill(colors.border_focus).width(1.))
                    .padding(Gaps::new_all(8.))
                    .vertical()
                    .content(Content::flex())
                    .spacing(6.)
                    .on_mouse_up(|event: Event<MouseEventData>| {
                        event.stop_propagation();
                    })
                    .child(
                        rect()
                            .width(Size::fill())
                            .horizontal()
                            .cross_align(Alignment::center())
                            .spacing(8.)
                            .child(
                                SvgViewer::new(("command", icon("command")))
                                    .color(colors.text_secondary)
                                    .width(Size::px(12.))
                                    .height(Size::px(12.)),
                            )
                            .child(
                                Input::new(query_state.into_writable())
                                    .placeholder("Type a command…")
                                    .width(Size::fill())
                                    .auto_focus(true)
                                    .on_pre_key_down(move |event: Event<KeyboardEventData>| {
                                        match event.key {
                                            Key::Named(NamedKey::ArrowDown) => {
                                                if result_count > 0 {
                                                    let next = (*selected_state.read() + 1)
                                                        .min(result_count - 1);
                                                    selected_state.set(next);
                                                }
                                                event.stop_propagation();
                                                event.prevent_default();
                                                false
                                            }
                                            Key::Named(NamedKey::ArrowUp) => {
                                                if result_count > 0 {
                                                    let previous =
                                                        selected_state.read().saturating_sub(1);
                                                    selected_state.set(previous);
                                                }
                                                event.stop_propagation();
                                                event.prevent_default();
                                                false
                                            }
                                            Key::Named(NamedKey::Escape) => {
                                                state.write().palette_open = false;
                                                event.stop_propagation();
                                                event.prevent_default();
                                                false
                                            }
                                            _ => true,
                                        }
                                    })
                                    .on_submit({
                                        let terminal = terminal.clone();
                                        let extension_host = extension_host.clone();
                                        move |_| {
                                            if let Some(entry) = submit_matches.get(selected_index)
                                            {
                                                run_command(
                                                    editor_context,
                                                    workspace,
                                                    terminal.clone(),
                                                    current_theme,
                                                    extension_host.clone(),
                                                    entry.action,
                                                );
                                            }
                                        }
                                    }),
                            ),
                    )
                    .child(
                        ScrollView::new()
                            .direction(torin::prelude::Direction::Vertical)
                            .height(Size::flex(1.))
                            .children(matches.into_iter().enumerate().map(|(index, entry)| {
                                let row_editor_context = editor_context;
                                let row_workspace = workspace;
                                let row_terminal = terminal.clone();
                                let row_theme = current_theme;
                                let row_extension_host = extension_host.clone();
                                let row_action = entry.action;
                                let row_title = entry.title;
                                let row_hint = entry.hint;
                                let is_selected = index == selected_index;
                                rect()
                                    .width(Size::fill())
                                    .min_height(Size::px(28.))
                                    .horizontal()
                                    .main_align(Alignment::space_between())
                                    .cross_align(Alignment::center())
                                    .padding(Gaps::new(0., 8., 0., 8.))
                                    .corner_radius(CornerRadius::new_all(4.))
                                    .background(if is_selected {
                                        colors.surface_tertiary
                                    } else {
                                        Color::TRANSPARENT
                                    })
                                    .on_all_press(move |_| {
                                        run_command(
                                            row_editor_context,
                                            row_workspace,
                                            row_terminal.clone(),
                                            row_theme,
                                            row_extension_host.clone(),
                                            row_action,
                                        );
                                    })
                                    .child(
                                        label()
                                            .color(colors.text_primary)
                                            .font_size(12.)
                                            .max_lines(1)
                                            .text(row_title),
                                    )
                                    .child(
                                        label()
                                            .color(colors.text_placeholder)
                                            .font_size(10.)
                                            .text(row_hint),
                                    )
                                    .into()
                            })),
                    )
                    .child(
                        label()
                            .color(colors.text_placeholder)
                            .font_size(10.)
                            .text("↑↓ navigate · Enter run · Esc close"),
                    ),
            )
    }
}

fn run_extension_debugger(
    mut state: State<AppState>,
    extension_host: ExtensionHost,
    extension_index: usize,
    debugger_index: usize,
) {
    let target = state
        .read()
        .extensions
        .get(extension_index)
        .filter(|extension| extension.status == ExtensionStatus::Ready)
        .and_then(|extension| {
            let debugger = extension
                .manifest
                .contributions
                .debuggers
                .get(debugger_index)?;
            let allowed = extension
                .manifest
                .permissions
                .iter()
                .any(|permission| permission.capability == "debugger");
            allowed.then(|| (extension.manifest.id.clone(), debugger.id.clone()))
        });
    let Some((extension_id, debugger_id)) = target else {
        state.write().show_notice(
            NoticeTone::Error,
            "Debugger is unavailable or not permitted",
        );
        return;
    };
    if !extension_host.has_granted_capability(&extension_id, "debugger") {
        state
            .write()
            .show_notice(NoticeTone::Error, "Debugger permission was not granted");
        return;
    }
    if !extension_host.is_debug_active(&extension_id, &debugger_id) {
        state
            .write()
            .show_notice(NoticeTone::Error, "Debug adapter is not active");
        return;
    }
    state.write().palette_open = false;
    let host = extension_host.clone();
    spawn(async move {
        let result = crate::run_blocking(move || {
            host.debug_request(&extension_id, &debugger_id, "launch", json!({}))
        })
        .await;
        let mut app_state = state.write();
        match result {
            Ok(Ok(_)) => {
                app_state.show_notice(NoticeTone::Success, "Debug session started".to_string())
            }
            Ok(Err(error)) => app_state.show_notice(NoticeTone::Error, error),
            Err(error) => {
                app_state.show_notice(NoticeTone::Error, format!("Debug worker failed: {error}"))
            }
        }
    });
}

fn run_extension_task(
    mut state: State<AppState>,
    extension_host: ExtensionHost,
    extension_index: usize,
    task_index: usize,
) {
    let target = state
        .read()
        .extensions
        .get(extension_index)
        .filter(|extension| extension.status == ExtensionStatus::Ready)
        .and_then(|extension| {
            let task = extension
                .manifest
                .contributions
                .tasks
                .get(task_index)
                .cloned()?;
            Some((extension.manifest.id.clone(), task))
        });
    let Some((extension_id, task)) = target else {
        state
            .write()
            .show_notice(NoticeTone::Error, "Task is unavailable or not permitted");
        return;
    };
    if !extension_host.has_granted_capability(&extension_id, "workspace.tasks") {
        state
            .write()
            .show_notice(NoticeTone::Error, "Task permission was not granted");
        return;
    }
    let root = state.read().workspace_root.clone();
    state.write().palette_open = false;
    spawn(async move {
        let result = crate::run_blocking(move || {
            crate::services::tasks::run_task(&root, &task, Duration::from_secs(120))
        })
        .await;
        let mut app_state = state.write();
        match result {
            Ok(Ok(task)) if task.timed_out => {
                app_state.show_notice(NoticeTone::Error, "Task timed out".to_string())
            }
            Ok(Ok(task)) if task.exit_code == Some(0) => app_state.show_notice(
                NoticeTone::Success,
                format!("Task completed: {}", task.stdout.trim()),
            ),
            Ok(Ok(task)) => app_state.show_notice(
                NoticeTone::Error,
                format!(
                    "Task exited with {:?}: {}",
                    task.exit_code,
                    task.stderr.trim()
                ),
            ),
            Ok(Err(error)) => app_state.show_notice(NoticeTone::Error, error),
            Err(error) => {
                app_state.show_notice(NoticeTone::Error, format!("Task worker failed: {error}"))
            }
        }
    });
}

fn run_extension_panel(
    mut state: State<AppState>,
    extension_index: usize,
    contribution_index: usize,
    kind: ExtensionPanelKind,
) {
    let panel = state
        .read()
        .extensions
        .get(extension_index)
        .filter(|extension| extension.status == ExtensionStatus::Ready)
        .and_then(|extension| {
            let (title, content) = match kind {
                ExtensionPanelKind::View => extension
                    .manifest
                    .contributions
                    .views
                    .get(contribution_index)
                    .map(|view| (view.title.clone(), view.content.clone())),
                ExtensionPanelKind::Panel => extension
                    .manifest
                    .contributions
                    .panels
                    .get(contribution_index)
                    .map(|panel| (panel.title.clone(), panel.content.clone())),
            }?;
            Some(ExtensionPanel {
                kind,
                title,
                content,
            })
        });
    if let Some(panel) = panel {
        let mut app_state = state.write();
        app_state.palette_open = false;
        app_state.extension_panel = Some(panel);
    } else {
        state
            .write()
            .show_notice(NoticeTone::Error, "Extension panel is unavailable");
    }
}

fn run_extension_theme(
    mut state: State<AppState>,
    mut current_theme: State<freya::prelude::Theme>,
    extension_index: usize,
    theme_index: usize,
) {
    let theme = state
        .read()
        .extensions
        .get(extension_index)
        .filter(|extension| extension.status == ExtensionStatus::Ready)
        .and_then(|extension| {
            extension
                .manifest
                .contributions
                .themes
                .get(theme_index)
                .cloned()
        });
    let Some(theme) = theme else {
        state
            .write()
            .show_notice(NoticeTone::Error, "Extension theme is unavailable");
        return;
    };
    let result = {
        let mut current = current_theme.write();
        crate::theme::apply_extension_tokens(&mut current, &theme.tokens)
    };
    match result {
        Ok(()) => {
            state.write().config.theme = theme.name.clone();
            if let Err(error) = state.write().config.save() {
                state
                    .write()
                    .show_notice(NoticeTone::Error, format!("Theme was not saved: {error}"));
            } else {
                state.write().show_notice(
                    NoticeTone::Success,
                    format!("Theme applied: {}", theme.name),
                );
            }
        }
        Err(error) => state
            .write()
            .show_notice(NoticeTone::Error, format!("Theme rejected: {error}")),
    }
}

fn definition_location(value: &serde_json::Value) -> Option<(String, usize)> {
    let location = value
        .as_array()
        .and_then(|locations| locations.first())
        .unwrap_or(value);
    let uri = location.get("uri")?.as_str()?.to_string();
    let line = location
        .pointer("/range/start/line")
        .and_then(serde_json::Value::as_u64)? as usize
        + 1;
    Some((uri, line))
}

fn run_lsp_command(
    editor_context: PaletteEditorContext,
    workspace: WorkspaceCommands,
    extension_host: ExtensionHost,
    action: PaletteAction,
) {
    let PaletteEditorContext {
        mut state,
        cursor_state,
    } = editor_context;
    let cursor_position = cursor_state.read().position;
    let active_document = state
        .read()
        .active_tab()
        .map(|tab| (tab.path.clone(), tab.content.clone(), cursor_position));
    let target = if matches!(action, PaletteAction::LspWorkspaceSymbols) {
        state.read().extensions.iter().find_map(|extension| {
            if extension.status != ExtensionStatus::Ready {
                return None;
            }
            extension
                .manifest
                .contributions
                .lsp
                .first()
                .map(|server| (extension.manifest.id.clone(), server.language.clone()))
        })
    } else {
        active_document
            .as_ref()
            .and_then(|(path, _, _)| extension_host.active_lsp_for_path(path))
    };
    let Some((extension_id, language)) = target else {
        state
            .write()
            .show_notice(NoticeTone::Error, "No active language server for this file");
        return;
    };
    if !extension_host.is_lsp_active(&extension_id, &language) {
        state
            .write()
            .show_notice(NoticeTone::Error, "Language server is not active");
        return;
    }
    let (file, line, character) = active_document
        .as_ref()
        .map(|(path, content, cursor_position)| {
            let (line, character) = utf16_position_to_line_character(content, *cursor_position);
            (path.to_string_lossy().to_string(), line, character)
        })
        .unwrap_or_default();
    if !matches!(action, PaletteAction::LspWorkspaceSymbols) && file.is_empty() {
        state
            .write()
            .show_notice(NoticeTone::Error, "Open a file first");
        return;
    }
    state.write().palette_open = false;
    let host = extension_host.clone();
    spawn(async move {
        let result = crate::run_blocking(move || match action {
            PaletteAction::LspWorkspaceSymbols => {
                host.lsp_workspace_symbols(&extension_id, &language, "")
            }
            PaletteAction::LspHover => {
                host.lsp_hover(&extension_id, &language, &file, line, character)
            }
            PaletteAction::LspDefinition => {
                host.lsp_definition(&extension_id, &language, &file, line, character)
            }
            _ => Err("unsupported language server action".to_string()),
        })
        .await;
        let mut app_state = state.write();
        match result {
            Ok(Ok(value)) => {
                let message = match action {
                    PaletteAction::LspWorkspaceSymbols => {
                        let count = value.as_array().map_or(0, Vec::len);
                        format!("Language server returned {count} symbols")
                    }
                    PaletteAction::LspHover => value
                        .pointer("/contents/value")
                        .and_then(serde_json::Value::as_str)
                        .map_or_else(
                            || "Language server returned no hover".to_string(),
                            ToString::to_string,
                        ),
                    PaletteAction::LspDefinition => match definition_location(&value) {
                        Some((uri, line)) => {
                            let path = file_uri_to_path(&uri)
                                .map_err(|error| format!("Invalid definition location: {error}"));
                            match path {
                                Ok(path) => {
                                    workspace.load_file_with_reveal(
                                        path.clone(),
                                        EditorReveal {
                                            path,
                                            line,
                                            start: 0,
                                            end: 0,
                                        },
                                    );
                                    format!("Opened definition at line {line}")
                                }
                                Err(error) => error,
                            }
                        }
                        None => "Language server returned no definition".to_string(),
                    },
                    _ => "Language server action completed".to_string(),
                };
                app_state.show_notice(NoticeTone::Success, message);
            }
            Ok(Err(error)) => app_state.show_notice(NoticeTone::Error, error),
            Err(error) => app_state.show_notice(
                NoticeTone::Error,
                format!("Language server worker failed: {error}"),
            ),
        }
    });
}

fn run_extension_command(
    editor_context: PaletteEditorContext,
    workspace: WorkspaceCommands,
    terminal: Option<TerminalRuntime>,
    current_theme: State<freya::prelude::Theme>,
    extension_host: ExtensionHost,
    extension_index: usize,
    command_index: usize,
) {
    let PaletteEditorContext { mut state, .. } = editor_context;
    let extension_command = {
        let app_state = state.read();
        app_state
            .extensions
            .get(extension_index)
            .filter(|extension| extension.status == ExtensionStatus::Ready)
            .and_then(|extension| {
                extension
                    .manifest
                    .contributions
                    .commands
                    .get(command_index)
                    .map(|command| {
                        (
                            extension.manifest.id.clone(),
                            command.id.clone(),
                            command.action.clone(),
                        )
                    })
            })
    };
    let Some((extension_id, command_id, extension_action)) = extension_command else {
        state
            .write()
            .show_notice(NoticeTone::Error, "Extension command is unavailable");
        return;
    };
    if extension_action == "process" {
        let host = extension_host.clone();
        let mut result_state = state;
        spawn(async move {
            let result = crate::run_blocking(move || {
                host.invoke(
                    &extension_id,
                    "command/invoke",
                    json!({ "command_id": command_id }),
                )
            })
            .await;
            let mut app_state = result_state.write();
            app_state.palette_open = false;
            match result {
                Ok(Ok(value)) => app_state.show_notice(
                    NoticeTone::Success,
                    value
                        .get("message")
                        .and_then(|message| message.as_str())
                        .unwrap_or("Extension command completed")
                        .to_string(),
                ),
                Ok(Err(error)) => app_state.show_notice(
                    NoticeTone::Error,
                    format!("Extension command failed: {error}"),
                ),
                Err(error) => app_state.show_notice(
                    NoticeTone::Error,
                    format!("Extension worker failed: {error}"),
                ),
            }
        });
        return;
    }
    let action = match extension_action.as_str() {
        "find_in_file" => Some(PaletteAction::FindInFile),
        "find_in_project" => Some(PaletteAction::FindInProject),
        "toggle_sidebar" => Some(PaletteAction::ToggleSidebar),
        "toggle_agent" => Some(PaletteAction::ToggleAgent),
        "toggle_dock" => Some(PaletteAction::ToggleDock),
        "open_terminal" => Some(PaletteAction::OpenTerminal),
        "open_settings" => Some(PaletteAction::OpenSettings),
        "refresh_workspace" => Some(PaletteAction::RefreshWorkspace),
        "new_file" => Some(PaletteAction::NewFile),
        "new_folder" => Some(PaletteAction::NewFolder),
        "toggle_theme" => Some(PaletteAction::ToggleTheme),
        _ => None,
    };
    if let Some(action) = action {
        run_command(
            editor_context,
            workspace,
            terminal,
            current_theme,
            extension_host,
            action,
        );
    } else {
        state.write().show_notice(
            NoticeTone::Error,
            "Extension action is not supported by this viewer",
        );
    }
}

fn run_command(
    editor_context: PaletteEditorContext,
    workspace: WorkspaceCommands,
    terminal: Option<TerminalRuntime>,
    mut current_theme: State<freya::prelude::Theme>,
    extension_host: ExtensionHost,
    action: PaletteAction,
) {
    let PaletteEditorContext { mut state, .. } = editor_context;
    if let PaletteAction::ExtensionDebugger(extension_index, debugger_index) = action {
        run_extension_debugger(state, extension_host, extension_index, debugger_index);
        return;
    }
    if let PaletteAction::ExtensionTask(extension_index, task_index) = action {
        run_extension_task(state, extension_host, extension_index, task_index);
        return;
    }
    if let PaletteAction::ExtensionPanel(extension_index, contribution_index, kind) = action {
        run_extension_panel(state, extension_index, contribution_index, kind);
        return;
    }
    if let PaletteAction::ExtensionTheme(extension_index, theme_index) = action {
        run_extension_theme(state, current_theme, extension_index, theme_index);
        return;
    }
    if matches!(
        action,
        PaletteAction::LspWorkspaceSymbols | PaletteAction::LspHover | PaletteAction::LspDefinition
    ) {
        run_lsp_command(editor_context, workspace, extension_host, action);
        return;
    }
    if let PaletteAction::Extension(extension_index, command_index) = action {
        run_extension_command(
            editor_context,
            workspace,
            terminal,
            current_theme,
            extension_host,
            extension_index,
            command_index,
        );
        return;
    }
    let mut app_state = state.write();
    app_state.palette_open = false;
    match action {
        PaletteAction::GoToFile => {
            app_state.quick_open_open = true;
        }
        PaletteAction::FindInFile => {
            app_state.find_open = true;
        }
        PaletteAction::FindInProject => {
            app_state.dock_view = DockView::Search;
            app_state.dock_open = true;
        }
        PaletteAction::ShowProblems => {
            app_state.dock_view = DockView::Problems;
            app_state.dock_open = true;
        }
        PaletteAction::ToggleSidebar => {
            app_state.sidebar_visible = !app_state.sidebar_visible;
        }
        PaletteAction::ToggleAgent => {
            app_state.agent_panel_visible = !app_state.agent_panel_visible;
        }
        PaletteAction::ToggleDock => {
            app_state.dock_open = !app_state.dock_open;
        }
        PaletteAction::OpenTerminal => {
            app_state.dock_view = DockView::Terminal;
            app_state.dock_open = true;
        }
        PaletteAction::RestartTerminal => {
            if let Some(terminal) = terminal {
                let _ = terminal.restart();
                app_state.terminal_output.clear();
                app_state.terminal_running = true;
            } else {
                app_state.show_notice(NoticeTone::Error, "No terminal is running");
            }
        }
        PaletteAction::OpenSettings => {
            app_state.config_open = true;
        }
        PaletteAction::ToggleTheme => {
            let next_name = if current_theme.read().name == "light" {
                "dark"
            } else {
                "light"
            };
            let next = if next_name == "light" {
                theme::light()
            } else {
                theme::dark()
            };
            current_theme.set(next);
            app_state.config.theme = next_name.to_string();
            if let Err(error) = app_state.config.save() {
                app_state.show_notice(
                    NoticeTone::Error,
                    format!("Theme preference was not saved: {error}"),
                );
            }
        }
        PaletteAction::ToggleFollowAgent => {
            let enabled = app_state.toggle_follow_agent();
            app_state.show_notice(
                NoticeTone::Info,
                if enabled {
                    "Following agent edits"
                } else {
                    "Follow agent off"
                },
            );
        }
        PaletteAction::SaveFile => {
            let _ = save_active_tab(&mut app_state);
        }
        PaletteAction::CloseTab => {
            if let Some(index) = app_state.active_tab_index {
                request_close_tab(&mut app_state, index);
            }
        }
        PaletteAction::RefreshWorkspace => {
            workspace.refresh();
        }
        PaletteAction::NewFile => {
            let root = app_state.workspace_root.clone();
            app_state.file_prompt = Some(FilePrompt {
                kind: FilePromptKind::NewFile(root),
                input: String::new(),
            });
        }
        PaletteAction::NewFolder => {
            let root = app_state.workspace_root.clone();
            app_state.file_prompt = Some(FilePrompt {
                kind: FilePromptKind::NewFolder(root),
                input: String::new(),
            });
        }
        PaletteAction::Extension(_, _)
        | PaletteAction::ExtensionTheme(_, _)
        | PaletteAction::ExtensionPanel(_, _, _)
        | PaletteAction::ExtensionTask(_, _)
        | PaletteAction::ExtensionDebugger(_, _)
        | PaletteAction::LspWorkspaceSymbols
        | PaletteAction::LspHover
        | PaletteAction::LspDefinition => {}
    }
}

#[cfg(test)]
mod tests {
    use super::{PALETTE_COMMANDS, definition_location};
    use serde_json::json;

    #[test]
    fn offers_a_command_to_detach_from_the_agent() {
        let titles: Vec<&str> = PALETTE_COMMANDS
            .iter()
            .map(|(_, title, _)| *title)
            .collect();

        assert!(titles.contains(&"Toggle Follow Agent"));
    }

    #[test]
    fn parses_single_and_array_definition_locations() {
        let single = json!({
            "uri": "file:///tmp/main.rs",
            "range": { "start": { "line": 4, "character": 0 } }
        });
        assert_eq!(
            definition_location(&single),
            Some(("file:///tmp/main.rs".to_string(), 5))
        );
        assert_eq!(
            definition_location(&json!([single])),
            Some(("file:///tmp/main.rs".to_string(), 5))
        );
    }
}
