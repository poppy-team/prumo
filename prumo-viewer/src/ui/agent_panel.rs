use crate::WorkspaceCommands;
use crate::client::protocol::PrumoClient;
use crate::services::run_observability;
use crate::services::search::{TextMatch, replace_workspace_text, search_workspace_text};
use crate::services::terminal::{TerminalRuntime, terminal_key_bytes};
use crate::services::workspace::refresh_workspace;
use crate::state::{
    AgentFileStatus, AgentMetrics, AgentStatus, AppState, ChatMessage, ChatMessageRole,
    ConnectionStatus, DiagnosticSeverity, DockView, EditorReveal, NoticeTone,
};
use crate::ui::chrome::IconButton;
use crate::ui::icons::icon;
use freya::prelude::*;
use std::time::Duration;
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
        let pending_details = state.read().pending_permission_details.clone();
        let agent_status = state.read().agent_status;
        let follow_agent = state.read().follow_agent;
        let conflict_count = state.read().follow_conflict_paths.len();
        let events = state.read().agent_events.clone();
        let messages = state.read().agent_messages.clone();
        let _last_prompt = state.read().prompt_history.last().cloned();
        let agent_log = state.read().agent_log.clone();
        let metrics = state.read().agent_metrics.clone();
        let sessions = state.read().agent_sessions.clone();
        let active_session_idx = state.read().active_session_index;
        let model_picker_open = state.read().agent_model_picker_open;
        let available_models = state.read().available_models.clone();
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
            .background(colors.background)
            .border(Border::new().fill(colors.border).width(BorderWidth {
                top: 0.,
                right: 0.,
                bottom: 0.,
                left: 1.,
            }))
            .on_pointer_down(move |_| {
                let is_explorer_focused = state.read().explorer_focused;
                if is_explorer_focused {
                    state.write().explorer_focused = false;
                }
            })
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::px(32.))
                    .horizontal()
                    .cross_align(Alignment::center())
                    .padding(Gaps::new(0., 8., 0., 8.))
                    .background(colors.background)
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
                            .font_weight(FontWeight::MEDIUM)
                            .text("Agent"),
                    )
                    .child(
                        rect()
                            .width(Size::fill())
                            .horizontal()
                            .main_align(Alignment::space_between())
                            .cross_align(Alignment::center())
                            .child(FollowToggle {
                                state,
                                follow_agent,
                                conflict_count,
                            })
                            .child(
                                rect()
                                    .horizontal()
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
                    ),
            )
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::px(28.))
                    .horizontal()
                    .cross_align(Alignment::center())
                    .main_align(Alignment::space_between())
                    .padding(Gaps::new(0., 4., 0., 4.))
                    .background(colors.surface_secondary)
                    .border(Border::new().fill(colors.border).width(BorderWidth {
                        top: 0.,
                        right: 0.,
                        bottom: 1.,
                        left: 0.,
                    }))
                    .child(
                        rect()
                            .width(Size::flex(1.))
                            .height(Size::fill())
                            .horizontal()
                            .child(
                                ScrollView::new()
                                    .direction(Direction::Horizontal)
                                    .children(sessions.iter().enumerate().map(|(idx, s)| {
                                        AgentSessionTab {
                                            state,
                                            title: s.title.clone(),
                                            status: s.status,
                                            index: idx,
                                            is_active: idx == active_session_idx,
                                            can_close: sessions.len() > 1,
                                        }
                                        .into()
                                    })),
                            ),
                    )
                    .child(
                        rect()
                            .width(Size::px(26.))
                            .height(Size::px(26.))
                            .center()
                            .child(
                                IconButton {
                                    icon: "plus",
                                    label: "New conversation",
                                    size: 11.,
                                    on_press: (move |_: Event<PressEventData>| {
                                        state.write().create_new_session();
                                    })
                                    .into(),
                                }
                            ),
                    ),
            )
            .child(
                AgentMetricsBar {
                    metrics: metrics.clone(),
                }
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
                                            .font_weight(FontWeight::MEDIUM)
                                            .text("Run"),
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
                                        rect()
                                            .width(Size::fill())
                                            .horizontal()
                                            .main_align(Alignment::space_between())
                                            .cross_align(Alignment::center())
                                            .child(
                                                label().color(colors.text_placeholder).font_size(10.).text(
                                                    format!(
                                                        "{} · {} files changed",
                                                        connection.label(),
                                                        changed_file_count
                                                    ),
                                                ),
                                            )
                                            .child(
                                                QuickModelPill {
                                                    state,
                                                    provider: runner_config.provider.clone(),
                                                    model: runner_model.clone(),
                                                },
                                            ),
                                    ),
                            )
                            .child(if pending_permissions.is_empty() {
                                Element::from(label().text(""))
                            } else {
                                Element::from(
                                    rect().width(Size::fill()).vertical().spacing(6.).children(
                                        pending_permissions.iter().map(|request_id| {
                                            let detail = pending_details
                                                .iter()
                                                .find(|d| &d.request_id == request_id);
                                            ApprovalCard {
                                                state,
                                                client: client.clone(),
                                                request_id: request_id.clone(),
                                                tool: detail
                                                    .map(|d| d.tool.clone())
                                                    .unwrap_or_default(),
                                                arguments_preview: detail
                                                    .map(|d| d.arguments_preview.clone())
                                                    .unwrap_or_default(),
                                                fingerprint: detail
                                                    .map(|d| d.fingerprint.clone())
                                                    .unwrap_or_default(),
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
                                    .font_weight(FontWeight::MEDIUM)
                                    .text(if !messages.is_empty() {
                                        "Conversation & Activity"
                                    } else {
                                        "Activity"
                                    }),
                            )
                            .child(if messages.is_empty() && events.is_empty() {
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
                                                        "No recent events. Start a conversation below."
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
                                                        .padding(Gaps::new(0., 9., 0., 9.))
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
                                    rect()
                                        .width(Size::fill())
                                        .vertical()
                                        .spacing(6.)
                                        .children(messages.iter().map(|msg| {
                                            ChatMessageView {
                                                state,
                                                message: msg.clone(),
                                                runner_model: runner_model.clone(),
                                            }
                                            .into()
                                        }))
                                        .maybe(!events.is_empty(), |r| {
                                            r.child(
                                                rect()
                                                    .width(Size::fill())
                                                    .vertical()
                                                    .spacing(4.)
                                                    .padding(Gaps::new(8., 0., 0., 0.))
                                                    .child(
                                                        label()
                                                            .color(colors.text_placeholder)
                                                            .font_size(9.5)
                                                            .font_weight(FontWeight::MEDIUM)
                                                            .text("TOOL CALLS & SYSTEM EVENTS"),
                                                    )
                                                    .children(events.iter().map(|event| {
                                                        AgentEventCard {
                                                            time: event.time.clone(),
                                                            kind: event.kind.clone(),
                                                            message: event.message.clone(),
                                                        }
                                                        .into()
                                                    })),
                                            )
                                        }),
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
                    .background(colors.background)
                    .border(Border::new().fill(colors.border).width(BorderWidth {
                        top: 1.,
                        right: 0.,
                        bottom: 0.,
                        left: 0.,
                    }))
                    .maybe(model_picker_open, |r| {
                        r.child(ModelPickerPopover {
                            state,
                            current_provider: runner_config.provider.clone(),
                            current_model: runner_model.clone(),
                            available_models: available_models.clone(),
                        })
                    })
                    .child(if can_steer {
                        Element::from(
                            rect()
                                .width(Size::fill())
                                .vertical()
                                .spacing(6.)
                                .padding(Gaps::new(8., 10., 6., 10.))
                                .background(colors.surface_secondary)
                                .border(Border::new().fill(colors.border).width(1.))
                                .corner_radius(CornerRadius::new_all(8.))
                                .child(
                                    rect()
                                        .width(Size::fill())
                                        .min_height(Size::px(38.))
                                        .font_size(12.)
                                        .child(
                                            Input::new(steer_state.into_writable())
                                                .placeholder("Steer active run (e.g. prioritize tests, skip docs)…")
                                                .width(Size::fill())
                                                .flat()
                                                .compact()
                                                .on_submit({
                                                    let client = client.clone();
                                                    let active_run = active_run.clone().unwrap_or_default();
                                                    let mut steer_state = steer_state;
                                                    move |value| {
                                                        steer_run(state, client.clone(), &active_run, value);
                                                        steer_state.set(String::new());
                                                    }
                                                }),
                                        ),
                                )
                                .child(
                                    rect()
                                        .width(Size::fill())
                                        .horizontal()
                                        .main_align(Alignment::space_between())
                                        .cross_align(Alignment::center())
                                        .child(
                                            rect()
                                                .horizontal()
                                                .spacing(6.)
                                                .cross_align(Alignment::center())
                                                .child(QuickModelPill {
                                                    state,
                                                    provider: runner_config.provider.clone(),
                                                    model: runner_model.clone(),
                                                })
                                                .child(
                                                    rect()
                                                        .padding(Gaps::new(2., 5., 2., 5.))
                                                        .corner_radius(CornerRadius::new_all(4.))
                                                        .background(colors.surface_tertiary)
                                                        .child(
                                                            label()
                                                                .color(colors.warning)
                                                                .font_size(9.5)
                                                                .font_weight(FontWeight::MEDIUM)
                                                                .text("⚡ Steer"),
                                                        ),
                                                ),
                                        )
                                        .child(
                                            rect()
                                                .horizontal()
                                                .spacing(6.)
                                                .cross_align(Alignment::center())
                                                .child(
                                                    Button::new()
                                                        .flat()
                                                        .compact()
                                                        .height(Size::px(26.))
                                                        .padding(Gaps::new(0., 9., 0., 9.))
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
                                                                .text("Cancel"),
                                                        ),
                                                )
                                                .child(
                                                    Button::new()
                                                        .compact()
                                                        .height(Size::px(26.))
                                                        .padding(Gaps::new(0., 11., 0., 11.))
                                                        .corner_radius(CornerRadius::new_all(5.))
                                                        .enabled(
                                                            connection == ConnectionStatus::Connected
                                                                && !steer.is_empty(),
                                                        )
                                                        .on_press({
                                                            let client = client.clone();
                                                            let active_run =
                                                                active_run.clone().unwrap_or_default();
                                                            let steer = steer.clone();
                                                            let mut steer_state = steer_state;
                                                            move |_| {
                                                                steer_run(
                                                                    state,
                                                                    client.clone(),
                                                                    &active_run,
                                                                    steer.clone(),
                                                                );
                                                                steer_state.set(String::new());
                                                            }
                                                        })
                                                        .child(
                                                            label()
                                                                .color(colors.text_primary)
                                                                .font_size(11.)
                                                                .font_weight(FontWeight::SEMI_BOLD)
                                                                .text("Send ↵"),
                                                        ),
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
                                .padding(Gaps::new(8., 10., 6., 10.))
                                .background(colors.surface_secondary)
                                .border(Border::new().fill(colors.border).width(1.))
                                .corner_radius(CornerRadius::new_all(8.))
                                .child(
                                    rect()
                                        .width(Size::fill())
                                        .min_height(Size::px(38.))
                                        .font_size(12.)
                                        .child(
                                            Input::new(goal_state.into_writable())
                                                .placeholder("Ask a question, request changes, or set a goal…")
                                                .width(Size::fill())
                                                .flat()
                                                .compact()
                                                .on_submit({
                                                    let client = client.clone();
                                                    let mut goal_state = goal_state;
                                                    move |value| {
                                                        start_run(state, client.clone(), value);
                                                        goal_state.set(String::new());
                                                    }
                                                }),
                                        ),
                                )
                                .child(
                                    rect()
                                        .width(Size::fill())
                                        .horizontal()
                                        .main_align(Alignment::space_between())
                                        .cross_align(Alignment::center())
                                        .child(
                                            rect()
                                                .horizontal()
                                                .spacing(6.)
                                                .cross_align(Alignment::center())
                                                .child(QuickModelPill {
                                                    state,
                                                    provider: runner_config.provider.clone(),
                                                    model: runner_model.clone(),
                                                })
                                                .child(
                                                    rect()
                                                        .padding(Gaps::new(2., 5., 2., 5.))
                                                        .corner_radius(CornerRadius::new_all(4.))
                                                        .background(colors.surface_tertiary)
                                                        .child(
                                                            label()
                                                                .color(colors.text_secondary)
                                                                .font_size(9.5)
                                                                .font_weight(FontWeight::MEDIUM)
                                                                .text("🎯 Goal"),
                                                        ),
                                                ),
                                        )
                                        .child(
                                            Button::new()
                                                .compact()
                                                .height(Size::px(26.))
                                                .padding(Gaps::new(0., 12., 0., 12.))
                                                .corner_radius(CornerRadius::new_all(5.))
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
                                                    label()
                                                        .color(if !goal.is_empty() {
                                                            colors.text_primary
                                                        } else {
                                                            colors.text_placeholder
                                                        })
                                                        .font_size(11.)
                                                        .font_weight(FontWeight::SEMI_BOLD)
                                                        .text("Run ↵"),
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
        let (error_count, warning_count) = state.read().diagnostic_counts();
        let findings = run_observability::findings(&state.read().agent_events);
        let quality_summary = run_observability::QualitySummary::of(&findings);
        let body: Element = match selected_view {
            DockView::Activity => ActivityDockView { state }.into_element(),
            DockView::Changes => ChangesDockView {
                state,
                workspace: self.workspace,
            }
            .into_element(),
            DockView::Search => SearchDockView {
                state,
                workspace: self.workspace,
            }
            .into_element(),
            DockView::Terminal => TerminalDockView {
                state,
                terminal: self.terminal.clone(),
            }
            .into_element(),
            DockView::Problems => ProblemsDockView {
                state,
                workspace: self.workspace,
            }
            .into_element(),
            DockView::Evidence => EvidenceDockView {
                state,
                workspace: self.workspace,
            }
            .into_element(),
            DockView::Quality => QualityDockView { state }.into_element(),
        };

        rect()
            .width(Size::fill())
            .height(Size::px(if expanded { 180. } else { 26. }))
            .background(colors.background)
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
                    .height(Size::px(26.))
                    .horizontal()
                    .cross_align(Alignment::center())
                    .padding(Gaps::new(0., 6., 0., 6.))
                    .spacing(2.)
                    .background(colors.background)
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .height(Size::px(24.))
                            .padding(Gaps::new(0., 8., 0., 8.))
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
                                    .font_weight(FontWeight::MEDIUM)
                                    .text("Activity"),
                            ),
                    )
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .height(Size::px(24.))
                            .padding(Gaps::new(0., 8., 0., 8.))
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
                                    .font_weight(FontWeight::MEDIUM)
                                    .text("Changes"),
                            ),
                    )
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .height(Size::px(24.))
                            .padding(Gaps::new(0., 8., 0., 8.))
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
                                    .font_weight(FontWeight::MEDIUM)
                                    .text("Terminal"),
                            ),
                    )
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .height(Size::px(22.))
                            .padding(Gaps::new(0., 6., 0., 6.))
                            .on_press(move |_| {
                                let mut state = state.write();
                                state.dock_view = DockView::Search;
                                state.dock_open = true;
                            })
                            .child(
                                label()
                                    .color(if selected_view == DockView::Search {
                                        colors.text_primary
                                    } else {
                                        colors.text_secondary
                                    })
                                    .font_size(10.)
                                    .font_weight(FontWeight::MEDIUM)
                                    .text("Search"),
                            ),
                    )
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .height(Size::px(22.))
                            .padding(Gaps::new(0., 6., 0., 6.))
                            .on_press(move |_| {
                                let mut state = state.write();
                                state.dock_view = DockView::Problems;
                                state.dock_open = true;
                            })
                            .child(
                                label()
                                    .color(if selected_view == DockView::Problems {
                                        colors.text_primary
                                    } else {
                                        colors.text_secondary
                                    })
                                    .font_size(10.)
                                    .font_weight(FontWeight::MEDIUM)
                                    .text(format!(
                                        "Problems ({error_count} errors, {warning_count} warnings)"
                                    )),
                            ),
                    )
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .height(Size::px(22.))
                            .padding(Gaps::new(0., 6., 0., 6.))
                            .on_press(move |_| {
                                let mut state = state.write();
                                state.dock_view = DockView::Evidence;
                                state.dock_open = true;
                            })
                            .child(
                                label()
                                    .color(if selected_view == DockView::Evidence {
                                        colors.text_primary
                                    } else {
                                        colors.text_secondary
                                    })
                                    .font_size(10.)
                                    .font_weight(FontWeight::MEDIUM)
                                    .text("Evidence"),
                            ),
                    )
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .height(Size::px(22.))
                            .padding(Gaps::new(0., 6., 0., 6.))
                            .on_press(move |_| {
                                let mut state = state.write();
                                state.dock_view = DockView::Quality;
                                state.dock_open = true;
                            })
                            .child(
                                label()
                                    .color(if selected_view == DockView::Quality {
                                        colors.text_primary
                                    } else {
                                        colors.text_secondary
                                    })
                                    .font_size(10.)
                                    .font_weight(FontWeight::MEDIUM)
                                    .text(format!(
                                        "Quality ({})",
                                        quality_summary.failures + quality_summary.warnings
                                    )),
                            ),
                    )
                    .child(
                        rect()
                            .expanded()
                            .horizontal()
                            .main_align(Alignment::end())
                            .child(IconButton {
                                icon: if expanded {
                                    "chevron_down"
                                } else {
                                    "chevron_up"
                                },
                                label: if expanded {
                                    "Collapse dock"
                                } else {
                                    "Expand dock"
                                },
                                size: 11.,
                                on_press: (move |_: Event<PressEventData>| {
                                    state.write().dock_open = !expanded;
                                })
                                .into(),
                            }),
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
            .padding(Gaps::new(0., 8., 0., 8.))
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
                    .content(Content::flex())
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

#[derive(Clone, PartialEq)]
struct WorkspaceSearchRequest {
    root: std::path::PathBuf,
    query: String,
    match_case: bool,
    use_regex: bool,
}

#[derive(PartialEq)]
struct SearchDockView {
    state: State<AppState>,
    workspace: WorkspaceCommands,
}

impl Component for SearchDockView {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let workspace = self.workspace;
        let search_query = use_state(|| state.read().search_query.clone());
        let search_replacement = use_state(|| state.read().search_replacement.clone());
        let mut search_case = use_state(|| state.read().search_match_case);
        let mut search_regex = use_state(|| state.read().search_use_regex);
        let mut armed = use_state(|| false);
        let mut armed_for = use_state(|| None::<(String, String)>);
        let mut results_state = use_state(Vec::<TextMatch>::new);
        let mut error_state = use_state(|| None::<String>);
        let mut loading_state = use_state(|| false);
        let mut search_generation = use_state(|| 0usize);
        let query = search_query.read().clone();
        let replacement = search_replacement.read().clone();
        let match_case = *search_case.read();
        let use_regex = *search_regex.read();
        let root = state.read().workspace_root.clone();
        let search_request = WorkspaceSearchRequest {
            root: root.clone(),
            query: query.clone(),
            match_case,
            use_regex,
        };
        use_side_effect_with_deps(&search_request, move |request| {
            let request = request.clone();
            let request_id = {
                let mut generation = search_generation.write();
                *generation = generation.saturating_add(1);
                *generation
            };
            results_state.set(Vec::new());
            error_state.set(None);
            if request.query.trim().is_empty() {
                loading_state.set(false);
                return;
            }
            loading_state.set(true);
            let generation = search_generation;
            let mut results_for_task = results_state;
            let mut errors_for_task = error_state;
            let mut loading_for_task = loading_state;
            spawn(async move {
                async_io::Timer::after(Duration::from_millis(180)).await;
                if *generation.peek() != request_id {
                    return;
                }
                let result = crate::run_blocking(move || {
                    search_workspace_text(
                        &request.root,
                        &request.query,
                        request.match_case,
                        request.use_regex,
                        200,
                    )
                })
                .await;
                if *generation.peek() != request_id {
                    return;
                }
                loading_for_task.set(false);
                match result {
                    Ok(Ok(results)) => {
                        results_for_task.set(results);
                        errors_for_task.set(None);
                    }
                    Ok(Err(error)) => errors_for_task.set(Some(error)),
                    Err(error) => {
                        errors_for_task.set(Some(format!("Search worker failed: {error}")))
                    }
                }
            });
        });
        let results = results_state.read().clone();
        let pattern_error = error_state.read().clone();
        let search_loading = *loading_state.read();
        let match_count = results.len();
        let effectively_armed =
            *armed.read() && *armed_for.read() == Some((query.clone(), replacement.clone()));

        rect()
            .width(Size::fill())
            .height(Size::fill())
            .vertical()
            .content(Content::flex())
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::px(30.))
                    .horizontal()
                    .content(Content::flex())
                    .cross_align(Alignment::center())
                    .spacing(4.)
                    .padding(Gaps::new(0., 6., 0., 6.))
                    .child(
                        Input::new(search_query.into_writable())
                            .placeholder("Search in project…")
                            .width(Size::flex(1.))
                            .auto_focus(true)
                            .on_submit(move |value| {
                                state.write().search_query = value;
                            }),
                    )
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .height(Size::px(22.))
                            .padding(Gaps::new(0., 6., 0., 6.))
                            .background(if match_case {
                                colors.surface_tertiary
                            } else {
                                Color::TRANSPARENT
                            })
                            .on_press(move |_| {
                                search_case.set(!match_case);
                                state.write().search_match_case = !match_case;
                            })
                            .child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(10.)
                                    .text("Aa"),
                            ),
                    )
                    .child(
                        Button::new()
                            .flat()
                            .compact()
                            .height(Size::px(22.))
                            .padding(Gaps::new(0., 6., 0., 6.))
                            .background(if use_regex {
                                colors.surface_tertiary
                            } else {
                                Color::TRANSPARENT
                            })
                            .on_press(move |_| {
                                search_regex.set(!use_regex);
                                state.write().search_use_regex = !use_regex;
                            })
                            .child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(10.)
                                    .text(".*"),
                            ),
                    )
                    .child(
                        label()
                            .color(if pattern_error.is_some() {
                                colors.error
                            } else {
                                colors.text_placeholder
                            })
                            .font_size(10.)
                            .text(if search_loading {
                                "Searching…".to_string()
                            } else {
                                pattern_error.clone().unwrap_or_else(|| {
                                    if match_count >= 200 {
                                        "200+ matches".to_string()
                                    } else {
                                        format!("{match_count} matches")
                                    }
                                })
                            }),
                    )
                    .child(
                        Button::new()
                            .compact()
                            .height(Size::px(22.))
                            .padding(Gaps::new(0., 8., 0., 8.))
                            .enabled(match_count > 0)
                            .on_press({
                                let replacement = replacement.clone();
                                move |_| {
                                    if effectively_armed {
                                        match replace_workspace_text(
                                            &root,
                                            &query,
                                            &replacement,
                                            match_case,
                                            use_regex,
                                        ) {
                                            Ok((replaced, files)) => {
                                                let mut app_state = state.write();
                                                let _ = refresh_workspace(&mut app_state);
                                                app_state.search_replacement =
                                                    replacement.clone();
                                                app_state.show_notice(
                                                    NoticeTone::Success,
                                                    format!(
                                                        "Replaced {replaced} matches in {files} files"
                                                    ),
                                                );
                                            }
                                            Err(error) => state.write().show_notice(
                                                NoticeTone::Error,
                                                error,
                                            ),
                                        }
                                        armed.set(false);
                                    } else {
                                        armed_for.set(Some((
                                            query.clone(),
                                            replacement.clone(),
                                        )));
                                        armed.set(true);
                                    }
                                }
                            })
                            .child(
                                label()
                                    .color(if effectively_armed {
                                        colors.error
                                    } else {
                                        colors.text_primary
                                    })
                                    .font_size(10.5)
                                    .text(if effectively_armed {
                                        format!("Confirm ({match_count})")
                                    } else {
                                        "Replace All".to_string()
                                    }),
                            ),
                    ),
            )
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::px(28.))
                    .horizontal()
                    .content(Content::flex())
                    .cross_align(Alignment::center())
                    .spacing(4.)
                    .padding(Gaps::new(0., 6., 0., 6.))
                    .child(
                        Input::new(search_replacement.into_writable())
                            .placeholder("Replace with…")
                            .width(Size::px(180.))
                            .on_submit(move |value| {
                                state.write().search_replacement = value;
                            }),
                    )
                    .child(
                        label()
                            .color(colors.text_placeholder)
                            .font_size(10.)
                            .text("Replace applies to every match above"),
                    ),
            )
            .child(
                ScrollView::new()
                    .direction(Direction::Vertical)
                    .height(Size::flex(1.))
                    .children(results.into_iter().map(|item| {
                        let row_workspace = workspace;
                        let row_path = item.path.clone();
                        let row_reveal = EditorReveal {
                            path: item.path.clone(),
                            line: item.line,
                            start: item.start,
                            end: item.end,
                        };
                        rect()
                            .width(Size::fill())
                            .min_height(Size::px(26.))
                            .horizontal()
                            .cross_align(Alignment::center())
                            .spacing(8.)
                            .padding(Gaps::new(0., 8., 0., 8.))
                            .on_all_press(move |_| {
                                row_workspace.load_file_with_reveal(
                                    row_path.clone(),
                                    row_reveal.clone(),
                                );
                            })
                            .child(
                                label()
                                    .color(colors.text_placeholder)
                                    .font_size(10.)
                                    .width(Size::px(36.))
                                    .text(format!(":{}", item.line)),
                            )
                            .child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(10.5)
                                    .max_lines(1)
                                    .text(item.rel_path.clone()),
                            )
                            .child(
                                label()
                                    .color(colors.text_primary)
                                    .font_size(10.5)
                                    .max_lines(1)
                                    .text(item.preview.clone()),
                            )
                            .into()
                    })),
            )
    }
}

