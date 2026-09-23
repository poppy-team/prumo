use crate::WorkspaceCommands;
use crate::client::protocol::PrumoClient;
use crate::services::terminal::TerminalRuntime;
use crate::state::{
    AgentFileStatus, AgentStatus, AppState, ConnectionStatus, DockView, NoticeTone,
};
use crate::ui::icons::icon;
use freya::prelude::*;
use torin::prelude::Direction;

#[derive(PartialEq)]
pub struct AgentPanel {
    pub state: State<AppState>,
    pub client: PrumoClient,
}

impl Component for AgentPanel {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let client = self.client.clone();
        let goal_state = use_state(String::new);
        let steer_state = use_state(String::new);
        let goal = goal_state.read().trim().to_string();
        let steer = steer_state.read().trim().to_string();
        let connection = state.read().connection_status;
        let active_run = state.read().active_run_id.clone();
        let can_steer = active_run.is_some()
            && matches!(
                state.read().agent_status,
                AgentStatus::Working | AgentStatus::AwaitingApproval
            );
        let active_run_phase = state.read().active_run_phase.clone();
        let changed_file_count = state.read().changed_files.len();
        let runner_config = state.read().config.clone();
        let runner_model = if runner_config.model.is_empty() {
            "default model".to_string()
        } else {
            runner_config.model.clone()
        };
        let pending_permissions = state.read().pending_permissions.clone();
        let agent_status = state.read().agent_status;
        let events = state.read().agent_events.clone();
        let agent_log = state.read().agent_log.clone();
        let status_color = match agent_status {
            AgentStatus::Disconnected => colors.error,
            AgentStatus::Idle => colors.text_secondary,
            AgentStatus::Working => colors.warning,
            AgentStatus::AwaitingApproval => colors.warning,
            AgentStatus::Completed => colors.success,
            AgentStatus::Failed => colors.error,
        };
        rect()
            .vertical()
            .width(Size::fill())
            .height(Size::fill())
            .content(Content::flex())
            .background(colors.surface_primary)
            .border(Border::new().fill(colors.border).width(BorderWidth {
                top: 0.,
                right: 0.,
                bottom: 0.,
                left: 1.,
            }))
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::px(40.))
                    .horizontal()
                    .cross_align(Alignment::center())
                    .padding(Gaps::new(0., 10., 0., 10.))
                    .background(colors.surface_primary)
                    .border(Border::new().fill(colors.border).width(BorderWidth {
                        top: 0.,
                        right: 0.,
                        bottom: 1.,
                        left: 0.,
                    }))
                    .child(
                        label()
                            .color(colors.text_secondary)
                            .font_size(11.)
                            .font_weight(FontWeight::BOLD)
                            .text("PRUMO AGENT / RUNNER"),
                    )
                    .child(
                        rect()
                            .expanded()
                            .horizontal()
                            .main_align(Alignment::end())
                            .spacing(6.)
                            .cross_align(Alignment::center())
                            .child(
                                rect()
                                    .width(Size::px(7.))
                                    .height(Size::px(7.))
                                    .corner_radius(CornerRadius::new_all(4.))
                                    .background(status_color),
                            )
                            .child(
                                label()
                                    .color(status_color)
                                    .font_size(10.5)
                                    .text(agent_status.label()),
                            ),
                    ),
            )
            .child(
                ScrollView::new()
                    .direction(Direction::Vertical)
                    .height(Size::flex(1.))
                    .child(
                        rect()
                            .width(Size::fill())
                            .padding(Gaps::new_all(10.))
                            .vertical()
                            .spacing(10.)
                            .child(
                                rect()
                                    .width(Size::fill())
                                    .vertical()
                                    .spacing(6.)
                                    .child(
                                        label()
                                            .color(colors.text_secondary)
                                            .font_size(10.)
                                            .font_weight(FontWeight::BOLD)
                                            .text("RUN INSPECTOR"),
                                    )
                                    .child(
                                        rect()
                                            .width(Size::fill())
                                            .horizontal()
                                            .main_align(Alignment::space_between())
                                            .child(
                                                label()
                                                    .color(colors.text_primary)
                                                    .font_size(11.5)
                                                    .font_weight(FontWeight::MEDIUM)
                                                    .text(
                                                        active_run.clone().unwrap_or_else(|| {
                                                            "No active run".into()
                                                        }),
                                                    ),
                                            )
                                            .child(
                                                label().color(status_color).font_size(10.5).text(
                                                    if active_run_phase.is_empty() {
                                                        "idle".to_string()
                                                    } else {
                                                        active_run_phase.clone()
                                                    },
                                                ),
                                            ),
                                    )
                                    .child(
                                        label().color(colors.text_placeholder).font_size(10.).text(
                                            format!(
                                        "{} · {} · {} · {} files changed",
                                        connection.label(),
                                        runner_config.provider,
                                        runner_model,
                                        changed_file_count
                                            ),
                                        ),
                                    ),
                            )
                            .child(if pending_permissions.is_empty() {
                                Element::from(label().text(""))
                            } else {
                                Element::from(
                                    rect().width(Size::fill()).vertical().spacing(6.).children(
                                        pending_permissions.iter().map(|request_id| {
                                            ApprovalCard {
                                                state,
                                                client: client.clone(),
                                                request_id: request_id.clone(),
                                            }
                                            .into()
                                        }),
                                    ),
                                )
                            })
                            .child(
                                rect()
                                    .width(Size::fill())
                                    .height(Size::px(1.))
                                    .background(colors.border),
                            )
                            .child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(10.)
                                    .font_weight(FontWeight::BOLD)
                                    .text("ACTIVITY"),
                            )
                            .child(if events.is_empty() {
                                Element::from(
                                    rect()
                                        .width(Size::fill())
                                        .vertical()
                                        .spacing(5.)
                                        .child(
                                            label()
                                                .color(colors.text_secondary)
                                                .font_size(11.5)
                                                .text(
                                                    if connection == ConnectionStatus::Connected {
                                                        "No recent events"
                                                    } else {
                                                        "Start Prumo Core from Terminal or run prumo agent serve"
                                                    },
                                                ),
                                        )
                                    .child(
                                        label()
                                            .color(colors.text_placeholder)
                                            .font_size(10.5)
                                            .text(agent_log),
                                    )
                                    .maybe(
                                        connection != ConnectionStatus::Connected,
                                        |rect| {
                                            rect.child(
                                                Button::new()
                                                    .flat()
                                                    .compact()
                                                    .height(Size::px(26.))
                                                    .padding(Gaps::new(9., 0., 9., 0.))
                                                    .on_press(move |_| {
                                                        let mut state = state.write();
                                                        state.dock_view = DockView::Terminal;
                                                        state.dock_open = true;
                                                    })
                                                    .child(
                                                        label()
                                                            .color(colors.primary)
                                                            .font_size(10.5)
                                                            .text("Open Terminal"),
                                                    ),
                                            )
                                        },
                                    ),
                                )
                            } else {
                                Element::from(
                                    rect().width(Size::fill()).vertical().spacing(5.).children(
                                        events.iter().map(|event| {
                                            rect()
                                                .width(Size::fill())
                                                .horizontal()
                                                .spacing(7.)
                                                .cross_align(Alignment::start())
                                                .child(
                                                    label()
                                                        .color(colors.text_placeholder)
                                                        .font_size(9.5)
                                                        .width(Size::px(42.))
                                                        .text(event.time.clone()),
                                                )
                                                .child(
                                                    label()
                                                        .color(colors.text_primary)
                                                        .font_size(10.5)
                                                        .text(event.message.clone()),
                                                )
                                                .into()
                                        }),
                                    ),
                                )
                            }),
                    ),
            )
            .child(
                rect()
                    .width(Size::fill())
                    .vertical()
                    .padding(Gaps::new_all(8.))
                    .spacing(6.)
                    .background(colors.surface_primary)
                    .border(Border::new().fill(colors.border).width(BorderWidth {
                        top: 1.,
                        right: 0.,
                        bottom: 0.,
                        left: 0.,
                    }))
                    .child(if can_steer {
                        Element::from(
                            rect()
                                .width(Size::fill())
                                .vertical()
                                .spacing(6.)
                                .child(
                                    Input::new(steer_state.into_writable())
                                        .placeholder("Steer the active run…")
                                        .width(Size::fill())
                                        .filled()
                                        .on_submit({
                                            let client = client.clone();
                                            let active_run = active_run.clone().unwrap_or_default();
                                            move |value| {
                                                steer_run(state, client.clone(), &active_run, value)
                                            }
                                        }),
                                )
                                .child(
                                    rect()
                                        .width(Size::fill())
                                        .horizontal()
                                        .main_align(Alignment::end())
                                        .spacing(6.)
                                        .child(
                                            Button::new()
                                                .flat()
                                                .compact()
                                                .height(Size::px(28.))
                                                .padding(Gaps::new(9., 0., 9., 0.))
                                                .enabled(connection == ConnectionStatus::Connected)
                                                .on_press({
                                                    let client = client.clone();
                                                    let active_run =
                                                        active_run.clone().unwrap_or_default();
                                                    move |_| {
                                                        cancel_run(
                                                            state,
                                                            client.clone(),
                                                            &active_run,
                                                        )
                                                    }
                                                })
                                                .child(
                                                    label()
                                                        .color(colors.error)
                                                        .font_size(11.)
                                                        .text("Cancel Run"),
                                                ),
                                        )
                                        .child(
                                            Button::new()
                                                .filled()
                                                .compact()
                                                .height(Size::px(28.))
                                                .padding(Gaps::new(10., 0., 10., 0.))
                                                .enabled(
                                                    connection == ConnectionStatus::Connected
                                                        && !steer.is_empty(),
                                                )
                                                .on_press({
                                                    let client = client.clone();
                                                    let active_run =
                                                        active_run.clone().unwrap_or_default();
                                                    let steer = steer.clone();
                                                    move |_| {
                                                        steer_run(
                                                            state,
                                                            client.clone(),
                                                            &active_run,
                                                            steer.clone(),
                                                        )
                                                    }
                                                })
                                                .child(
                                                    label()
                                                        .color(colors.text_primary)
                                                        .font_size(11.)
                                                        .text("Send"),
                                                ),
                                        ),
                                ),
                        )
                    } else {
                        Element::from(
                            rect()
                                .width(Size::fill())
                                .vertical()
                                .spacing(6.)
                                .child(
                                    Input::new(goal_state.into_writable())
                                        .placeholder("Goal for the agent…")
                                        .width(Size::fill())
                                        .filled()
                                        .on_submit({
                                            let client = client.clone();
                                            move |value| start_run(state, client.clone(), value)
                                        }),
                                )
                                .child(
                                    rect()
                                        .width(Size::fill())
                                        .horizontal()
                                        .main_align(Alignment::end())
                                        .child(
                                            Button::new()
                                                .filled()
                                                .enabled(
                                                    connection == ConnectionStatus::Connected
                                                        && !goal.is_empty(),
                                                )
                                                .on_press({
                                                    let client = client.clone();
                                                    let goal = goal.clone();
                                                    let mut goal_state = goal_state;
                                                    move |_| {
                                                        start_run(
                                                            state,
                                                            client.clone(),
                                                            goal.clone(),
                                                        );
                                                        goal_state.set(String::new());
                                                    }
                                                })
                                                .child(
                                                    rect()
                                                        .horizontal()
                                                        .spacing(6.)
                                                        .cross_align(Alignment::center())
                                                        .child(
                                                            SvgViewer::new(("play", icon("play")))
                                                                .color(colors.primary)
                                                                .width(Size::px(13.))
                                                                .height(Size::px(13.)),
                                                        )
                                                        .child(
                                                            label()
                                                                .color(colors.text_primary)
                                                                .font_size(11.)
                                                                .text("Start Run"),
                                                        ),
                                                ),
                                        ),
                                ),
                        )
                    }),
            )
    }
}

