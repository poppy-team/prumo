use crate::client::protocol::PrumoClient;
use crate::extensions::registry::{
    ExtensionRecord, ExtensionStatus, ExtensionTemplateKind, rescan_workspace_extensions,
    scaffold_extension,
};
use crate::services::config::{ViewerConfig, config_path};
use crate::state::{AppState, NoticeTone};
use crate::theme::{CustomTheme, ThemeDescriptor, available_themes, resolve_theme};
use crate::ui::chrome::{IconButton, hairline, section_label};
use crate::ui::icons::icon;
use freya::prelude::*;
use std::collections::BTreeMap;
use std::path::PathBuf;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SettingsCategory {
    Editor,
    Agent,
    Appearance,
    Extensions,
    Json,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum JsonScope {
    User,
    Workspace,
}

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
        let workspace_root = state.read().workspace_root.clone();

        let active_category = use_state(|| SettingsCategory::Editor);
        let json_scope = use_state(|| JsonScope::User);

        // Form state for Editor
        let whitespace_state = use_state(|| config.show_whitespace);
        let line_numbers_state = use_state(|| config.line_numbers);
        let word_wrap_state = use_state(|| config.word_wrap);
        let font_size_state = use_state(|| config.font_size.to_string());
        let tab_size_state = use_state(|| config.tab_size.to_string());
        let poll_state = use_state(|| config.poll_interval_seconds.to_string());
        let socket_state = use_state(|| config.socket_path.clone().unwrap_or_default());
        let shell_state = use_state(|| config.terminal_shell.clone().unwrap_or_default());

        // Form state for Agent
        let provider_state = use_state(|| config.provider.clone());
        let model_state = use_state(|| config.model.clone());
        let api_key_state = use_state(|| config.api_key.clone().unwrap_or_default());
        let base_url_state = use_state(|| config.base_url.clone().unwrap_or_default());
        let acf_token_state = use_state(|| config.acf_token.clone().unwrap_or_default());
        let available_models = state.read().available_models.clone();
        let provider = provider_state.read().trim().to_string();

        // Form state for Appearance / Themes
        let selected_theme = use_state(|| config.theme.clone());
        let show_create_theme = use_state(|| false);
        let new_theme_name = use_state(String::new);
        let new_theme_id = use_state(String::new);
        let new_theme_base = use_state(|| "dark".to_string());
        let new_theme_accent = use_state(|| "#78a9e8".to_string());
        let new_theme_bg = use_state(|| "#1b1e24".to_string());
        let new_theme_surface = use_state(|| "#282c35".to_string());
        let new_theme_border = use_state(|| "#394152".to_string());
        let new_theme_text = use_state(|| "#dce0e6".to_string());

        // Form state for Extensions
        let extension_filter = use_state(String::new);
        let show_create_ext = use_state(|| false);
        let new_ext_id = use_state(String::new);
        let new_ext_name = use_state(String::new);
        let new_ext_desc = use_state(String::new);
        let new_ext_publisher = use_state(|| "local".to_string());
        let new_ext_kind = use_state(|| ExtensionTemplateKind::Theme);
        let extensions = state.read().extensions.clone();
        let extension_errors = state.read().extension_errors.clone();

        // Form state for JSON settings
        let user_json_path = config_path().unwrap_or_else(|_| PathBuf::from("viewer.json"));
        let ws_json_path = ViewerConfig::workspace_config_path(&workspace_root);
        let user_json_str = config.to_json_pretty().unwrap_or_default();

        let raw_json_state = use_state(|| user_json_str);
        let window = Platform::get().root_size.read();

        rect()
            .width(Size::px(window.width))
            .height(Size::px(window.height))
            .position(Position::new_global().top(0.).left(0.))
            .layer(Layer::Overlay)
            .background(Color::from_argb(120, 0, 0, 0))
            .horizontal()
            .main_align(Alignment::center())
            .cross_align(Alignment::center())
            .on_press(move |_| state.write().config_open = false)
            .child(
                rect()
                    .width(Size::px(880.))
                    .height(Size::px(640.))
                    .background(colors.surface_primary)
                    .border(Border::new().fill(colors.border_focus).width(1.))
                    .corner_radius(CornerRadius::new_all(8.))
                    .vertical()
                    .content(Content::flex())
                    .on_mouse_up(|event: Event<MouseEventData>| {
                        event.stop_propagation();
                    })
                    // --- MODAL HEADER ---
                    .child(
                        rect()
                            .width(Size::fill())
                            .height(Size::px(44.))
                            .horizontal()
                            .main_align(Alignment::space_between())
                            .cross_align(Alignment::center())
                            .padding(Gaps::new(0., 12., 0., 16.))
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
                                        SvgViewer::new(("settings", icon("settings")))
                                            .color(colors.primary)
                                            .width(Size::px(16.))
                                            .height(Size::px(16.)),
                                    )
                                    .child(
                                        label()
                                            .color(colors.text_primary)
                                            .font_size(14.)
                                            .font_weight(FontWeight::BOLD)
                                            .text("Prumo Settings"),
                                    ),
                            )
                            .child(IconButton {
                                icon: "x",
                                label: "Close settings",
                                size: 12.,
                                on_press: (move |_: Event<PressEventData>| {
                                    state.write().config_open = false;
                                })
                                .into(),
                            }),
                    )
                    // --- BODY: SIDEBAR + CONTENT ---
                    .child(
                        rect()
                            .width(Size::fill())
                            .height(Size::flex(1.))
                            .horizontal()
                            // LEFT SIDEBAR TABS
                            .child(
                                rect()
                                    .width(Size::px(200.))
                                    .height(Size::fill())
                                    .background(colors.surface_secondary)
                                    .border(Border::new().fill(colors.border).width(BorderWidth {
                                        top: 0.,
                                        right: 1.,
                                        bottom: 0.,
                                        left: 0.,
                                    }))
                                    .padding(Gaps::new_all(8.))
                                    .vertical()
                                    .spacing(4.)
                                    .child(sidebar_tab_button(
                                        &colors,
                                        "settings",
                                        "Editor",
                                        *active_category.read() == SettingsCategory::Editor,
                                        {
                                            let mut active_category = active_category;
                                            move |_| {
                                                active_category.set(SettingsCategory::Editor);
                                            }
                                        },
                                    ))
                                    .child(sidebar_tab_button(
                                        &colors,
                                        "bot",
                                        "Agent & Models",
                                        *active_category.read() == SettingsCategory::Agent,
                                        {
                                            let mut active_category = active_category;
                                            move |_| {
                                                active_category.set(SettingsCategory::Agent);
                                            }
                                        },
                                    ))
                                    .child(sidebar_tab_button(
                                        &colors,
                                        "sun",
                                        "Appearance",
                                        *active_category.read() == SettingsCategory::Appearance,
                                        {
                                            let mut active_category = active_category;
                                            move |_| {
                                                active_category.set(SettingsCategory::Appearance);
                                            }
                                        },
                                    ))
                                    .child(sidebar_tab_button(
                                        &colors,
                                        "package",
                                        "Extensions",
                                        *active_category.read() == SettingsCategory::Extensions,
                                        {
                                            let mut active_category = active_category;
                                            move |_| {
                                                active_category.set(SettingsCategory::Extensions);
                                            }
                                        },
                                    ))
                                    .child(sidebar_tab_button(
                                        &colors,
                                        "file_code",
                                        "Settings (JSON)",
                                        *active_category.read() == SettingsCategory::Json,
                                        {
                                            let mut active_category = active_category;
                                            let mut raw_json_state = raw_json_state;
                                            let config = config.clone();
                                            move |_| {
                                                active_category.set(SettingsCategory::Json);
                                                raw_json_state.set(
                                                    config.to_json_pretty().unwrap_or_default(),
                                                );
                                            }
                                        },
                                    )),
                            )
                            // RIGHT CONTENT AREA
                            .child(
                                rect()
                                    .width(Size::flex(1.))
                                    .height(Size::fill())
                                    .padding(Gaps::new_all(16.))
                                    .child(ScrollView::new().direction(torin::prelude::Direction::Vertical).child(
                                        match *active_category.read() {
                                            SettingsCategory::Editor => render_editor_tab(
                                                whitespace_state,
                                                line_numbers_state,
                                                word_wrap_state,
                                                font_size_state,
                                                tab_size_state,
                                                poll_state,
                                                socket_state,
                                                shell_state,
                                            )
                                            .into_element(),
                                            SettingsCategory::Agent => render_agent_tab(
                                                &colors,
                                                state,
                                                client.clone(),
                                                provider_state,
                                                model_state,
                                                api_key_state,
                                                base_url_state,
                                                acf_token_state,
                                                available_models,
                                                &provider,
                                            )
                                            .into_element(),
                                            SettingsCategory::Appearance => render_appearance_tab(
                                                &colors,
                                                state,
                                                current_theme,
                                                &config,
                                                selected_theme,
                                                show_create_theme,
                                                new_theme_name,
                                                new_theme_id,
                                                new_theme_base,
                                                new_theme_accent,
                                                new_theme_bg,
                                                new_theme_surface,
                                                new_theme_border,
                                                new_theme_text,
                                            )
                                            .into_element(),
                                            SettingsCategory::Extensions => render_extensions_tab(
                                                &colors,
                                                state,
                                                workspace_root.clone(),
                                                extensions,
                                                extension_errors,
                                                extension_filter,
                                                show_create_ext,
                                                new_ext_id,
                                                new_ext_name,
                                                new_ext_desc,
                                                new_ext_publisher,
                                                new_ext_kind,
                                            )
                                            .into_element(),
                                            SettingsCategory::Json => render_json_tab(
                                                &colors,
                                                state,
                                                workspace_root.clone(),
                                                json_scope,
                                                raw_json_state,
                                                user_json_path.clone(),
                                                ws_json_path.clone(),
                                            )
                                            .into_element(),
                                        },
                                    )),
                            ),
                    )
                    // --- MODAL FOOTER ---
                    .child(
                        rect()
                            .width(Size::fill())
                            .height(Size::px(48.))
                            .horizontal()
                            .main_align(Alignment::space_between())
                            .cross_align(Alignment::center())
                            .padding(Gaps::new(0., 16., 0., 16.))
                            .border(Border::new().fill(colors.border).width(BorderWidth {
                                top: 1.,
                                right: 0.,
                                bottom: 0.,
                                left: 0.,
                            }))
                            .child(
                                rect()
                                    .horizontal()
                                    .spacing(6.)
                                    .cross_align(Alignment::center())
                                    .child(
                                        SvgViewer::new(("file_text", icon("file_text")))
                                            .color(colors.text_placeholder)
                                            .width(Size::px(12.))
                                            .height(Size::px(12.)),
                                    )
                                    .child(
                                        label()
                                            .color(colors.text_placeholder)
                                            .font_size(10.5)
                                            .text(user_json_path.display().to_string()),
                                    ),
                            )
                            .child(
                                rect()
                                    .horizontal()
                                    .spacing(8.)
                                    .child(
                                        Button::new()
                                            .flat()
                                            .compact()
                                            .height(Size::px(28.))
                                            .padding(Gaps::new(0., 12., 0., 12.))
                                            .on_press(move |_| state.write().config_open = false)
                                            .child(
                                                label()
                                                    .color(colors.text_secondary)
                                                    .font_size(11.)
                                                    .text("Close"),
                                            ),
                                    )
                                    .child(
                                        Button::new()
                                            .compact()
                                            .height(Size::px(28.))
                                            .padding(Gaps::new(0., 16., 0., 16.))
                                            .enabled(!provider_state.read().trim().is_empty())
                                            .on_press({
                                                let state = state;
                                                move |_| {
                                                    save_all_settings(
                                                        state,
                                                        current_theme,
                                                        selected_theme.read().clone(),
                                                        provider_state.read().clone(),
                                                        model_state.read().clone(),
                                                        api_key_state.read().clone(),
                                                        base_url_state.read().clone(),
                                                        acf_token_state.read().clone(),
                                                        socket_state.read().clone(),
                                                        shell_state.read().clone(),
                                                        poll_state.read().clone(),
                                                        font_size_state.read().clone(),
                                                        tab_size_state.read().clone(),
                                                        *whitespace_state.read(),
                                                        *line_numbers_state.read(),
                                                        *word_wrap_state.read(),
                                                    );
                                                }
                                            })
                                            .child(
                                                label()
                                                    .color(colors.text_primary)
                                                    .font_size(11.)
                                                    .font_weight(FontWeight::MEDIUM)
                                                    .text("Save Settings"),
                                            ),
                                    ),
                            ),
                    ),
            )
    }
}