#[derive(PartialEq)]
struct ProblemsDockView {
    state: State<AppState>,
    workspace: WorkspaceCommands,
}

impl Component for ProblemsDockView {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let state = self.state;
        let diagnostics = state.read().diagnostics.clone();
        let body: Element = if diagnostics.is_empty() {
            rect()
                .width(Size::fill())
                .height(Size::fill())
                .vertical()
                .main_align(Alignment::center())
                .cross_align(Alignment::center())
                .spacing(5.)
                .child(
                    label()
                        .color(colors.success)
                        .font_size(12.)
                        .font_weight(FontWeight::MEDIUM)
                        .text("No problems"),
                )
                .child(
                    label()
                        .color(colors.text_placeholder)
                        .font_size(10.5)
                        .text("Diagnostics from language servers and tasks appear here."),
                )
                .into_element()
        } else {
            ScrollView::new()
                .direction(Direction::Vertical)
                .height(Size::flex(1.))
                .children(diagnostics.into_iter().map(|diagnostic| {
                    let row_workspace = self.workspace;
                    let path = diagnostic.path.clone();
                    let reveal = EditorReveal {
                        path: path.clone(),
                        line: diagnostic.line,
                        start: 0,
                        end: 0,
                    };
                    let color = match diagnostic.severity {
                        DiagnosticSeverity::Error => colors.error,
                        DiagnosticSeverity::Warning => colors.warning,
                        DiagnosticSeverity::Information => colors.primary,
                        DiagnosticSeverity::Hint => colors.text_secondary,
                    };
                    let code = diagnostic
                        .code
                        .as_ref()
                        .map(|code| format!(" ({code})"))
                        .unwrap_or_default();
                    rect()
                        .width(Size::fill())
                        .min_height(Size::px(30.))
                        .horizontal()
                        .content(Content::flex())
                        .cross_align(Alignment::center())
                        .spacing(8.)
                        .padding(Gaps::new(0., 8., 0., 8.))
                        .on_all_press(move |_| {
                            row_workspace.load_file_with_reveal(path.clone(), reveal.clone());
                        })
                        .child(
                            label()
                                .color(color)
                                .font_size(10.)
                                .font_weight(FontWeight::BOLD)
                                .width(Size::px(76.))
                                .text(format!("{}{code}", diagnostic.severity.label())),
                        )
                        .child(
                            rect().width(Size::flex(1.)).content(Content::flex()).child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(10.5)
                                    .max_lines(1)
                                    .text(diagnostic.message),
                            ),
                        )
                        .child(
                            rect().width(Size::px(180.)).child(
                                label()
                                    .color(colors.text_placeholder)
                                    .font_size(10.)
                                    .max_lines(1)
                                    .text(format!(
                                        "{}:{} · {}",
                                        diagnostic
                                            .path
                                            .file_name()
                                            .and_then(|name| name.to_str())
                                            .unwrap_or("file"),
                                        diagnostic.line,
                                        diagnostic.source
                                    )),
                            ),
                        )
                        .into()
                }))
                .into_element()
        };

        rect()
            .width(Size::fill())
            .height(Size::fill())
            .vertical()
            .content(Content::flex())
            .background(colors.background)
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::px(26.))
                    .horizontal()
                    .cross_align(Alignment::center())
                    .padding(Gaps::new(0., 8., 0., 8.))
                    .child(
                        label()
                            .color(colors.text_secondary)
                            .font_size(10.)
                            .font_weight(FontWeight::MEDIUM)
                            .text("Problems"),
                    )
                    .child(rect().expanded()),
            )
            .child(body)
    }
}