#[derive(PartialEq)]
pub struct BottomDock {
    pub state: State<AppState>,
    pub workspace: WorkspaceCommands,
    pub terminal: Option<TerminalRuntime>,
}

impl Component for BottomDock {
    fn render(&self) -> impl IntoElement {
        let mut state = self.state;
        let colors = get_theme_or_default().read().colors.clone();
        let expanded = state.read().dock_open;
        let selected_view = state.read().dock_view;
        let body: Element = match selected_view {
            DockView::Activity => ActivityDockView { state }.into_element(),
            DockView::Changes => ChangesDockView {
                state,
                workspace: self.workspace,
            }
            .into_element(),
            DockView::Terminal => TerminalDockView {
                state,
                terminal: self.terminal.clone(),
            }
            .into_element(),
        };

        rect()
            .width(Size::fill())
            .height(Size::px(if expanded { 190. } else { 30. }))
            .background(colors.surface_primary)
            .content(Content::flex())
            .border(Border::new().fill(colors.border).width(BorderWidth {
                top: 1.,
                right: 0.,
                bottom: 0.,
                left: 0.,
            }))
            .vertical()
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::px(30.))
                    .horizontal()
                    .cross_align(Alignment::center())
                    .padding(Gaps::new(8., 0., 8., 0.))
                    .spacing(4.)
                    .background(colors.surface_inverse)
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .height(Size::px(24.))
                            .padding(Gaps::new(8., 0., 8., 0.))
                            .on_press(move |_| {
                                let mut state = state.write();
                                state.dock_view = DockView::Activity;
                                state.dock_open = true;
                            })
                            .child(
                                label()
                                    .color(if selected_view == DockView::Activity {
                                        colors.text_primary
                                    } else {
                                        colors.text_secondary
                                    })
                                    .font_size(10.)
                                    .font_weight(FontWeight::BOLD)
                                    .text("ACTIVITY"),
                            ),
                    )
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .height(Size::px(24.))
                            .padding(Gaps::new(8., 0., 8., 0.))
                            .on_press(move |_| {
                                let mut state = state.write();
                                state.dock_view = DockView::Changes;
                                state.dock_open = true;
                            })
                            .child(
                                label()
                                    .color(if selected_view == DockView::Changes {
                                        colors.text_primary
                                    } else {
                                        colors.text_secondary
                                    })
                                    .font_size(10.)
                                    .font_weight(FontWeight::BOLD)
                                    .text("CHANGES"),
                            ),
                    )
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .height(Size::px(24.))
                            .padding(Gaps::new(8., 0., 8., 0.))
                            .on_press(move |_| {
                                let mut state = state.write();
                                state.dock_view = DockView::Terminal;
                                state.dock_open = true;
                            })
                            .child(
                                label()
                                    .color(if selected_view == DockView::Terminal {
                                        colors.text_primary
                                    } else {
                                        colors.text_secondary
                                    })
                                    .font_size(10.)
                                    .font_weight(FontWeight::BOLD)
                                    .text("TERMINAL"),
                            ),
                    )
                    .child(
                        rect()
                            .width(Size::fill())
                            .height(Size::px(1.))
                            .background(colors.border),
                    )
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .width(Size::px(76.))
                            .height(Size::px(24.))
                            .padding(0.)
                            .on_press(move |_| state.write().dock_open = !expanded)
                            .child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(10.)
                                    .text(if expanded { "Hide dock" } else { "Show dock" }),
                            ),
                    ),
            )
            .maybe(expanded, |parent| {
                parent.child(
                    rect()
                        .width(Size::fill())
                        .height(Size::flex(1.))
                        .child(body),
                )
            })
    }
}