// -----------------------------------------------------------------------------
// TAB 1: EDITOR PREFERENCES
// -----------------------------------------------------------------------------
#[allow(clippy::too_many_arguments)]
fn render_editor_tab(
    whitespace_state: State<bool>,
    line_numbers_state: State<bool>,
    word_wrap_state: State<bool>,
    font_size_state: State<String>,
    tab_size_state: State<String>,
    poll_state: State<String>,
    socket_state: State<String>,
    shell_state: State<String>,
) -> impl IntoElement {
    let colors = get_theme_or_default().read().colors.clone();

    rect()
        .width(Size::fill())
        .vertical()
        .spacing(12.)
        .child(section_header(
            "Editor Preferences",
            "Configure typography, whitespace visibility, and workspace behavior",
        ))
        // TOGGLE SWITCHES ROW
        .child(
            rect()
                .width(Size::fill())
                .horizontal()
                .spacing(8.)
                .child(toggle_card(
                    &colors,
                    "Show Whitespace",
                    "Render space & tab markers",
                    *whitespace_state.read(),
                    {
                        let mut whitespace_state = whitespace_state;
                        move |_| {
                            whitespace_state.set(!*whitespace_state.read());
                        }
                    },
                ))
                .child(toggle_card(
                    &colors,
                    "Line Numbers",
                    "Display editor line gutter",
                    *line_numbers_state.read(),
                    {
                        let mut line_numbers_state = line_numbers_state;
                        move |_| {
                            line_numbers_state.set(!*line_numbers_state.read());
                        }
                    },
                ))
                .child(toggle_card(
                    &colors,
                    "Word Wrap",
                    "Wrap lines at viewport width",
                    *word_wrap_state.read(),
                    {
                        let mut word_wrap_state = word_wrap_state;
                        move |_| {
                            word_wrap_state.set(!*word_wrap_state.read());
                        }
                    },
                )),
        )
        .child(hairline())
        .child(section_label("Typography & Spacing"))
        .child(
            rect()
                .width(Size::fill())
                .horizontal()
                .spacing(12.)
                .child(
                    rect()
                        .width(Size::flex(1.))
                        .child(SettingsInput {
                            label: "Font Size (pt)",
                            placeholder: "13.0",
                            value: font_size_state,
                        }),
                )
                .child(
                    rect()
                        .width(Size::flex(1.))
                        .child(SettingsInput {
                            label: "Tab Size (spaces)",
                            placeholder: "4",
                            value: tab_size_state,
                        }),
                ),
        )
        .child(hairline())
        .child(section_label("Runtime & Terminal"))
        .child(SettingsInput {
            label: "Daemon Poll Interval (seconds)",
            placeholder: "2",
            value: poll_state,
        })
        .child(SettingsInput {
            label: "Daemon Socket Override",
            placeholder: "Auto (defaults to .prumo/agent.sock)",
            value: socket_state,
        })
        .child(SettingsInput {
            label: "Terminal Shell Override",
            placeholder: "Auto (defaults to $SHELL)",
            value: shell_state,
        })
}

