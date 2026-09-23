use crate::client::protocol::PrumoClient;
use crate::services::config::{ViewerConfig, config_path};
use crate::state::{AppState, NoticeTone};
use crate::theme;
use freya::prelude::*;

#[derive(PartialEq)]
pub struct SettingsModal {
    pub state: State<AppState>,
    pub client: PrumoClient,
}

impl Component for SettingsModal {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let current_theme = use_theme();
        let mut state = self.state;
        let client = self.client.clone();
        let config = state.read().config.clone();
        let provider_state = use_state(|| config.provider.clone());
        let model_state = use_state(|| config.model.clone());
        let socket_state = use_state(|| config.socket_path.clone().unwrap_or_default());
        let shell_state = use_state(|| config.terminal_shell.clone().unwrap_or_default());
        let poll_state = use_state(|| config.poll_interval_seconds.to_string());
        let mut whitespace_state = use_state(|| config.show_whitespace);
        let selected_theme = config.theme.clone();
        let provider = provider_state.read().trim().to_string();
        let available_models = state.read().available_models.clone();
        let path_label = config_path()
            .map(|path| path.display().to_string())
            .unwrap_or_else(|error| format!("Config unavailable: {error}"));

        rect()
            .width(Size::fill())
            .height(Size::fill())
            .position(Position::new_global().top(0.).left(0.))
            .layer(Layer::Overlay)
            .background(Color::from_argb(170, 0, 0, 0))
            .horizontal()
            .main_align(Alignment::center())
            .cross_align(Alignment::center())
            .on_press(move |_| state.write().config_open = false)
            .child(
                rect()
                    .width(Size::px(620.))
                    .max_height(Size::px(760.))
                    .background(colors.surface_primary)
                    .border(Border::new().fill(colors.border_focus).width(1.))
                    .corner_radius(CornerRadius::new_all(8.))
                    .vertical()
                    .content(Content::flex())
                    .on_mouse_up(|event: Event<MouseEventData>| {
                        event.stop_propagation();
                    })
                    .child(
                        rect()
                            .width(Size::fill())
                            .height(Size::px(48.))
                            .horizontal()
                            .main_align(Alignment::space_between())
                            .cross_align(Alignment::center())
                            .padding(Gaps::new(16., 0., 10., 0.))
                            .border(Border::new().fill(colors.border).width(BorderWidth {
                                top: 0.,
                                right: 0.,
                                bottom: 1.,
                                left: 0.,
                            }))
                            .child(
                                label()
                                    .color(colors.text_primary)
                                    .font_size(15.)
                                    .font_weight(FontWeight::BOLD)
                                    .text("Prumo Viewer Settings"),
                            )
                            .child(
                                Button::new()
                                    .flat()
                                    .compact()
                                    .width(Size::px(70.))
                                    .height(Size::px(28.))
                                    .padding(0.)
                                    .on_press(move |_| state.write().config_open = false)
                                    .child("Close"),
                            ),
                    )
                    .child(
                        ScrollView::new()
                            .direction(torin::prelude::Direction::Vertical)
                            .height(Size::flex(1.))
                            .child(
                                rect()
                                    .width(Size::fill())
                                    .vertical()
                                    .padding(Gaps::new_all(16.))
                                    .spacing(14.)
                                    .child(
                                        label()
                                            .color(colors.text_secondary)
                                            .font_size(11.)
                                            .font_weight(FontWeight::BOLD)
                                            .text("APPEARANCE"),
                                    )
                                    .child(
                                        rect()
                                            .width(Size::fill())
                                            .horizontal()
                                            .spacing(8.)
                                            .child(ThemeButton {
                                                state,
                                                current_theme,
                                                label: "System",
                                                value: "system",
                                                selected: selected_theme == "system",
                                            })
                                            .child(ThemeButton {
                                                state,
                                                current_theme,
                                                label: "Light",
                                                value: "light",
                                                selected: selected_theme == "light",
                                            })
                                            .child(ThemeButton {
                                                state,
                                                current_theme,
                                                label: "Dark",
                                                value: "dark",
                                                selected: selected_theme == "dark",
                                            }),
                                    )
                                    .child(
                                        rect()
                                            .width(Size::fill())
                                            .horizontal()
                                            .main_align(Alignment::space_between())
                                            .cross_align(Alignment::center())
                                            .child(
                                                label()
                                                    .color(colors.text_primary)
                                                    .font_size(12.)
                                                    .text("Show whitespace"),
                                            )
                                            .child(
                                                Button::new()
                                                    .flat()
                                                    .compact()
                                                    .height(Size::px(28.))
                                                    .padding(Gaps::new(10., 0., 10., 0.))
                                                    .on_press(move |_| {
                                                        whitespace_state
                                                            .set(!*whitespace_state.read())
                                                    })
                                                    .child(
                                                        label()
                                                            .color(colors.text_primary)
                                                            .font_size(11.)
                                                            .text(if *whitespace_state.read() {
                                                                "ON"
                                                            } else {
                                                                "OFF"
                                                            }),
                                                    ),
                                            ),
                                    )
                                    .child(
                                        label()
                                            .color(colors.text_secondary)
                                            .font_size(11.)
                                            .font_weight(FontWeight::BOLD)
                                            .text("AGENT RUNNER"),
                                    )
                                    .child(
                                        rect()
                                            .width(Size::fill())
                                            .vertical()
                                            .spacing(6.)
                                            .child(
                                                label()
                                                    .color(colors.text_secondary)
                                                    .font_size(11.)
                                                    .text("Provider"),
                                            )
                                            .child(
                                                Input::new(provider_state.into_writable())
                                                    .placeholder("fake, openai-compat, anthropic, opencode")
                                                    .width(Size::fill())
                                                    .filled(),
                                            )
                                            .child(
                                                rect()
                                                    .width(Size::fill())
                                                    .horizontal()
                                                    .main_align(Alignment::end())
                                                    .child(
                                                        Button::new()
                                                            .flat()
                                                            .compact()
                                                            .height(Size::px(30.))
                                                            .padding(Gaps::new(10., 0., 10., 0.))
                                                            .on_press({
                                                                let client = client.clone();
                                                                let state = state;
                                                                let provider = provider.clone();
                                                                move |_| {
                                                                    load_models(
                                                                        state,
                                                                        client.clone(),
                                                                        provider.clone(),
                                                                    )
                                                                }
                                                            })
                                                            .child(
                                                                label()
                                                                    .color(colors.primary)
                                                                    .font_size(11.)
                                                                    .text("Load models"),
                                                            ),
                                                    ),
                                            ),
                                    )
                                    .child(
                                        rect()
                                            .width(Size::fill())
                                            .vertical()
                                            .spacing(6.)
                                            .child(
                                                label()
                                                    .color(colors.text_secondary)
                                                    .font_size(11.)
                                                    .text("Model"),
                                            )
                                            .child(
                                                Input::new(model_state.into_writable())
                                                    .placeholder("Provider default")
                                                    .width(Size::fill())
                                                    .filled(),
                                            )
                                            .maybe(!available_models.is_empty(), |rect| {
                                                rect.child(
                                                    ScrollView::new()
                                                        .direction(torin::prelude::Direction::Vertical)
                                                        .height(Size::px(96.))
                                                        .children(available_models.into_iter().map(
                                                            |model| {
                                                                Button::new()
                                                                    .flat()
                                                                    .expanded()
                                                                    .height(Size::px(26.))
                                                                    .padding(Gaps::new(8., 0., 8., 0.))
                                                                    .on_press({
                                                                        let mut model_state =
                                                                            model_state;
                                                                        let mut state = state;
                                                                        let selected =
                                                                            model.clone();
                                                                        move |_| {
                                                                            model_state.set(
                                                                                selected.clone(),
                                                                            );
                                                                            state.write().config.model =
                                                                                selected.clone();
                                                                        }
                                                                    })
                                                                    .child(
                                                                        label()
                                                                            .color(colors.text_primary)
                                                                            .font_size(11.)
                                                                            .text(model),
                                                                    )
                                                                    .into_element()
                                                            },
                                                        )),
                                                )
                                            }),
                                    )
                                    .child(
                                        label()
                                            .color(colors.text_secondary)
                                            .font_size(11.)
                                            .font_weight(FontWeight::BOLD)
                                            .text("CONNECTION AND TERMINAL"),
                                    )
                                    .child(SettingsInput {
                                        label: "Poll interval in seconds",
                                        placeholder: "2",
                                        value: poll_state,
                                    })
                                    .child(SettingsInput {
                                        label: "Daemon socket override",
                                        placeholder: "Auto from workspace",
                                        value: socket_state,
                                    })
                                    .child(SettingsInput {
                                        label: "Terminal shell",
                                        placeholder: "Auto from SHELL",
                                        value: shell_state,
                                    })
                                    .child(
                                        label()
                                            .color(colors.text_placeholder)
                                            .font_size(10.)
                                            .text(path_label),
                                    ),
                            ),
                    )
                    .child(
                        rect()
                            .width(Size::fill())
                            .height(Size::px(52.))
                            .horizontal()
                            .main_align(Alignment::end())
                            .cross_align(Alignment::center())
                            .spacing(8.)
                            .padding(Gaps::new(10., 0., 12., 0.))
                            .border(Border::new().fill(colors.border).width(BorderWidth {
                                top: 1.,
                                right: 0.,
                                bottom: 0.,
                                left: 0.,
                            }))
                            .child(
                                Button::new()
                                    .filled()
                                    .enabled(!provider_state.read().trim().is_empty())
                                    .on_press({
                                        let state = state;
                                        move |_| {
                                            save_settings(
                                                state,
                                                provider_state.read().clone(),
                                                model_state.read().clone(),
                                                socket_state.read().clone(),
                                                shell_state.read().clone(),
                                                poll_state.read().clone(),
                                                *whitespace_state.read(),
                                            )
                                        }
                                    })
                                    .child(
                                        label()
                                            .color(colors.text_primary)
                                            .font_size(12.)
                                            .text("Save settings"),
                                    ),
                            ),
                    ),
            )
    }
}

