use serde::{Deserialize, Serialize};
use std::{
    fs,
    io::Write,
    path::{Path, PathBuf},
};

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ViewerConfig {
    pub version: u32,
    pub theme: String,
    pub provider: String,
    pub model: String,
    #[serde(default)]
    pub socket_path: Option<String>,
    #[serde(default)]
    pub terminal_shell: Option<String>,
    pub show_whitespace: bool,
    pub poll_interval_seconds: u64,
}

impl Default for ViewerConfig {
    fn default() -> Self {
        Self {
            version: 1,
            theme: "system".to_string(),
            provider: "fake".to_string(),
            model: String::new(),
            socket_path: None,
            terminal_shell: None,
            show_whitespace: false,
            poll_interval_seconds: 2,
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

    pub fn validate(&self) -> Result<(), String> {
        if self.version != 1 {
            return Err(format!(
                "unsupported viewer config version: {}",
                self.version
            ));
        }
        if !matches!(self.theme.as_str(), "system" | "light" | "dark") {
            return Err(format!("unsupported theme: {}", self.theme));
        }
        if self.provider.trim().is_empty() {
            return Err("provider cannot be empty".to_string());
        }
        if !(1..=60).contains(&self.poll_interval_seconds) {
            return Err("poll interval must be between 1 and 60 seconds".to_string());
        }
        Ok(())
    }

    fn load_from_path(path: &Path) -> Result<Self, String> {
        let content = fs::read_to_string(path)
            .map_err(|error| format!("failed to read {}: {error}", path.display()))?;
        let config: Self = serde_json::from_str(&content)
            .map_err(|error| format!("invalid viewer config {}: {error}", path.display()))?;
        config.validate()?;
        Ok(config)
    }

    fn save_to_path(&self, path: &Path) -> Result<(), String> {
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
}
