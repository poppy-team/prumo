use freya::code_editor::{
    CodeEditorThemeExt, EditorSyntaxTheme, EditorTheme, EditorThemePreference,
};
use freya::prelude::*;
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

fn rgb(r: u8, g: u8, b: u8) -> Color {
    Color::from_rgb(r, g, b)
}

fn rgba(r: u8, g: u8, b: u8, a: u8) -> Color {
    Color::from_argb(a, r, g, b)
}

pub fn parse_color(value: &str) -> Result<Color, String> {
    let value = value.strip_prefix('#').unwrap_or(value);
    if value.len() != 6 && value.len() != 8 {
        return Err(format!("invalid theme color: {value}"));
    }
    let channel = |start: usize| {
        u8::from_str_radix(&value[start..start + 2], 16)
            .map_err(|_| format!("invalid theme color: {value}"))
    };
    if value.len() == 6 {
        Ok(Color::from_rgb(channel(0)?, channel(2)?, channel(4)?))
    } else {
        Ok(Color::from_argb(
            channel(0)?,
            channel(2)?,
            channel(4)?,
            channel(6)?,
        ))
    }
}

pub fn apply_extension_tokens(
    theme: &mut Theme,
    tokens: &std::collections::BTreeMap<String, String>,
) -> Result<(), String> {
    let mut parsed = Vec::with_capacity(tokens.len());
    for (name, value) in tokens {
        let normalized = match name.as_str() {
            "accent" | "primary" => "accent",
            "background" | "bg" => "background",
            "surface-primary" | "surface_primary" | "surface1" => "surface-primary",
            "surface-secondary" | "surface_secondary" | "surface2" => "surface-secondary",
            "surface-tertiary" | "surface_tertiary" | "surface3" => "surface-tertiary",
            "border" => "border",
            "border-focus" | "border_focus" => "border-focus",
            "text-primary" | "text_primary" | "text" => "text-primary",
            "text-secondary" | "text_secondary" => "text-secondary",
            "text-placeholder" | "text_placeholder" => "text-placeholder",
            "success" => "success",
            "warning" => "warning",
            "error" => "error",
            _ => return Err(format!("unknown theme token: {name}")),
        };
        parsed.push((normalized, parse_color(value)?));
    }
    for (name, color) in parsed {
        match name {
            "accent" => theme.colors.primary = color,
            "background" => theme.colors.background = color,
            "surface-primary" => theme.colors.surface_primary = color,
            "surface-secondary" => theme.colors.surface_secondary = color,
            "surface-tertiary" => theme.colors.surface_tertiary = color,
            "border" => theme.colors.border = color,
            "border-focus" => theme.colors.border_focus = color,
            "text-primary" => theme.colors.text_primary = color,
            "text-secondary" => theme.colors.text_secondary = color,
            "text-placeholder" => theme.colors.text_placeholder = color,
            "success" => theme.colors.success = color,
            "warning" => theme.colors.warning = color,
            "error" => theme.colors.error = color,
            _ => return Err(format!("unknown theme token: {name}")),
        }
    }
    Ok(())
}