// -----------------------------------------------------------------------------
// TAB 2: AGENT & MODELS
// -----------------------------------------------------------------------------
#[allow(clippy::too_many_arguments)]
fn render_agent_tab(
    colors: &ColorsSheet,
    state: State<AppState>,
    client: PrumoClient,
    provider_state: State<String>,
    model_state: State<String>,
    api_key_state: State<String>,
    base_url_state: State<String>,
    acf_token_state: State<String>,
    available_models: Vec<String>,
    provider: &str,
) -> impl IntoElement {
    rect()
        .width(Size::fill())
        .vertical()
        .spacing(12.)
        .child(section_header(
            "Agent & LLM Connectivity",
            "Configure model providers, authentication keys, and remote endpoints",
        ))
        .child(section_label("Provider Presets"))
        .child(
            rect()
                .width(Size::fill())
                .horizontal()
                .spacing(6.)
                .child(PresetButton {
                    label: "OpenAI",
                    provider: "openai-compat",
                    base_url: "https://api.openai.com/v1",
                    default_model: "gpt-4o",
                    provider_state,
                    base_url_state,
                    model_state,
                })
                .child(PresetButton {
                    label: "Anthropic",
                    provider: "anthropic",
                    base_url: "",
                    default_model: "claude-3-7-sonnet",
                    provider_state,
                    base_url_state,
                    model_state,
                })
                .child(PresetButton {
                    label: "Gemini",
                    provider: "gemini",
                    base_url: "https://generativelanguage.googleapis.com/v1beta/openai",
                    default_model: "gemini-2.5-pro",
                    provider_state,
                    base_url_state,
                    model_state,
                })
                .child(PresetButton {
                    label: "ACF / Antigravity",
                    provider: "antigravity",
                    base_url: "",
                    default_model: "gemini-2.5-pro",
                    provider_state,
                    base_url_state,
                    model_state,
                })
                .child(PresetButton {
                    label: "Ollama (Local)",
                    provider: "openai-compat",
                    base_url: "http://127.0.0.1:11434/v1",
                    default_model: "deepseek-coder",
                    provider_state,
                    base_url_state,
                    model_state,
                })
                .child(PresetButton {
                    label: "Fake",
                    provider: "fake",
                    base_url: "",
                    default_model: "",
                    provider_state,
                    base_url_state,
                    model_state,
                }),
        )
        .child(hairline())
        .child(SettingsInput {
            label: "Provider ID",
            placeholder: "openai-compat, anthropic, gemini, antigravity, fake",
            value: provider_state,
        })
        .child(
            rect()
                .width(Size::fill())
                .vertical()
                .spacing(6.)
                .child(
                    rect()
                        .width(Size::fill())
                        .horizontal()
                        .main_align(Alignment::space_between())
                        .cross_align(Alignment::center())
                        .child(section_label("Model ID"))
                        .child(
                            Button::new()
                                .flat()
                                .compact()
                                .height(Size::px(22.))
                                .padding(Gaps::new(0., 8., 0., 8.))
                                .on_press({
                                    let client = client.clone();
                                    let provider = provider.to_string();
                                    let api_key = api_key_state.read().clone();
                                    let base_url = base_url_state.read().clone();
                                    let acf_token = acf_token_state.read().clone();
                                    move |_| {
                                        let key = if !api_key.trim().is_empty() {
                                            Some(api_key.clone())
                                        } else if !acf_token.trim().is_empty() {
                                            Some(acf_token.clone())
                                        } else {
                                            None
                                        };
                                        let url =
                                            (!base_url.trim().is_empty()).then(|| base_url.clone());
                                        load_models(
                                            state,
                                            client.clone(),
                                            provider.clone(),
                                            key,
                                            url,
                                        );
                                    }
                                })
                                .child(
                                    label()
                                        .color(colors.primary)
                                        .font_size(10.5)
                                        .text("Discover Available Models"),
                                ),
                        ),
                )
                .child(
                    Input::new(model_state.into_writable())
                        .placeholder("gpt-4o, claude-3-7-sonnet, gemini-2.5-pro")
                        .width(Size::fill()),
                )
                .maybe(!available_models.is_empty(), |r| {
                    r.child(
                        rect()
                            .width(Size::fill())
                            .max_height(Size::px(90.))
                            .background(colors.surface_secondary)
                            .border(Border::new().fill(colors.border).width(1.))
                            .corner_radius(CornerRadius::new_all(4.))
                            .padding(Gaps::new_all(4.))
                            .child(ScrollView::new().direction(torin::prelude::Direction::Vertical).children(
                                available_models.into_iter().map(|model_name| {
                                    let m = model_name.clone();
                                    let mut model_state = model_state;
                                    Button::new()
                                        .flat()
                                        .expanded()
                                        .height(Size::px(22.))
                                        .padding(Gaps::new(0., 6., 0., 6.))
                                        .on_press(move |_| {
                                            model_state.set(m.clone());
                                        })
                                        .child(
                                            label()
                                                .color(colors.text_primary)
                                                .font_size(10.5)
                                                .text(model_name),
                                        )
                                        .into_element()
                                }),
                            )),
                    )
                }),
        )
        .child(SettingsInput {
            label: "API Key (Provider Secret)",
            placeholder: "sk-... (stored securely in local config)",
            value: api_key_state,
        })
        .child(SettingsInput {
            label: "Base URL (API Endpoint Override)",
            placeholder: "https://api.openai.com/v1 or http://127.0.0.1:11434/v1",
            value: base_url_state,
        })
        .child(SettingsInput {
            label: "ACF / Antigravity Token",
            placeholder: "Antigravity Context Fabric session token",
            value: acf_token_state,
        })
}

