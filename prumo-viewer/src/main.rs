mod client;
mod projections;
mod services;
mod state;
mod theme;
mod ui;

use std::collections::HashSet;
use std::env;
use std::path::PathBuf;
use std::time::Duration;

use freya::prelude::*;
use tokio::runtime::Builder;
use torin::prelude::Direction;

use crate::client::protocol::PrumoClient;
use crate::projections::run::{apply_client_error, apply_daemon_snapshot};
use crate::services::config::ViewerConfig;
use crate::services::document::{activate_path, open_file, request_close_tab, save_active_tab};
use crate::services::terminal::{TerminalEvent, TerminalRuntime};
use crate::services::workspace::refresh_workspace;
use crate::state::{AppState, NoticeTone};
use crate::ui::agent_panel::{AgentPanel, BottomDock};
use crate::ui::diff_view::DiffModal;
use crate::ui::editor_area::EditorArea;
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
                    state.clear_notice();
                }
                Ok(Err(error)) => state
                    .write()
                    .show_notice(NoticeTone::Error, format!("Could not open file: {error}")),
                Err(error) => state
                    .write()
                    .show_notice(NoticeTone::Error, format!("File reader failed: {error}")),
            }
        });
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
    launch(
        LaunchConfig::new().with_window(
            WindowConfig::new(move || {
                app(
                    initial_state.clone(),
                    app_client.clone(),
                    app_terminal.clone(),
                )
            })
            .with_title(leaked_title)
            .with_size(1440., 900.)
            .with_min_size(720., 540.),
        ),
    );
}