#[derive(PartialEq)]
struct ActivityDockView {
    state: State<AppState>,
}

impl Component for ActivityDockView {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut events = self.state.read().agent_events.clone();
        events.reverse();
        events.truncate(100);
        let events: Vec<Element> = events
            .into_iter()
            .map(|event| {
                rect()
                    .width(Size::fill())
                    .height(Size::px(24.))
                    .horizontal()
                    .cross_align(Alignment::center())
                    .spacing(10.)
                    .child(
                        label()
                            .color(colors.text_placeholder)
                            .font_size(9.5)
                            .width(Size::px(54.))
                            .text(event.time),
                    )
                    .child(
                        label()
                            .color(colors.text_secondary)
                            .font_size(9.5)
                            .width(Size::px(100.))
                            .text(event.kind.replace(['.', '_'], " ")),
                    )
                    .child(
                        label()
                            .color(colors.text_primary)
                            .font_size(10.5)
                            .max_lines(1)
                            .text(event.message),
                    )
                    .into_element()
            })
            .collect();

        if events.is_empty() {
            rect()
                .width(Size::fill())
                .height(Size::fill())
                .horizontal()
                .main_align(Alignment::center())
                .child(
                    label()
                        .color(colors.text_secondary)
                        .font_size(11.)
                        .text("No run activity yet"),
                )
                .into_element()
        } else {
            ScrollView::new()
                .direction(Direction::Vertical)
                .children(events)
                .into_element()
        }
    }
}