// -----------------------------------------------------------------------------
// TAB 3: APPEARANCE & THEMES
// -----------------------------------------------------------------------------
#[allow(clippy::too_many_arguments)]
fn render_appearance_tab(
    colors: &ColorsSheet,
    state: State<AppState>,
    current_theme: State<Theme>,
    config: &ViewerConfig,
    selected_theme: State<String>,
    show_create_theme: State<bool>,
    new_theme_name: State<String>,
    new_theme_id: State<String>,
    new_theme_base: State<String>,
    new_theme_accent: State<String>,
    new_theme_bg: State<String>,
    new_theme_surface: State<String>,
    new_theme_border: State<String>,
    new_theme_text: State<String>,
) -> impl IntoElement {
    let all_themes = available_themes(&config.custom_themes);
    let cur_theme_val = selected_theme.read().clone();

    rect()
        .width(Size::fill())
        .vertical()
        .spacing(12.)
        .child(
            rect()
                .width(Size::fill())
                .horizontal()
                .main_align(Alignment::space_between())
                .cross_align(Alignment::center())
                .child(section_header(
                    "Appearance & Themes",
                    "Choose built-in color themes or create custom JSON themes",
                ))
                .child(
                    Button::new()
                        .compact()
                        .height(Size::px(26.))
                        .padding(Gaps::new(0., 10., 0., 10.))
                        .on_press({
                            let mut show_create_theme = show_create_theme;
                            move |_| {
                                show_create_theme.set(!*show_create_theme.read());
                            }
                        })
                        .child(
                            label()
                                .color(colors.primary)
                                .font_size(11.)
                                .text(if *show_create_theme.read() {
                                    "Cancel Theme Creator"
                                } else {
                                    "+ Create Custom Theme"
                                }),
                        ),
                ),
        )
        // THEME CREATOR DRAWER
        .maybe(*show_create_theme.read(), |r| {
            let custom_themes_ref = config.custom_themes.clone();
            let mut new_theme_base_mut = new_theme_base;
            r.child(
                rect()
                    .width(Size::fill())
                    .background(colors.surface_secondary)
                    .border(Border::new().fill(colors.border_focus).width(1.))
                    .corner_radius(CornerRadius::new_all(6.))
                    .padding(Gaps::new_all(12.))
                    .vertical()
                    .spacing(8.)
                    .child(
                        label()
                            .color(colors.text_primary)
                            .font_size(12.)
                            .font_weight(FontWeight::BOLD)
                            .text("Custom Theme Builder"),
                    )
                    .child(
                        rect()
                            .width(Size::fill())
                            .horizontal()
                            .spacing(8.)
                            .child(
                                rect()
                                    .width(Size::flex(1.))
                                    .child(SettingsInput {
                                        label: "Theme Name",
                                        placeholder: "e.g. Cyberpunk Neon",
                                        value: new_theme_name,
                                    }),
                            )
                            .child(
                                rect()
                                    .width(Size::flex(1.))
                                    .child(SettingsInput {
                                        label: "Theme ID (slug)",
                                        placeholder: "e.g. cyberpunk-neon",
                                        value: new_theme_id,
                                    }),
                            ),
                    )
                    .child(
                        rect()
                            .width(Size::fill())
                            .horizontal()
                            .spacing(6.)
                            .cross_align(Alignment::center())
                            .child(section_label("Base Template:"))
                            .child(
                                Button::new()
                                    .flat()
                                    .compact()
                                    .background(if new_theme_base.read().as_str() == "dark" {
                                        colors.surface_tertiary
                                    } else {
                                        Color::TRANSPARENT
                                    })
                                    .on_press(move |_| new_theme_base_mut.set("dark".to_string()))
                                    .child(label().font_size(10.5).text("Dark Base")),
                            )
                            .child(
                                Button::new()
                                    .flat()
                                    .compact()
                                    .background(if new_theme_base.read().as_str() == "light" {
                                        colors.surface_tertiary
                                    } else {
                                        Color::TRANSPARENT
                                    })
                                    .on_press(move |_| new_theme_base_mut.set("light".to_string()))
                                    .child(label().font_size(10.5).text("Light Base")),
                            ),
                    )
                    .child(
                        rect()
                            .width(Size::fill())
                            .horizontal()
                            .spacing(8.)
                            .child(
                                rect()
                                    .width(Size::flex(1.))
                                    .child(SettingsInput {
                                        label: "Accent Hex",
                                        placeholder: "#78a9e8",
                                        value: new_theme_accent,
                                    }),
                            )
                            .child(
                                rect()
                                    .width(Size::flex(1.))
                                    .child(SettingsInput {
                                        label: "Background Hex",
                                        placeholder: "#1b1e24",
                                        value: new_theme_bg,
                                    }),
                            )
                            .child(
                                rect()
                                    .width(Size::flex(1.))
                                    .child(SettingsInput {
                                        label: "Surface Hex",
                                        placeholder: "#282c35",
                                        value: new_theme_surface,
                                    }),
                            ),
                    )
                    .child(
                        rect()
                            .width(Size::fill())
                            .horizontal()
                            .spacing(8.)
                            .child(
                                rect()
                                    .width(Size::flex(1.))
                                    .child(SettingsInput {
                                        label: "Border Hex",
                                        placeholder: "#394152",
                                        value: new_theme_border,
                                    }),
                            )
                            .child(
                                rect()
                                    .width(Size::flex(1.))
                                    .child(SettingsInput {
                                        label: "Text Hex",
                                        placeholder: "#dce0e6",
                                        value: new_theme_text,
                                    }),
                            ),
                    )
                    .child(
                        rect()
                            .width(Size::fill())
                            .horizontal()
                            .main_align(Alignment::end())
                            .child(
                                Button::new()
                                    .compact()
                                    .height(Size::px(26.))
                                    .padding(Gaps::new(0., 12., 0., 12.))
                                    .on_press({
                                        let name = new_theme_name.read().clone();
                                        let id = new_theme_id.read().clone();
                                        let appearance = new_theme_base.read().clone();
                                        let accent = new_theme_accent.read().clone();
                                        let bg = new_theme_bg.read().clone();
                                        let surface = new_theme_surface.read().clone();
                                        let border = new_theme_border.read().clone();
                                        let text = new_theme_text.read().clone();
                                        let mut custom_themes = custom_themes_ref;
                                        let mut state = state;
                                        let mut current_theme = current_theme;
                                        let mut selected_theme = selected_theme;
                                        let mut show_create_theme = show_create_theme;
                                        move |_| {
                                            if name.trim().is_empty() {
                                                state.write().show_notice(
                                                    NoticeTone::Error,
                                                    "Theme name is required",
                                                );
                                                return;
                                            }
                                            let tid = if id.trim().is_empty() {
                                                name.trim().to_lowercase().replace(' ', "-")
                                            } else {
                                                id.trim().to_string()
                                            };
                                            let mut tokens = BTreeMap::new();
                                            tokens.insert("accent".to_string(), accent.clone());
                                            tokens.insert("background".to_string(), bg.clone());
                                            tokens.insert(
                                                "surface-primary".to_string(),
                                                surface.clone(),
                                            );
                                            tokens.insert(
                                                "surface-secondary".to_string(),
                                                surface.clone(),
                                            );
                                            tokens.insert("border".to_string(), border.clone());
                                            tokens.insert(
                                                "text-primary".to_string(),
                                                text.clone(),
                                            );

                                            let new_custom = CustomTheme {
                                                id: tid.clone(),
                                                name: name.clone(),
                                                appearance: appearance.clone(),
                                                tokens,
                                            };
                                            custom_themes.retain(|t| t.id != tid);
                                            custom_themes.push(new_custom.clone());

                                            let mut app_state = state.write();
                                            app_state.config.custom_themes = custom_themes.clone();
                                            app_state.config.theme = tid.clone();
                                            let _ = app_state.config.save();

                                            let next = resolve_theme(
                                                &tid,
                                                &app_state.config.custom_themes,
                                                None,
                                            );
                                            current_theme.set(next);
                                            selected_theme.set(tid);
                                            show_create_theme.set(false);
                                            app_state.show_notice(
                                                NoticeTone::Success,
                                                format!("Theme '{name}' created and applied!"),
                                            );
                                        }
                                    })
                                    .child(
                                        label()
                                            .color(colors.text_primary)
                                            .font_size(11.)
                                            .text("Save & Apply Theme"),
                                    ),
                            ),
                    ),
            )
        })
        .child(hairline())
        .child(section_label("Available Themes"))
        // THEMES GRID (2 COLUMNS)
        .child(
            rect()
                .width(Size::fill())
                .horizontal()
                .spacing(8.)
                .children(all_themes.chunks(2).map(|row| {
                    rect()
                        .width(Size::flex(1.))
                        .vertical()
                        .spacing(8.)
                        .children(row.iter().map(|desc| {
                            let tid = desc.id.clone();
                            let is_active = cur_theme_val == tid;
                            let custom_list = config.custom_themes.clone();
                            let mut selected_theme = selected_theme;
                            let mut current_theme = current_theme;
                            let mut state = state;
                            theme_card(colors, desc.clone(), is_active, move |_| {
                                selected_theme.set(tid.clone());
                                let next = resolve_theme(&tid, &custom_list, None);
                                current_theme.set(next);
                                state.write().config.theme = tid.clone();
                            })
                            .into_element()
                        }))
                        .into_element()
                })),
        )
}

