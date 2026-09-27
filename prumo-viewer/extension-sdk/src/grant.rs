use crate::manifest::ExtensionManifest;
use serde::{Deserialize, Serialize};
use std::collections::{BTreeMap, BTreeSet};

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct ExtensionGrant {
    pub extension_id: String,
    pub extension_version: String,
    pub manifest_digest: String,
    pub capabilities: Vec<String>,
    #[serde(default)]
    pub scopes: BTreeMap<String, String>,
    #[serde(default)]
    pub expires_at: Option<String>,
}

impl ExtensionGrant {
    pub fn validate_for(
        &self,
        manifest: &ExtensionManifest,
        actual_package_digest: &str,
    ) -> Result<(), Vec<String>> {
        let mut errors = Vec::new();
        if self.extension_id != manifest.id {
            errors.push("grant extension_id does not match manifest".to_string());
        }
        if self.extension_version != manifest.version {
            errors.push("grant extension_version does not match manifest".to_string());
        }
        if self.manifest_digest.len() != 64
            || !self
                .manifest_digest
                .chars()
                .all(|character| character.is_ascii_hexdigit())
        {
            errors.push("grant manifest_digest must be a SHA-256 hex digest".to_string());
        }
        if self.manifest_digest != actual_package_digest {
            errors.push("grant manifest_digest does not match the active package".to_string());
        }
        let declared: BTreeSet<&str> = manifest
            .capabilities
            .iter()
            .map(|capability| capability.id.as_str())
            .collect();
        let mut granted = BTreeSet::new();
        for capability in &self.capabilities {
            if !declared.contains(capability.as_str()) {
                errors.push(format!(
                    "grant references undeclared capability {capability}"
                ));
            }
            if !granted.insert(capability) {
                errors.push(format!("grant repeats capability {capability}"));
            }
        }
        for capability in &self.capabilities {
            if !self.scopes.contains_key(capability) {
                errors.push(format!("grant is missing scope for {capability}"));
            }
        }
        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::manifest::{
        Capability, CommandContribution, Contributions, EXTENSION_API, HostRequirement,
        RuntimeKind, RuntimeSpec,
    };
    use std::collections::BTreeMap;

    fn manifest() -> ExtensionManifest {
        ExtensionManifest {
            schema_version: 1,
            id: "com.example.grant".to_string(),
            name: "Grant".to_string(),
            description: "Grant test".to_string(),
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
            permissions: Vec::new(),
            contributions: Contributions {
                commands: vec![CommandContribution {
                    id: "com.example.grant.command".to_string(),
                    title: "Command".to_string(),
                    description: "Command".to_string(),
                    action: "toggle_dock".to_string(),
                    keybinding: None,
                }],
                ..Contributions::default()
            },
        }
    }

    #[test]
    fn accepts_matching_grant() {
        let grant = ExtensionGrant {
            extension_id: "com.example.grant".to_string(),
            extension_version: "0.1.0".to_string(),
            manifest_digest: "a".repeat(64),
            capabilities: vec!["ui.command".to_string()],
            scopes: BTreeMap::from([("ui.command".to_string(), "workspace".to_string())]),
            expires_at: None,
        };
        assert!(grant.validate_for(&manifest(), &"a".repeat(64)).is_ok());
    }
}