#[derive(PartialEq)]
struct ChangesDockView {
    state: State<AppState>,
    workspace: WorkspaceCommands,
}

impl Component for ChangesDockView {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let files = self.state.read().changed_files.clone();
        let mut rows = Vec::new();
        for file in files {
            rows.push(
                DockChangeRow {
                    state: self.state,
                    file,
                    workspace: self.workspace,
                }
                .into_element(),
            );
        }

        if rows.is_empty() {
            rect()
                .width(Size::fill())
                .height(Size::fill())
                .horizontal()
                .main_align(Alignment::center())
                .child(
                    label()
                        .color(colors.text_secondary)
                        .font_size(11.)
                        .text("No files changed in the active run"),
                )
                .into_element()
        } else {
            ScrollView::new()
                .direction(Direction::Horizontal)
                .children(rows)
                .into_element()
        }
    }
}

#[derive(PartialEq)]
struct DockChangeRow {
    state: State<AppState>,
    file: crate::state::ChangedFile,
    workspace: WorkspaceCommands,
}

impl Component for DockChangeRow {
    fn render(&self) -> impl IntoElement {
        let mut state = self.state;
        let status = self.file.status;
        let path = std::path::PathBuf::from(&self.file.path);
        let deleted_path = self.file.path.clone();
        let workspace = self.workspace;
        let colors = get_theme_or_default().read().colors.clone();
        let status_color = match status {
            AgentFileStatus::Created => colors.success,
            AgentFileStatus::Modified => colors.warning,
            AgentFileStatus::Deleted => colors.error,
        };

        Button::new()
            .flat()
            .compact()
            .width(Size::px(280.))
            .height(Size::px(30.))
            .padding(Gaps::new(8., 0., 8., 0.))
            .on_press(move |_| {
                if status == AgentFileStatus::Deleted {
                    state.write().show_notice(
                        NoticeTone::Error,
                        format!("{} was deleted by the active run", deleted_path),
                    );
                } else {
                    let absolute_path = state.read().workspace_root.join(&path);
                    workspace.load_file(absolute_path);
                }
            })
            .child(
                rect()
                    .width(Size::fill())
                    .horizontal()
                    .cross_align(Alignment::center())
                    .spacing(8.)
                    .child(
                        label()
                            .color(status_color)
                            .font_size(10.5)
                            .font_weight(FontWeight::BOLD)
                            .text(status.marker()),
                    )
                    .child(
                        label()
                            .color(colors.text_primary)
                            .font_size(10.5)
                            .max_lines(1)
                            .text(self.file.path.clone()),
                    ),
            )
    }
}