pub fn dark_syntax() -> EditorSyntaxTheme {
    EditorSyntaxTheme {
        text: rgb(0xD4, 0xD8, 0xE0),
        whitespace: rgba(0xA8, 0xAF, 0xBD, 0x3D),
        attribute: rgb(0x78, 0xA9, 0xE8),
        boolean: rgb(0xC9, 0x9C, 0x6A),
        comment: rgb(0x6B, 0x72, 0x7E),
        constant: rgb(0xDF, 0xC4, 0x88),
        constructor: rgb(0x78, 0xA9, 0xE8),
        escape: rgb(0x8D, 0x95, 0xA3),
        function: rgb(0x78, 0xA9, 0xE8),
        function_macro: rgb(0xBC, 0x82, 0xD6),
        function_method: rgb(0x78, 0xA9, 0xE8),
        keyword: rgb(0xBC, 0x82, 0xD6),
        label: rgb(0x78, 0xA9, 0xE8),
        module: rgb(0xDC, 0xE0, 0xE6),
        number: rgb(0xC9, 0x9C, 0x6A),
        operator: rgb(0x72, 0xB5, 0xC2),
        property: rgb(0xD2, 0x7D, 0x82),
        punctuation: rgb(0xB8, 0xBF, 0xCB),
        punctuation_bracket: rgb(0xC3, 0xC9, 0xD3),
        punctuation_delimiter: rgb(0xC3, 0xC9, 0xD3),
        punctuation_special: rgb(0xB6, 0x63, 0x58),
        string: rgb(0xA7, 0xC8, 0x8B),
        string_escape: rgb(0x8D, 0x95, 0xA3),
        string_special: rgb(0xC9, 0x9C, 0x6A),
        tag: rgb(0x78, 0xA9, 0xE8),
        text_literal: rgb(0xA7, 0xC8, 0x8B),
        text_reference: rgb(0xD4, 0xD8, 0xE0),
        text_title: rgb(0xD2, 0x7D, 0x82),
        text_uri: rgb(0x72, 0xB5, 0xC2),
        text_emphasis: rgb(0x78, 0xA9, 0xE8),
        type_: rgb(0x72, 0xB5, 0xC2),
        variable: rgb(0xD4, 0xD8, 0xE0),
        variable_builtin: rgb(0xD2, 0x7D, 0x82),
        variable_parameter: rgb(0xD2, 0x7D, 0x82),
    }
}

pub fn light_syntax() -> EditorSyntaxTheme {
    EditorSyntaxTheme {
        text: rgb(0x2B, 0x2D, 0x32),
        whitespace: rgba(0x4A, 0x4E, 0x58, 0x30),
        attribute: rgb(0x5C, 0x78, 0xE2),
        boolean: rgb(0xA8, 0x6D, 0x25),
        comment: rgb(0x8A, 0x8E, 0x98),
        constant: rgb(0xA8, 0x6D, 0x25),
        constructor: rgb(0x5C, 0x78, 0xE2),
        escape: rgb(0x75, 0x78, 0x82),
        function: rgb(0x5C, 0x78, 0xE2),
        function_macro: rgb(0x8A, 0x4F, 0xAD),
        function_method: rgb(0x5C, 0x78, 0xE2),
        keyword: rgb(0x8A, 0x4F, 0xAD),
        label: rgb(0x5C, 0x78, 0xE2),
        module: rgb(0x2B, 0x2D, 0x32),
        number: rgb(0xA8, 0x6D, 0x25),
        operator: rgb(0x3A, 0x82, 0xB7),
        property: rgb(0xCB, 0x5C, 0x50),
        punctuation: rgb(0x4C, 0x50, 0x59),
        punctuation_bracket: rgb(0x4C, 0x50, 0x59),
        punctuation_delimiter: rgb(0x4C, 0x50, 0x59),
        punctuation_special: rgb(0xB0, 0x55, 0x4B),
        string: rgb(0x5E, 0x91, 0x50),
        string_escape: rgb(0x75, 0x78, 0x82),
        string_special: rgb(0xA8, 0x6D, 0x25),
        tag: rgb(0x5C, 0x78, 0xE2),
        text_literal: rgb(0x5E, 0x91, 0x50),
        text_reference: rgb(0x2B, 0x2D, 0x32),
        text_title: rgb(0xCB, 0x5C, 0x50),
        text_uri: rgb(0x3A, 0x82, 0xB7),
        text_emphasis: rgb(0x5C, 0x78, 0xE2),
        type_: rgb(0x3A, 0x82, 0xB7),
        variable: rgb(0x2B, 0x2D, 0x32),
        variable_builtin: rgb(0xCB, 0x5C, 0x50),
        variable_parameter: rgb(0xCB, 0x5C, 0x50),
    }
}