fn app(
    initial_state: AppState,
    client: PrumoClient,
    terminal: Option<TerminalRuntime>,
) -> impl IntoElement {
    let configured_theme = initial_state.config.theme.clone();
    use_init_theme(move || match configured_theme.as_str() {
        "light" => theme::light(),
        "dark" => theme::dark(),
        _ => match &*Platform::get().preferred_theme.read() {
            PreferredTheme::Light => theme::light(),
            PreferredTheme::Dark => theme::dark(),
        },
    });

    let state = use_state(move || initial_state);
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
        let mut polling_state = state;
        let polling_client = client.clone();
        let configured_poll_interval = state.read().config.poll_interval_seconds;
        spawn_forever(async move {
            let mut poll_delay = Duration::from_secs(configured_poll_interval);
            loop {
                async_io::Timer::after(poll_delay).await;
                let (known_event_ids, known_event_count): (HashSet<String>, usize) = {
                    let app_state = polling_state.read();
                    let events: Vec<String> = app_state
                        .agent_events
                        .iter()
                        .map(|event| event.id.clone())
                        .collect();
                    (events.iter().cloned().collect(), events.len())
                };
                let mut worker_state = polling_state.read().clone();
                let worker_client = polling_client.clone();
                let result = tokio::task::spawn_blocking(move || {
                    let snapshot_result = worker_client.snapshot();
                    let should_refresh = snapshot_result.as_ref().is_ok_and(|snapshot| {
                        snapshot.events.iter().any(|event| {
                            event.kind == "file.changed" && !known_event_ids.contains(&event.id)
                        })
                    });
                    if should_refresh {
                        let _ = refresh_workspace(&mut worker_state);
                    }
                    (snapshot_result, worker_state, should_refresh)
                })
                .await;

                match result {
                    Ok((snapshot_result, worker_state, should_refresh)) => {
                        let snapshot_event_count = snapshot_result
                            .as_ref()
                            .map(|snapshot| snapshot.events.len())
                            .unwrap_or_default();
                        let mut app_state = polling_state.write();
                        if should_refresh {
                            app_state.tree = worker_state.tree;
                            app_state.flattened_tree = worker_state.flattened_tree;
                            app_state.file_candidates = worker_state.file_candidates;
                        }
                        match snapshot_result {
                            Ok(snapshot) => {
                                apply_daemon_snapshot(&mut app_state, &snapshot);
                                poll_delay = if snapshot_event_count > known_event_count {
                                    Duration::from_secs(configured_poll_interval)
                                } else {
                                    poll_delay.saturating_mul(2).min(Duration::from_secs(10))
                                };
                            }
                            Err(error) => {
                                apply_client_error(&mut app_state, error.to_string());
                                poll_delay = Duration::from_secs(5);
                            }
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
    let is_diff_open = state.read().diff_open;
    let has_notice = state.read().notice.is_some();
    let has_close_dialog = state.read().pending_close_tab.is_some();
    let has_config = state.read().config_open;
    let mut state_for_keys = state;
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
            let control = event.modifiers.contains(Modifiers::CONTROL);
            match &event.key {
                Key::Character(character) if control && character.eq_ignore_ascii_case("p") => {
                    let mut app_state = state_for_keys.write();
                    app_state.quick_open_open = !app_state.quick_open_open;
                    app_state.diff_open = false;
                    app_state.clear_notice();
                }
                Key::Character(character) if control && character.eq_ignore_ascii_case("s") => {
                    let mut app_state = state_for_keys.write();
                    let _ = save_active_tab(&mut app_state);
                }
                Key::Character(character) if control && character.eq_ignore_ascii_case("w") => {
                    let mut app_state = state_for_keys.write();
                    if let Some(index) = app_state.active_tab_index {
                        request_close_tab(&mut app_state, index);
                    }
                }
                Key::Character(character) if control && character.eq_ignore_ascii_case("b") => {
                    let sidebar_visible = state_for_keys.read().sidebar_visible;
                    state_for_keys.write().sidebar_visible = !sidebar_visible;
                }
                Key::Character(character) if control && character.eq_ignore_ascii_case("j") => {
                    let dock_open = state_for_keys.read().dock_open;
                    state_for_keys.write().dock_open = !dock_open;
                }
                Key::Character(character)
                    if control
                        && event.modifiers.contains(Modifiers::SHIFT)
                        && character.eq_ignore_ascii_case("j") =>
                {
                    let agent_visible = state_for_keys.read().agent_panel_visible;
                    state_for_keys.write().agent_panel_visible = !agent_visible;
                }
                Key::Named(NamedKey::Escape) => {
                    let mut app_state = state_for_keys.write();
                    app_state.quick_open_open = false;
                    app_state.diff_open = false;
                    app_state.pending_close_tab = None;
                    app_state.config_open = false;
                }
                _ => {}
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
                    client: client.clone(),
                    workspace,
                    terminal: terminal.clone(),
                }),
        )
        .child(StatusBar { state })
        .child(if is_quick_open {
            Element::from(QuickOpenModal { state, workspace })
        } else {
            Element::from(rect())
        })
        .child(if is_diff_open {
            Element::from(DiffModal { state })
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
}

#[derive(PartialEq)]
struct Body {
    state: State<AppState>,
    client: PrumoClient,
    workspace: WorkspaceCommands,
    terminal: Option<TerminalRuntime>,
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

        let editor_content = ResizablePanel::new(PanelSize::percent(1.))
            .min_size(0.25)
            .key("editor-panel")
            .order(0usize)
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::fill())
                    .vertical()
                    .content(Content::flex())
                    .child(TabBar { state })
                    .child(
                        rect()
                            .width(Size::fill())
                            .height(Size::flex(1.))
                            .content(Content::flex())
                            .child(EditorArea { state }),
                    ),
            );
        let agent_container = ResizableContainer::new()
            .direction(Direction::Horizontal)
            .panel(editor_content);
        let agent_container = if agent_visible {
            agent_container.panel(
                ResizablePanel::new(PanelSize::px(340.))
                    .min_size(300.)
                    .key("agent-panel")
                    .order(1usize)
                    .child(AgentPanel {
                        state,
                        client: client.clone(),
                    }),
            )
        } else {
            agent_container
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
                    .child(
                        rect()
                            .width(Size::fill())
                            .height(Size::flex(1.))
                            .content(Content::flex())
                            .child(agent_container),
                    )
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
                    ResizablePanel::new(PanelSize::px(286.))
                        .min_size(240.)
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
            .min_height(Size::px(30.))
            .horizontal()
            .cross_align(Alignment::center())
            .padding(Gaps::new(0., 10., 0., 8.))
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
                    .font_size(11.)
                    .text(notice.message),
            )
            .child(
                Button::new()
                    .on_press(move |_| state.write().clear_notice())
                    .child(
                        label()
                            .color(colors.text_secondary)
                            .font_size(11.)
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

        rect()
            .width(Size::fill())
            .height(Size::fill())
            .position(Position::new_global().top(0.).left(0.))
            .layer(Layer::Overlay)
            .background(Color::from_argb(170, 0, 0, 0))
            .horizontal()
            .main_align(Alignment::center())
            .cross_align(Alignment::center())
            .on_mouse_up(move |_| {
                state.write().pending_close_tab = None;
            })
            .child(
                rect()
                    .width(Size::px(440.))
                    .vertical()
                    .padding(Gaps::new_all(16.))
                    .spacing(12.)
                    .background(colors.surface_primary)
                    .corner_radius(CornerRadius::new_all(8.))
                    .border(Border::new().fill(colors.border_focus).width(1.))
                    .on_mouse_up(|event: Event<MouseEventData>| {
                        event.stop_propagation();
                    })
                    .child(
                        label()
                            .color(colors.text_primary)
                            .font_size(15.)
                            .font_weight(FontWeight::BOLD)
                            .text("Unsaved changes"),
                    )
                    .child(
                        label()
                            .color(colors.text_secondary)
                            .font_size(12.)
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
                                    .on_press(move |_| {
                                        state.write().pending_close_tab = None;
                                    })
                                    .child(
                                        label()
                                            .color(colors.text_secondary)
                                            .font_size(11.5)
                                            .text("Cancel"),
                                    ),
                            )
                            .child(
                                Button::new()
                                    .on_press(move |_| {
                                        let mut app_state = state.write();
                                        crate::services::document::discard_pending_tab(
                                            &mut app_state,
                                        );
                                    })
                                    .child(
                                        label().color(colors.error).font_size(11.5).text("Discard"),
                                    ),
                            )
                            .child(
                                Button::new()
                                    .on_press(move |_| {
                                        let mut app_state = state.write();
                                        crate::services::document::save_and_close_pending_tab(
                                            &mut app_state,
                                        );
                                    })
                                    .child(
                                        label()
                                            .color(colors.primary)
                                            .font_size(11.5)
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
        let (mut runner, state) = TestingRunner::new(
            move || {
                use_init_theme(theme::dark);
                let state = use_consume::<State<AppState>>();
                Body {
                    state,
                    client: client.clone(),
                    workspace: WorkspaceCommands { state },
                    terminal: None,
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
            .write("printf '__prumo_terminal_visible__\\n'\r")
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
        let (mut runner, _) = TestingRunner::new(
            move || {
                use_init_theme(theme::dark);
                let state = use_consume::<State<AppState>>();
                Body {
                    state,
                    client: client.clone(),
                    workspace: WorkspaceCommands { state },
                    terminal: None,
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
}