#[derive(PartialEq)]
struct TerminalDockView {
    state: State<AppState>,
    terminal: Option<TerminalRuntime>,
}

impl Component for TerminalDockView {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let output = state.read().terminal_output.clone();
        let running = state.read().terminal_running;
        let command_state = use_state(String::new);
        let command = command_state.read().trim().to_string();
        let terminal = self.terminal.clone();
        let restart_runtime = terminal.clone();
        let resize_terminal = terminal.clone();

        rect()
            .width(Size::fill())
            .height(Size::fill())
            .vertical()
            .content(Content::flex())
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::px(26.))
                    .horizontal()
                    .main_align(Alignment::space_between())
                    .cross_align(Alignment::center())
                    .padding(Gaps::new(10., 0., 6., 0.))
                    .child(
                        rect()
                            .horizontal()
                            .cross_align(Alignment::center())
                            .spacing(6.)
                            .child(
                                rect()
                                    .width(Size::px(7.))
                                    .height(Size::px(7.))
                                    .corner_radius(CornerRadius::new_all(4.))
                                    .background(if running {
                                        colors.success
                                    } else {
                                        colors.error
                                    }),
                            )
                            .child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(10.)
                                    .text("Local shell · user privileges"),
                            ),
                    )
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .height(Size::px(22.))
                            .padding(Gaps::new(7., 0., 7., 0.))
                            .enabled(terminal.is_some())
                            .on_press(move |_| restart_terminal(state, restart_runtime.clone()))
                            .child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(10.)
                                    .text("Restart"),
                            ),
                    ),
            )
            .child(
                ScrollView::new()
                    .direction(Direction::Vertical)
                    .height(Size::flex(1.))
                    .child(
                        label()
                            .width(Size::fill())
                            .color(colors.text_primary)
                            .font_size(11.)
                            .font_family("Jetbrains Mono")
                            .text(if output.is_empty() {
                                "Starting terminal…".to_string()
                            } else {
                                output
                            }),
                    ),
            )
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::px(34.))
                    .horizontal()
                    .cross_align(Alignment::center())
                    .spacing(6.)
                    .padding(Gaps::new(8., 0., 8., 0.))
                    .background(colors.surface_secondary)
                    .border(Border::new().fill(colors.border).width(BorderWidth {
                        top: 1.,
                        right: 0.,
                        bottom: 0.,
                        left: 0.,
                    }))
                    .child(
                        Input::new(command_state.into_writable())
                            .placeholder("Type a terminal command…")
                            .width(Size::flex(1.))
                            .filled()
                            .on_submit({
                                let terminal = terminal.clone();
                                let mut command_state = command_state;
                                move |value| {
                                    if let Some(terminal) = &terminal
                                        && let Err(error) = terminal.write(&format!("{value}\r"))
                                    {
                                        state.write().show_notice(NoticeTone::Error, error);
                                    }
                                    command_state.set(String::new());
                                }
                            }),
                    )
                    .child(
                        Button::new()
                            .filled()
                            .compact()
                            .height(Size::px(26.))
                            .padding(Gaps::new(10., 0., 10., 0.))
                            .enabled(running && !command.is_empty())
                            .on_press({
                                let terminal = terminal.clone();
                                let mut command_state = command_state;
                                let command = command.clone();
                                move |_| {
                                    if let Some(terminal) = &terminal
                                        && let Err(error) =
                                            terminal.write(&format!("{}\r", command.clone()))
                                    {
                                        state.write().show_notice(NoticeTone::Error, error);
                                    }
                                    command_state.set(String::new());
                                }
                            })
                            .child(
                                label()
                                    .color(colors.text_primary)
                                    .font_size(10.5)
                                    .text("Send"),
                            ),
                    ),
            )
            .on_sized(move |event: Event<SizedEventData>| {
                if let Some(terminal) = &resize_terminal {
                    let size = event.visible_area.size;
                    let columns = (size.width / 7.).clamp(20., 240.) as u16;
                    let rows = (size.height / 17.).clamp(2., 100.) as u16;
                    let _ = terminal.resize(rows, columns);
                }
            })
    }
}