#[derive(PartialEq)]
struct EvidenceDockView {
    state: State<AppState>,
    workspace: WorkspaceCommands,
}

impl Component for EvidenceDockView {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let entries = run_observability::evidence(
            &self.state.read().agent_events,
            &self.state.read().changed_files,
        );
        let body: Element = if entries.is_empty() {
            empty_dock_state(
                &colors,
                "No evidence yet",
                "Files the run creates or changes, plus the tools it used, appear here",
            )
        } else {
            ScrollView::new()
                .direction(Direction::Vertical)
                .children(entries.into_iter().map(|entry| {
                    let path = entry.path.clone().and_then(|relative| {
                        let root = self.state.read().workspace_root.clone();
                        let absolute = root.join(&relative);
                        absolute.is_file().then_some(absolute)
                    });
                    let workspace = self.workspace;
                    rect()
                        .width(Size::fill())
                        .height(Size::px(24.))
                        .horizontal()
                        .cross_align(Alignment::center())
                        .spacing(10.)
                        .on_press(move |_| {
                            if let Some(path) = path.clone() {
                                workspace.load_file(path);
                            }
                        })
                        .child(
                            label()
                                .color(colors.text_placeholder)
                                .font_size(9.5)
                                .width(Size::px(80.))
                                .text(match entry.kind {
                                    run_observability::EvidenceKind::File => "file",
                                    run_observability::EvidenceKind::Tool => "tool",
                                    run_observability::EvidenceKind::Approval => "approval",
                                }),
                        )
                        .child(
                            label()
                                .color(colors.text_primary)
                                .font_size(10.5)
                                .max_lines(1)
                                .text(entry.title),
                        )
                        .child(
                            label()
                                .color(colors.text_secondary)
                                .font_size(9.5)
                                .text(entry.detail),
                        )
                        .into_element()
                }))
                .into_element()
        };

