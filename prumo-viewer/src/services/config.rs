use crate::theme::CustomTheme;
use serde::{Deserialize, Serialize};
use std::{
    collections::BTreeMap,
    fs,
    io::Write,
    path::{Path, PathBuf},
};

fn default_version() -> u32 {
    1
}

fn default_theme() -> String {
    "system".to_string()
}

fn default_provider() -> String {
    "fake".to_string()
}

fn default_poll_interval() -> u64 {
    2
}

fn default_font_size() -> f32 {
    13.0
}

fn default_tab_size() -> u32 {
    4
}

fn default_line_numbers() -> bool {
    true
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ViewerConfig {
    #[serde(default = "default_version")]
    pub version: u32,
    #[serde(default = "default_theme")]
    pub theme: String,
    #[serde(default = "default_provider")]
    pub provider: String,
    #[serde(default)]
    pub model: String,
    #[serde(default)]
    pub api_key: Option<String>,
    #[serde(default)]
    pub base_url: Option<String>,
    #[serde(default)]
    pub acf_token: Option<String>,
    #[serde(default)]
    pub socket_path: Option<String>,
    #[serde(default)]
    pub terminal_shell: Option<String>,
    #[serde(default)]
    pub show_whitespace: bool,
    #[serde(default = "default_poll_interval")]
    pub poll_interval_seconds: u64,
    #[serde(default = "default_font_size")]
    pub font_size: f32,
    #[serde(default = "default_tab_size")]
    pub tab_size: u32,
    #[serde(default = "default_line_numbers")]
    pub line_numbers: bool,
    #[serde(default)]
    pub word_wrap: bool,
    #[serde(default)]
    pub custom_themes: Vec<CustomTheme>,
    #[serde(default)]
    pub extensions_enabled: BTreeMap<String, bool>,
}

impl Default for ViewerConfig {
    fn default() -> Self {
        Self {
            version: 1,
            theme: "system".to_string(),
            provider: "fake".to_string(),
            model: String::new(),
            api_key: None,
            base_url: None,
            acf_token: None,
            socket_path: None,
            terminal_shell: None,
            show_whitespace: false,
            poll_interval_seconds: 2,
            font_size: 13.0,
            tab_size: 4,
            line_numbers: true,
            word_wrap: false,
            custom_themes: Vec::new(),
            extensions_enabled: BTreeMap::new(),
        }
    }
}

impl ViewerConfig {
    pub fn load() -> Result<Self, String> {
        let path = config_path()?;
        if !path.exists() {
            return Self::from_environment();
        }
        Self::load_from_path(&path)
    }

    pub fn save(&self) -> Result<PathBuf, String> {
        self.validate()?;
        let path = config_path()?;
        self.save_to_path(&path)?;
        Ok(path)
    }

    pub fn from_environment() -> Result<Self, String> {
        let mut config = Self::default();
        if let Some(theme) = env_string("PRUMO_VIEWER_THEME") {
            config.theme = theme;
        }
        if let Some(provider) = env_string("PRUMO_PROVIDER") {
            config.provider = provider;
        }
        if let Some(model) = env_string("PRUMO_MODEL") {
            config.model = model;
        }
        if let Some(api_key) = env_string("PRUMO_MODEL_API_KEY")
            .or_else(|| env_string("OPENAI_API_KEY"))
            .or_else(|| env_string("ANTHROPIC_API_KEY"))
            .or_else(|| env_string("GEMINI_API_KEY"))
        {
            config.api_key = Some(api_key);
        }
        if let Some(base_url) =
            env_string("PRUMO_MODEL_BASE_URL").or_else(|| env_string("ACF_ENDPOINT"))
        {
            config.base_url = Some(base_url);
        }
        if let Some(acf_token) = env_string("ACF_TOKEN").or_else(|| env_string("ANTIGRAVITY_TOKEN"))
        {
            config.acf_token = Some(acf_token);
        }
        if let Some(socket_path) = env_string("PRUMO_SOCKET") {
            config.socket_path = Some(socket_path);
        }
        if let Some(shell) = env_string("PRUMO_TERMINAL_SHELL") {
            config.terminal_shell = Some(shell);
        }
        if let Some(value) = env_string("PRUMO_VIEWER_SHOW_WHITESPACE") {
            config.show_whitespace = value == "1" || value.eq_ignore_ascii_case("true");
        }
        if let Some(value) = env_string("PRUMO_VIEWER_POLL_INTERVAL")
            && let Ok(seconds) = value.parse()
        {
            config.poll_interval_seconds = seconds;
        }
        config.validate()?;
        Ok(config)
    }

    pub fn socket(&self) -> Option<PathBuf> {
        self.socket_path
            .as_ref()
            .filter(|path| !path.trim().is_empty())
            .map(PathBuf::from)
    }

    pub fn shell(&self) -> Option<String> {
        self.terminal_shell.clone().or_else(|| env_string("SHELL"))
    }

    pub fn workspace_config_path(workspace_root: &Path) -> PathBuf {
        workspace_root.join(".prumo/settings.json")
    }

    pub fn to_json_pretty(&self) -> Result<String, String> {
        serde_json::to_string_pretty(self)
            .map_err(|error| format!("failed to encode viewer config: {error}"))
    }

    pub fn from_json_str(content: &str) -> Result<Self, String> {
        let config: Self = serde_json::from_str(content)
            .map_err(|error| format!("invalid viewer config: {error}"))?;
        config.validate()?;
        Ok(config)
    }

    pub fn load_workspace(workspace_root: &Path) -> Result<Option<Self>, String> {
        let path = Self::workspace_config_path(workspace_root);
        if !path.is_file() {
            return Ok(None);
        }
        Self::load_from_path(&path).map(Some)
    }

    pub fn save_workspace(workspace_root: &Path, config: &Self) -> Result<PathBuf, String> {
        config.validate()?;
        let path = Self::workspace_config_path(workspace_root);
        config.save_to_path(&path)?;
        Ok(path)
    }

    pub fn validate(&self) -> Result<(), String> {
        if self.version != 1 {
            return Err(format!(
                "unsupported viewer config version: {}",
                self.version
            ));
        }
        if self.theme.trim().is_empty() {
            return Err("theme cannot be empty".to_string());
        }
        if self.provider.trim().is_empty() {
            return Err("provider cannot be empty".to_string());
        }
        if !(1..=60).contains(&self.poll_interval_seconds) {
            return Err("poll interval must be between 1 and 60 seconds".to_string());
        }
        if self.font_size < 8.0 || self.font_size > 36.0 {
            return Err("font size must be between 8 and 36".to_string());
        }
        if self.tab_size < 1 || self.tab_size > 16 {
            return Err("tab size must be between 1 and 16".to_string());
        }
        Ok(())
    }

    pub fn load_from_path(path: &Path) -> Result<Self, String> {
        let content = fs::read_to_string(path)
            .map_err(|error| format!("failed to read {}: {error}", path.display()))?;
        let config: Self = serde_json::from_str(&content)
            .map_err(|error| format!("invalid viewer config {}: {error}", path.display()))?;
        config.validate()?;
        Ok(config)
    }

    pub fn save_to_path(&self, path: &Path) -> Result<(), String> {
        let parent = path
            .parent()
            .ok_or_else(|| "viewer config has no parent directory".to_string())?;
        fs::create_dir_all(parent)
            .map_err(|error| format!("failed to create {}: {error}", parent.display()))?;
        let temporary_path = path.with_extension("json.tmp");
        let content = serde_json::to_vec_pretty(self)
            .map_err(|error| format!("failed to encode viewer config: {error}"))?;
        let mut file = fs::File::create(&temporary_path).map_err(|error| {
            format!(
                "failed to create temporary viewer config {}: {error}",
                temporary_path.display()
            )
        })?;
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            file.set_permissions(fs::Permissions::from_mode(0o600))
                .map_err(|error| format!("failed to secure viewer config: {error}"))?;
        }
        file.write_all(&content)
            .map_err(|error| format!("failed to write viewer config: {error}"))?;
        file.sync_all()
            .map_err(|error| format!("failed to sync viewer config: {error}"))?;
        fs::rename(&temporary_path, path).map_err(|error| {
            format!(
                "failed to replace viewer config {}: {error}",
                path.display()
            )
        })
    }
}