fn restart_terminal(mut state: State<AppState>, terminal: Option<TerminalRuntime>) {
    let Some(terminal) = terminal else {
        return;
    };
    state.write().terminal_output.clear();
    spawn(async move {
        let result = tokio::task::spawn_blocking(move || terminal.restart()).await;
        match result {
            Ok(Ok(())) => {
                let mut state = state.write();
                state.terminal_running = true;
                state.show_notice(NoticeTone::Success, "Terminal restarted");
            }
            Ok(Err(error)) => state.write().show_notice(
                NoticeTone::Error,
                format!("Terminal restart failed: {error}"),
            ),
            Err(error) => state.write().show_notice(
                NoticeTone::Error,
                format!("Terminal restart worker failed: {error}"),
            ),
        }
    });
}

#[derive(PartialEq)]
struct ApprovalCard {
    state: State<AppState>,
    client: PrumoClient,
    request_id: String,
}

impl Component for ApprovalCard {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let state = self.state;
        let client = self.client.clone();
        let request_id = self.request_id.clone();
        let run_id = state.read().active_run_id.clone().unwrap_or_default();

        rect()
            .width(Size::fill())
            .vertical()
            .padding(Gaps::new_all(8.))
            .spacing(6.)
            .background(colors.surface_secondary)
            .corner_radius(CornerRadius::new_all(5.))
            .child(
                label()
                    .color(colors.warning)
                    .font_size(10.5)
                    .font_weight(FontWeight::MEDIUM)
                    .text("Approval required"),
            )
            .child(
                label()
                    .color(colors.text_secondary)
                    .font_size(9.5)
                    .max_lines(2)
                    .text(request_id.clone()),
            )
            .child(
                rect()
                    .width(Size::fill())
                    .horizontal()
                    .main_align(Alignment::end())
                    .spacing(6.)
                    .child(
                        Button::new()
                            .on_press({
                                let client = client.clone();
                                let run_id = run_id.clone();
                                let request_id = request_id.clone();
                                move |_| {
                                    answer_permission(
                                        state,
                                        client.clone(),
                                        &run_id,
                                        &request_id,
                                        true,
                                    )
                                }
                            })
                            .child(
                                label()
                                    .color(colors.success)
                                    .font_size(10.5)
                                    .text("Approve"),
                            ),
                    )
                    .child(
                        Button::new()
                            .on_press(move |_| {
                                answer_permission(
                                    state,
                                    client.clone(),
                                    &run_id,
                                    &request_id,
                                    false,
                                )
                            })
                            .child(label().color(colors.error).font_size(10.5).text("Deny")),
                    ),
            )
    }
}