        dock_panel(&colors, "Evidence", body)
    }
}

#[derive(PartialEq)]
struct QualityDockView {
    state: State<AppState>,
}

impl Component for QualityDockView {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let (agent_status, events) = {
            let state = self.state.read();
            (state.agent_status, state.agent_events.clone())
        };
        let findings = run_observability::findings(&events);
        let summary = run_observability::QualitySummary::of(&findings);
        let verdict = run_observability::verdict(agent_status, &summary);
        let body: Element = if findings.is_empty() {
            empty_dock_state(
                &colors,
                verdict,
                "Failed tools, refused side effects, scope violations and budget stops appear here",
            )
        } else {
            ScrollView::new()
                .direction(Direction::Vertical)
                .children(findings.into_iter().map(|finding| {
                    rect()
                        .width(Size::fill())
                        .height(Size::px(24.))
                        .horizontal()
                        .cross_align(Alignment::center())
                        .spacing(10.)
                        .child(
                            label()
                                .color(match finding.severity {
                                    run_observability::QualitySeverity::Failure => colors.error,
                                    run_observability::QualitySeverity::Warning => colors.warning,
                                    run_observability::QualitySeverity::Info => {
                                        colors.text_secondary
                                    }
                                })
                                .font_size(9.5)
                                .width(Size::px(58.))
                                .text(finding.severity.label()),
                        )
                        .child(
                            label()
                                .color(colors.text_primary)
                                .font_size(10.5)
                                .max_lines(1)
                                .text(finding.summary),
                        )
                        .into_element()
                }))
                .into_element()
        };

        dock_panel(&colors, &format!("Quality · {verdict}"), body)
    }
}

fn empty_dock_state(colors: &freya::prelude::ColorsSheet, title: &str, hint: &str) -> Element {
    rect()
        .width(Size::fill())
        .height(Size::fill())
        .vertical()
        .main_align(Alignment::center())
        .cross_align(Alignment::center())
        .spacing(4.)
        .child(
            label()
                .color(colors.text_secondary)
                .font_size(11.)
                .text(title.to_string()),
        )
        .child(
            label()
                .color(colors.text_placeholder)
                .font_size(9.5)
                .text(hint.to_string()),
        )
        .into_element()
}

fn dock_panel(colors: &freya::prelude::ColorsSheet, title: &str, body: Element) -> Element {
    rect()
        .width(Size::fill())
        .height(Size::fill())
        .vertical()
        .content(Content::flex())
        .background(colors.background)
        .child(
            rect()
                .width(Size::fill())
                .height(Size::px(26.))
                .horizontal()
                .cross_align(Alignment::center())
                .padding(Gaps::new(0., 8., 0., 8.))
                .child(
                    label()
                        .color(colors.text_secondary)
                        .font_size(10.)
                        .font_weight(FontWeight::MEDIUM)
                        .text(title.to_string()),
                )
                .child(rect().expanded()),
        )
        .child(body)
        .into_element()
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
        let key_terminal = self.terminal.clone();
        let restart_runtime = self.terminal.clone();
        let resize_terminal = self.terminal.clone();
        let a11y_id = use_a11y();
        let focus = use_focus(a11y_id);
        let keyboard_focused = focus() == Focus::Keyboard;

        rect()
            .width(Size::fill())
            .height(Size::fill())
            .vertical()
            .content(Content::flex())
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::px(24.))
                    .horizontal()
                    .main_align(Alignment::space_between())
                    .cross_align(Alignment::center())
                    .padding(Gaps::new(0., 6., 0., 8.))
                    .child(
                        rect()
                            .horizontal()
                            .cross_align(Alignment::center())
                            .spacing(6.)
                            .child(
                                rect()
                                    .width(Size::px(6.))
                                    .height(Size::px(6.))
                                    .corner_radius(CornerRadius::new_all(3.))
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
                                    .text("Terminal"),
                            ),
                    )
                    .child(IconButton {
                        icon: "refresh",
                        label: "Restart terminal",
                        size: 11.,
                        on_press: (move |_: Event<PressEventData>| {
                            restart_terminal(state, restart_runtime.clone());
                        })
                        .into(),
                    }),
            )
            .child(
                ScrollView::new()
                    .direction(Direction::Vertical)
                    .height(Size::flex(1.))
                    .child(
                        rect()
                            .width(Size::fill())
                            .a11y_id(a11y_id)
                            .a11y_focusable(true)
                            .a11y_role(AccessibilityRole::Terminal)
                            .a11y_alt("Integrated terminal. Click to focus, then type directly.")
                            .border(
                                Border::new()
                                    .fill(if keyboard_focused {
                                        colors.border_focus
                                    } else {
                                        Color::TRANSPARENT
                                    })
                                    .width(1.),
                            )
                            .on_mouse_down(move |_| a11y_id.request_focus())
                            .on_key_down(move |event: Event<KeyboardEventData>| {
                                if let Some(bytes) =
                                    terminal_key_bytes(&event.key, &event.modifiers)
                                {
                                    if let Some(terminal) = &key_terminal
                                        && let Err(error) = terminal.write_bytes(&bytes)
                                    {
                                        state.write().show_notice(NoticeTone::Error, error);
                                    }
                                    event.prevent_default();
                                }
                            })
                            .child(
                                label()
                                    .width(Size::fill())
                                    .color(colors.text_primary)
                                    .font_size(11.)
                                    .font_family("Jetbrains Mono")
                                    .text(if output.is_empty() {
                                        if running {
                                            "Starting terminal…".to_string()
                                        } else {
                                            "Shell exited. Restart opens a new shell.".to_string()
                                        }
                                    } else {
                                        output
                                    }),
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
struct FollowToggle {
    state: State<AppState>,
    follow_agent: bool,
    conflict_count: usize,
}

impl Component for FollowToggle {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let following = self.follow_agent;
        let text = match (following, self.conflict_count) {
            (true, 0) => "Following agent edits".to_string(),
            (true, count) => format!("Following · {count} unsaved"),
            (false, _) => "Follow agent off".to_string(),
        };
        let tone = match (following, self.conflict_count) {
            (true, 0) => colors.success,
            (true, _) => colors.warning,
            (false, _) => colors.text_placeholder,
        };

        rect()
            .height(Size::px(22.))
            .padding(Gaps::new(0., 6., 0., 6.))
            .horizontal()
            .spacing(5.)
            .cross_align(Alignment::center())
            .corner_radius(CornerRadius::new_all(4.))
            .background(if following {
                colors.surface_secondary
            } else {
                Color::TRANSPARENT
            })
            .on_press(move |_| {
                let mut app_state = state.write();
                let enabled = app_state.toggle_follow_agent();
                app_state.show_notice(
                    NoticeTone::Info,
                    if enabled {
                        "Following agent edits"
                    } else {
                        "Follow agent off"
                    },
                );
            })
            .child(
                rect()
                    .width(Size::px(6.))
                    .height(Size::px(6.))
                    .corner_radius(CornerRadius::new_all(3.))
                    .background(tone),
            )
            .child(label().color(tone).font_size(10.5).text(text))
    }
}

#[derive(PartialEq)]
struct AgentSessionTab {
    state: State<AppState>,
    title: String,
    status: AgentStatus,
    index: usize,
    is_active: bool,
    can_close: bool,
}

impl Component for AgentSessionTab {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let a11y_id = use_a11y();
        let mut hovered = use_state(|| false);
        let mut state = self.state;
        let index = self.index;
        let is_active = self.is_active;
        let can_close = self.can_close;
        let dot_color = match self.status {
            AgentStatus::Disconnected => colors.error,
            AgentStatus::Idle => colors.text_secondary,
            AgentStatus::Working | AgentStatus::AwaitingApproval => colors.warning,
            AgentStatus::Completed => colors.success,
            AgentStatus::Failed => colors.error,
        };

        rect()
            .min_width(Size::px(60.))
            .max_width(Size::px(120.))
            .height(Size::px(26.))
            .horizontal()
            .cross_align(Alignment::center())
            .spacing(4.)
            .padding(Gaps::new(0., 6., 0., 6.))
            .a11y_id(a11y_id)
            .a11y_focusable(false)
            .a11y_role(AccessibilityRole::Tab)
            .background(if is_active {
                colors.surface_primary
            } else {
                colors.surface_secondary
            })
            .border(
                Border::new()
                    .fill(if is_active {
                        colors.primary
                    } else {
                        colors.border
                    })
                    .width(BorderWidth {
                        top: 0.,
                        right: 1.,
                        bottom: if is_active { 2. } else { 0. },
                        left: 0.,
                    }),
            )
            .on_all_press(move |_| {
                state.write().select_session(index);
            })
            .on_pointer_enter(move |_| hovered.set(true))
            .on_pointer_leave(move |_| hovered.set(false))
            .child(
                rect()
                    .width(Size::px(6.))
                    .height(Size::px(6.))
                    .corner_radius(CornerRadius::new_all(3.))
                    .background(dot_color),
            )
            .child(
                label()
                    .color(if is_active {
                        colors.text_primary
                    } else {
                        colors.text_secondary
                    })
                    .font_size(10.)
                    .max_lines(1)
                    .text(self.title.clone()),
            )
            .child(if can_close && (*hovered.read() || is_active) {
                Element::from(IconButton {
                    icon: "x",
                    label: "Close session",
                    size: 9.,
                    on_press: (move |_: Event<PressEventData>| {
                        state.write().close_session(index);
                    })
                    .into(),
                })
            } else {
                Element::from(label().text(""))
            })
    }
}

#[derive(PartialEq)]
struct AgentMetricsBar {
    metrics: AgentMetrics,
}

impl Component for AgentMetricsBar {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let m = &self.metrics;
        let in_str = AgentMetrics::format_tokens(m.input_tokens);
        let out_str = AgentMetrics::format_tokens(m.output_tokens);
        let cache_str = AgentMetrics::format_tokens(m.cache_read_tokens);
        let hit_rate = m.cache_hit_percent().unwrap_or(0);
        let cost_str = m.format_cost();

        rect()
            .width(Size::fill())
            .height(Size::px(22.))
            .horizontal()
            .cross_align(Alignment::center())
            .main_align(Alignment::space_between())
            .padding(Gaps::new(0., 8., 0., 8.))
            .background(colors.surface_secondary)
            .border(Border::new().fill(colors.border).width(BorderWidth {
                top: 0.,
                right: 0.,
                bottom: 1.,
                left: 0.,
            }))
            .child(
                rect()
                    .horizontal()
                    .spacing(8.)
                    .cross_align(Alignment::center())
                    .child(
                        label()
                            .color(colors.text_secondary)
                            .font_size(9.5)
                            .text(format!("In: {in_str}")),
                    )
                    .child(
                        label()
                            .color(if m.cache_read_tokens > 0 {
                                colors.success
                            } else {
                                colors.text_placeholder
                            })
                            .font_size(9.5)
                            .text(format!("Cache: {cache_str} ({hit_rate}%)")),
                    )
                    .child(
                        label()
                            .color(colors.text_secondary)
                            .font_size(9.5)
                            .text(format!("Out: {out_str}")),
                    ),
            )
            .child(
                rect()
                    .horizontal()
                    .spacing(6.)
                    .cross_align(Alignment::center())
                    .child(
                        label()
                            .color(colors.warning)
                            .font_size(9.5)
                            .font_weight(FontWeight::MEDIUM)
                            .text(cost_str),
                    )
                    .maybe(m.tool_calls > 0, |r| {
                        r.child(
                            label()
                                .color(colors.primary)
                                .font_size(9.5)
                                .text(format!("⚡ {}", m.tool_calls)),
                        )
                    }),
            )
    }
}