#[derive(PartialEq)]
struct ThemeButton {
    state: State<AppState>,
    current_theme: State<freya::prelude::Theme>,
    label: &'static str,
    value: &'static str,
    selected: bool,
}

impl Component for ThemeButton {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut state = self.state;
        let mut current_theme = self.current_theme;
        let value = self.value;
        let selected = self.selected;
        Button::new()
            .flat()
            .compact()
            .height(Size::px(30.))
            .padding(Gaps::new(12., 0., 12., 0.))
            .background(if selected {
                colors.surface_tertiary
            } else {
                Color::TRANSPARENT
            })
            .on_press(move |_| {
                let next = match value {
                    "light" => theme::light(),
                    "dark" => theme::dark(),
                    _ => match &*Platform::get().preferred_theme.read() {
                        PreferredTheme::Light => theme::light(),
                        PreferredTheme::Dark => theme::dark(),
                    },
                };
                state.write().config.theme = value.to_string();
                current_theme.set(next);
            })
            .child(
                label()
                    .color(if selected {
                        colors.text_primary
                    } else {
                        colors.text_secondary
                    })
                    .font_size(11.)
                    .text(self.label),
            )
    }
}

#[derive(PartialEq)]
struct SettingsInput {
    label: &'static str,
    placeholder: &'static str,
    value: State<String>,
}