fn start_run(mut state: State<AppState>, client: PrumoClient, goal: String) {
    let goal = goal.trim().to_string();
    if goal.is_empty() {
        return;
    }
    let workspace_root = state.read().workspace_root.clone();
    let config = state.read().config.clone();
    state.write().show_notice(
        NoticeTone::Info,
        format!("Starting Run with {}…", config.provider),
    );
    spawn(async move {
        let worker_client = client.clone();
        let worker_workspace = workspace_root.clone();
        let result = tokio::task::spawn_blocking(move || {
            worker_client.start_goal_with_options(
                &goal,
                &worker_workspace,
                Some(&config.provider),
                Some(&config.model),
            )
        })
        .await;
        match result {
            Ok(Ok(run_id)) => {
                let mut app_state = state.write();
                app_state.active_run_id = Some(run_id.clone());
                app_state.agent_status = AgentStatus::Working;
                app_state.show_notice(NoticeTone::Success, format!("Run {run_id} started"));
            }
            Ok(Err(error)) => {
                state
                    .write()
                    .show_notice(NoticeTone::Error, format!("Could not start run: {error}"));
            }
            Err(error) => {
                state
                    .write()
                    .show_notice(NoticeTone::Error, format!("Agent worker failed: {error}"));
            }
        }
    });
}

fn cancel_run(mut state: State<AppState>, client: PrumoClient, run_id: &str) {
    if run_id.is_empty() {
        return;
    }
    let run_id = run_id.to_string();
    spawn(async move {
        let worker_client = client.clone();
        let worker_run_id = run_id.clone();
        let result =
            tokio::task::spawn_blocking(move || worker_client.cancel(&worker_run_id)).await;
        match result {
            Ok(Ok(())) => state
                .write()
                .show_notice(NoticeTone::Success, format!("Run {run_id} cancelled")),
            Ok(Err(error)) => state
                .write()
                .show_notice(NoticeTone::Error, format!("Cancel failed: {error}")),
            Err(error) => state
                .write()
                .show_notice(NoticeTone::Error, format!("Cancel worker failed: {error}")),
        }
    });
}

fn steer_run(mut state: State<AppState>, client: PrumoClient, run_id: &str, message: String) {
    let message = message.trim().to_string();
    if run_id.is_empty() || message.is_empty() {
        return;
    }
    let run_id = run_id.to_string();
    spawn(async move {
        let worker_client = client.clone();
        let worker_run_id = run_id.clone();
        let worker_message = message.clone();
        let result = tokio::task::spawn_blocking(move || {
            worker_client.steer(&worker_run_id, &worker_message)
        })
        .await;
        match result {
            Ok(Ok(())) => state
                .write()
                .show_notice(NoticeTone::Success, "Run steering message sent"),
            Ok(Err(error)) => state
                .write()
                .show_notice(NoticeTone::Error, format!("Steering failed: {error}")),
            Err(error) => state.write().show_notice(
                NoticeTone::Error,
                format!("Steering worker failed: {error}"),
            ),
        }
    });
}

fn answer_permission(
    mut state: State<AppState>,
    client: PrumoClient,
    run_id: &str,
    request_id: &str,
    approved: bool,
) {
    let run_id = run_id.to_string();
    let request_id = request_id.to_string();
    spawn(async move {
        let worker_client = client.clone();
        let worker_run_id = run_id.clone();
        let worker_request_id = request_id.clone();
        let result = tokio::task::spawn_blocking(move || {
            if approved {
                worker_client.approve(&worker_run_id, &worker_request_id)
            } else {
                worker_client.deny(&worker_run_id, &worker_request_id)
            }
        })
        .await;
        match result {
            Ok(Ok(())) => {
                let mut app_state = state.write();
                app_state
                    .pending_permissions
                    .retain(|pending| pending != &request_id);
                app_state.show_notice(
                    NoticeTone::Success,
                    if approved {
                        "Permission approved"
                    } else {
                        "Permission denied"
                    },
                );
            }
            Ok(Err(error)) => {
                state
                    .write()
                    .show_notice(NoticeTone::Error, format!("Permission failed: {error}"));
            }
            Err(error) => {
                state.write().show_notice(
                    NoticeTone::Error,
                    format!("Permission worker failed: {error}"),
                );
            }
        }
    });
}