struct ModelPreset {
    provider: &'static str,
    model: &'static str,
    name: &'static str,
    desc: &'static str,
    category: &'static str,
    badges: &'static [&'static str],
}

const MODEL_PRESETS: &[ModelPreset] = &[
    // Recommended & Reasoning
    ModelPreset {
        provider: "anthropic",
        model: "claude-3-7-sonnet",
        name: "Claude 3.7 Sonnet",
        desc: "Hybrid reasoning & cutting-edge coding agent",
        category: "reasoning",
        badges: &["Thinking", "Agentic"],
    },
    ModelPreset {
        provider: "anthropic",
        model: "claude-3-5-sonnet",
        name: "Claude 3.5 Sonnet",
        desc: "Most widely used frontier model for code generation",
        category: "reasoning",
        badges: &["Coding"],
    },
    ModelPreset {
        provider: "gemini",
        model: "gemini-2.5-pro",
        name: "Gemini 2.5 Pro",
        desc: "1M+ context window & deep architectural reasoning",
        category: "reasoning",
        badges: &["1M Context", "Thinking"],
    },
    ModelPreset {
        provider: "openai",
        model: "gpt-4o",
        name: "GPT-4o",
        desc: "Frontier multimodal reasoning & high speed",
        category: "reasoning",
        badges: &["Vision", "Fast"],
    },
    ModelPreset {
        provider: "openai",
        model: "o3-mini",
        name: "o3-mini",
        desc: "Deep reasoning model optimized for code and math",
        category: "reasoning",
        badges: &["Reasoning", "STEM"],
    },
    // Fast & Low Latency
    ModelPreset {
        provider: "gemini",
        model: "gemini-2.5-flash",
        name: "Gemini 2.5 Flash",
        desc: "Ultra-fast response & low latency code edits",
        category: "fast",
        badges: &["Ultra-Fast", "Low Cost"],
    },
    ModelPreset {
        provider: "openai",
        model: "gpt-4o-mini",
        name: "GPT-4o mini",
        desc: "Fast, cost-efficient everyday coding assistant",
        category: "fast",
        badges: &["Fast", "Efficient"],
    },
    ModelPreset {
        provider: "anthropic",
        model: "claude-3-5-haiku",
        name: "Claude 3.5 Haiku",
        desc: "Speed-optimized agentic model for rapid execution",
        category: "fast",
        badges: &["Speed"],
    },
    // Local & Offline
    ModelPreset {
        provider: "ollama",
        model: "deepseek-coder:6.7b",
        name: "DeepSeek Coder 6.7B",
        desc: "Local privacy-first coding via Ollama",
        category: "local",
        badges: &["Local", "Offline"],
    },
    ModelPreset {
        provider: "ollama",
        model: "deepseek-r1:8b",
        name: "DeepSeek R1 8B",
        desc: "Open reasoning model running 100% on your machine",
        category: "local",
        badges: &["Local Reasoning"],
    },
    ModelPreset {
        provider: "acf",
        model: "antigravity",
        name: "Antigravity ACF",
        desc: "Google DeepMind Agentic Coding Framework",
        category: "local",
        badges: &["ACF Protocol"],
    },
    ModelPreset {
        provider: "fake",
        model: "test-model",
        name: "Prumo Fake",
        desc: "Offline local simulation for test harness",
        category: "local",
        badges: &["Dev / Mock"],
    },
];

fn friendly_model_name(provider: &str, model: &str) -> String {
    for preset in MODEL_PRESETS {
        if preset.provider == provider && preset.model == model {
            return preset.name.to_string();
        }
    }
    if model.is_empty() {
        provider.to_string()
    } else {
        model.to_string()
    }
}

#[derive(PartialEq)]
struct QuickModelPill {
    state: State<AppState>,
    provider: String,
    model: String,
}

impl Component for QuickModelPill {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let p = &self.provider;
        let m = &self.model;
        let is_open = state.read().agent_model_picker_open;
        let display_name = friendly_model_name(p, m);

        rect()
            .height(Size::px(24.))
            .horizontal()
            .cross_align(Alignment::center())
            .spacing(5.)
            .padding(Gaps::new(0., 8., 0., 8.))
            .corner_radius(CornerRadius::new_all(5.))
            .background(if is_open {
                colors.surface_tertiary
            } else {
                colors.surface_primary
            })
            .border(Border::new().fill(if is_open { colors.primary } else { colors.border }).width(1.))
            .on_press(move |_| {
                let mut app_state = state.write();
                let cur = app_state.agent_model_picker_open;
                app_state.agent_model_picker_open = !cur;
            })
            .child(
                SvgViewer::new(("bot", icon("bot")))
                    .color(colors.primary)
                    .width(Size::px(12.))
                    .height(Size::px(12.)),
            )
            .child(
                label()
                    .color(colors.text_primary)
                    .font_size(10.5)
                    .font_weight(FontWeight::MEDIUM)
                    .max_lines(1)
                    .text(display_name),
            )
            .child(
                SvgViewer::new(("chevron_down", icon("chevron_down")))
                    .color(colors.text_secondary)
                    .width(Size::px(9.))
                    .height(Size::px(9.)),
            )
    }
}

#[derive(PartialEq)]
struct ModelPickerPopover {
    state: State<AppState>,
    current_provider: String,
    current_model: String,
    available_models: Vec<String>,
}

impl Component for ModelPickerPopover {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let cur_provider = self.current_provider.clone();
        let cur_model = self.current_model.clone();
        let available_models = self.available_models.clone();

        let search_query = use_state(String::new);
        let mut selected_tab = use_state(|| "all".to_string());

        let q = search_query.read().to_lowercase().trim().to_string();
        let tab = selected_tab.read().clone();

        let categories: &[(&'static str, &'static str)] = &[
            ("reasoning", "RECOMMENDED & REASONING"),
            ("fast", "FAST & LOW LATENCY"),
            ("local", "LOCAL & OFFLINE"),
        ];

