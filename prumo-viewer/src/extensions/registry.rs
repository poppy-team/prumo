use prumo_extension_sdk::manifest::{ExtensionManifest, RuntimeKind};
use prumo_extension_sdk::package::{Lockfile, directory_digest, read_lockfile};
use std::collections::HashSet;
use std::env;
use std::fs;
use std::path::{Path, PathBuf};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ExtensionSource {
    Workspace,
    Environment,
    Development,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ExtensionStatus {
    Ready,
    Deferred,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ExtensionRecord {
    pub manifest: ExtensionManifest,
    pub root: PathBuf,
    pub source: ExtensionSource,
    pub status: ExtensionStatus,
    pub manifest_digest: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ExtensionLoadError {
    pub root: PathBuf,
    pub message: String,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ExtensionRegistry {
    pub records: Vec<ExtensionRecord>,
    pub errors: Vec<ExtensionLoadError>,
    pub lockfile: Option<Lockfile>,
}

pub fn discover_extensions(
    workspace_root: &Path,
    development_root: Option<&Path>,
) -> ExtensionRegistry {
    let mut roots = Vec::new();
    roots.push((
        workspace_root.join(".prumo/extensions"),
        ExtensionSource::Workspace,
    ));
    if let Some(development_root) = development_root {
        roots.push((development_root.to_path_buf(), ExtensionSource::Development));
    }
    if let Some(value) = env::var_os("PRUMO_VIEWER_EXTENSIONS") {
        roots.extend(env::split_paths(&value).map(|path| (path, ExtensionSource::Environment)));
    }

    let mut registry = ExtensionRegistry::default();
    let mut ids = HashSet::new();
    for (root, source) in roots {
        load_extension_root(root, source, &mut ids, &mut registry);
    }
    registry
}

impl ExtensionRegistry {
    pub fn load_lockfile(&mut self, path: &Path) {
        if !path.is_file() {
            return;
        }
        match read_lockfile(path) {
            Ok(lockfile) => {
                for entry in &lockfile.extensions {
                    if let Some(record) = self
                        .records
                        .iter_mut()
                        .find(|record| record.manifest.id == entry.extension_id)
                        && record.manifest_digest != entry.digest
                    {
                        record.status = ExtensionStatus::Deferred;
                        self.errors.push(ExtensionLoadError {
                            root: record.root.clone(),
                            message: "extension digest does not match lockfile".to_string(),
                        });
                    }
                }
                self.lockfile = Some(lockfile);
            }
            Err(error) => self.errors.push(ExtensionLoadError {
                root: path.to_path_buf(),
                message: error,
            }),
        }
    }
}

fn load_extension_root(
    root: PathBuf,
    source: ExtensionSource,
    ids: &mut HashSet<String>,
    registry: &mut ExtensionRegistry,
) {
    if !root.is_dir() {
        return;
    }
    let mut children = match fs::read_dir(&root) {
        Ok(children) => children
            .filter_map(Result::ok)
            .map(|entry| entry.path())
            .collect::<Vec<_>>(),
        Err(error) => {
            registry.errors.push(ExtensionLoadError {
                root,
                message: format!("could not read extension directory: {error}"),
            });
            return;
        }
    };
    children.sort();
    for child in children {
        if !child.is_dir() {
            continue;
        }
        load_extension_directory(child, source, ids, registry);
    }
}

fn load_extension_directory(
    root: PathBuf,
    source: ExtensionSource,
    ids: &mut HashSet<String>,
    registry: &mut ExtensionRegistry,
) {
    let manifest_path = root.join("manifest.json");
    match fs::symlink_metadata(&manifest_path) {
        Ok(metadata) if metadata.file_type().is_symlink() => {
            registry.errors.push(ExtensionLoadError {
                root,
                message: "manifest.json must not be a symlink".to_string(),
            });
            return;
        }
        Ok(_) => (),
        Err(error) => {
            registry.errors.push(ExtensionLoadError {
                root,
                message: format!("missing manifest.json: {error}"),
            });
            return;
        }
    };
    let manifest = match fs::read_to_string(&manifest_path) {
        Ok(contents) => match serde_json::from_str::<ExtensionManifest>(&contents) {
            Ok(manifest) => manifest,
            Err(error) => {
                registry.errors.push(ExtensionLoadError {
                    root,
                    message: format!("invalid manifest: {error}"),
                });
                return;
            }
        },
        Err(error) => {
            registry.errors.push(ExtensionLoadError {
                root,
                message: format!("missing manifest.json: {error}"),
            });
            return;
        }
    };
    if let Err(errors) = manifest.validate() {
        registry.errors.push(ExtensionLoadError {
            root,
            message: errors.join("; "),
        });
        return;
    }
    if !manifest.supports_host(env!("CARGO_PKG_VERSION")) {
        registry.errors.push(ExtensionLoadError {
            root,
            message: format!(
                "extension host range does not include viewer {}",
                env!("CARGO_PKG_VERSION")
            ),
        });
        return;
    }
    if !ids.insert(manifest.id.clone()) {
        registry.errors.push(ExtensionLoadError {
            root,
            message: format!("duplicate extension id: {}", manifest.id),
        });
        return;
    }
    let manifest_digest = match directory_digest(&root) {
        Ok(digest) => digest,
        Err(error) => {
            registry.errors.push(ExtensionLoadError {
                root,
                message: error,
            });
            return;
        }
    };
    let status = match manifest.runtime.kind {
        RuntimeKind::Declarative => ExtensionStatus::Ready,
        RuntimeKind::Process => ExtensionStatus::Deferred,
    };
    registry.records.push(ExtensionRecord {
        manifest,
        root,
        source,
        status,
        manifest_digest,
    });
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ExtensionTemplateKind {
    Theme,
    Commands,
    Process,
}

pub fn scaffold_extension(
    workspace_root: &Path,
    id: &str,
    name: &str,
    description: &str,
    publisher: &str,
    template: ExtensionTemplateKind,
) -> Result<PathBuf, String> {
    let clean_id = id.trim().to_lowercase().replace(' ', "-");
    if clean_id.is_empty() {
        return Err("extension id cannot be empty".to_string());
    }
    let clean_publisher = if publisher.trim().is_empty() {
        "workspace".to_string()
    } else {
        publisher.trim().to_lowercase().replace(' ', "-")
    };
    let sanitized_id = if clean_id.contains('.') {
        clean_id
    } else {
        format!("{clean_publisher}.{clean_id}")
    };
    let folder_name = sanitized_id.replace('.', "-");
    let ext_dir = workspace_root.join(".prumo/extensions").join(&folder_name);
    if ext_dir.exists() {
        return Err(format!(
            "extension directory already exists: {}",
            ext_dir.display()
        ));
    }
    fs::create_dir_all(&ext_dir)
        .map_err(|e| format!("failed to create extension directory: {e}"))?;

    let manifest_json = match template {
        ExtensionTemplateKind::Theme => serde_json::json!({
            "schema_version": 1,
            "id": sanitized_id,
            "name": if name.trim().is_empty() { &sanitized_id } else { name },
            "description": description,
            "version": "0.1.0",
            "publisher": clean_publisher,
            "extension_api": "prumo.viewer.extensions/v1",
            "host": { "min_version": "0.1.0", "max_version": "0.2.0" },
            "runtime": { "kind": "declarative" },
            "contributions": {
                "themes": [{
                    "id": format!("{sanitized_id}.theme"),
                    "name": if name.trim().is_empty() { &sanitized_id } else { name },
                    "tokens": {
                        "accent": "#78a9e8",
                        "background": "#1e222b",
                        "surface-primary": "#262b36",
                        "surface-secondary": "#2e3442",
                        "surface-tertiary": "#394152",
                        "border": "#394152",
                        "border-focus": "#78a9e8",
                        "text-primary": "#dce0e6",
                        "text-secondary": "#a8afbd",
                        "text-placeholder": "#79818e",
                        "success": "#a7c88b",
                        "warning": "#dfc488",
                        "error": "#d27d82"
                    }
                }]
            }
        }),
        ExtensionTemplateKind::Commands => serde_json::json!({
            "schema_version": 1,
            "id": sanitized_id,
            "name": if name.trim().is_empty() { &sanitized_id } else { name },
            "description": description,
            "version": "0.1.0",
            "publisher": clean_publisher,
            "extension_api": "prumo.viewer.extensions/v1",
            "host": { "min_version": "0.1.0", "max_version": "0.2.0" },
            "runtime": { "kind": "declarative" },
            "capabilities": [{ "id": "ui.command", "required": true, "since": "1.0.0" }],
            "permissions": [{
                "capability": "ui.command",
                "scope": "workspace",
                "risk": "low",
                "justification": "Contributes custom actions."
            }],
            "contributions": {
                "commands": [{
                    "id": format!("{sanitized_id}.action"),
                    "title": format!("{}: Custom Action", if name.trim().is_empty() { &sanitized_id } else { name }),
                    "description": description,
                    "action": "refresh_workspace"
                }]
            }
        }),
        ExtensionTemplateKind::Process => serde_json::json!({
            "schema_version": 1,
            "id": sanitized_id,
            "name": if name.trim().is_empty() { &sanitized_id } else { name },
            "description": description,
            "version": "0.1.0",
            "publisher": clean_publisher,
            "extension_api": "prumo.viewer.extensions/v1",
            "host": { "min_version": "0.1.0", "max_version": "0.2.0" },
            "runtime": {
                "kind": "process",
                "protocol": "json-rpc-2.0",
                "entrypoint": { "linux": "run.sh" }
            },
            "capabilities": [{ "id": "workspace.tasks", "required": true, "since": "1.0.0" }],
            "permissions": [{
                "capability": "workspace.tasks",
                "scope": "workspace",
                "risk": "medium",
                "justification": "Executes background analysis tasks."
            }]
        }),
    };

    let manifest_path = ext_dir.join("manifest.json");
    let content = serde_json::to_string_pretty(&manifest_json)
        .map_err(|e| format!("failed to format manifest json: {e}"))?;
    fs::write(&manifest_path, content)
        .map_err(|e| format!("failed to write manifest.json: {e}"))?;

    if matches!(template, ExtensionTemplateKind::Process) {
        let script_path = ext_dir.join("run.sh");
        let script = "#!/bin/sh\n# Prumo Extension Process Runtime stub\nread line\n";
        let _ = fs::write(&script_path, script);
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            let _ = fs::set_permissions(&script_path, fs::Permissions::from_mode(0o755));
        }
    }

    Ok(ext_dir)
}

pub fn rescan_workspace_extensions(
    workspace_root: &Path,
) -> (Vec<ExtensionRecord>, Vec<ExtensionLoadError>) {
    let development_root = option_env!("CARGO_MANIFEST_DIR")
        .map(std::path::Path::new)
        .map(|path| path.join("extensions"));
    let mut registry = discover_extensions(workspace_root, development_root.as_deref());
    registry.load_lockfile(&workspace_root.join(".prumo/extensions.lock.json"));
    (registry.records, registry.errors)
}

#[cfg(test)]
mod tests {
    use super::*;
    use prumo_extension_sdk::manifest::{
        Capability, CommandContribution, Contributions, EXTENSION_API, HostRequirement, Permission,
        Risk, RuntimeSpec,
    };
    use prumo_extension_sdk::package::{LockEntry, Lockfile, write_lockfile};
    use std::collections::BTreeMap;

    fn write_manifest(root: &Path, id: &str) {
        fs::create_dir_all(root).unwrap();
        let manifest = ExtensionManifest {
            schema_version: 1,
            id: id.to_string(),
            name: "Example".to_string(),
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
                justification: "Provides a command".to_string(),
            }],
            contributions: Contributions {
                commands: vec![CommandContribution {
                    id: format!("{id}.find"),
                    title: "Find".to_string(),
                    description: "Open find".to_string(),
                    action: "find_in_file".to_string(),
                    keybinding: None,
                }],
                ..Contributions::default()
            },
        };
        fs::write(
            root.join("manifest.json"),
            serde_json::to_vec_pretty(&manifest).unwrap(),
        )
        .unwrap();
    }

    #[test]
    fn discovers_valid_declarative_extensions() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().join("extensions");
        write_manifest(&root.join("example"), "com.example.quick-actions");
        let registry = discover_extensions(directory.path(), Some(&root));
        assert_eq!(registry.records.len(), 1);
        assert_eq!(registry.records[0].status, ExtensionStatus::Ready);
        assert_eq!(registry.records[0].manifest_digest.len(), 64);
        assert!(registry.errors.is_empty());
    }

    #[test]
    fn loads_reference_editor_navigation_package() {
        let root = PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("extensions");
        let registry = discover_extensions(Path::new("."), Some(&root));
        assert_eq!(registry.records.len(), 2);
        assert!(
            registry
                .records
                .iter()
                .any(|record| record.manifest.id == "com.prumo.editor-navigation")
        );
        let rust_analyzer = registry
            .records
            .iter()
            .find(|record| record.manifest.id == "com.prumo.rust-analyzer")
            .unwrap();
        assert_eq!(rust_analyzer.manifest.contributions.lsp[0].language, "rust");
        assert_eq!(
            rust_analyzer.manifest.contributions.lsp[0].command,
            "rust-analyzer"
        );
        assert!(registry.errors.is_empty());
    }

    #[test]
    fn accepts_lock_digest_from_package_directory() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().join("extensions");
        let extension_root = root.join("example");
        write_manifest(&extension_root, "com.example.locked");
        let mut registry = discover_extensions(directory.path(), Some(&root));
        let lockfile = Lockfile {
            schema_version: 1,
            extensions: vec![LockEntry {
                extension_id: "com.example.locked".to_string(),
                version: "0.1.0".to_string(),
                digest: directory_digest(&extension_root).unwrap(),
                source: "test".to_string(),
                signature_key_id: None,
            }],
        };
        let lock_path = directory.path().join("extensions.lock.json");
        write_lockfile(&lock_path, &lockfile).unwrap();
        registry.load_lockfile(&lock_path);
        assert_eq!(registry.records[0].status, ExtensionStatus::Ready);
        assert!(registry.errors.is_empty());
    }

    #[test]
    fn defers_extension_when_lock_digest_differs() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().join("extensions");
        write_manifest(&root.join("example"), "com.example.locked");
        let mut registry = discover_extensions(directory.path(), Some(&root));
        let lockfile = Lockfile {
            schema_version: 1,
            extensions: vec![LockEntry {
                extension_id: "com.example.locked".to_string(),
                version: "0.1.0".to_string(),
                digest: "d".repeat(64),
                source: "test".to_string(),
                signature_key_id: None,
            }],
        };
        let lock_path = directory.path().join("extensions.lock.json");
        write_lockfile(&lock_path, &lockfile).unwrap();
        registry.load_lockfile(&lock_path);
        assert_eq!(registry.records[0].status, ExtensionStatus::Deferred);
        assert_eq!(registry.errors.len(), 1);
    }

    #[test]
    fn reports_invalid_manifest_without_loading_it() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().join("extensions");
        fs::create_dir_all(root.join("broken")).unwrap();
        fs::write(root.join("broken/manifest.json"), "{not-json").unwrap();
        let registry = discover_extensions(directory.path(), Some(&root));
        assert!(registry.records.is_empty());
        assert_eq!(registry.errors.len(), 1);
    }

    #[test]
    fn scaffolds_and_rescans_workspace_extensions() {
        let directory = tempfile::tempdir().unwrap();
        let ws_root = directory.path();

        let ext_dir = scaffold_extension(
            ws_root,
            "custom-tools",
            "Custom Tools",
            "Workspace tools",
            "developer",
            ExtensionTemplateKind::Commands,
        )
        .unwrap();

        assert!(ext_dir.join("manifest.json").is_file());

        let (extensions, errors) = rescan_workspace_extensions(ws_root);

        assert!(errors.is_empty(), "unexpected errors: {errors:?}");
        let found = extensions
            .iter()
            .find(|e| e.manifest.id == "developer.custom-tools");
        assert!(found.is_some());
        assert_eq!(found.unwrap().status, ExtensionStatus::Ready);
    }
}