impl Component for SettingsInput {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        rect()
            .width(Size::fill())
            .vertical()
            .spacing(6.)
            .child(
                label()
                    .color(colors.text_secondary)
                    .font_size(11.)
                    .text(self.label),
            )
            .child(
                Input::new(self.value.into_writable())
                    .placeholder(self.placeholder)
                    .width(Size::fill())
                    .filled(),
            )
    }
}

fn load_models(mut state: State<AppState>, client: PrumoClient, provider: String) {
    if provider.trim().is_empty() {
        state
            .write()
            .show_notice(NoticeTone::Error, "Provider is required");
        return;
    }
    spawn(async move {
        let result = tokio::task::spawn_blocking(move || client.models(&provider)).await;
        match result {
            Ok(Ok(models)) => {
                let mut state = state.write();
                state.available_models = models.models;
                state.show_notice(
                    NoticeTone::Success,
                    format!("Loaded models from {}", models.provider),
                );
            }
            Ok(Err(error)) => state.write().show_notice(
                NoticeTone::Error,
                format!("Model discovery failed: {error}"),
            ),
            Err(error) => state
                .write()
                .show_notice(NoticeTone::Error, format!("Model worker failed: {error}")),
        }
    });
}

fn save_settings(
    mut state: State<AppState>,
    provider: String,
    model: String,
    socket_path: String,
    terminal_shell: String,
    poll_interval: String,
    show_whitespace: bool,
) {
    let poll_interval_seconds = match poll_interval.trim().parse::<u64>() {
        Ok(value) => value,
        Err(_) => {
            state
                .write()
                .show_notice(NoticeTone::Error, "Poll interval must be a number");
            return;
        }
    };
    let selected_theme = state.read().config.theme.clone();
    let config = ViewerConfig {
        version: 1,
        theme: selected_theme,
        provider,
        model,
        socket_path: (!socket_path.trim().is_empty()).then_some(socket_path),
        terminal_shell: (!terminal_shell.trim().is_empty()).then_some(terminal_shell),
        show_whitespace,
        poll_interval_seconds,
    };
    match config.validate().and_then(|()| config.save()) {
        Ok(path) => {
            let mut state = state.write();
            state.config = config;
            state.config_open = false;
            state.show_notice(
                NoticeTone::Success,
                format!(
                    "Settings saved to {}. Restart to apply socket or shell changes",
                    path.display()
                ),
            );
        }
        Err(error) => state
            .write()
            .show_notice(NoticeTone::Error, format!("Settings save failed: {error}")),
    }
}