        let tabs = [
            ("all", "All"),
            ("anthropic", "Anthropic"),
            ("openai", "OpenAI"),
            ("gemini", "Google"),
            ("local", "Local"),
        ];

        let mut any_found = false;

        let category_blocks: Vec<Element> = categories
            .iter()
            .filter_map(|(cat_id, cat_title)| {
                let cat_models: Vec<&ModelPreset> = MODEL_PRESETS
                    .iter()
                    .filter(|p| p.category == *cat_id)
                    .filter(|p| {
                        let tab_ok = match tab.as_str() {
                            "all" => true,
                            "anthropic" => p.provider == "anthropic",
                            "openai" => p.provider == "openai",
                            "gemini" => p.provider == "gemini",
                            "local" => p.category == "local",
                            _ => true,
                        };
                        let q_ok = if q.is_empty() {
                            true
                        } else {
                            p.name.to_lowercase().contains(&q)
                                || p.model.to_lowercase().contains(&q)
                                || p.provider.to_lowercase().contains(&q)
                                || p.desc.to_lowercase().contains(&q)
                                || p.badges.iter().any(|b| b.to_lowercase().contains(&q))
                        };
                        tab_ok && q_ok
                    })
                    .collect();

                if cat_models.is_empty() {
                    return None;
                }

                any_found = true;

                let rows: Vec<Element> = cat_models
                    .into_iter()
                    .map(|preset| {
                        let is_selected = cur_provider == preset.provider && cur_model == preset.model;
                        let p = preset.provider.to_string();
                        let m = preset.model.to_string();
                        let name = preset.name.to_string();
                        let d = preset.desc.to_string();
                        let badges = preset.badges;

                        rect()
                            .width(Size::fill())
                            .padding(Gaps::new(6., 8., 6., 8.))
                            .corner_radius(CornerRadius::new_all(5.))
                            .background(if is_selected {
                                colors.surface_secondary
                            } else {
                                Color::TRANSPARENT
                            })
                            .border(Border::new().fill(if is_selected { colors.primary } else { Color::TRANSPARENT }).width(1.))
                            .horizontal()
                            .main_align(Alignment::space_between())
                            .cross_align(Alignment::center())
                            .on_press(move |_| {
                                let mut app_state = state.write();
                                app_state.config.provider = p.clone();
                                app_state.config.model = m.clone();
                                let idx = app_state.active_session_index;
                                if let Some(session) = app_state.agent_sessions.get_mut(idx) {
                                    session.provider = p.clone();
                                    session.model = m.clone();
                                }
                                app_state.agent_model_picker_open = false;
                                app_state.show_notice(NoticeTone::Success, format!("Switched to {name} ({p}/{m})"));
                            })
                            .child(
                                rect()
                                    .vertical()
                                    .spacing(3.)
                                    .child(
                                        rect()
                                            .horizontal()
                                            .spacing(6.)
                                            .cross_align(Alignment::center())
                                            .child(
                                                label()
                                                    .color(if is_selected { colors.primary } else { colors.text_primary })
                                                    .font_size(11.)
                                                    .font_weight(FontWeight::SEMI_BOLD)
                                                    .text(preset.name),
                                            )
                                            .child(
                                                rect()
                                                    .padding(Gaps::new(1., 4., 1., 4.))
                                                    .corner_radius(CornerRadius::new_all(3.))
                                                    .background(colors.surface_secondary)
                                                    .child(
                                                        label()
                                                            .color(colors.text_secondary)
                                                            .font_size(8.5)
                                                            .text(preset.provider),
                                                    ),
                                            )
                                            .children(badges.iter().map(|b| {
                                                rect()
                                                    .padding(Gaps::new(1., 4., 1., 4.))
                                                    .corner_radius(CornerRadius::new_all(3.))
                                                    .background(colors.surface_tertiary)
                                                    .child(
                                                        label()
                                                            .color(colors.primary)
                                                            .font_size(8.5)
                                                            .font_weight(FontWeight::MEDIUM)
                                                            .text(*b),
                                                    )
                                                    .into()
                                            })),
                                    )
                                    .child(
                                        label()
                                            .color(colors.text_placeholder)
                                            .font_size(9.5)
                                            .text(d),
                                    ),
                            )
                            .child(if is_selected {
                                Element::from(
                                    SvgViewer::new(("check", icon("check")))
                                        .color(colors.primary)
                                        .width(Size::px(13.))
                                        .height(Size::px(13.)),
                                )
                            } else {
                                Element::from(label().text(""))
                            })
                            .into()
                    })
                    .collect();

                Some(
                    rect()
                        .width(Size::fill())
                        .vertical()
                        .spacing(2.)
                        .child(
                            rect()
                                .width(Size::fill())
                                .padding(Gaps::new(5., 6., 2., 6.))
                                .child(
                                    label()
                                        .color(colors.text_placeholder)
                                        .font_size(9.)
                                        .font_weight(FontWeight::SEMI_BOLD)
                                        .text(*cat_title),
                                ),
                        )
                        .children(rows)
                        .into(),
                )
            })
            .collect();

        // Custom models from workspace if not in presets
        let extra_models: Vec<String> = available_models
            .iter()
            .filter(|m| !MODEL_PRESETS.iter().any(|p| p.model == *m || &format!("{}/{}", p.provider, p.model) == *m))
            .filter(|m| q.is_empty() || m.to_lowercase().contains(&q))
            .cloned()
            .collect();

        let custom_block: Option<Element> = if !extra_models.is_empty() && (tab == "all" || tab == "local") {
            any_found = true;
            let rows: Vec<Element> = extra_models
                .into_iter()
                .map(|m| {
                    let is_selected = cur_model == m;
                    let m_clone = m.clone();
                    rect()
                        .width(Size::fill())
                        .padding(Gaps::new(6., 8., 6., 8.))
                        .corner_radius(CornerRadius::new_all(5.))
                        .background(if is_selected {
                            colors.surface_secondary
                        } else {
                            Color::TRANSPARENT
                        })
                        .horizontal()
                        .main_align(Alignment::space_between())
                        .cross_align(Alignment::center())
                        .on_press(move |_| {
                            let mut app_state = state.write();
                            app_state.config.model = m_clone.clone();
                            let idx = app_state.active_session_index;
                            if let Some(session) = app_state.agent_sessions.get_mut(idx) {
                                session.model = m_clone.clone();
                            }
                            app_state.agent_model_picker_open = false;
                        })
                        .child(
                            label()
                                .color(if is_selected { colors.primary } else { colors.text_primary })
                                .font_size(11.)
                                .text(m),
                        )
                        .child(if is_selected {
                            Element::from(
                                SvgViewer::new(("check", icon("check")))
                                    .color(colors.primary)
                                    .width(Size::px(13.))
                                    .height(Size::px(13.)),
                            )
                        } else {
                            Element::from(label().text(""))
                        })
                        .into()
                })
                .collect();

            Some(
                rect()
                    .width(Size::fill())
                    .vertical()
                    .spacing(2.)
                    .child(
                        rect()
                            .width(Size::fill())
                            .padding(Gaps::new(5., 6., 2., 6.))
                            .child(
                                label()
                                    .color(colors.text_placeholder)
                                    .font_size(9.)
                                    .font_weight(FontWeight::SEMI_BOLD)
                                    .text("CUSTOM / WORKSPACE MODELS"),
                            ),
                    )
                    .children(rows)
                    .into(),
            )
        } else {
            None
        };

        rect()
            .width(Size::fill())
            .vertical()
            .spacing(6.)
            .padding(Gaps::new_all(8.))
            .background(colors.surface_primary)
            .corner_radius(CornerRadius::new_all(8.))
            .border(Border::new().fill(colors.primary).width(1.))
            .child(
                rect()
                    .width(Size::fill())
                    .horizontal()
                    .main_align(Alignment::space_between())
                    .cross_align(Alignment::center())
                    .child(
                        label()
                            .color(colors.text_primary)
                            .font_size(11.)
                            .font_weight(FontWeight::SEMI_BOLD)
                            .text("Select Model & Provider"),
                    )
                    .child(
                        IconButton {
                            icon: "x",
                            label: "Close picker",
                            size: 11.,
                            on_press: (move |_: Event<PressEventData>| {
                                state.write().agent_model_picker_open = false;
                            })
                            .into(),
                        }
                    ),
            )
            .child(
                rect()
                    .width(Size::fill())
                    .height(Size::px(30.))
                    .horizontal()
                    .cross_align(Alignment::center())
                    .spacing(6.)
                    .padding(Gaps::new(0., 8., 0., 8.))
                    .background(colors.surface_secondary)
                    .corner_radius(CornerRadius::new_all(5.))
                    .border(Border::new().fill(colors.border).width(1.))
                    .child(
                        SvgViewer::new(("search", icon("search")))
                            .color(colors.text_placeholder)
                            .width(Size::px(12.))
                            .height(Size::px(12.)),
                    )
                    .child(
                        rect()
                            .width(Size::flex(1.))
                            .font_size(11.)
                            .child(
                                Input::new(search_query.into_writable())
                                    .placeholder("Search models or providers…")
                                    .width(Size::fill())
                                    .flat()
                                    .compact()
                            ),
                    ),
            )
            .child(
                rect()
                    .width(Size::fill())
                    .horizontal()
                    .spacing(4.)
                    .children(tabs.iter().map(|(id, title)| {
                        let is_tab_active = *selected_tab.read() == *id;
                        let tab_id = id.to_string();
                        rect()
                            .padding(Gaps::new(3., 7., 3., 7.))
                            .corner_radius(CornerRadius::new_all(4.))
                            .background(if is_tab_active {
                                colors.primary
                            } else {
                                colors.surface_secondary
                            })
                            .on_press(move |_| {
                                selected_tab.set(tab_id.clone());
                            })
                            .child(
                                label()
                                    .color(if is_tab_active {
                                        colors.text_primary
                                    } else {
                                        colors.text_secondary
                                    })
                                    .font_size(10.)
                                    .font_weight(if is_tab_active { FontWeight::SEMI_BOLD } else { FontWeight::NORMAL })
                                    .text(*title),
                            )
                            .into()
                    })),
            )
            .child(
                ScrollView::new()
                    .direction(Direction::Vertical)
                    .height(Size::px(210.))
                    .children(if any_found {
                        let mut all = category_blocks;
                        if let Some(c) = custom_block {
                            all.push(c);
                        }
                        all
                    } else {
                        vec![
                            rect()
                                .width(Size::fill())
                                .height(Size::px(60.))
                                .center()
                                .child(
                                    label()
                                        .color(colors.text_placeholder)
                                        .font_size(10.5)
                                        .text("No models match your search"),
                                )
                                .into()
                        ]
                    }),
            )
            .child(
                rect()
                    .width(Size::fill())
                    .padding(Gaps::new(6., 4., 2., 4.))
                    .border(Border::new().fill(colors.border).width(BorderWidth {
                        top: 1.,
                        right: 0.,
                        bottom: 0.,
                        left: 0.,
                    }))
                    .horizontal()
                    .main_align(Alignment::space_between())
                    .cross_align(Alignment::center())
                    .child(
                        rect()
                            .horizontal()
                            .spacing(5.)
                            .cross_align(Alignment::center())
                            .on_press(move |_| {
                                let mut app_state = state.write();
                                app_state.config_open = true;
                                app_state.agent_model_picker_open = false;
                            })
                            .child(
                                SvgViewer::new(("settings", icon("settings")))
                                    .color(colors.text_secondary)
                                    .width(Size::px(11.))
                                    .height(Size::px(11.)),
                            )
                            .child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(10.)
                                    .text("Configure API Keys & Custom Models…"),
                            ),
                    )
                    .child(
                        label()
                            .color(colors.text_placeholder)
                            .font_size(9.5)
                            .text("Esc to close"),
                    ),
            )
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FormattedBlock {
    Paragraph(String),
    CodeBlock { language: String, code: String },
}

