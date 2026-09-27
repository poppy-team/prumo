use crate::manifest::ExtensionManifest;
use ed25519_dalek::{Signature, Signer, SigningKey, Verifier, VerifyingKey};
use flate2::read::GzDecoder;
use flate2::write::GzEncoder;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, BTreeSet};
use std::fs::{self, File};
use std::io::{Read, Write};
use std::path::{Component, Path};
use tar::{Archive, Builder, EntryType, Header};

pub const ARCHIVE_FORMAT: &str = "prumo.viewer.archive/v1";
pub const MAX_ARCHIVE_BYTES: u64 = 64 * 1024 * 1024;
pub const MAX_ENTRY_BYTES: u64 = 16 * 1024 * 1024;
pub const MAX_ARCHIVE_ENTRIES: usize = 2048;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct ExtensionPackage {
    pub format: String,
    pub manifest_bytes: Vec<u8>,
    pub files: BTreeMap<String, Vec<u8>>,
    pub digest: String,
}

pub fn directory_digest(root: &Path) -> Result<String, String> {
    let manifest_bytes = read_manifest_bytes(root)?;
    let mut files = BTreeMap::new();
    collect_package_files(root, root, &mut files)?;
    Ok(package_digest(&manifest_bytes, &files))
}

impl ExtensionPackage {
    pub fn from_directory(root: &Path) -> Result<Self, String> {
        let manifest_bytes = read_manifest_bytes(root)?;
        let manifest: ExtensionManifest = serde_json::from_slice(&manifest_bytes)
            .map_err(|error| format!("could not parse extension manifest: {error}"))?;
        manifest
            .validate()
            .map_err(|errors| format!("invalid extension manifest: {}", errors.join("; ")))?;
        let mut files = BTreeMap::new();
        collect_package_files(root, root, &mut files)?;
        let digest = package_digest(&manifest_bytes, &files);
        Ok(Self {
            format: ARCHIVE_FORMAT.to_string(),
            manifest_bytes,
            files,
            digest,
        })
    }

    pub fn manifest(&self) -> Result<ExtensionManifest, String> {
        serde_json::from_slice(&self.manifest_bytes)
            .map_err(|error| format!("could not parse packaged manifest: {error}"))
    }

    pub fn write_archive(&self, destination: &Path) -> Result<(), String> {
        let file = File::create(destination)
            .map_err(|error| format!("could not create extension archive: {error}"))?;
        let encoder = GzEncoder::new(file, flate2::Compression::default());
        let mut builder = Builder::new(encoder);
        append_entry(
            &mut builder,
            "metadata.json",
            metadata_bytes(&self.format).as_slice(),
        )?;
        append_entry(&mut builder, "manifest.json", &self.manifest_bytes)?;
        for (path, contents) in &self.files {
            let archive_path = format!("payload/{path}");
            validate_archive_path(&archive_path)?;
            append_entry(&mut builder, &archive_path, contents)?;
        }
        builder
            .into_inner()
            .and_then(|encoder| encoder.finish())
            .map(|_| ())
            .map_err(|error| format!("could not finish extension archive: {error}"))
    }

    pub fn install_to(&self, destination: &Path) -> Result<(), String> {
        if destination.exists() {
            return Err(format!(
                "extension destination already exists: {}",
                destination.display()
            ));
        }
        let temporary = destination.with_extension("installing");
        if temporary.exists() {
            fs::remove_dir_all(&temporary)
                .map_err(|error| format!("could not clear extension staging directory: {error}"))?;
        }
        fs::create_dir_all(&temporary)
            .map_err(|error| format!("could not create extension staging directory: {error}"))?;
        write_installed_file(&temporary.join("manifest.json"), &self.manifest_bytes)?;
        for (path, contents) in &self.files {
            validate_relative_path(path)?;
            write_installed_file(&temporary.join(path), contents)?;
        }
        #[cfg(unix)]
        if let Ok(manifest) = self.manifest() {
            use std::os::unix::fs::PermissionsExt;
            for entrypoint in manifest.runtime.entrypoint.values() {
                let path = temporary.join(entrypoint);
                if path.is_file() {
                    let mut permissions = fs::metadata(&path)
                        .map_err(|error| {
                            format!("could not inspect extension entrypoint: {error}")
                        })?
                        .permissions();
                    permissions.set_mode(0o755);
                    fs::set_permissions(&path, permissions).map_err(|error| {
                        format!("could not set extension entrypoint mode: {error}")
                    })?;
                }
            }
        }
        fs::rename(&temporary, destination)
            .map_err(|error| format!("could not commit extension installation: {error}"))
    }