fn dark_editor() -> EditorTheme {
    EditorTheme {
        background: rgb(0x1B, 0x1E, 0x24),
        gutter_selected: rgb(0xAF, 0xB7, 0xC3),
        gutter_unselected: rgb(0x5E, 0x66, 0x72),
        line_selected_background: rgb(0x25, 0x2A, 0x33),
        cursor: rgb(0x78, 0xA9, 0xE8),
        highlight: rgb(0x30, 0x38, 0x45),
        text: rgb(0xD4, 0xD8, 0xE0),
        whitespace: rgba(0xA8, 0xAF, 0xBD, 0x3D),
    }
}

fn light_editor() -> EditorTheme {
    EditorTheme {
        background: rgb(0xFC, 0xFC, 0xFD),
        gutter_selected: rgb(0x4A, 0x4E, 0x58),
        gutter_unselected: rgb(0xA0, 0xA4, 0xAC),
        line_selected_background: rgb(0xF2, 0xF4, 0xF7),
        cursor: rgb(0x3A, 0x6F, 0xD8),
        highlight: rgb(0xDD, 0xE7, 0xFA),
        text: rgb(0x2B, 0x2D, 0x32),
        whitespace: rgba(0x4A, 0x4E, 0x58, 0x30),
    }
}

pub fn dark() -> Theme {
    let mut theme = dark_theme();
    theme.colors = ColorsSheet {
        primary: rgb(0x78, 0xA9, 0xE8),
        secondary: rgb(0x72, 0xB5, 0xC2),
        tertiary: rgb(0xBC, 0x82, 0xD6),
        success: rgb(0xA7, 0xC8, 0x8B),
        warning: rgb(0xDF, 0xC4, 0x88),
        error: rgb(0xD2, 0x7D, 0x82),
        info: rgb(0x78, 0xA9, 0xE8),
        background: rgb(0x1B, 0x1E, 0x24),
        surface_primary: rgb(0x21, 0x24, 0x2B),
        surface_secondary: rgb(0x28, 0x2C, 0x35),
        surface_tertiary: rgb(0x32, 0x38, 0x44),
        surface_inverse: rgb(0x26, 0x2A, 0x32),
        surface_inverse_secondary: rgb(0x3D, 0x5D, 0x91),
        surface_inverse_tertiary: rgb(0x46, 0x4E, 0x5C),
        border: rgb(0x2D, 0x32, 0x3B),
        border_focus: rgb(0x78, 0xA9, 0xE8),
        border_disabled: rgb(0x3A, 0x40, 0x4A),
        text_primary: rgb(0xDC, 0xE0, 0xE6),
        text_secondary: rgb(0xA8, 0xAF, 0xBD),
        text_placeholder: rgb(0x79, 0x81, 0x8E),
        text_inverse: rgb(0x1B, 0x1E, 0x24),
        text_highlight: rgb(0x78, 0xA9, 0xE8),
        focus: rgb(0x78, 0xA9, 0xE8),
        active: rgb(0x32, 0x38, 0x44),
        disabled: rgb(0x5E, 0x66, 0x72),
        overlay: rgba(0x00, 0x00, 0x00, 0x88),
        shadow: rgba(0x00, 0x00, 0x00, 0x66),
    };
    theme = theme.with_dark_code_editor();
    theme.set("code_editor", EditorThemePreference::from(dark_editor()));
    theme.set("code_editor_syntax", dark_syntax());
    theme.set(
        "resizable_handle",
        ResizableHandleThemePreference {
            background: Preference::Specific(rgb(0x1B, 0x1E, 0x24)),
            hover_background: Preference::Specific(rgb(0x78, 0xA9, 0xE8)),
            corner_radius: Preference::Specific(CornerRadius::new_all(1.)),
        },
    );
    theme
}