// -----------------------------------------------------------------------------
// TAB 4: EXTENSIONS & PLUGINS
// -----------------------------------------------------------------------------
#[allow(clippy::too_many_arguments)]
fn render_extensions_tab(
    colors: &ColorsSheet,
    mut state: State<AppState>,
    workspace_root: PathBuf,
    extensions: Vec<ExtensionRecord>,
    extension_errors: Vec<crate::extensions::registry::ExtensionLoadError>,
    extension_filter: State<String>,
    show_create_ext: State<bool>,
    new_ext_id: State<String>,
    new_ext_name: State<String>,
    new_ext_desc: State<String>,
    new_ext_publisher: State<String>,
    new_ext_kind: State<ExtensionTemplateKind>,
) -> impl IntoElement {
    let filter = extension_filter.read().to_lowercase();
    let filtered_exts: Vec<_> = extensions
        .into_iter()
        .filter(|ext| {
            if filter.is_empty() {
                true
            } else {
                ext.manifest.name.to_lowercase().contains(&filter)
                    || ext.manifest.id.to_lowercase().contains(&filter)
                    || ext.manifest.description.to_lowercase().contains(&filter)
            }
        })
        .collect();

    rect()
        .width(Size::fill())
        .vertical()
        .spacing(12.)
        .child(
            rect()
                .width(Size::fill())
                .horizontal()
                .main_align(Alignment::space_between())
                .cross_align(Alignment::center())
                .child(section_header(
                    "Plugins & Extensions",
                    "Manage installed packages, review permissions, and scaffold new extensions",
                ))
                .child(
                    rect()
                        .horizontal()
                        .spacing(6.)
                        .child(
                            Button::new()
                                .flat()
                                .compact()
                                .height(Size::px(26.))
                                .padding(Gaps::new(0., 10., 0., 10.))
                                .on_press({
                                    let ws = workspace_root.clone();
                                    move |_| {
                                        let (exts, errs) = rescan_workspace_extensions(&ws);
                                        let mut app_state = state.write();
                                        app_state.extensions = exts;
                                        app_state.extension_errors = errs;
                                        app_state.show_notice(
                                            NoticeTone::Success,
                                            "Extensions rescanned successfully",
                                        );
                                    }
                                })
                                .child(
                                    label()
                                        .color(colors.text_secondary)
                                        .font_size(10.5)
                                        .text("Reload"),
                                ),
                        )
                        .child(
                            Button::new()
                                .compact()
                                .height(Size::px(26.))
                                .padding(Gaps::new(0., 10., 0., 10.))
                                .on_press({
                                    let mut show_create_ext = show_create_ext;
                                    move |_| {
                                        show_create_ext.set(!*show_create_ext.read());
                                    }
                                })
                                .child(
                                    label()
                                        .color(colors.primary)
                                        .font_size(10.5)
                                        .text(if *show_create_ext.read() {
                                            "Close Wizard"
                                        } else {
                                            "+ New Extension"
                                        }),
                                ),
                        ),
                ),
        )
        // EXTENSION CREATOR WIZARD
        .maybe(*show_create_ext.read(), |r| {
            let ws = workspace_root.clone();
            let mut new_ext_kind_mut = new_ext_kind;
            r.child(
                rect()
                    .width(Size::fill())
                    .background(colors.surface_secondary)
                    .border(Border::new().fill(colors.border_focus).width(1.))
                    .corner_radius(CornerRadius::new_all(6.))
                    .padding(Gaps::new_all(12.))
                    .vertical()
                    .spacing(8.)
                    .child(
                        label()
                            .color(colors.text_primary)
                            .font_size(12.)
                            .font_weight(FontWeight::BOLD)
                            .text("Scaffold New Extension"),
                    )
                    .child(
                        rect()
                            .width(Size::fill())
                            .horizontal()
                            .spacing(8.)
                            .child(
                                rect()
                                    .width(Size::flex(1.))
                                    .child(SettingsInput {
                                        label: "Extension Name",
                                        placeholder: "e.g. Workspace Quick Actions",
                                        value: new_ext_name,
                                    }),
                            )
                            .child(
                                rect()
                                    .width(Size::flex(1.))
                                    .child(SettingsInput {
                                        label: "Extension ID",
                                        placeholder: "e.g. quick-actions",
                                        value: new_ext_id,
                                    }),
                            ),
                    )
                    .child(SettingsInput {
                        label: "Description",
                        placeholder: "Short summary of what this extension provides",
                        value: new_ext_desc,
                    })
                    .child(
                        rect()
                            .width(Size::fill())
                            .horizontal()
                            .spacing(8.)
                            .child(
                                rect()
                                    .width(Size::flex(1.))
                                    .child(SettingsInput {
                                        label: "Publisher",
                                        placeholder: "e.g. my-team or local",
                                        value: new_ext_publisher,
                                    }),
                            )
                            .child(
                                rect()
                                    .width(Size::flex(1.))
                                    .vertical()
                                    .spacing(6.)
                                    .child(section_label("Template Type"))
                                    .child(
                                        rect()
                                            .width(Size::fill())
                                            .horizontal()
                                            .spacing(4.)
                                            .child(
                                                Button::new()
                                                    .flat()
                                                    .compact()
                                                    .background(if *new_ext_kind.read()
                                                        == ExtensionTemplateKind::Theme
                                                    {
                                                        colors.surface_tertiary
                                                    } else {
                                                        Color::TRANSPARENT
                                                    })
                                                    .on_press(move |_| {
                                                        new_ext_kind_mut
                                                            .set(ExtensionTemplateKind::Theme)
                                                    })
                                                    .child(
                                                        label()
                                                            .font_size(10.5)
                                                            .text("Theme Template"),
                                                    ),
                                            )
                                            .child(
                                                Button::new()
                                                    .flat()
                                                    .compact()
                                                    .background(if *new_ext_kind.read()
                                                        == ExtensionTemplateKind::Commands
                                                    {
                                                        colors.surface_tertiary
                                                    } else {
                                                        Color::TRANSPARENT
                                                    })
                                                    .on_press(move |_| {
                                                        new_ext_kind_mut
                                                            .set(ExtensionTemplateKind::Commands)
                                                    })
                                                    .child(
                                                        label()
                                                            .font_size(10.5)
                                                            .text("Command Template"),
                                                    ),
                                            )
                                            .child(
                                                Button::new()
                                                    .flat()
                                                    .compact()
                                                    .background(if *new_ext_kind.read()
                                                        == ExtensionTemplateKind::Process
                                                    {
                                                        colors.surface_tertiary
                                                    } else {
                                                        Color::TRANSPARENT
                                                    })
                                                    .on_press(move |_| {
                                                        new_ext_kind_mut
                                                            .set(ExtensionTemplateKind::Process)
                                                    })
                                                    .child(
                                                        label()
                                                            .font_size(10.5)
                                                            .text("Process / LSP"),
                                                    ),
                                            ),
                                    ),
                            ),
                    )
                    .child(
                        rect()
                            .width(Size::fill())
                            .horizontal()
                            .main_align(Alignment::end())
                            .child(
                                Button::new()
                                    .compact()
                                    .height(Size::px(26.))
                                    .padding(Gaps::new(0., 12., 0., 12.))
                                    .on_press({
                                        let name = new_ext_name.read().clone();
                                        let id = new_ext_id.read().clone();
                                        let desc = new_ext_desc.read().clone();
                                        let publisher = new_ext_publisher.read().clone();
                                        let kind = *new_ext_kind.read();
                                        let mut show_create_ext = show_create_ext;
                                        move |_| {
                                            if id.trim().is_empty() {
                                                state.write().show_notice(
                                                    NoticeTone::Error,
                                                    "Extension ID is required",
                                                );
                                                return;
                                            }
                                            match scaffold_extension(
                                                &ws, &id, &name, &desc, &publisher, kind,
                                            ) {
                                                Ok(path) => {
                                                    let (exts, errs) =
                                                        rescan_workspace_extensions(&ws);
                                                    let mut app_state = state.write();
                                                    app_state.extensions = exts;
                                                    app_state.extension_errors = errs;
                                                    show_create_ext.set(false);
                                                    app_state.show_notice(
                                                        NoticeTone::Success,
                                                        format!(
                                                            "Extension created at {}",
                                                            path.display()
                                                        ),
                                                    );
                                                }
                                                Err(e) => {
                                                    state.write().show_notice(
                                                        NoticeTone::Error,
                                                        format!("Scaffold failed: {e}"),
                                                    );
                                                }
                                            }
                                        }
                                    })
                                    .child(
                                        label()
                                            .color(colors.text_primary)
                                            .font_size(11.)
                                            .text("Generate Extension Package"),
                                    ),
                            ),
                    ),
            )
        })
        .child(
            Input::new(extension_filter.into_writable())
                .placeholder("Filter installed extensions by name, ID, or description...")
                .width(Size::fill()),
        )
        // ERRORS IF ANY
        .maybe(!extension_errors.is_empty(), |r| {
            r.child(
                rect()
                    .width(Size::fill())
                    .background(Color::from_argb(30, 255, 100, 100))
                    .border(Border::new().fill(colors.error).width(1.))
                    .corner_radius(CornerRadius::new_all(4.))
                    .padding(Gaps::new_all(8.))
                    .vertical()
                    .spacing(4.)
                    .children(extension_errors.into_iter().map(|err| {
                        label()
                            .color(colors.error)
                            .font_size(10.5)
                            .text(format!("{}: {}", err.root.display(), err.message))
                            .into_element()
                    })),
            )
        })
        // EXTENSIONS CARDS LIST
        .child(
            rect()
                .width(Size::fill())
                .vertical()
                .spacing(8.)
                .children(if filtered_exts.is_empty() {
                    vec![rect()
                        .width(Size::fill())
                        .height(Size::px(60.))
                        .horizontal()
                        .main_align(Alignment::center())
                        .cross_align(Alignment::center())
                        .child(
                            label()
                                .color(colors.text_placeholder)
                                .font_size(12.)
                                .text("No extensions match the current search filter"),
                        )
                        .into_element()]
                } else {
                    filtered_exts
                        .into_iter()
                        .map(|ext| ExtensionCard { record: ext }.into_element())
                        .collect()
                }),
        )
}