    pub fn read_archive(source: &Path) -> Result<Self, String> {
        let file = File::open(source)
            .map_err(|error| format!("could not open extension archive: {error}"))?;
        if file
            .metadata()
            .map_err(|error| format!("could not inspect extension archive: {error}"))?
            .len()
            > MAX_ARCHIVE_BYTES
        {
            return Err("extension archive exceeds maximum size".to_string());
        }
        let mut archive = Archive::new(GzDecoder::new(file));
        let mut entries = 0usize;
        let mut format = None;
        let mut manifest_bytes = None;
        let mut files = BTreeMap::new();
        let mut seen_paths = BTreeSet::new();
        let mut total_bytes = 0u64;
        for entry in archive
            .entries()
            .map_err(|error| format!("could not read extension archive: {error}"))?
        {
            entries += 1;
            if entries > MAX_ARCHIVE_ENTRIES {
                return Err("extension archive has too many entries".to_string());
            }
            let mut entry = entry
                .map_err(|error| format!("could not read extension archive entry: {error}"))?;
            let path = entry
                .path()
                .map_err(|error| format!("invalid extension archive path: {error}"))?
                .to_string_lossy()
                .to_string();
            validate_archive_path(&path)?;
            if !seen_paths.insert(path.clone()) {
                return Err(format!("duplicate extension archive entry: {path}"));
            }
            let size = entry
                .header()
                .size()
                .map_err(|error| format!("could not read extension archive size: {error}"))?;
            if size > MAX_ENTRY_BYTES {
                return Err(format!("extension archive entry is too large: {path}"));
            }
            if !entry.header().entry_type().is_file() {
                return Err(format!("extension archive entry is not a file: {path}"));
            }
            let mut contents = Vec::with_capacity(size as usize);
            entry
                .read_to_end(&mut contents)
                .map_err(|error| format!("could not read extension archive entry: {error}"))?;
            if contents.len() as u64 > MAX_ENTRY_BYTES {
                return Err(format!("extension archive entry is too large: {path}"));
            }
            total_bytes = total_bytes
                .checked_add(contents.len() as u64)
                .ok_or_else(|| "extension archive size overflow".to_string())?;
            if total_bytes > MAX_ARCHIVE_BYTES {
                return Err("extension archive payload exceeds maximum size".to_string());
            }
            match path.as_str() {
                "metadata.json" => {
                    let metadata: ArchiveMetadata = serde_json::from_slice(&contents)
                        .map_err(|error| format!("invalid extension metadata: {error}"))?;
                    format = Some(metadata.format);
                }
                "manifest.json" => manifest_bytes = Some(contents),
                path if path.starts_with("payload/") => {
                    let relative = path.trim_start_matches("payload/").to_string();
                    validate_relative_path(&relative)?;
                    files.insert(relative, contents);
                }
                _ => return Err(format!("unknown extension archive entry: {path}")),
            }
        }
        let format = format.ok_or_else(|| "extension archive has no metadata".to_string())?;
        if format != ARCHIVE_FORMAT {
            return Err(format!("unsupported extension archive format: {format}"));
        }
        let manifest_bytes =
            manifest_bytes.ok_or_else(|| "extension archive has no manifest".to_string())?;
        let manifest: ExtensionManifest = serde_json::from_slice(&manifest_bytes)
            .map_err(|error| format!("could not parse packaged manifest: {error}"))?;
        manifest
            .validate()
            .map_err(|errors| format!("invalid packaged manifest: {}", errors.join("; ")))?;
        let digest = package_digest(&manifest_bytes, &files);
        Ok(Self {
            format,
            manifest_bytes,
            files,
            digest,
        })
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct SignatureEnvelope {
    pub algorithm: String,
    pub key_id: String,
    pub digest: String,
    pub signature: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct TrustedKey {
    pub key_id: String,
    pub public_key: String,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize, PartialEq, Eq)]
pub struct TrustStore {
    pub keys: Vec<TrustedKey>,
    pub revoked_key_ids: Vec<String>,
}

impl TrustStore {
    pub fn validate(&self) -> Result<(), String> {
        let mut ids = BTreeSet::new();
        for key in &self.keys {
            if !is_hex(&key.key_id, 32) || !ids.insert(&key.key_id) {
                return Err(format!("invalid or duplicate trusted key: {}", key.key_id));
            }
            decode_hex::<32>(&key.public_key)?;
        }
        for key_id in &self.revoked_key_ids {
            if !is_hex(key_id, 32) {
                return Err(format!("invalid revoked key id: {key_id}"));
            }
        }
        Ok(())
    }

    pub fn verify(
        &self,
        package: &ExtensionPackage,
        envelope: &SignatureEnvelope,
    ) -> Result<(), String> {
        self.validate()?;
        if envelope.algorithm != "ed25519" {
            return Err(format!(
                "unsupported signature algorithm: {}",
                envelope.algorithm
            ));
        }
        if envelope.digest != package.digest {
            return Err("extension signature digest does not match package".to_string());
        }
        if self
            .revoked_key_ids
            .iter()
            .any(|key_id| key_id == &envelope.key_id)
        {
            return Err(format!("signing key is revoked: {}", envelope.key_id));
        }
        let trusted = self
            .keys
            .iter()
            .find(|key| key.key_id == envelope.key_id)
            .ok_or_else(|| format!("signing key is not trusted: {}", envelope.key_id))?;
        let public_key = decode_hex::<32>(&trusted.public_key)?;
        let verifying_key = VerifyingKey::from_bytes(&public_key)
            .map_err(|error| format!("invalid trusted public key: {error}"))?;
        if key_id(&public_key) != envelope.key_id {
            return Err("trusted key id does not match public key".to_string());
        }
        let signature_bytes = decode_hex_variable(&envelope.signature)?;
        let signature = Signature::from_slice(&signature_bytes)
            .map_err(|error| format!("invalid extension signature: {error}"))?;
        verifying_key
            .verify(package.digest.as_bytes(), &signature)
            .map_err(|_| "extension signature verification failed".to_string())
    }
}

pub fn sign_package(
    package: &ExtensionPackage,
    signing_key: &SigningKey,
) -> Result<(SignatureEnvelope, TrustedKey), String> {
    let public_key = signing_key.verifying_key().to_bytes();
    let signature = signing_key.sign(package.digest.as_bytes());
    let envelope = SignatureEnvelope {
        algorithm: "ed25519".to_string(),
        key_id: key_id(&public_key),
        digest: package.digest.clone(),
        signature: encode_hex(&signature.to_bytes()),
    };
    let trusted = TrustedKey {
        key_id: envelope.key_id.clone(),
        public_key: encode_hex(&public_key),
    };
    Ok((envelope, trusted))
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct Lockfile {
    pub schema_version: u32,
    pub extensions: Vec<LockEntry>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct LockEntry {
    pub extension_id: String,
    pub version: String,
    pub digest: String,
    pub source: String,
    pub signature_key_id: Option<String>,
}

impl Default for Lockfile {
    fn default() -> Self {
        Self {
            schema_version: 1,
            extensions: Vec::new(),
        }
    }
}

impl Lockfile {
    pub fn upsert(&mut self, entry: LockEntry) {
        self.schema_version = 1;
        self.extensions
            .retain(|current| current.extension_id != entry.extension_id);
        self.extensions.push(entry);
        self.extensions
            .sort_by(|left, right| left.extension_id.cmp(&right.extension_id));
    }

    pub fn remove(&mut self, extension_id: &str) -> bool {
        let before = self.extensions.len();
        self.extensions
            .retain(|entry| entry.extension_id != extension_id);
        before != self.extensions.len()
    }

    pub fn validate(&self) -> Result<(), String> {
        if self.schema_version != 1 {
            return Err(format!(
                "unsupported extension lockfile schema: {}",
                self.schema_version
            ));
        }
        let mut ids = BTreeSet::new();
        for entry in &self.extensions {
            if entry.extension_id.trim().is_empty() || !ids.insert(&entry.extension_id) {
                return Err(format!(
                    "invalid or duplicate lockfile extension: {}",
                    entry.extension_id
                ));
            }
            if entry.version.trim().is_empty() || entry.source.trim().is_empty() {
                return Err(format!(
                    "lockfile entry is missing version or source: {}",
                    entry.extension_id
                ));
            }
            if !is_hex(&entry.digest, 64) {
                return Err(format!(
                    "lockfile entry has invalid digest: {}",
                    entry.extension_id
                ));
            }
            if entry
                .signature_key_id
                .as_deref()
                .is_some_and(|key_id| !is_hex(key_id, 32))
            {
                return Err(format!(
                    "lockfile entry has invalid signature key: {}",
                    entry.extension_id
                ));
            }
        }
        Ok(())
    }

    pub fn entry(&self, extension_id: &str) -> Option<&LockEntry> {
        self.extensions
            .iter()
            .find(|entry| entry.extension_id == extension_id)
    }
}

pub fn read_trust_store(path: &Path) -> Result<TrustStore, String> {
    let bytes = fs::read(path).map_err(|error| format!("could not read trust store: {error}"))?;
    let trust_store: TrustStore =
        serde_json::from_slice(&bytes).map_err(|error| format!("invalid trust store: {error}"))?;
    trust_store.validate()?;
    Ok(trust_store)
}

pub fn write_trust_store(path: &Path, trust_store: &TrustStore) -> Result<(), String> {
    trust_store.validate()?;
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)
            .map_err(|error| format!("could not create trust store directory: {error}"))?;
    }
    let bytes = serde_json::to_vec_pretty(trust_store)
        .map_err(|error| format!("could not encode trust store: {error}"))?;
    fs::write(path, bytes).map_err(|error| format!("could not write trust store: {error}"))
}

pub fn read_lockfile(path: &Path) -> Result<Lockfile, String> {
    let bytes =
        fs::read(path).map_err(|error| format!("could not read extension lockfile: {error}"))?;
    let lockfile: Lockfile = serde_json::from_slice(&bytes)
        .map_err(|error| format!("invalid extension lockfile: {error}"))?;
    lockfile.validate()?;
    Ok(lockfile)
}

pub fn write_lockfile(path: &Path, lockfile: &Lockfile) -> Result<(), String> {
    lockfile.validate()?;
    let parent = path.parent().unwrap_or_else(|| Path::new("."));
    fs::create_dir_all(parent)
        .map_err(|error| format!("could not create lockfile directory: {error}"))?;
    let temporary = path.with_extension("tmp");
    let mut file = File::create(&temporary)
        .map_err(|error| format!("could not create extension lockfile: {error}"))?;
    serde_json::to_writer_pretty(&mut file, lockfile)
        .map_err(|error| format!("could not write extension lockfile: {error}"))?;
    file.write_all(b"\n")
        .map_err(|error| format!("could not finish extension lockfile: {error}"))?;
    fs::rename(&temporary, path)
        .map_err(|error| format!("could not commit extension lockfile: {error}"))
}

pub fn read_signature(path: &Path) -> Result<SignatureEnvelope, String> {
    let bytes = fs::read(path).map_err(|error| format!("could not read signature: {error}"))?;
    serde_json::from_slice(&bytes).map_err(|error| format!("invalid signature: {error}"))
}

pub fn write_signature(path: &Path, signature: &SignatureEnvelope) -> Result<(), String> {
    if let Some(parent) = path.parent() {
        fs::create_dir_all(parent)
            .map_err(|error| format!("could not create signature directory: {error}"))?;
    }
    let bytes = serde_json::to_vec_pretty(signature)
        .map_err(|error| format!("could not encode signature: {error}"))?;
    fs::write(path, bytes).map_err(|error| format!("could not write signature: {error}"))
}

fn metadata_bytes(format: &str) -> Vec<u8> {
    serde_json::to_vec(&ArchiveMetadata {
        format: format.to_string(),
    })
    .unwrap_or_else(|_| b"{\"format\":\"prumo.viewer.archive/v1\"}".to_vec())
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct ArchiveMetadata {
    format: String,
}

fn read_manifest_bytes(root: &Path) -> Result<Vec<u8>, String> {
    let path = root.join("manifest.json");
    let metadata = fs::symlink_metadata(&path)
        .map_err(|error| format!("could not inspect extension manifest: {error}"))?;
    if metadata.file_type().is_symlink() {
        return Err("extension manifest must not be a symlink".to_string());
    }
    fs::read(&path).map_err(|error| format!("could not read extension manifest: {error}"))
}

fn collect_package_files(
    root: &Path,
    current: &Path,
    files: &mut BTreeMap<String, Vec<u8>>,
) -> Result<(), String> {
    for entry in fs::read_dir(current)
        .map_err(|error| format!("could not read extension package: {error}"))?
    {
        let entry =
            entry.map_err(|error| format!("could not read extension package entry: {error}"))?;
        let path = entry.path();
        let metadata = fs::symlink_metadata(&path)
            .map_err(|error| format!("could not inspect extension package entry: {error}"))?;
        if metadata.file_type().is_symlink() {
            return Err(format!(
                "extension package contains symlink: {}",
                path.display()
            ));
        }
        if path == root.join("manifest.json") {
            continue;
        }
        if path.is_dir() {
            collect_package_files(root, &path, files)?;
            continue;
        }
        if !path.is_file() {
            continue;
        }
        let relative = path
            .strip_prefix(root)
            .map_err(|_| "extension package path escaped root".to_string())?
            .to_string_lossy()
            .to_string();
        validate_relative_path(&relative)?;
        let contents = fs::read(&path)
            .map_err(|error| format!("could not read extension package file: {error}"))?;
        if contents.len() as u64 > MAX_ENTRY_BYTES {
            return Err(format!("extension package file is too large: {relative}"));
        }
        files.insert(relative, contents);
        if files.len() > MAX_ARCHIVE_ENTRIES {
            return Err("extension package has too many files".to_string());
        }
    }
    let total_bytes = files.values().try_fold(0u64, |total, contents| {
        total.checked_add(contents.len() as u64)
    });
    if total_bytes.is_none_or(|total| total > MAX_ARCHIVE_BYTES) {
        return Err("extension package payload exceeds maximum size".to_string());
    }
    Ok(())
}

fn write_installed_file(path: &Path, contents: &[u8]) -> Result<(), String> {
    let parent = path
        .parent()
        .ok_or_else(|| "extension package file has no parent".to_string())?;
    fs::create_dir_all(parent)
        .map_err(|error| format!("could not create extension package directory: {error}"))?;
    fs::write(path, contents)
        .map_err(|error| format!("could not write extension package file: {error}"))
}

fn append_entry(
    builder: &mut Builder<GzEncoder<File>>,
    path: &str,
    contents: &[u8],
) -> Result<(), String> {
    let mut header = Header::new_gnu();
    header.set_entry_type(EntryType::Regular);
    header.set_mode(0o644);
    header.set_size(contents.len() as u64);
    header.set_cksum();
    builder
        .append_data(&mut header, path, contents)
        .map_err(|error| format!("could not add extension archive entry: {error}"))
}

fn package_digest(manifest: &[u8], files: &BTreeMap<String, Vec<u8>>) -> String {
    let mut digest = Sha256::new();
    digest.update(manifest);
    for (path, contents) in files {
        digest.update(path.as_bytes());
        digest.update([0]);
        digest.update((contents.len() as u64).to_le_bytes());
        digest.update(contents);
    }
    encode_hex(&digest.finalize())
}

fn key_id(public_key: &[u8; 32]) -> String {
    encode_hex(&Sha256::digest(public_key))[..32].to_string()
}

fn validate_archive_path(path: &str) -> Result<(), String> {
    if path.is_empty() || path.starts_with('/') || path.contains('\\') {
        return Err(format!("invalid extension archive path: {path}"));
    }
    validate_relative_path(path)
}

fn validate_relative_path(path: &str) -> Result<(), String> {
    let path = Path::new(path);
    if path.as_os_str().is_empty()
        || path.is_absolute()
        || path.components().any(|component| {
            matches!(
                component,
                Component::ParentDir | Component::RootDir | Component::Prefix(_)
            )
        })
    {
        return Err(format!(
            "extension package path is unsafe: {}",
            path.display()
        ));
    }
    Ok(())
}

fn is_hex(value: &str, length: usize) -> bool {
    value.len() == length && value.chars().all(|character| character.is_ascii_hexdigit())
}

fn encode_hex(bytes: &[u8]) -> String {
    bytes.iter().map(|byte| format!("{byte:02x}")).collect()
}

fn decode_hex_variable(value: &str) -> Result<Vec<u8>, String> {
    if !value.len().is_multiple_of(2) {
        return Err("hex value has odd length".to_string());
    }
    (0..value.len())
        .step_by(2)
        .map(|index| {
            u8::from_str_radix(&value[index..index + 2], 16)
                .map_err(|_| "invalid hex value".to_string())
        })
        .collect()
}

fn decode_hex<const N: usize>(value: &str) -> Result<[u8; N], String> {
    let bytes = decode_hex_variable(value)?;
    if bytes.len() != N {
        return Err(format!("hex value must contain {N} bytes"));
    }
    bytes
        .try_into()
        .map_err(|_| format!("hex value must contain {N} bytes"))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::BTreeMap;

    #[test]
    fn signs_and_verifies_package() {
        let package = ExtensionPackage {
            format: ARCHIVE_FORMAT.to_string(),
            manifest_bytes: b"{}".to_vec(),
            files: BTreeMap::from([("payload.txt".to_string(), b"hello".to_vec())]),
            digest: String::new(),
        };
        let mut package = package;
        package.digest = package_digest(&package.manifest_bytes, &package.files);
        let signing_key = SigningKey::from_bytes(&[7; 32]);
        let (signature, trusted) = sign_package(&package, &signing_key).unwrap();
        let trust = TrustStore {
            keys: vec![trusted],
            revoked_key_ids: Vec::new(),
        };
        trust.verify(&package, &signature).unwrap();
    }

    #[test]
    fn archive_round_trip_and_lockfile_round_trip() {
        let directory = tempfile::tempdir().unwrap();
        let package_directory = directory.path().join("package");
        fs::create_dir_all(&package_directory).unwrap();
        let manifest = include_str!("../../extensions/editor-navigation/manifest.json");
        fs::write(package_directory.join("manifest.json"), manifest).unwrap();
        fs::write(package_directory.join("README.md"), "extension").unwrap();
        let package = ExtensionPackage::from_directory(&package_directory).unwrap();
        let archive = directory.path().join("extension.prumoext");
        package.write_archive(&archive).unwrap();
        let unpacked = ExtensionPackage::read_archive(&archive).unwrap();
        let installed = directory.path().join("installed");
        unpacked.install_to(&installed).unwrap();
        assert!(installed.join("README.md").is_file());
        assert_eq!(unpacked.digest, package.digest);
        assert_eq!(
            unpacked.files.get("README.md"),
            package.files.get("README.md")
        );
        let lock_path = directory.path().join("extensions.lock.json");
        let mut lockfile = Lockfile::default();
        lockfile.upsert(LockEntry {
            extension_id: "com.prumo.editor-navigation".to_string(),
            version: "0.1.0".to_string(),
            digest: package.digest.clone(),
            source: archive.display().to_string(),
            signature_key_id: None,
        });
        write_lockfile(&lock_path, &lockfile).unwrap();
        assert_eq!(read_lockfile(&lock_path).unwrap(), lockfile);
        assert!(lockfile.remove("com.prumo.editor-navigation"));
    }

    #[test]
    fn rejects_revoked_signature() {
        let package = ExtensionPackage {
            format: ARCHIVE_FORMAT.to_string(),
            manifest_bytes: b"{}".to_vec(),
            files: BTreeMap::new(),
            digest: "digest".to_string(),
        };
        let signing_key = SigningKey::from_bytes(&[8; 32]);
        let (signature, trusted) = sign_package(&package, &signing_key).unwrap();
        let trust = TrustStore {
            keys: vec![trusted],
            revoked_key_ids: vec![signature.key_id.clone()],
        };
        assert!(trust.verify(&package, &signature).is_err());
    }
}