pub fn light() -> Theme {
    let mut theme = light_theme();
    theme.colors = ColorsSheet {
        primary: rgb(0x5C, 0x78, 0xE2),
        secondary: rgb(0x3A, 0x82, 0xB7),
        tertiary: rgb(0x8A, 0x4F, 0xAD),
        success: rgb(0x5E, 0x91, 0x50),
        warning: rgb(0xA8, 0x6D, 0x25),
        error: rgb(0xCB, 0x5C, 0x50),
        info: rgb(0x5C, 0x78, 0xE2),
        background: rgb(0xF7, 0xF8, 0xFA),
        surface_primary: rgb(0xF1, 0xF3, 0xF5),
        surface_secondary: rgb(0xE8, 0xEB, 0xEF),
        surface_tertiary: rgb(0xDD, 0xE1, 0xE6),
        surface_inverse: rgb(0xE5, 0xE8, 0xEC),
        surface_inverse_secondary: rgb(0x5C, 0x78, 0xE2),
        surface_inverse_tertiary: rgb(0xC4, 0xC9, 0xD0),
        border: rgb(0xD8, 0xDC, 0xE2),
        border_focus: rgb(0x5C, 0x78, 0xE2),
        border_disabled: rgb(0xC7, 0xCC, 0xD3),
        text_primary: rgb(0x2B, 0x2D, 0x32),
        text_secondary: rgb(0x5D, 0x62, 0x6B),
        text_placeholder: rgb(0x8A, 0x8E, 0x98),
        text_inverse: rgb(0xFF, 0xFF, 0xFF),
        text_highlight: rgb(0x5C, 0x78, 0xE2),
        focus: rgb(0x5C, 0x78, 0xE2),
        active: rgb(0xDD, 0xE1, 0xE6),
        disabled: rgb(0xA0, 0xA4, 0xAC),
        overlay: rgba(0x20, 0x24, 0x2A, 0x44),
        shadow: rgba(0x20, 0x24, 0x2A, 0x22),
    };
    theme = theme.with_light_code_editor();
    theme.set("code_editor", EditorThemePreference::from(light_editor()));
    theme.set("code_editor_syntax", light_syntax());
    theme.set(
        "resizable_handle",
        ResizableHandleThemePreference {
            background: Preference::Specific(rgb(0xF7, 0xF8, 0xFA)),
            hover_background: Preference::Specific(rgb(0x5C, 0x78, 0xE2)),
            corner_radius: Preference::Specific(CornerRadius::new_all(1.)),
        },
    );
    theme
}

pub fn nord() -> Theme {
    let mut theme = dark();
    theme.name = "nord";
    theme.colors.primary = rgb(0x88, 0xC0, 0xD0);
    theme.colors.secondary = rgb(0x81, 0xA1, 0xC1);
    theme.colors.tertiary = rgb(0xB4, 0x8E, 0xAD);
    theme.colors.success = rgb(0xA3, 0xBE, 0x8C);
    theme.colors.warning = rgb(0xEB, 0xCB, 0x8B);
    theme.colors.error = rgb(0xBF, 0x61, 0x6A);
    theme.colors.info = rgb(0x88, 0xC0, 0xD0);
    theme.colors.background = rgb(0x2E, 0x34, 0x40);
    theme.colors.surface_primary = rgb(0x3B, 0x42, 0x52);
    theme.colors.surface_secondary = rgb(0x43, 0x4C, 0x5E);
    theme.colors.surface_tertiary = rgb(0x4C, 0x56, 0x6A);
    theme.colors.border = rgb(0x43, 0x4C, 0x5E);
    theme.colors.border_focus = rgb(0x88, 0xC0, 0xD0);
    theme.colors.text_primary = rgb(0xEC, 0xEF, 0xF4);
    theme.colors.text_secondary = rgb(0xD8, 0xDE, 0xE9);
    theme.colors.text_placeholder = rgb(0x4C, 0x56, 0x6A);
    theme
}