// -----------------------------------------------------------------------------
// TAB 5: SETTINGS (JSON)
// -----------------------------------------------------------------------------
fn render_json_tab(
    colors: &ColorsSheet,
    state: State<AppState>,
    workspace_root: PathBuf,
    json_scope: State<JsonScope>,
    raw_json_state: State<String>,
    user_json_path: PathBuf,
    ws_json_path: PathBuf,
) -> impl IntoElement {
    let is_user = *json_scope.read() == JsonScope::User;
    let active_path = if is_user {
        &user_json_path
    } else {
        &ws_json_path
    };

    let validation_result = serde_json::from_str::<ViewerConfig>(&raw_json_state.read());
    let is_valid = validation_result.is_ok();
    let error_msg = validation_result.err().map(|e| e.to_string());

    rect()
        .width(Size::fill())
        .vertical()
        .spacing(10.)
        .child(
            rect()
                .width(Size::fill())
                .horizontal()
                .main_align(Alignment::space_between())
                .cross_align(Alignment::center())
                .child(section_header(
                    "Settings (JSON)",
                    "Directly view and edit raw settings configuration files",
                ))
                .child(
                    rect()
                        .horizontal()
                        .spacing(6.)
                        .child(
                            Button::new()
                                .flat()
                                .compact()
                                .height(Size::px(24.))
                                .background(if is_user {
                                    colors.surface_tertiary
                                } else {
                                    Color::TRANSPARENT
                                })
                                .on_press({
                                    let mut json_scope = json_scope;
                                    let mut raw = raw_json_state;
                                    let path = user_json_path.clone();
                                    move |_| {
                                        json_scope.set(JsonScope::User);
                                        if let Ok(c) = ViewerConfig::load_from_path(&path) {
                                            raw.set(c.to_json_pretty().unwrap_or_default());
                                        }
                                    }
                                })
                                .child(label().font_size(10.5).text("User (Global)")),
                        )
                        .child(
                            Button::new()
                                .flat()
                                .compact()
                                .height(Size::px(24.))
                                .background(if !is_user {
                                    colors.surface_tertiary
                                } else {
                                    Color::TRANSPARENT
                                })
                                .on_press({
                                    let mut json_scope = json_scope;
                                    let mut raw = raw_json_state;
                                    let ws = workspace_root.clone();
                                    move |_| {
                                        json_scope.set(JsonScope::Workspace);
                                        let c = ViewerConfig::load_workspace(&ws)
                                            .ok()
                                            .flatten()
                                            .and_then(|c| c.to_json_pretty().ok())
                                            .unwrap_or_else(|| {
                                                "{\n  \"version\": 1\n}\n".to_string()
                                            });
                                        raw.set(c);
                                    }
                                })
                                .child(label().font_size(10.5).text("Workspace (Local)")),
                        ),
                ),
        )
        // FILE PATH & STATUS INDICATOR
        .child(
            rect()
                .width(Size::fill())
                .horizontal()
                .main_align(Alignment::space_between())
                .cross_align(Alignment::center())
                .child(
                    label()
                        .color(colors.text_secondary)
                        .font_size(10.5)
                        .text(format!("File: {}", active_path.display())),
                )
                .child(
                    rect()
                        .horizontal()
                        .spacing(4.)
                        .cross_align(Alignment::center())
                        .child(
                            rect()
                                .width(Size::px(8.))
                                .height(Size::px(8.))
                                .corner_radius(CornerRadius::new_all(4.))
                                .background(if is_valid {
                                    colors.success
                                } else {
                                    colors.error
                                }),
                        )
                        .child(
                            label()
                                .color(if is_valid {
                                    colors.success
                                } else {
                                    colors.error
                                })
                                .font_size(10.5)
                                .text(if is_valid {
                                    "Valid JSON Configuration"
                                } else {
                                    "Syntax / Contract Error"
                                }),
                        ),
                ),
        )
        .maybe(error_msg.is_some(), |r| {
            let msg = error_msg.unwrap_or_default();
            r.child(
                rect()
                    .width(Size::fill())
                    .padding(Gaps::new_all(6.))
                    .background(Color::from_argb(30, 255, 80, 80))
                    .corner_radius(CornerRadius::new_all(4.))
                    .child(
                        label()
                            .color(colors.error)
                            .font_size(10.)
                            .max_lines(2)
                            .text(msg),
                    ),
            )
        })
        // MONOSPACE RAW JSON EDITOR
        .child(
            rect()
                .width(Size::fill())
                .height(Size::px(320.))
                .background(colors.surface_secondary)
                .border(
                    Border::new()
                        .fill(if is_valid {
                            colors.border
                        } else {
                            colors.error
                        })
                        .width(1.),
                )
                .corner_radius(CornerRadius::new_all(4.))
                .padding(Gaps::new_all(8.))
                .child(
                    Input::new(raw_json_state.into_writable())
                        .placeholder("{\n  \"version\": 1\n}")
                        .width(Size::fill()),
                ),
        )
        // JSON ACTION BUTTONS
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
                        .on_press({
                            let mut raw = raw_json_state;
                            move |_| {
                                if let Ok(val) =
                                    serde_json::from_str::<serde_json::Value>(&raw.read())
                                    && let Ok(formatted) = serde_json::to_string_pretty(&val)
                                {
                                    raw.set(formatted);
                                }
                            }
                        })
                        .child(
                            label()
                                .color(colors.text_secondary)
                                .font_size(10.5)
                                .text("Format JSON"),
                        ),
                )
                .child(
                    Button::new()
                        .compact()
                        .height(Size::px(26.))
                        .padding(Gaps::new(0., 12., 0., 12.))
                        .enabled(is_valid)
                        .on_press({
                            let ws = workspace_root.clone();
                            let user_path = user_json_path.clone();
                            let raw = raw_json_state.read().clone();
                            let mut state = state;
                            move |_| match ViewerConfig::from_json_str(&raw) {
                                Ok(cfg) => {
                                    let res = if is_user {
                                        cfg.save_to_path(&user_path).map(|_| user_path.clone())
                                    } else {
                                        ViewerConfig::save_workspace(&ws, &cfg)
                                    };
                                    match res {
                                        Ok(p) => {
                                            state.write().config = cfg;
                                            state.write().show_notice(
                                                NoticeTone::Success,
                                                format!("Saved configuration to {}", p.display()),
                                            );
                                        }
                                        Err(e) => {
                                            state.write().show_notice(
                                                NoticeTone::Error,
                                                format!("Save failed: {e}"),
                                            );
                                        }
                                    }
                                }
                                Err(e) => {
                                    state.write().show_notice(
                                        NoticeTone::Error,
                                        format!("Invalid JSON: {e}"),
                                    );
                                }
                            }
                        })
                        .child(
                            label()
                                .color(colors.text_primary)
                                .font_size(10.5)
                                .text("Save File Directly"),
                        ),
                ),
        )
}

