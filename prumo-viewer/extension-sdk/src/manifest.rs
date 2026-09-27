use serde::{Deserialize, Serialize};
use std::collections::{BTreeMap, BTreeSet};
use std::path::{Component, Path};

pub const MANIFEST_SCHEMA_VERSION: u32 = 1;
pub const EXTENSION_API: &str = "prumo.viewer.extensions/v1";
pub const PROCESS_PROTOCOL: &str = "json-rpc-2.0";
pub const PROCESS_COMMAND_ACTION: &str = "process";
pub const LSP_CAPABILITY: &str = "editor.lsp";
pub const TASK_CAPABILITY: &str = "workspace.tasks";
pub const DEBUG_CAPABILITY: &str = "debugger";

pub const SUPPORTED_DECLARATIVE_ACTIONS: &[&str] = &[
    "find_in_file",
    "find_in_project",
    "toggle_sidebar",
    "toggle_agent",
    "toggle_dock",
    "open_terminal",
    "open_settings",
    "refresh_workspace",
    "new_file",
    "new_folder",
    "toggle_theme",
];

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct ExtensionManifest {
    pub schema_version: u32,
    pub id: String,
    pub name: String,
    pub description: String,
    pub version: String,
    pub publisher: String,
    pub extension_api: String,
    pub host: HostRequirement,
    pub runtime: RuntimeSpec,
    #[serde(default)]
    pub capabilities: Vec<Capability>,
    #[serde(default)]
    pub permissions: Vec<Permission>,
    #[serde(default)]
    pub contributions: Contributions,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct HostRequirement {
    pub min_version: String,
    pub max_version: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct RuntimeSpec {
    pub kind: RuntimeKind,
    #[serde(default)]
    pub protocol: Option<String>,
    #[serde(default)]
    pub entrypoint: BTreeMap<String, String>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct KeybindingChord {
    pub key: String,
    pub control: bool,
    pub shift: bool,
    pub alt: bool,
}

pub fn parse_keybinding(value: &str) -> Result<KeybindingChord, String> {
    let parts: Vec<&str> = value
        .split('+')
        .map(str::trim)
        .filter(|part| !part.is_empty())
        .collect();
    if parts.is_empty() {
        return Err("keybinding must not be empty".to_string());
    }
    let key = parts.last().expect("parts is non-empty").to_lowercase();
    if key.is_empty() {
        return Err("keybinding must include a key".to_string());
    }
    let mut control = false;
    let mut shift = false;
    let mut alt = false;
    for modifier in &parts[..parts.len() - 1] {
        match modifier.to_lowercase().as_str() {
            "ctrl" | "control" => control = true,
            "shift" => shift = true,
            "alt" | "option" => alt = true,
            _ => return Err(format!("unknown keybinding modifier: {modifier}")),
        }
    }
    Ok(KeybindingChord {
        key,
        control,
        shift,
        alt,
    })
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum RuntimeKind {
    Declarative,
    Process,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct Capability {
    pub id: String,
    #[serde(default = "default_true")]
    pub required: bool,
    pub since: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct Permission {
    pub capability: String,
    pub scope: String,
    pub risk: Risk,
    pub justification: String,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum Risk {
    Low,
    Medium,
    High,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize, PartialEq, Eq)]
pub struct Contributions {
    #[serde(default)]
    pub commands: Vec<CommandContribution>,
    #[serde(default)]
    pub views: Vec<ViewContribution>,
    #[serde(default)]
    pub panels: Vec<PanelContribution>,
    #[serde(default)]
    pub themes: Vec<ThemeContribution>,
    #[serde(default)]
    pub keybindings: Vec<KeybindingContribution>,
    #[serde(default)]
    pub lsp: Vec<LspContribution>,
    #[serde(default)]
    pub tasks: Vec<TaskContribution>,
    #[serde(default)]
    pub debuggers: Vec<DebuggerContribution>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct CommandContribution {
    pub id: String,
    pub title: String,
    pub description: String,
    pub action: String,
    #[serde(default)]
    pub keybinding: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct ViewContribution {
    pub id: String,
    pub title: String,
    #[serde(default)]
    pub content: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct PanelContribution {
    pub id: String,
    pub title: String,
    #[serde(default)]
    pub content: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct ThemeContribution {
    pub id: String,
    pub name: String,
    pub tokens: BTreeMap<String, String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct KeybindingContribution {
    pub id: String,
    pub command: String,
    pub key: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct LspContribution {
    pub id: String,
    pub language: String,
    pub command: String,
    #[serde(default)]
    pub args: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct TaskContribution {
    pub id: String,
    pub title: String,
    pub command: String,
    #[serde(default)]
    pub args: Vec<String>,
    #[serde(default)]
    pub working_directory: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct DebuggerContribution {
    pub id: String,
    pub title: String,
    pub command: String,
    #[serde(default)]
    pub args: Vec<String>,
}

impl ExtensionManifest {
    pub fn supports_host(&self, host_version: &str) -> bool {
        let Some(host) = parse_semver(host_version) else {
            return false;
        };
        let Some(minimum) = parse_semver(&self.host.min_version) else {
            return false;
        };
        let Some(maximum) = parse_semver(&self.host.max_version) else {
            return false;
        };
        host >= minimum && host <= maximum
    }

    pub fn contributions(&self) -> &Contributions {
        &self.contributions
    }

    pub fn validate(&self) -> Result<(), Vec<String>> {
        let mut errors = Vec::new();
        if self.schema_version != MANIFEST_SCHEMA_VERSION {
            errors.push(format!(
                "unsupported schema_version {}; expected {}",
                self.schema_version, MANIFEST_SCHEMA_VERSION
            ));
        }
        validate_id("extension.id", &self.id, &mut errors);
        if self.name.trim().is_empty() {
            errors.push("extension.name must not be empty".to_string());
        }
        if self.description.trim().is_empty() {
            errors.push("extension.description must not be empty".to_string());
        }
        validate_semver("extension.version", &self.version, &mut errors);
        if self.publisher.trim().is_empty() {
            errors.push("extension.publisher must not be empty".to_string());
        }
        if self.extension_api != EXTENSION_API {
            errors.push(format!(
                "unsupported extension_api {}; expected {}",
                self.extension_api, EXTENSION_API
            ));
        }
        validate_semver("host.min_version", &self.host.min_version, &mut errors);
        validate_semver("host.max_version", &self.host.max_version, &mut errors);
        match self.runtime.kind {
            RuntimeKind::Declarative => {
                if self.runtime.protocol.is_some() || !self.runtime.entrypoint.is_empty() {
                    errors.push(
                        "declarative extensions cannot define protocol or entrypoint".to_string(),
                    );
                }
            }
            RuntimeKind::Process => {
                if self.runtime.protocol.as_deref() != Some(PROCESS_PROTOCOL) {
                    errors.push(format!(
                        "process extensions must use protocol {}",
                        PROCESS_PROTOCOL
                    ));
                }
                if self.runtime.entrypoint.is_empty() {
                    errors.push("process extensions require at least one entrypoint".to_string());
                }
                for (platform, entrypoint) in &self.runtime.entrypoint {
                    if platform.trim().is_empty() {
                        errors.push("entrypoint platform must not be empty".to_string());
                    }
                    validate_relative_path(entrypoint, &mut errors);
                }
            }
        }

        let capability_ids: BTreeSet<&str> = self
            .capabilities
            .iter()
            .map(|capability| capability.id.as_str())
            .collect();
        if capability_ids.len() != self.capabilities.len() {
            errors.push("capability ids must be unique".to_string());
        }
        for capability in &self.capabilities {
            if capability.id.trim().is_empty() {
                errors.push("capability id must not be empty".to_string());
            }
            validate_semver(
                &format!("capability {} since", capability.id),
                &capability.since,
                &mut errors,
            );
        }
        for permission in &self.permissions {
            if !capability_ids.contains(permission.capability.as_str()) {
                errors.push(format!(
                    "permission references undeclared capability {}",
                    permission.capability
                ));
            }
            if permission.scope.trim().is_empty() || permission.justification.trim().is_empty() {
                errors.push("permission scope and justification are required".to_string());
            }
        }

        let mut contribution_ids = BTreeSet::new();
        for command in &self.contributions.commands {
            validate_id("command.id", &command.id, &mut errors);
            validate_unique_id(&mut contribution_ids, &command.id, &mut errors);
            if command.title.trim().is_empty() || command.description.trim().is_empty() {
                errors.push(format!(
                    "command {} requires title and description",
                    command.id
                ));
            }
            match self.runtime.kind {
                RuntimeKind::Declarative
                    if !SUPPORTED_DECLARATIVE_ACTIONS.contains(&command.action.as_str()) =>
                {
                    errors.push(format!(
                        "command {} uses unsupported declarative action {}",
                        command.id, command.action
                    ));
                }
                RuntimeKind::Process if command.action != PROCESS_COMMAND_ACTION => {
                    errors.push(format!(
                        "process command {} must use the process action",
                        command.id
                    ));
                }
                _ => {}
            }
        }
        if !self.contributions.lsp.is_empty() {
            if self.runtime.kind != RuntimeKind::Declarative {
                errors.push("LSP contributions require a declarative runtime".to_string());
            }
            if !capability_ids.contains(LSP_CAPABILITY) {
                errors.push(format!(
                    "LSP contributions require the {LSP_CAPABILITY} capability"
                ));
            }
            for server in &self.contributions.lsp {
                validate_id("lsp.id", &server.id, &mut errors);
                validate_unique_id(&mut contribution_ids, &server.id, &mut errors);
                if server.language.trim().is_empty() || server.command.trim().is_empty() {
                    errors.push(format!(
                        "LSP contribution {} requires language and command",
                        server.id
                    ));
                }
                if server.args.len() > 32
                    || server.args.iter().any(|argument| argument.contains('\0'))
                {
                    errors.push(format!(
                        "LSP contribution {} has invalid arguments",
                        server.id
                    ));
                }
            }
        }
        if !self.contributions.tasks.is_empty() && !capability_ids.contains(TASK_CAPABILITY) {
            errors.push(format!(
                "task contributions require the {TASK_CAPABILITY} capability"
            ));
        }
        for task in &self.contributions.tasks {
            validate_id("task.id", &task.id, &mut errors);
            validate_unique_id(&mut contribution_ids, &task.id, &mut errors);
            if task.title.trim().is_empty() || task.command.trim().is_empty() {
                errors.push(format!("task {} requires title and command", task.id));
            }
            if task.args.len() > 64
                || task.args.iter().any(|argument| argument.contains('\0'))
                || task.command.contains('\0')
            {
                errors.push(format!("task {} has invalid arguments", task.id));
            }
            if let Some(directory) = &task.working_directory {
                validate_relative_path(directory, &mut errors);
            }
        }
        if !self.contributions.debuggers.is_empty() && !capability_ids.contains(DEBUG_CAPABILITY) {
            errors.push(format!(
                "debugger contributions require the {DEBUG_CAPABILITY} capability"
            ));
        }
        for debugger in &self.contributions.debuggers {
            validate_id("debugger.id", &debugger.id, &mut errors);
            validate_unique_id(&mut contribution_ids, &debugger.id, &mut errors);
            if debugger.title.trim().is_empty() || debugger.command.trim().is_empty() {
                errors.push(format!(
                    "debugger {} requires title and command",
                    debugger.id
                ));
            }
            if debugger.args.len() > 32
                || debugger.args.iter().any(|argument| argument.contains('\0'))
                || debugger.command.contains('\0')
            {
                errors.push(format!("debugger {} has invalid arguments", debugger.id));
            }
        }
        for view in &self.contributions.views {
            validate_id("view.id", &view.id, &mut errors);
            validate_unique_id(&mut contribution_ids, &view.id, &mut errors);
            validate_content(&view.content, "view", &view.id, &mut errors);
        }
        for panel in &self.contributions.panels {
            validate_id("panel.id", &panel.id, &mut errors);
            validate_unique_id(&mut contribution_ids, &panel.id, &mut errors);
            validate_content(&panel.content, "panel", &panel.id, &mut errors);
        }
        for theme in &self.contributions.themes {
            validate_id("theme.id", &theme.id, &mut errors);
            validate_unique_id(&mut contribution_ids, &theme.id, &mut errors);
            if theme.name.trim().is_empty() || theme.tokens.is_empty() {
                errors.push(format!("theme {} requires name and tokens", theme.id));
            }
            if theme.tokens.len() > 64 {
                errors.push(format!("theme {} has too many tokens", theme.id));
            }
            for (token, value) in &theme.tokens {
                if token.trim().is_empty() || !valid_theme_token(token) {
                    errors.push(format!("theme {} has invalid token {token}", theme.id));
                }
                if !valid_theme_color(value) {
                    errors.push(format!("theme {} has invalid color {value}", theme.id));
                }
            }
        }
        for keybinding in &self.contributions.keybindings {
            validate_id("keybinding.id", &keybinding.id, &mut errors);
            validate_unique_id(&mut contribution_ids, &keybinding.id, &mut errors);
            if !self
                .contributions
                .commands
                .iter()
                .any(|command| command.id == keybinding.command)
            {
                errors.push(format!(
                    "keybinding {} references unknown command {}",
                    keybinding.id, keybinding.command
                ));
            }
            if keybinding.key.trim().is_empty() {
                errors.push(format!("keybinding {} requires a key", keybinding.id));
            } else if let Err(error) = parse_keybinding(&keybinding.key) {
                errors.push(format!("keybinding {} is invalid: {error}", keybinding.id));
            }
        }
        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors)
        }
    }
}

fn validate_content(value: &str, kind: &str, id: &str, errors: &mut Vec<String>) {
    if value.len() > 16_384 {
        errors.push(format!("{kind} {id} content is too long"));
    }
    if value
        .chars()
        .any(|character| character.is_control() && character != '\n' && character != '\t')
    {
        errors.push(format!("{kind} {id} content contains control characters"));
    }
}

fn valid_theme_token(value: &str) -> bool {
    matches!(
        value,
        "accent"
            | "background"
            | "surface-primary"
            | "surface-secondary"
            | "surface-tertiary"
            | "border"
            | "border-focus"
            | "text-primary"
            | "text-secondary"
            | "text-placeholder"
            | "success"
            | "warning"
            | "error"
    )
}

fn valid_theme_color(value: &str) -> bool {
    let value = value.strip_prefix('#').unwrap_or(value);
    (value.len() == 6 || value.len() == 8)
        && value.chars().all(|character| character.is_ascii_hexdigit())
}

fn default_true() -> bool {
    true
}

fn validate_id(field: &str, value: &str, errors: &mut Vec<String>) {
    let valid = value.contains('.')
        && value.split('.').all(|part| {
            !part.is_empty()
                && part
                    .chars()
                    .all(|character| character.is_ascii_alphanumeric() || character == '-')
        });
    if !valid {
        errors.push(format!("{field} must be a namespaced identifier: {value}"));
    }
}

fn validate_unique_id(ids: &mut BTreeSet<String>, value: &str, errors: &mut Vec<String>) {
    if !ids.insert(value.to_string()) {
        errors.push(format!("duplicate contribution id: {value}"));
    }
}

fn parse_semver(value: &str) -> Option<(u64, u64, u64)> {
    let core = value.split_once('-').map_or(value, |(core, _)| core);
    let mut parts = core.split('.');
    let major = parts.next()?.parse().ok()?;
    let minor = parts.next()?.parse().ok()?;
    let patch = parts.next()?.parse().ok()?;
    if parts.next().is_some() {
        return None;
    }
    Some((major, minor, patch))
}

fn validate_semver(field: &str, value: &str, errors: &mut Vec<String>) {
    let core = value.split_once('-').map_or(value, |(core, _)| core);
    let parts: Vec<&str> = core.split('.').collect();
    let valid = parts.len() == 3
        && parts.iter().all(|part| {
            !part.is_empty() && part.chars().all(|character| character.is_ascii_digit())
        });
    if !valid {
        errors.push(format!("{field} must use major.minor.patch: {value}"));
    }
}

fn validate_relative_path(value: &str, errors: &mut Vec<String>) {
    let path = Path::new(value);
    if value.trim().is_empty()
        || path.is_absolute()
        || path.components().any(|component| {
            matches!(
                component,
                Component::ParentDir | Component::RootDir | Component::Prefix(_)
            )
        })
    {
        errors.push(format!("entrypoint must be a safe relative path: {value}"));
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn manifest() -> ExtensionManifest {
        ExtensionManifest {
            schema_version: 1,
            id: "com.example.quick-actions".to_string(),
            name: "Quick Actions".to_string(),
            description: "Example extension".to_string(),
            version: "0.1.0".to_string(),
            publisher: "Example".to_string(),
            extension_api: EXTENSION_API.to_string(),
            host: HostRequirement {
                min_version: "0.1.0".to_string(),
                max_version: "0.2.0".to_string(),
            },
            runtime: RuntimeSpec {
                kind: RuntimeKind::Declarative,
                protocol: None,
                entrypoint: BTreeMap::new(),
            },
            capabilities: vec![Capability {
                id: "ui.command".to_string(),
                required: true,
                since: "1.0.0".to_string(),
            }],
            permissions: vec![Permission {
                capability: "ui.command".to_string(),
                scope: "workspace".to_string(),
                risk: Risk::Low,
                justification: "Provides user-invoked commands".to_string(),
            }],
            contributions: Contributions {
                commands: vec![CommandContribution {
                    id: "com.example.quick-actions.find".to_string(),
                    title: "Find".to_string(),
                    description: "Open find".to_string(),
                    action: "find_in_file".to_string(),
                    keybinding: None,
                }],
                ..Contributions::default()
            },
        }
    }

    #[test]
    fn validates_reference_manifest() {
        assert!(manifest().validate().is_ok());
        assert!(manifest().supports_host("0.1.5"));
        assert!(!manifest().supports_host("0.2.1"));
    }

    #[test]
    fn parses_keybindings() {
        assert_eq!(
            parse_keybinding("Ctrl+Shift+P").unwrap(),
            KeybindingChord {
                key: "p".to_string(),
                control: true,
                shift: true,
                alt: false,
            }
        );
        assert!(parse_keybinding("Hyper+P").is_err());
    }

    #[test]
    fn accepts_debugger_contributions() {
        let mut value = manifest();
        value.capabilities.push(Capability {
            id: DEBUG_CAPABILITY.to_string(),
            required: true,
            since: "1.0.0".to_string(),
        });
        value.permissions.push(Permission {
            capability: DEBUG_CAPABILITY.to_string(),
            scope: "workspace".to_string(),
            risk: Risk::High,
            justification: "Starts a user-invoked debug adapter".to_string(),
        });
        value.contributions.debuggers.push(DebuggerContribution {
            id: "com.example.quick-actions.debugger".to_string(),
            title: "Debug".to_string(),
            command: "debug-adapter".to_string(),
            args: vec!["--stdio".to_string()],
        });
        assert!(value.validate().is_ok());
    }

    #[test]
    fn accepts_task_contributions() {
        let mut value = manifest();
        value.capabilities.push(Capability {
            id: TASK_CAPABILITY.to_string(),
            required: true,
            since: "1.0.0".to_string(),
        });
        value.permissions.push(Permission {
            capability: TASK_CAPABILITY.to_string(),
            scope: "workspace".to_string(),
            risk: Risk::High,
            justification: "Runs a user-invoked task".to_string(),
        });
        value.contributions.tasks.push(TaskContribution {
            id: "com.example.quick-actions.task".to_string(),
            title: "Check".to_string(),
            command: "cargo".to_string(),
            args: vec!["check".to_string()],
            working_directory: None,
        });
        assert!(value.validate().is_ok());
    }

    #[test]
    fn accepts_panels_and_keybindings() {
        let mut value = manifest();
        value.contributions.views.push(ViewContribution {
            id: "com.example.quick-actions.view".to_string(),
            title: "View".to_string(),
            content: "Hello".to_string(),
        });
        value.contributions.panels.push(PanelContribution {
            id: "com.example.quick-actions.panel".to_string(),
            title: "Panel".to_string(),
            content: "World".to_string(),
        });
        value
            .contributions
            .keybindings
            .push(KeybindingContribution {
                id: "com.example.quick-actions.key".to_string(),
                command: "com.example.quick-actions.find".to_string(),
                key: "Ctrl+Shift+P".to_string(),
            });
        assert!(value.validate().is_ok());
    }

    #[test]
    fn accepts_theme_tokens() {
        let mut value = manifest();
        value.contributions.themes.push(ThemeContribution {
            id: "com.example.quick-actions.theme".to_string(),
            name: "Example".to_string(),
            tokens: BTreeMap::from([("accent".to_string(), "#123456".to_string())]),
        });
        assert!(value.validate().is_ok());
    }

    #[test]
    fn accepts_lsp_contribution() {
        let mut value = manifest();
        value.capabilities.push(Capability {
            id: LSP_CAPABILITY.to_string(),
            required: true,
            since: "1.0.0".to_string(),
        });
        value.permissions.push(Permission {
            capability: LSP_CAPABILITY.to_string(),
            scope: "workspace".to_string(),
            risk: Risk::High,
            justification: "Starts a language server".to_string(),
        });
        value.contributions.lsp.push(LspContribution {
            id: "com.example.quick-actions.rust".to_string(),
            language: "rust".to_string(),
            command: "rust-analyzer".to_string(),
            args: vec!["--stdio".to_string()],
        });
        assert!(value.validate().is_ok());
    }

    #[test]
    fn accepts_process_command_action() {
        let mut value = manifest();
        value.runtime.kind = RuntimeKind::Process;
        value.runtime.protocol = Some(PROCESS_PROTOCOL.to_string());
        value
            .runtime
            .entrypoint
            .insert("linux".to_string(), "bin/extension".to_string());
        value.contributions.commands[0].action = PROCESS_COMMAND_ACTION.to_string();
        assert!(value.validate().is_ok());
    }

    #[test]
    fn rejects_unsafe_entrypoint_and_unknown_action() {
        let mut value = manifest();
        value.runtime.kind = RuntimeKind::Process;
        value.runtime.protocol = Some(PROCESS_PROTOCOL.to_string());
        value
            .runtime
            .entrypoint
            .insert("linux".to_string(), "../outside".to_string());
        value.contributions.commands[0].action = "arbitrary_code".to_string();
        let errors = value.validate().expect_err("manifest must be rejected");
        assert!(
            errors
                .iter()
                .any(|error| error.contains("safe relative path"))
        );
        assert!(errors.iter().any(|error| error.contains("process action")));
    }
}