pub fn dracula() -> Theme {
    let mut theme = dark();
    theme.name = "dracula";
    theme.colors.primary = rgb(0xBD, 0x93, 0xF9);
    theme.colors.secondary = rgb(0x8B, 0xEA, 0xFD);
    theme.colors.tertiary = rgb(0xFF, 0x79, 0xC6);
    theme.colors.success = rgb(0x50, 0xFA, 0x7B);
    theme.colors.warning = rgb(0xFF, 0xB8, 0x6C);
    theme.colors.error = rgb(0xFF, 0x55, 0x55);
    theme.colors.info = rgb(0xBD, 0x93, 0xF9);
    theme.colors.background = rgb(0x28, 0x2A, 0x36);
    theme.colors.surface_primary = rgb(0x21, 0x22, 0x2C);
    theme.colors.surface_secondary = rgb(0x19, 0x1A, 0x21);
    theme.colors.surface_tertiary = rgb(0x44, 0x47, 0x5A);
    theme.colors.border = rgb(0x44, 0x47, 0x5A);
    theme.colors.border_focus = rgb(0xBD, 0x93, 0xF9);
    theme.colors.text_primary = rgb(0xF8, 0xF8, 0xF2);
    theme.colors.text_secondary = rgb(0x62, 0x72, 0xA4);
    theme.colors.text_placeholder = rgb(0x62, 0x72, 0xA4);
    theme
}

pub fn monokai() -> Theme {
    let mut theme = dark();
    theme.name = "monokai";
    theme.colors.primary = rgb(0xFF, 0xD8, 0x66);
    theme.colors.secondary = rgb(0x78, 0xDC, 0xE8);
    theme.colors.tertiary = rgb(0xAB, 0x9D, 0xF2);
    theme.colors.success = rgb(0xA9, 0xDC, 0x76);
    theme.colors.warning = rgb(0xFC, 0x98, 0x67);
    theme.colors.error = rgb(0xFF, 0x61, 0x88);
    theme.colors.info = rgb(0x78, 0xDC, 0xE8);
    theme.colors.background = rgb(0x2D, 0x2A, 0x2E);
    theme.colors.surface_primary = rgb(0x22, 0x1F, 0x22);
    theme.colors.surface_secondary = rgb(0x19, 0x18, 0x1A);
    theme.colors.surface_tertiary = rgb(0x40, 0x3E, 0x41);
    theme.colors.border = rgb(0x40, 0x3E, 0x41);
    theme.colors.border_focus = rgb(0xFF, 0xD8, 0x66);
    theme.colors.text_primary = rgb(0xFC, 0xFC, 0xFA);
    theme.colors.text_secondary = rgb(0x93, 0x92, 0x93);
    theme.colors.text_placeholder = rgb(0x72, 0x70, 0x72);
    theme
}