// -----------------------------------------------------------------------------
// HELPER COMPONENTS
// -----------------------------------------------------------------------------

fn section_header(title: &'static str, description: &'static str) -> impl IntoElement {
    let colors = get_theme_or_default().read().colors.clone();
    rect()
        .width(Size::fill())
        .vertical()
        .spacing(2.)
        .child(
            label()
                .color(colors.text_primary)
                .font_size(14.)
                .font_weight(FontWeight::BOLD)
                .text(title),
        )
        .child(
            label()
                .color(colors.text_placeholder)
                .font_size(11.)
                .text(description),
        )
}

fn sidebar_tab_button(
    colors: &ColorsSheet,
    icon_name: &'static str,
    title: &'static str,
    selected: bool,
    on_press: impl FnMut(Event<PressEventData>) + 'static,
) -> impl IntoElement {
    Button::new()
        .flat()
        .expanded()
        .height(Size::px(32.))
        .padding(Gaps::new(0., 10., 0., 10.))
        .background(if selected {
            colors.surface_tertiary
        } else {
            Color::TRANSPARENT
        })
        .corner_radius(CornerRadius::new_all(4.))
        .on_press(on_press)
        .child(
            rect()
                .width(Size::fill())
                .horizontal()
                .spacing(8.)
                .cross_align(Alignment::center())
                .child(
                    SvgViewer::new((icon_name, icon(icon_name)))
                        .color(if selected {
                            colors.primary
                        } else {
                            colors.text_secondary
                        })
                        .width(Size::px(14.))
                        .height(Size::px(14.)),
                )
                .child(
                    label()
                        .color(if selected {
                            colors.text_primary
                        } else {
                            colors.text_secondary
                        })
                        .font_size(11.5)
                        .font_weight(if selected {
                            FontWeight::BOLD
                        } else {
                            FontWeight::NORMAL
                        })
                        .text(title),
                ),
        )
}

fn toggle_card(
    colors: &ColorsSheet,
    title: &'static str,
    description: &'static str,
    active: bool,
    on_press: impl FnMut(Event<PressEventData>) + 'static,
) -> impl IntoElement {
    rect()
        .width(Size::flex(1.))
        .height(Size::px(52.))
        .background(colors.surface_secondary)
        .border(
            Border::new()
                .fill(if active {
                    colors.border_focus
                } else {
                    colors.border
                })
                .width(1.),
        )
        .corner_radius(CornerRadius::new_all(6.))
        .padding(Gaps::new(6., 10., 6., 10.))
        .child(
            Button::new()
                .flat()
                .expanded()
                .height(Size::fill())
                .on_press(on_press)
                .child(
                    rect()
                        .width(Size::fill())
                        .height(Size::fill())
                        .horizontal()
                        .main_align(Alignment::space_between())
                        .cross_align(Alignment::center())
                        .child(
                            rect()
                                .vertical()
                                .spacing(2.)
                                .child(
                                    label()
                                        .color(colors.text_primary)
                                        .font_size(11.)
                                        .font_weight(FontWeight::MEDIUM)
                                        .text(title),
                                )
                                .child(
                                    label()
                                        .color(colors.text_placeholder)
                                        .font_size(9.5)
                                        .text(description),
                                ),
                        )
                        .child(
                            rect()
                                .padding(Gaps::new(2., 6., 2., 6.))
                                .corner_radius(CornerRadius::new_all(4.))
                                .background(if active {
                                    colors.primary
                                } else {
                                    colors.surface_tertiary
                                })
                                .child(
                                    label()
                                        .color(if active {
                                            colors.text_inverse
                                        } else {
                                            colors.text_secondary
                                        })
                                        .font_size(9.5)
                                        .font_weight(FontWeight::BOLD)
                                        .text(if active { "ON" } else { "OFF" }),
                                ),
                        ),
                ),
        )
}

fn theme_card(
    colors: &ColorsSheet,
    desc: ThemeDescriptor,
    active: bool,
    on_press: impl FnMut(Event<PressEventData>) + 'static,
) -> impl IntoElement {
    rect()
        .width(Size::fill())
        .height(Size::px(54.))
        .background(colors.surface_secondary)
        .border(
            Border::new()
                .fill(if active {
                    colors.primary
                } else {
                    colors.border
                })
                .width(if active { 1.5 } else { 1. }),
        )
        .corner_radius(CornerRadius::new_all(6.))
        .padding(Gaps::new(6., 10., 6., 10.))
        .child(
            Button::new()
                .flat()
                .expanded()
                .height(Size::fill())
                .on_press(on_press)
                .child(
                    rect()
                        .width(Size::fill())
                        .height(Size::fill())
                        .horizontal()
                        .main_align(Alignment::space_between())
                        .cross_align(Alignment::center())
                        .child(
                            rect()
                                .vertical()
                                .spacing(2.)
                                .child(
                                    rect()
                                        .horizontal()
                                        .spacing(6.)
                                        .cross_align(Alignment::center())
                                        .child(
                                            label()
                                                .color(colors.text_primary)
                                                .font_size(11.5)
                                                .font_weight(FontWeight::MEDIUM)
                                                .text(desc.name.clone()),
                                        )
                                        .maybe(desc.is_custom, |r| {
                                            r.child(
                                                rect()
                                                    .padding(Gaps::new(1., 4., 1., 4.))
                                                    .corner_radius(CornerRadius::new_all(2.))
                                                    .background(colors.surface_tertiary)
                                                    .child(
                                                        label()
                                                            .color(colors.primary)
                                                            .font_size(8.5)
                                                            .text("CUSTOM"),
                                                    ),
                                            )
                                        }),
                                )
                                .child(
                                    label()
                                        .color(colors.text_placeholder)
                                        .font_size(9.5)
                                        .text(desc.description.clone()),
                                ),
                        )
                        // COLOR SWATCH PILLS
                        .child(
                            rect()
                                .horizontal()
                                .spacing(4.)
                                .cross_align(Alignment::center())
                                .child(
                                    rect()
                                        .width(Size::px(12.))
                                        .height(Size::px(12.))
                                        .corner_radius(CornerRadius::new_all(6.))
                                        .background(desc.preview_bg)
                                        .border(Border::new().fill(colors.border).width(1.)),
                                )
                                .child(
                                    rect()
                                        .width(Size::px(12.))
                                        .height(Size::px(12.))
                                        .corner_radius(CornerRadius::new_all(6.))
                                        .background(desc.preview_surface),
                                )
                                .child(
                                    rect()
                                        .width(Size::px(12.))
                                        .height(Size::px(12.))
                                        .corner_radius(CornerRadius::new_all(6.))
                                        .background(desc.preview_accent),
                                )
                                .maybe(active, |r| {
                                    r.child(
                                        SvgViewer::new(("check", icon("check")))
                                            .color(colors.primary)
                                            .width(Size::px(14.))
                                            .height(Size::px(14.)),
                                    )
                                }),
                        ),
                ),
        )
}