pub fn parse_markdown_blocks(text: &str) -> Vec<FormattedBlock> {
    let mut blocks = Vec::new();
    let mut in_code = false;
    let mut current_lang = String::new();
    let mut current_code = String::new();
    let mut current_paragraph = String::new();

    for line in text.lines() {
        let trimmed = line.trim();
        if trimmed.starts_with("```") {
            if in_code {
                blocks.push(FormattedBlock::CodeBlock {
                    language: if current_lang.is_empty() {
                        "code".to_string()
                    } else {
                        current_lang.clone()
                    },
                    code: current_code.trim_end().to_string(),
                });
                current_code.clear();
                current_lang.clear();
                in_code = false;
            } else {
                if !current_paragraph.trim().is_empty() {
                    blocks.push(FormattedBlock::Paragraph(current_paragraph.trim().to_string()));
                    current_paragraph.clear();
                }
                current_lang = trimmed.trim_start_matches('`').trim().to_string();
                in_code = true;
            }
        } else if in_code {
            current_code.push_str(line);
            current_code.push('\n');
        } else {
            if !current_paragraph.is_empty() {
                current_paragraph.push('\n');
            }
            current_paragraph.push_str(line);
        }
    }

    if in_code && !current_code.is_empty() {
        blocks.push(FormattedBlock::CodeBlock {
            language: if current_lang.is_empty() {
                "code".to_string()
            } else {
                current_lang
            },
            code: current_code.trim_end().to_string(),
        });
    } else if !current_paragraph.trim().is_empty() {
        blocks.push(FormattedBlock::Paragraph(current_paragraph.trim().to_string()));
    }

    if blocks.is_empty() && !text.trim().is_empty() {
        blocks.push(FormattedBlock::Paragraph(text.trim().to_string()));
    }

    blocks
}

#[derive(PartialEq)]
struct ChatMessageView {
    state: State<AppState>,
    message: ChatMessage,
    runner_model: String,
}

impl Component for ChatMessageView {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let is_user = self.message.role == ChatMessageRole::User;
        let is_system = self.message.role == ChatMessageRole::System;
        let is_streaming = self.message.is_streaming;
        let blocks = parse_markdown_blocks(&self.message.content);

        rect()
            .width(Size::fill())
            .vertical()
            .spacing(5.)
            .padding(Gaps::new_all(8.))
            .background(if is_user {
                colors.surface_secondary
            } else if is_system {
                colors.surface_tertiary
            } else {
                colors.surface_primary
            })
            .corner_radius(CornerRadius::new_all(6.))
            .border(
                Border::new()
                    .fill(if is_user {
                        colors.border
                    } else if is_system {
                        colors.warning
                    } else {
                        colors.border_focus
                    })
                    .width(1.),
            )
            .child(
                rect()
                    .width(Size::fill())
                    .horizontal()
                    .main_align(Alignment::space_between())
                    .cross_align(Alignment::center())
                    .child(
                        rect()
                            .horizontal()
                            .spacing(6.)
                            .cross_align(Alignment::center())
                            .child(
                                label()
                                    .color(if is_user {
                                        colors.primary
                                    } else if is_system {
                                        colors.warning
                                    } else {
                                        colors.success
                                    })
                                    .font_size(10.5)
                                    .font_weight(FontWeight::BOLD)
                                    .text(if is_user {
                                        "👤 You"
                                    } else if is_system {
                                        "⚙️ System"
                                    } else {
                                        "🤖 Agent"
                                    }),
                            )
                            .maybe(!is_user && !self.runner_model.is_empty(), |r| {
                                r.child(
                                    label()
                                        .color(colors.text_placeholder)
                                        .font_size(9.5)
                                        .text(format!("({})", self.runner_model)),
                                )
                            }),
                    )
                    .child(
                        rect()
                            .horizontal()
                            .spacing(6.)
                            .cross_align(Alignment::center())
                            .maybe(is_streaming, |r| {
                                r.child(
                                    rect()
                                        .padding(Gaps::new(1., 4., 1., 4.))
                                        .corner_radius(CornerRadius::new_all(3.))
                                        .background(colors.warning)
                                        .child(
                                            label()
                                                .color(colors.background)
                                                .font_size(8.5)
                                                .font_weight(FontWeight::BOLD)
                                                .text("STREAMING"),
                                        ),
                                )
                            })
                            .child(
                                label()
                                    .color(colors.text_placeholder)
                                    .font_size(9.)
                                    .text(self.message.timestamp.clone()),
                            ),
                    ),
            )
            .maybe(
                self.message
                    .reasoning
                    .as_ref()
                    .is_some_and(|r| !r.is_empty()),
                |r| {
                    let reasoning_text = self.message.reasoning.clone().unwrap_or_default();
                    r.child(
                        rect()
                            .width(Size::fill())
                            .padding(Gaps::new(4., 6., 4., 6.))
                            .background(colors.surface_tertiary)
                            .corner_radius(CornerRadius::new_all(4.))
                            .vertical()
                            .spacing(2.)
                            .child(
                                label()
                                    .color(colors.warning)
                                    .font_size(9.)
                                    .font_weight(FontWeight::SEMI_BOLD)
                                    .text("💭 Thinking Process"),
                            )
                            .child(
                                label()
                                    .color(colors.text_secondary)
                                    .font_size(9.5)
                                    .font_family("Jetbrains Mono")
                                    .text(reasoning_text),
                            ),
                    )
                },
            )
            .child(
                rect()
                    .width(Size::fill())
                    .vertical()
                    .spacing(6.)
                    .children(if blocks.is_empty() && is_streaming {
                        vec![rect()
                            .padding(Gaps::new(4., 0., 4., 0.))
                            .child(
                                label()
                                    .color(colors.text_placeholder)
                                    .font_size(10.5)
                                    .text("Thinking…"),
                            )
                            .into()]
                    } else {
                        blocks
                            .into_iter()
                            .map(|block| match block {
                                FormattedBlock::Paragraph(text) => rect()
                                    .width(Size::fill())
                                    .child(
                                        label()
                                            .width(Size::fill())
                                            .color(colors.text_primary)
                                            .font_size(11.)
                                            .text(text),
                                    )
                                    .into(),
                                FormattedBlock::CodeBlock { language, code } => {
                                    let copy_snippet = code.clone();
                                    rect()
                                        .width(Size::fill())
                                        .vertical()
                                        .background(colors.background)
                                        .border(Border::new().fill(colors.border).width(1.))
                                        .corner_radius(CornerRadius::new_all(4.))
                                        .child(
                                            rect()
                                                .width(Size::fill())
                                                .height(Size::px(22.))
                                                .horizontal()
                                                .main_align(Alignment::space_between())
                                                .cross_align(Alignment::center())
                                                .padding(Gaps::new(0., 8., 0., 8.))
                                                .background(colors.surface_secondary)
                                                .border(Border::new().fill(colors.border).width(
                                                    BorderWidth {
                                                        top: 0.,
                                                        right: 0.,
                                                        bottom: 1.,
                                                        left: 0.,
                                                    },
                                                ))
                                                .child(
                                                    label()
                                                        .color(colors.text_secondary)
                                                        .font_size(9.)
                                                        .font_weight(FontWeight::BOLD)
                                                        .text(language.to_uppercase()),
                                                )
                                                .child(
                                                    Button::new()
                                                        .flat()
                                                        .compact()
                                                        .height(Size::px(18.))
                                                        .padding(Gaps::new(0., 6., 0., 6.))
                                                        .on_press(move |_| {
                                                            let mut app_state = state.write();
                                                            app_state.show_notice(
                                                                NoticeTone::Success,
                                                                format!(
                                                                    "Code snippet copied ({} lines)",
                                                                    copy_snippet.lines().count()
                                                                ),
                                                            );
                                                        })
                                                        .child(
                                                            label()
                                                                .color(colors.primary)
                                                                .font_size(9.)
                                                                .text("Copy"),
                                                        ),
                                                ),
                                        )
                                        .child(
                                            rect()
                                                .width(Size::fill())
                                                .padding(Gaps::new_all(6.))
                                                .child(
                                                    label()
                                                        .width(Size::fill())
                                                        .color(colors.text_primary)
                                                        .font_size(10.5)
                                                        .font_family("Jetbrains Mono")
                                                        .text(code),
                                                ),
                                        )
                                        .into()
                                }
                            })
                            .collect::<Vec<Element>>()
                    }),
            )
    }
}