pub fn github_dark() -> Theme {
    let mut theme = dark();
    theme.name = "github-dark";
    theme.colors.primary = rgb(0x58, 0xA6, 0xFF);
    theme.colors.secondary = rgb(0x79, 0xC0, 0xFF);
    theme.colors.tertiary = rgb(0xD2, 0xA8, 0xFF);
    theme.colors.success = rgb(0x3F, 0xB9, 0x50);
    theme.colors.warning = rgb(0xD2, 0x99, 0x22);
    theme.colors.error = rgb(0xF8, 0x51, 0x49);
    theme.colors.info = rgb(0x58, 0xA6, 0xFF);
    theme.colors.background = rgb(0x0D, 0x11, 0x17);
    theme.colors.surface_primary = rgb(0x16, 0x1B, 0x22);
    theme.colors.surface_secondary = rgb(0x21, 0x26, 0x2D);
    theme.colors.surface_tertiary = rgb(0x30, 0x36, 0x3D);
    theme.colors.border = rgb(0x30, 0x36, 0x3D);
    theme.colors.border_focus = rgb(0x58, 0xA6, 0xFF);
    theme.colors.text_primary = rgb(0xC9, 0xD1, 0xD9);
    theme.colors.text_secondary = rgb(0x8B, 0x94, 0x9E);
    theme.colors.text_placeholder = rgb(0x6E, 0x76, 0x81);
    theme
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct CustomTheme {
    pub id: String,
    pub name: String,
    #[serde(default = "default_appearance_dark")]
    pub appearance: String,
    pub tokens: BTreeMap<String, String>,
}

fn default_appearance_dark() -> String {
    "dark".to_string()
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ThemeDescriptor {
    pub id: String,
    pub name: String,
    pub description: String,
    pub appearance: String,
    pub preview_bg: Color,
    pub preview_surface: Color,
    pub preview_accent: Color,
    pub is_custom: bool,
}

pub fn available_themes(custom_themes: &[CustomTheme]) -> Vec<ThemeDescriptor> {
    let mut themes = vec![
        ThemeDescriptor {
            id: "system".to_string(),
            name: "System Default".to_string(),
            description: "Follows system theme preference".to_string(),
            appearance: "auto".to_string(),
            preview_bg: rgb(0x1B, 0x1E, 0x24),
            preview_surface: rgb(0x28, 0x2C, 0x35),
            preview_accent: rgb(0x78, 0xA9, 0xE8),
            is_custom: false,
        },
        ThemeDescriptor {
            id: "dark".to_string(),
            name: "Prumo Dark".to_string(),
            description: "Classic high-contrast dark theme".to_string(),
            appearance: "dark".to_string(),
            preview_bg: rgb(0x1B, 0x1E, 0x24),
            preview_surface: rgb(0x28, 0x2C, 0x35),
            preview_accent: rgb(0x78, 0xA9, 0xE8),
            is_custom: false,
        },
        ThemeDescriptor {
            id: "light".to_string(),
            name: "Prumo Light".to_string(),
            description: "Clean modern light theme".to_string(),
            appearance: "light".to_string(),
            preview_bg: rgb(0xF7, 0xF8, 0xFA),
            preview_surface: rgb(0xE8, 0xEB, 0xEF),
            preview_accent: rgb(0x5C, 0x78, 0xE2),
            is_custom: false,
        },
        ThemeDescriptor {
            id: "nord".to_string(),
            name: "Nord Frost".to_string(),
            description: "Arctic, north-bluish palette".to_string(),
            appearance: "dark".to_string(),
            preview_bg: rgb(0x2E, 0x34, 0x40),
            preview_surface: rgb(0x3B, 0x42, 0x52),
            preview_accent: rgb(0x88, 0xC0, 0xD0),
            is_custom: false,
        },
        ThemeDescriptor {
            id: "dracula".to_string(),
            name: "Dracula".to_string(),
            description: "Vibrant purple and cyan dark theme".to_string(),
            appearance: "dark".to_string(),
            preview_bg: rgb(0x28, 0x2A, 0x36),
            preview_surface: rgb(0x21, 0x22, 0x2C),
            preview_accent: rgb(0xBD, 0x93, 0xF9),
            is_custom: false,
        },
        ThemeDescriptor {
            id: "monokai".to_string(),
            name: "Monokai Pro".to_string(),
            description: "Warm high-contrast palette".to_string(),
            appearance: "dark".to_string(),
            preview_bg: rgb(0x2D, 0x2A, 0x2E),
            preview_surface: rgb(0x22, 0x1F, 0x22),
            preview_accent: rgb(0xFF, 0xD8, 0x66),
            is_custom: false,
        },
        ThemeDescriptor {
            id: "github-dark".to_string(),
            name: "GitHub Dark".to_string(),
            description: "Modern GitHub dark palette".to_string(),
            appearance: "dark".to_string(),
            preview_bg: rgb(0x0D, 0x11, 0x17),
            preview_surface: rgb(0x16, 0x1B, 0x22),
            preview_accent: rgb(0x58, 0xA6, 0xFF),
            is_custom: false,
        },
    ];

    for custom in custom_themes {
        let accent = custom
            .tokens
            .get("accent")
            .or_else(|| custom.tokens.get("primary"))
            .and_then(|c| parse_color(c).ok())
            .unwrap_or(rgb(0x78, 0xA9, 0xE8));
        let bg = custom
            .tokens
            .get("background")
            .or_else(|| custom.tokens.get("bg"))
            .and_then(|c| parse_color(c).ok())
            .unwrap_or(rgb(0x1B, 0x1E, 0x24));
        let surface = custom
            .tokens
            .get("surface-primary")
            .or_else(|| custom.tokens.get("surface_primary"))
            .and_then(|c| parse_color(c).ok())
            .unwrap_or(rgb(0x28, 0x2C, 0x35));

        themes.push(ThemeDescriptor {
            id: custom.id.clone(),
            name: custom.name.clone(),
            description: format!("Custom user theme ({})", custom.appearance),
            appearance: custom.appearance.clone(),
            preview_bg: bg,
            preview_surface: surface,
            preview_accent: accent,
            is_custom: true,
        });
    }

    themes
}

pub fn resolve_theme(
    theme_id: &str,
    custom_themes: &[CustomTheme],
    preferred: Option<PreferredTheme>,
) -> Theme {
    match theme_id {
        "light" => light(),
        "dark" => dark(),
        "nord" => nord(),
        "dracula" => dracula(),
        "monokai" => monokai(),
        "github-dark" => github_dark(),
        "system" => match preferred.unwrap_or(PreferredTheme::Dark) {
            PreferredTheme::Light => light(),
            PreferredTheme::Dark => dark(),
        },
        custom_id => {
            if let Some(custom) = custom_themes.iter().find(|t| t.id == custom_id) {
                let mut base = if custom.appearance == "light" {
                    light()
                } else {
                    dark()
                };
                base.name = "custom";
                let _ = apply_extension_tokens(&mut base, &custom.tokens);
                base
            } else {
                dark()
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use freya_testing::prelude::TestingRunner;

    #[test]
    fn parses_hex_colors() {
        assert_eq!(parse_color("#1B1E24").unwrap(), rgb(0x1B, 0x1E, 0x24));
        assert_eq!(parse_color("5C78E2").unwrap(), rgb(0x5C, 0x78, 0xE2));
        assert_eq!(
            parse_color("#88000000").unwrap(),
            rgba(0x00, 0x00, 0x00, 0x88)
        );
        assert!(parse_color("#invalid").is_err());
    }

    #[test]
    fn applies_tokens_and_resolves_custom_theme() {
        let (mut runner, _) = TestingRunner::new(
            move || {
                let mut tokens = BTreeMap::new();
                tokens.insert("primary".to_string(), "#FF0077".to_string());
                tokens.insert("background".to_string(), "#0D0221".to_string());
                tokens.insert("surface-primary".to_string(), "#190B3A".to_string());

                let custom = CustomTheme {
                    id: "neon".to_string(),
                    name: "Neon Glow".to_string(),
                    appearance: "dark".to_string(),
                    tokens,
                };

                let resolved = resolve_theme("neon", &[custom], None);
                assert_eq!(resolved.colors.primary, rgb(0xFF, 0x00, 0x77));
                assert_eq!(resolved.colors.background, rgb(0x0D, 0x02, 0x21));
                assert_eq!(resolved.colors.surface_primary, rgb(0x19, 0x0B, 0x3A));
                rect()
            },
            (100., 100.).into(),
            |_| {},
            1.,
        );
        runner.sync_and_update();
    }

    #[test]
    fn loads_all_builtin_presets() {
        let (mut runner, _) = TestingRunner::new(
            move || {
                let presets = ["dark", "light", "nord", "dracula", "monokai", "github-dark"];
                for preset in presets {
                    let theme = resolve_theme(preset, &[], None);
                    assert!(!theme.name.is_empty());
                }
                rect()
            },
            (100., 100.).into(),
            |_| {},
            1.,
        );
        runner.sync_and_update();
    }
}