#[derive(PartialEq)]
struct ExtensionCard {
    record: ExtensionRecord,
}

impl Component for ExtensionCard {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let m = &self.record.manifest;
        let is_ready = self.record.status == ExtensionStatus::Ready;

        rect()
            .width(Size::fill())
            .background(colors.surface_secondary)
            .border(Border::new().fill(colors.border).width(1.))
            .corner_radius(CornerRadius::new_all(6.))
            .padding(Gaps::new_all(10.))
            .vertical()
            .spacing(4.)
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
                                    .color(colors.text_primary)
                                    .font_size(12.)
                                    .font_weight(FontWeight::BOLD)
                                    .text(m.name.clone()),
                            )
                            .child(
                                label()
                                    .color(colors.text_placeholder)
                                    .font_size(10.)
                                    .text(format!("v{}", m.version)),
                            ),
                    )
                    .child(
                        rect()
                            .horizontal()
                            .spacing(4.)
                            .child(
                                rect()
                                    .padding(Gaps::new(1., 5., 1., 5.))
                                    .corner_radius(CornerRadius::new_all(3.))
                                    .background(colors.surface_tertiary)
                                    .child(
                                        label()
                                            .color(colors.text_secondary)
                                            .font_size(9.)
                                            .text(match m.runtime.kind {
                                                prumo_extension_sdk::manifest::RuntimeKind::Declarative => "Declarative",
                                                prumo_extension_sdk::manifest::RuntimeKind::Process => "Process",
                                            }),
                                    ),
                            )
                            .child(
                                rect()
                                    .padding(Gaps::new(1., 5., 1., 5.))
                                    .corner_radius(CornerRadius::new_all(3.))
                                    .background(if is_ready {
                                        Color::from_argb(40, 100, 200, 100)
                                    } else {
                                        Color::from_argb(40, 255, 100, 100)
                                    })
                                    .child(
                                        label()
                                            .color(if is_ready {
                                                colors.success
                                            } else {
                                                colors.error
                                            })
                                            .font_size(9.)
                                            .text(if is_ready { "Ready" } else { "Deferred" }),
                                    ),
                            ),
                    ),
            )
            .child(
                label()
                    .color(colors.text_secondary)
                    .font_size(10.5)
                    .text(m.description.clone()),
            )
            .child(
                label()
                    .color(colors.text_placeholder)
                    .font_size(9.5)
                    .text(format!(
                        "Publisher: {} · ID: {} · Path: {}",
                        m.publisher,
                        m.id,
                        self.record.root.display()
                    )),
            )
    }
}

#[derive(PartialEq)]
struct PresetButton {
    label: &'static str,
    provider: &'static str,
    base_url: &'static str,
    default_model: &'static str,
    provider_state: State<String>,
    base_url_state: State<String>,
    model_state: State<String>,
}

impl Component for PresetButton {
    fn render(&self) -> impl IntoElement {
        let colors = get_theme_or_default().read().colors.clone();
        let mut provider_state = self.provider_state;
        let mut base_url_state = self.base_url_state;
        let mut model_state = self.model_state;
        let p = self.provider;
        let u = self.base_url;
        let m = self.default_model;
        let is_selected = provider_state.read().as_str() == p;

        Button::new()
            .flat()
            .compact()
            .height(Size::px(22.))
            .padding(Gaps::new(0., 8., 0., 8.))
            .background(if is_selected {
                colors.surface_tertiary
            } else {
                Color::TRANSPARENT
            })
            .on_press(move |_| {
                provider_state.set(p.to_string());
                base_url_state.set(u.to_string());
                if !m.is_empty() {
                    model_state.set(m.to_string());
                }
            })
            .child(
                label()
                    .color(if is_selected {
                        colors.text_primary
                    } else {
                        colors.text_secondary
                    })
                    .font_size(10.5)
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
            .spacing(4.)
            .child(
                label()
                    .color(colors.text_secondary)
                    .font_size(11.)
                    .text(self.label),
            )
            .child(
                Input::new(self.value.into_writable())
                    .placeholder(self.placeholder)
                    .width(Size::fill()),
            )
    }
}

// -----------------------------------------------------------------------------
// WORKERS & ACTIONS
// -----------------------------------------------------------------------------

fn load_models(
    mut state: State<AppState>,
    client: PrumoClient,
    provider: String,
    api_key: Option<String>,
    base_url: Option<String>,
) {
    if provider.trim().is_empty() {
        state
            .write()
            .show_notice(NoticeTone::Error, "Provider is required");
        return;
    }
    spawn(async move {
        let result = tokio::task::spawn_blocking(move || {
            client.models_with_options(&provider, api_key.as_deref(), base_url.as_deref())
        })
        .await;
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

#[allow(clippy::too_many_arguments)]
fn save_all_settings(
    mut state: State<AppState>,
    mut current_theme: State<Theme>,
    theme_name: String,
    provider: String,
    model: String,
    api_key: String,
    base_url: String,
    acf_token: String,
    socket_path: String,
    terminal_shell: String,
    poll_interval: String,
    font_size: String,
    tab_size: String,
    show_whitespace: bool,
    line_numbers: bool,
    word_wrap: bool,
) {
    let poll_interval_seconds = match poll_interval.trim().parse::<u64>() {
        Ok(v) => v,
        Err(_) => {
            state
                .write()
                .show_notice(NoticeTone::Error, "Poll interval must be a number");
            return;
        }
    };
    let parsed_font_size = font_size.trim().parse::<f32>().unwrap_or(13.0);
    let parsed_tab_size = tab_size.trim().parse::<u32>().unwrap_or(4);

    let existing_custom_themes = state.read().config.custom_themes.clone();
    let existing_extensions = state.read().config.extensions_enabled.clone();

    let config = ViewerConfig {
        version: 1,
        theme: theme_name.clone(),
        provider,
        model,
        api_key: (!api_key.trim().is_empty()).then_some(api_key),
        base_url: (!base_url.trim().is_empty()).then_some(base_url),
        acf_token: (!acf_token.trim().is_empty()).then_some(acf_token),
        socket_path: (!socket_path.trim().is_empty()).then_some(socket_path),
        terminal_shell: (!terminal_shell.trim().is_empty()).then_some(terminal_shell),
        show_whitespace,
        poll_interval_seconds,
        font_size: parsed_font_size,
        tab_size: parsed_tab_size,
        line_numbers,
        word_wrap,
        custom_themes: existing_custom_themes.clone(),
        extensions_enabled: existing_extensions,
    };

    match config.validate().and_then(|()| config.save()) {
        Ok(path) => {
            let next_theme = resolve_theme(&theme_name, &existing_custom_themes, None);
            current_theme.set(next_theme);
            let mut app_state = state.write();
            app_state.config = config;
            app_state.config_open = false;
            app_state.show_notice(
                NoticeTone::Success,
                format!("Settings saved to {}", path.display()),
            );
        }
        Err(error) => {
            state
                .write()
                .show_notice(NoticeTone::Error, format!("Settings save failed: {error}"));
        }
    }
}