#[derive(PartialEq)]
struct AgentEventCard {
    time: String,
    kind: String,
    message: String,
}

impl Component for AgentEventCard {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let is_tool = self.kind == "tool_call_ready" || self.kind == "tool.call" || self.kind.starts_with("tool");
        let is_reasoning = self.kind == "reasoning_delta" || self.kind.contains("reasoning");
        let is_file = self.kind == "file.changed" || self.kind == "file.patch" || self.message.contains("file");
        let is_run = self.kind == "run.started" || self.kind == "run.finished";
        let is_usage = self.kind == "usage" || self.kind.starts_with("budget");

        let border_color = if is_tool {
            colors.primary
        } else if is_reasoning {
            colors.warning
        } else if is_file {
            colors.success
        } else {
            colors.border
        };

        let badge_text = if is_tool {
            "TOOL"
        } else if is_reasoning {
            "THINKING"
        } else if is_file {
            "EDIT"
        } else if is_run {
            "RUN"
        } else if is_usage {
            "USAGE"
        } else {
            ""
        };

        rect()
            .width(Size::fill())
            .padding(Gaps::new_all(5.))
            .background(if is_tool || is_reasoning || is_run {
                colors.surface_secondary
            } else {
                Color::TRANSPARENT
            })
            .corner_radius(CornerRadius::new_all(4.))
            .border(Border::new().fill(border_color).width(BorderWidth {
                top: 0.,
                right: 0.,
                bottom: 0.,
                left: if is_tool || is_reasoning || is_file { 2. } else { 0. },
            }))
            .vertical()
            .spacing(2.)
            .child(
                rect()
                    .width(Size::fill())
                    .horizontal()
                    .main_align(Alignment::space_between())
                    .cross_align(Alignment::center())
                    .child(
                        rect()
                            .horizontal()
                            .spacing(5.)
                            .cross_align(Alignment::center())
                            .maybe(!badge_text.is_empty(), |r| {
                                r.child(
                                    rect()
                                        .padding(Gaps::new(1., 3., 1., 3.))
                                        .corner_radius(CornerRadius::new_all(2.))
                                        .background(border_color)
                                        .child(
                                            label()
                                                .color(colors.background)
                                                .font_size(8.)
                                                .font_weight(FontWeight::BOLD)
                                                .text(badge_text),
                                        ),
                                )
                            })
                            .child(
                                label()
                                    .color(colors.text_placeholder)
                                    .font_size(9.)
                                    .text(self.time.clone()),
                            ),
                    ),
            )
            .child(
                label()
                    .color(if is_reasoning { colors.text_secondary } else { colors.text_primary })
                    .font_size(10.)
                    .text(self.message.clone()),
            )
    }
}

#[derive(PartialEq)]
struct ApprovalCard {
    state: State<AppState>,
    client: PrumoClient,
    request_id: String,
    tool: String,
    arguments_preview: String,
    fingerprint: String,
}

impl Component for ApprovalCard {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let state = self.state;
        let client = self.client.clone();
        let request_id = self.request_id.clone();
        let tool = self.tool.clone();
        let arguments = self.arguments_preview.clone();
        let fingerprint = self.fingerprint.clone();
        let run_id = state.read().active_run_id.clone().unwrap_or_default();

        rect()
            .width(Size::fill())
            .vertical()
            .padding(Gaps::new_all(8.))
            .spacing(5.)
            .background(colors.surface_secondary)
            .corner_radius(CornerRadius::new_all(5.))
            .border(Border::new().fill(colors.warning).width(BorderWidth {
                top: 0.,
                right: 0.,
                bottom: 0.,
                left: 2.,
            }))
            .child(
                rect()
                    .width(Size::fill())
                    .horizontal()
                    .main_align(Alignment::space_between())
                    .cross_align(Alignment::center())
                    .child(
                        label()
                            .color(colors.warning)
                            .font_size(10.5)
                            .font_weight(FontWeight::MEDIUM)
                            .text(format!("Approval: {}", if tool.is_empty() { "tool" } else { &tool })),
                    )
                    .child(
                        label()
                            .color(colors.text_placeholder)
                            .font_size(9.)
                            .text(request_id.clone()),
                    ),
            )
            .maybe(!arguments.is_empty(), |rect| {
                rect.child(
                    label()
                        .color(colors.text_primary)
                        .font_size(10.)
                        .max_lines(2)
                        .text(arguments.clone()),
                )
            })
            .child(
                rect()
                    .width(Size::fill())
                    .horizontal()
                    .main_align(Alignment::end())
                    .spacing(6.)
                    .child(
                        Button::new()
                            .compact()
                            .on_press({
                                let client = client.clone();
                                let run_id = run_id.clone();
                                let request_id = request_id.clone();
                                let fingerprint = fingerprint.clone();
                                move |_| {
                                    answer_permission(
                                        state,
                                        client.clone(),
                                        &run_id,
                                        &request_id,
                                        Some(fingerprint.clone()),
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
                            .compact()
                            .on_press({
                                let client = client.clone();
                                let run_id = run_id.clone();
                                let request_id = request_id.clone();
                                let fingerprint = fingerprint.clone();
                                move |_| {
                                    answer_permission(
                                        state,
                                        client.clone(),
                                        &run_id,
                                        &request_id,
                                        Some(fingerprint.clone()),
                                        false,
                                    )
                                }
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
    {
        let mut app_state = state.write();
        let idx = app_state.active_session_index;
        if let Some(session) = app_state.agent_sessions.get_mut(idx)
            && (session.title.starts_with("Chat ") || session.title.starts_with("Session ")) {
                session.title = if goal.chars().count() > 22 {
                    format!("{}…", goal.chars().take(22).collect::<String>())
                } else {
                    goal.clone()
                };
            }
        app_state.add_user_message(&goal);
        app_state.start_assistant_streaming_message();
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
                config.api_key.as_deref().or(config.acf_token.as_deref()),
                config.base_url.as_deref(),
            )
        })
        .await;
        match result {
            Ok(Ok(run_id)) => {
                let mut app_state = state.write();
                app_state.track_run(Some(run_id.clone()));
                app_state.agent_status = AgentStatus::Working;
                app_state.show_notice(NoticeTone::Success, format!("Run {run_id} started"));
            }
            Ok(Err(error)) => {
                let mut app_state = state.write();
                app_state.finish_assistant_streaming();
                app_state.show_notice(NoticeTone::Error, format!("Could not start run: {error}"));
            }
            Err(error) => {
                let mut app_state = state.write();
                app_state.finish_assistant_streaming();
                app_state.show_notice(NoticeTone::Error, format!("Agent worker failed: {error}"));
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
            Ok(Ok(())) => {
                let mut app_state = state.write();
                app_state.finish_assistant_streaming();
                app_state.show_notice(NoticeTone::Success, format!("Run {run_id} cancelled"));
            }
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
    {
        let mut app_state = state.write();
        let steer_msg = format!("⚡ {}", message);
        app_state.add_user_message(&steer_msg);
        app_state.start_assistant_streaming_message();
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
    fingerprint: Option<String>,
    approved: bool,
) {
    let run_id = run_id.to_string();
    let request_id = request_id.to_string();
    spawn(async move {
        let worker_client = client.clone();
        let worker_run_id = run_id.clone();
        let worker_request_id = request_id.clone();
        let worker_fingerprint = fingerprint.clone();
        let result = tokio::task::spawn_blocking(move || {
            if approved {
                worker_client.approve(
                    &worker_run_id,
                    &worker_request_id,
                    worker_fingerprint.as_deref(),
                )
            } else {
                worker_client.deny(
                    &worker_run_id,
                    &worker_request_id,
                    worker_fingerprint.as_deref(),
                )
            }
        })
        .await;
        match result {
            Ok(Ok(())) => {
                let mut app_state = state.write();
                app_state
                    .pending_permissions
                    .retain(|pending| pending != &request_id);
                app_state
                    .pending_permission_details
                    .retain(|p| p.request_id != request_id);
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_friendly_model_name_mapping() {
        assert_eq!(
            friendly_model_name("anthropic", "claude-3-7-sonnet"),
            "Claude 3.7 Sonnet"
        );
        assert_eq!(
            friendly_model_name("gemini", "gemini-2.5-pro"),
            "Gemini 2.5 Pro"
        );
        assert_eq!(
            friendly_model_name("openai", "gpt-4o"),
            "GPT-4o"
        );
        assert_eq!(
            friendly_model_name("ollama", "deepseek-coder:6.7b"),
            "DeepSeek Coder 6.7B"
        );
        assert_eq!(
            friendly_model_name("custom", "my-custom-model"),
            "my-custom-model"
        );
    }

    #[test]
    fn test_model_presets_integrity() {
        assert!(!MODEL_PRESETS.is_empty());
        for preset in MODEL_PRESETS {
            assert!(!preset.provider.is_empty());
            assert!(!preset.model.is_empty());
            assert!(!preset.name.is_empty());
            assert!(!preset.desc.is_empty());
            assert!(
                matches!(preset.category, "reasoning" | "fast" | "local"),
                "unknown category: {}",
                preset.category
            );
            assert!(!preset.badges.is_empty());
        }
    }
}
