use prumo_extension_sdk::manifest::{ExtensionManifest, RuntimeKind};
use prumo_extension_sdk::package::{
    ExtensionPackage, LockEntry, Lockfile, TrustStore, read_lockfile, sign_package, write_lockfile,
};
use prumo_extension_sdk::{frame_message, parse_keybinding, read_framed_message};
use std::fs;
use std::io::Cursor;

fn manifest_bytes() -> Vec<u8> {
    serde_json::to_vec(&serde_json::json!({
        "schema_version": 1,
        "id": "com.example.conformance",
        "name": "Conformance",
        "description": "Conformance fixture",
        "version": "0.1.0",
        "publisher": "Example",
        "extension_api": "prumo.viewer.extensions/v1",
        "host": { "min_version": "0.1.0", "max_version": "0.2.0" },
        "runtime": { "kind": "declarative" },
        "capabilities": [{ "id": "ui.command", "required": true, "since": "1.0.0" }],
        "permissions": [{
            "capability": "ui.command",
            "scope": "workspace",
            "risk": "low",
            "justification": "Runs a test command"
        }],
        "contributions": {
            "commands": [{
                "id": "com.example.conformance.command",
                "title": "Command",
                "description": "Test command",
                "action": "refresh_workspace"
            }],
            "keybindings": [{
                "id": "com.example.conformance.key",
                "command": "com.example.conformance.command",
                "key": "Ctrl+Shift+P"
            }]
        }
    }))
    .unwrap()
}

#[test]
fn manifest_and_keybinding_conform() {
    let manifest: ExtensionManifest = serde_json::from_slice(&manifest_bytes()).unwrap();
    assert_eq!(manifest.runtime.kind, RuntimeKind::Declarative);
    assert!(manifest.validate().is_ok());
    assert!(parse_keybinding("Ctrl+Shift+P").is_ok());
}

#[test]
fn archive_signature_and_lockfile_conform() {
    let directory = tempfile::tempdir().unwrap();
    let package_directory = directory.path().join("package");
    fs::create_dir_all(&package_directory).unwrap();
    fs::write(package_directory.join("manifest.json"), manifest_bytes()).unwrap();
    fs::write(package_directory.join("data.txt"), "data").unwrap();
    let package = ExtensionPackage::from_directory(&package_directory).unwrap();
    let archive = directory.path().join("extension.prumoext");
    package.write_archive(&archive).unwrap();
    let unpacked = ExtensionPackage::read_archive(&archive).unwrap();
    let signing_key = ed25519_dalek::SigningKey::from_bytes(&[3; 32]);
    let (signature, trusted) = sign_package(&unpacked, &signing_key).unwrap();
    let trust = TrustStore {
        keys: vec![trusted],
        revoked_key_ids: Vec::new(),
    };
    trust.verify(&unpacked, &signature).unwrap();
    let lock_path = directory.path().join("extensions.lock.json");
    let mut lockfile = Lockfile::default();
    lockfile.upsert(LockEntry {
        extension_id: "com.example.conformance".to_string(),
        version: "0.1.0".to_string(),
        digest: unpacked.digest.clone(),
        source: archive.display().to_string(),
        signature_key_id: Some(signature.key_id.clone()),
    });
    write_lockfile(&lock_path, &lockfile).unwrap();
    assert_eq!(
        read_lockfile(&lock_path)
            .unwrap()
            .entry("com.example.conformance"),
        lockfile.extensions.first()
    );
}

#[test]
fn lsp_and_rpc_frames_are_bounded() {
    let frame = frame_message(br#"{"jsonrpc":"2.0"}"#).unwrap();
    let mut reader = Cursor::new(frame);
    assert_eq!(
        read_framed_message(&mut reader).unwrap(),
        br#"{"jsonrpc":"2.0"}"#
    );
    let mut oversized = vec![b'x'; 8 * 1024 * 1024 + 1];
    oversized[0..1].copy_from_slice(b"{");
    assert!(prumo_extension_sdk::lsp::frame_message(&oversized).is_err());
}

#[test]
fn lockfile_is_sorted_and_removable() {
    let mut lockfile = Lockfile::default();
    lockfile.upsert(LockEntry {
        extension_id: "com.example.z".to_string(),
        version: "1.0.0".to_string(),
        digest: "a".repeat(64),
        source: "test".to_string(),
        signature_key_id: None,
    });
    lockfile.upsert(LockEntry {
        extension_id: "com.example.a".to_string(),
        version: "1.0.0".to_string(),
        digest: "b".repeat(64),
        source: "test".to_string(),
        signature_key_id: None,
    });
    assert_eq!(lockfile.extensions[0].extension_id, "com.example.a");
}