pub fn config_path() -> Result<PathBuf, String> {
    if let Some(path) = env_string("PRUMO_VIEWER_CONFIG") {
        return Ok(PathBuf::from(path));
    }
    if let Some(path) = env_string("XDG_CONFIG_HOME") {
        return Ok(PathBuf::from(path).join("prumo/viewer.json"));
    }
    if let Some(path) = env_string("APPDATA") {
        return Ok(PathBuf::from(path).join("Prumo/viewer.json"));
    }
    if let Some(home) = env_string("HOME") {
        return Ok(PathBuf::from(home).join(".config/prumo/viewer.json"));
    }
    Err("cannot resolve the viewer config directory".to_string())
}

fn env_string(name: &str) -> Option<String> {
    std::env::var(name)
        .ok()
        .filter(|value| !value.trim().is_empty())
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    #[test]
    fn saves_and_loads_valid_config() {
        let directory = tempdir().unwrap();
        let path = directory.path().join("viewer.json");
        let config = ViewerConfig {
            theme: "dark".to_string(),
            provider: "openai-compat".to_string(),
            model: "model-a".to_string(),
            poll_interval_seconds: 5,
            ..ViewerConfig::default()
        };
        config.save_to_path(&path).unwrap();
        assert_eq!(ViewerConfig::load_from_path(&path).unwrap(), config);
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            assert_eq!(
                fs::metadata(&path).unwrap().permissions().mode() & 0o777,
                0o600
            );
        }
    }

    #[test]
    fn rejects_invalid_provider_and_interval() {
        let mut config = ViewerConfig {
            provider: " ".to_string(),
            ..ViewerConfig::default()
        };
        assert!(config.validate().is_err());
        config.provider = "fake".to_string();
        config.poll_interval_seconds = 0;
        assert!(config.validate().is_err());
    }

    #[test]
    fn serializes_and_deserializes_json_with_editor_and_theme_fields() {
        let mut custom_tokens = BTreeMap::new();
        custom_tokens.insert("accent".to_string(), "#00ffcc".to_string());
        let custom = CustomTheme {
            id: "cyberpunk".to_string(),
            name: "Cyberpunk".to_string(),
            appearance: "dark".to_string(),
            tokens: custom_tokens,
        };
        let config = ViewerConfig {
            theme: "cyberpunk".to_string(),
            font_size: 14.5,
            tab_size: 2,
            line_numbers: false,
            word_wrap: true,
            custom_themes: vec![custom],
            ..ViewerConfig::default()
        };

        let json = config.to_json_pretty().unwrap();
        assert!(json.contains("\"cyberpunk\""));
        assert!(json.contains("\"font_size\": 14.5"));
        assert!(json.contains("\"tab_size\": 2"));

        let loaded = ViewerConfig::from_json_str(&json).unwrap();
        assert_eq!(loaded.theme, "cyberpunk");
        assert_eq!(loaded.font_size, 14.5);
        assert_eq!(loaded.tab_size, 2);
        assert!(!loaded.line_numbers);
        assert!(loaded.word_wrap);
        assert_eq!(loaded.custom_themes.len(), 1);
    }

    #[test]
    fn saves_and_loads_workspace_settings() {
        let directory = tempdir().unwrap();
        let ws_root = directory.path();
        let config = ViewerConfig {
            theme: "nord".to_string(),
            provider: "gemini".to_string(),
            model: "gemini-2.5-pro".to_string(),
            ..ViewerConfig::default()
        };

        let saved_path = ViewerConfig::save_workspace(ws_root, &config).unwrap();
        assert_eq!(saved_path, ws_root.join(".prumo/settings.json"));
        assert!(saved_path.is_file());

        let loaded = ViewerConfig::load_workspace(ws_root).unwrap().unwrap();
        assert_eq!(loaded.theme, "nord");
        assert_eq!(loaded.provider, "gemini");
    }
}
