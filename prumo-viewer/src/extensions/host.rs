use crate::extensions::registry::ExtensionRecord;
use prumo_extension_sdk::dap::DapClient;
use prumo_extension_sdk::grant::ExtensionGrant;
use prumo_extension_sdk::lsp::{
    LspCancellation, LspClient, LspCodeAction, LspCompletionResponse, LspRange, LspTextEdit,
};
use prumo_extension_sdk::manifest::{LSP_CAPABILITY, RuntimeKind};
use prumo_extension_sdk::process::ProcessRuntime;
use prumo_extension_sdk::rpc::{RpcError, RpcRequest};
use serde_json::Value;
use std::collections::{HashMap, HashSet};
use std::fs::File;
use std::path::{Path, PathBuf};
use std::sync::{Arc, Mutex, mpsc};
use std::thread;
use std::time::Duration;

#[derive(Clone)]
pub struct ExtensionHost {
    inner: Arc<Mutex<ExtensionHostInner>>,
}

struct ExtensionHostInner {
    workspace_root: PathBuf,
    processes: HashMap<String, ProcessEntry>,
    lsp_servers: HashMap<String, LspEntry>,
    debug_servers: HashMap<String, DapEntry>,
    granted_capabilities: HashMap<String, HashSet<String>>,
}

struct ProcessEntry {
    runtime: ProcessRuntime,
    capabilities: HashSet<String>,
}

const LSP_COMMAND_CAPACITY: usize = 32;
const LSP_REQUEST_TIMEOUT: Duration = Duration::from_secs(10);

struct LspEntry {
    extension_id: String,
    language: String,
    actor: LspActor,
}

struct LspDocument {
    version: i64,
    content: String,
}

enum LspResponse {
    Value(Value),
    Completion(LspCompletionResponse),
    Formatting(Vec<LspTextEdit>),
    CodeActions(Vec<LspCodeAction>),
}

enum LspRequest {
    WorkspaceSymbols {
        query: String,
    },
    Hover {
        file: String,
        line: usize,
        character: usize,
    },
    Definition {
        file: String,
        line: usize,
        character: usize,
    },
    Completion {
        file: String,
        line: usize,
        character: usize,
    },
    Formatting {
        file: String,
        tab_size: u32,
        insert_spaces: bool,
    },
    CodeActions {
        file: String,
        range: LspRange,
        diagnostics: Vec<Value>,
    },
}

enum LspCommand {
    Request {
        request: LspRequest,
        cancellation: LspCancellation,
        reply: mpsc::SyncSender<Result<LspResponse, String>>,
    },
    SyncDocument {
        path: PathBuf,
        content: String,
        version: u64,
        reply: mpsc::SyncSender<Result<(), String>>,
    },
    Save {
        path: PathBuf,
        reply: mpsc::SyncSender<Result<(), String>>,
    },
    Close {
        path: PathBuf,
        reply: mpsc::SyncSender<Result<(), String>>,
    },
    Drain(mpsc::SyncSender<Vec<Value>>),
}

#[derive(Clone)]
struct LspActor {
    sender: mpsc::SyncSender<LspCommand>,
}

impl LspActor {
    fn start(mut client: LspClient, language: String) -> Self {
        let (sender, receiver) = mpsc::sync_channel(LSP_COMMAND_CAPACITY);
        thread::spawn(move || {
            let mut documents = HashMap::<PathBuf, LspDocument>::new();
            while let Ok(command) = receiver.recv() {
                match command {
                    LspCommand::Request {
                        request,
                        cancellation,
                        reply,
                    } => {
                        let result = match request {
                            LspRequest::WorkspaceSymbols { query } => client
                                .workspace_symbols_with_cancellation(
                                    &query,
                                    LSP_REQUEST_TIMEOUT,
                                    &cancellation,
                                )
                                .map(LspResponse::Value),
                            LspRequest::Hover {
                                file,
                                line,
                                character,
                            } => client
                                .hover_with_cancellation(
                                    &file,
                                    line,
                                    character,
                                    LSP_REQUEST_TIMEOUT,
                                    &cancellation,
                                )
                                .map(LspResponse::Value),
                            LspRequest::Definition {
                                file,
                                line,
                                character,
                            } => client
                                .definition_with_cancellation(
                                    &file,
                                    line,
                                    character,
                                    LSP_REQUEST_TIMEOUT,
                                    &cancellation,
                                )
                                .map(LspResponse::Value),
                            LspRequest::Completion {
                                file,
                                line,
                                character,
                            } => client
                                .completion_with_cancellation(
                                    &file,
                                    line,
                                    character,
                                    LSP_REQUEST_TIMEOUT,
                                    &cancellation,
                                )
                                .map(LspResponse::Completion),
                            LspRequest::Formatting {
                                file,
                                tab_size,
                                insert_spaces,
                            } => client
                                .formatting_with_cancellation(
                                    &file,
                                    tab_size,
                                    insert_spaces,
                                    LSP_REQUEST_TIMEOUT,
                                    &cancellation,
                                )
                                .map(LspResponse::Formatting),
                            LspRequest::CodeActions {
                                file,
                                range,
                                diagnostics,
                            } => client
                                .code_actions_with_cancellation(
                                    &file,
                                    range,
                                    diagnostics,
                                    LSP_REQUEST_TIMEOUT,
                                    &cancellation,
                                )
                                .map(LspResponse::CodeActions),
                        };
                        let _ = reply.send(result);
                    }
                    LspCommand::SyncDocument {
                        path,
                        content,
                        version,
                        reply,
                    } => {
                        let version = i64::try_from(version).unwrap_or(i64::MAX);
                        let result = match documents.get(&path) {
                            None => client.did_open(
                                &path.to_string_lossy(),
                                &language,
                                &content,
                                version,
                            ),
                            Some(document)
                                if document.version != version || document.content != content =>
                            {
                                client.did_change(&path.to_string_lossy(), version, &content)
                            }
                            Some(_) => Ok(()),
                        };
                        if result.is_ok() {
                            documents.insert(path, LspDocument { version, content });
                        }
                        let _ = reply.send(result);
                    }
                    LspCommand::Save { path, reply } => {
                        let result = client.did_save(&path.to_string_lossy());
                        let _ = reply.send(result);
                    }
                    LspCommand::Close { path, reply } => {
                        let result = if documents.remove(&path).is_some() {
                            client.did_close(&path.to_string_lossy())
                        } else {
                            Ok(())
                        };
                        let _ = reply.send(result);
                    }
                    LspCommand::Drain(reply) => {
                        let _ = reply.send(client.drain_notifications());
                    }
                }
            }
            client.stop();
        });
        Self { sender }
    }

    fn request(&self, request: LspRequest) -> Result<LspResponse, String> {
        let cancellation = LspCancellation::new();
        let (reply, response) = mpsc::sync_channel(1);
        self.send(LspCommand::Request {
            request,
            cancellation: cancellation.clone(),
            reply,
        })?;
        match response.recv_timeout(LSP_REQUEST_TIMEOUT) {
            Ok(result) => result,
            Err(mpsc::RecvTimeoutError::Timeout) => {
                cancellation.cancel();
                Err("LSP request timed out".to_string())
            }
            Err(mpsc::RecvTimeoutError::Disconnected) => Err("LSP server stopped".to_string()),
        }
    }

    fn sync_document(&self, path: &Path, content: &str, version: u64) -> Result<(), String> {
        let (reply, response) = mpsc::sync_channel(1);
        self.send(LspCommand::SyncDocument {
            path: path.to_path_buf(),
            content: content.to_string(),
            version,
            reply,
        })?;
        response
            .recv_timeout(LSP_REQUEST_TIMEOUT)
            .map_err(|_| "LSP document sync timed out".to_string())?
    }

    fn save(&self, path: &Path) -> Result<(), String> {
        let (reply, response) = mpsc::sync_channel(1);
        self.send(LspCommand::Save {
            path: path.to_path_buf(),
            reply,
        })?;
        response
            .recv_timeout(LSP_REQUEST_TIMEOUT)
            .map_err(|_| "LSP save notification timed out".to_string())?
    }

    fn close(&self, path: &Path) -> Result<(), String> {
        let (reply, response) = mpsc::sync_channel(1);
        self.send(LspCommand::Close {
            path: path.to_path_buf(),
            reply,
        })?;
        response
            .recv_timeout(LSP_REQUEST_TIMEOUT)
            .map_err(|_| "LSP close notification timed out".to_string())?
    }

    fn drain_notifications(&self) -> Vec<Value> {
        let (reply, response) = mpsc::sync_channel(1);
        if self.send(LspCommand::Drain(reply)).is_err() {
            return Vec::new();
        }
        response
            .recv_timeout(Duration::from_millis(100))
            .unwrap_or_default()
    }

    fn send(&self, command: LspCommand) -> Result<(), String> {
        self.sender.try_send(command).map_err(|error| match error {
            mpsc::TrySendError::Full(_) => "LSP request queue is full".to_string(),
            mpsc::TrySendError::Disconnected(_) => "LSP server stopped".to_string(),
        })
    }
}

#[derive(Debug, Clone, PartialEq)]
pub struct LspNotification {
    pub extension_id: String,
    pub language: String,
    pub method: String,
    pub params: Value,
}

struct DapEntry {
    client: DapClient,
}

#[derive(Debug, Clone, PartialEq)]
pub struct ExtensionRequest {
    pub extension_id: String,
    pub request: RpcRequest,
}

impl ExtensionHost {
    pub fn new(workspace_root: PathBuf) -> Self {
        Self {
            inner: Arc::new(Mutex::new(ExtensionHostInner {
                workspace_root,
                processes: HashMap::new(),
                lsp_servers: HashMap::new(),
                debug_servers: HashMap::new(),
                granted_capabilities: HashMap::new(),
            })),
        }
    }

    pub fn start(&self, records: &[ExtensionRecord], grants: &[ExtensionGrant]) -> Vec<String> {
        let mut errors = Vec::new();
        if let Ok(mut inner) = self.inner.lock() {
            for record in records {
                if let Some(grant) = grants
                    .iter()
                    .find(|grant| grant.extension_id == record.manifest.id)
                    && grant
                        .validate_for(&record.manifest, &record.manifest_digest)
                        .is_ok()
                {
                    inner.granted_capabilities.insert(
                        record.manifest.id.clone(),
                        grant.capabilities.iter().cloned().collect(),
                    );
                }
            }
        }
        for record in records {
            if record.manifest.runtime.kind != RuntimeKind::Process {
                continue;
            }
            if let Err(manifest_errors) = record.manifest.validate() {
                errors.push(format!(
                    "{}: invalid manifest: {}",
                    record.manifest.id,
                    manifest_errors.join("; ")
                ));
                continue;
            }
            if !record.manifest.supports_host(env!("CARGO_PKG_VERSION")) {
                errors.push(format!(
                    "{}: host version is not supported",
                    record.manifest.id
                ));
                continue;
            }
            let Some(grant) = grants
                .iter()
                .find(|grant| grant.extension_id == record.manifest.id)
            else {
                continue;
            };
            if let Err(grant_errors) = grant.validate_for(&record.manifest, &record.manifest_digest)
            {
                errors.push(format!(
                    "{}: invalid grant: {}",
                    record.manifest.id,
                    grant_errors.join("; ")
                ));
                continue;
            }
            let workspace_root = self
                .inner
                .lock()
                .map_err(|_| "extension host lock was poisoned".to_string())
                .map(|inner| inner.workspace_root.clone());
            let workspace_root = match workspace_root {
                Ok(root) => root,
                Err(error) => {
                    errors.push(error);
                    continue;
                }
            };
            match ProcessRuntime::start(
                &record.manifest,
                &record.root,
                &workspace_root,
                &grant.capabilities,
            ) {
                Ok(runtime) => {
                    if let Ok(mut inner) = self.inner.lock() {
                        inner.processes.insert(
                            record.manifest.id.clone(),
                            ProcessEntry {
                                runtime,
                                capabilities: grant.capabilities.iter().cloned().collect(),
                            },
                        );
                    }
                }
                Err(error) => errors.push(format!("{}: {error}", record.manifest.id)),
            }
        }
        self.start_lsp_servers(records, grants, &mut errors);
        self.start_debuggers(records, grants, &mut errors);
        errors
    }

    fn start_lsp_servers(
        &self,
        records: &[ExtensionRecord],
        grants: &[ExtensionGrant],
        errors: &mut Vec<String>,
    ) {
        for record in records {
            if record.manifest.contributions.lsp.is_empty() {
                continue;
            }
            if let Err(manifest_errors) = record.manifest.validate() {
                errors.push(format!(
                    "{}: invalid manifest: {}",
                    record.manifest.id,
                    manifest_errors.join("; ")
                ));
                continue;
            }
            if !record.manifest.supports_host(env!("CARGO_PKG_VERSION")) {
                errors.push(format!(
                    "{}: host version is not supported",
                    record.manifest.id
                ));
                continue;
            }
            let Some(grant) = grants
                .iter()
                .find(|grant| grant.extension_id == record.manifest.id)
            else {
                continue;
            };
            if let Err(grant_errors) = grant.validate_for(&record.manifest, &record.manifest_digest)
            {
                errors.push(format!(
                    "{}: invalid grant: {}",
                    record.manifest.id,
                    grant_errors.join("; ")
                ));
                continue;
            }
            if !grant
                .capabilities
                .iter()
                .any(|capability| capability == LSP_CAPABILITY)
            {
                errors.push(format!(
                    "{}: grant does not include {LSP_CAPABILITY}",
                    record.manifest.id
                ));
                continue;
            }
            let workspace_root = match self.inner.lock().map(|inner| inner.workspace_root.clone()) {
                Ok(root) => root,
                Err(_) => {
                    errors.push("extension host lock was poisoned".to_string());
                    continue;
                }
            };
            for server in &record.manifest.contributions.lsp {
                let key = lsp_key(&record.manifest.id, &server.language);
                let command = resolve_lsp_command(&record.root, &server.command);
                let client = match command {
                    Ok(command) => LspClient::start(
                        command,
                        &server.args,
                        &workspace_root,
                        std::time::Duration::from_secs(10),
                    ),
                    Err(error) => Err(error),
                };
                match client {
                    Ok(client) => {
                        let actor = LspActor::start(client, server.language.clone());
                        if let Ok(mut inner) = self.inner.lock() {
                            inner.lsp_servers.insert(
                                key,
                                LspEntry {
                                    extension_id: record.manifest.id.clone(),
                                    language: server.language.clone(),
                                    actor,
                                },
                            );
                        }
                    }
                    Err(error) => errors.push(format!(
                        "{}: could not start {}: {error}",
                        record.manifest.id, server.language
                    )),
                }
            }
        }
    }

    fn lsp_actor(&self, extension_id: &str, language: &str) -> Result<LspActor, String> {
        let inner = self
            .inner
            .lock()
            .map_err(|_| "extension host lock was poisoned".to_string())?;
        inner
            .lsp_servers
            .get(&lsp_key(extension_id, language))
            .map(|entry| entry.actor.clone())
            .ok_or_else(|| format!("LSP server is not active: {extension_id}/{language}"))
    }

    pub fn lsp_workspace_symbols(
        &self,
        extension_id: &str,
        language: &str,
        query: &str,
    ) -> Result<Value, String> {
        let actor = self.lsp_actor(extension_id, language)?;
        match actor.request(LspRequest::WorkspaceSymbols {
            query: query.to_string(),
        })? {
            LspResponse::Value(value) => Ok(value),
            _ => Err("LSP returned an unexpected response".to_string()),
        }
    }

    pub fn lsp_hover(
        &self,
        extension_id: &str,
        language: &str,
        file: &str,
        line: usize,
        character: usize,
    ) -> Result<Value, String> {
        let actor = self.lsp_actor(extension_id, language)?;
        match actor.request(LspRequest::Hover {
            file: file.to_string(),
            line,
            character,
        })? {
            LspResponse::Value(value) => Ok(value),
            _ => Err("LSP returned an unexpected response".to_string()),
        }
    }

    pub fn lsp_definition(
        &self,
        extension_id: &str,
        language: &str,
        file: &str,
        line: usize,
        character: usize,
    ) -> Result<Value, String> {
        let actor = self.lsp_actor(extension_id, language)?;
        match actor.request(LspRequest::Definition {
            file: file.to_string(),
            line,
            character,
        })? {
            LspResponse::Value(value) => Ok(value),
            _ => Err("LSP returned an unexpected response".to_string()),
        }
    }

    pub fn lsp_completion(
        &self,
        extension_id: &str,
        language: &str,
        file: &str,
        line: usize,
        character: usize,
    ) -> Result<LspCompletionResponse, String> {
        let actor = self.lsp_actor(extension_id, language)?;
        match actor.request(LspRequest::Completion {
            file: file.to_string(),
            line,
            character,
        })? {
            LspResponse::Completion(response) => Ok(response),
            _ => Err("LSP returned an unexpected response".to_string()),
        }
    }

    pub fn lsp_formatting(
        &self,
        extension_id: &str,
        language: &str,
        file: &str,
        tab_size: u32,
        insert_spaces: bool,
    ) -> Result<Vec<LspTextEdit>, String> {
        let actor = self.lsp_actor(extension_id, language)?;
        match actor.request(LspRequest::Formatting {
            file: file.to_string(),
            tab_size,
            insert_spaces,
        })? {
            LspResponse::Formatting(edits) => Ok(edits),
            _ => Err("LSP returned an unexpected response".to_string()),
        }
    }

    pub fn lsp_code_actions(
        &self,
        extension_id: &str,
        language: &str,
        file: &str,
        range: LspRange,
        diagnostics: Vec<Value>,
    ) -> Result<Vec<LspCodeAction>, String> {
        let actor = self.lsp_actor(extension_id, language)?;
        match actor.request(LspRequest::CodeActions {
            file: file.to_string(),
            range,
            diagnostics,
        })? {
            LspResponse::CodeActions(actions) => Ok(actions),
            _ => Err("LSP returned an unexpected response".to_string()),
        }
    }

    pub fn drain_lsp_notifications(&self) -> Vec<LspNotification> {
        let entries = self
            .inner
            .lock()
            .map(|inner| {
                inner
                    .lsp_servers
                    .values()
                    .map(|entry| {
                        (
                            entry.actor.clone(),
                            entry.extension_id.clone(),
                            entry.language.clone(),
                        )
                    })
                    .collect::<Vec<_>>()
            })
            .unwrap_or_default();
        let mut notifications = Vec::new();
        for (actor, extension_id, language) in entries {
            for notification in actor.drain_notifications() {
                let Some(method) = notification.get("method").and_then(Value::as_str) else {
                    continue;
                };
                notifications.push(LspNotification {
                    extension_id: extension_id.clone(),
                    language: language.clone(),
                    method: method.to_string(),
                    params: notification
                        .get("params")
                        .cloned()
                        .unwrap_or_else(|| Value::Object(Default::default())),
                });
            }
        }
        notifications
    }

    pub fn sync_active_document(
        &self,
        path: &Path,
        language: &str,
        content: &str,
        version: u64,
    ) -> Result<(), String> {
        let actors = self
            .inner
            .lock()
            .map(|inner| {
                inner
                    .lsp_servers
                    .values()
                    .filter(|entry| entry.language == language)
                    .map(|entry| entry.actor.clone())
                    .collect::<Vec<_>>()
            })
            .map_err(|_| "extension host lock was poisoned".to_string())?;
        if actors.is_empty() {
            return Err(format!("No active language server for {language}"));
        }
        for actor in actors {
            actor.sync_document(path, content, version)?;
        }
        Ok(())
    }

    pub fn notify_lsp_save(&self, path: &Path, language: &str) -> Result<(), String> {
        let actors = self
            .inner
            .lock()
            .map(|inner| {
                inner
                    .lsp_servers
                    .values()
                    .filter(|entry| entry.language == language)
                    .map(|entry| entry.actor.clone())
                    .collect::<Vec<_>>()
            })
            .map_err(|_| "extension host lock was poisoned".to_string())?;
        if actors.is_empty() {
            return Err(format!("No active language server for {language}"));
        }
        for actor in actors {
            actor.save(path)?;
        }
        Ok(())
    }

    pub fn close_lsp_document(&self, path: &Path, language: &str) -> Result<(), String> {
        let actors = self
            .inner
            .lock()
            .map(|inner| {
                inner
                    .lsp_servers
                    .values()
                    .filter(|entry| entry.language == language)
                    .map(|entry| entry.actor.clone())
                    .collect::<Vec<_>>()
            })
            .map_err(|_| "extension host lock was poisoned".to_string())?;
        if actors.is_empty() {
            return Err(format!("No active language server for {language}"));
        }
        for actor in actors {
            actor.close(path)?;
        }
        Ok(())
    }

    pub fn active_lsp_for_path(&self, path: &Path) -> Option<(String, String)> {
        let language = language_for_path(path)?;
        let inner = self.inner.lock().ok()?;
        inner
            .lsp_servers
            .values()
            .find(|entry| entry.language == language)
            .map(|entry| (entry.extension_id.clone(), entry.language.clone()))
    }

    fn start_debuggers(
        &self,
        records: &[ExtensionRecord],
        grants: &[ExtensionGrant],
        errors: &mut Vec<String>,
    ) {
        for record in records {
            if record.manifest.contributions.debuggers.is_empty() {
                continue;
            }
            if let Err(manifest_errors) = record.manifest.validate() {
                errors.push(format!(
                    "{}: invalid manifest: {}",
                    record.manifest.id,
                    manifest_errors.join("; ")
                ));
                continue;
            }
            if !record.manifest.supports_host(env!("CARGO_PKG_VERSION")) {
                errors.push(format!(
                    "{}: host version is not supported",
                    record.manifest.id
                ));
                continue;
            }
            let Some(grant) = grants
                .iter()
                .find(|grant| grant.extension_id == record.manifest.id)
            else {
                continue;
            };
            if let Err(grant_errors) = grant.validate_for(&record.manifest, &record.manifest_digest)
            {
                errors.push(format!(
                    "{}: invalid grant: {}",
                    record.manifest.id,
                    grant_errors.join("; ")
                ));
                continue;
            }
            if !grant
                .capabilities
                .iter()
                .any(|capability| capability == prumo_extension_sdk::manifest::DEBUG_CAPABILITY)
            {
                errors.push(format!(
                    "{}: grant does not include debugger capability",
                    record.manifest.id
                ));
                continue;
            }
            let workspace_root = match self.inner.lock().map(|inner| inner.workspace_root.clone()) {
                Ok(root) => root,
                Err(_) => {
                    errors.push("extension host lock was poisoned".to_string());
                    continue;
                }
            };
            for debugger in &record.manifest.contributions.debuggers {
                let key = format!("{}:{}", record.manifest.id, debugger.id);
                let command = match resolve_lsp_command(&record.root, &debugger.command) {
                    Ok(command) => command,
                    Err(error) => {
                        errors.push(format!("{}: {error}", record.manifest.id));
                        continue;
                    }
                };
                match DapClient::start(
                    command,
                    &debugger.args,
                    &workspace_root,
                    std::time::Duration::from_secs(10),
                ) {
                    Ok(client) => {
                        if let Ok(mut inner) = self.inner.lock() {
                            inner.debug_servers.insert(key, DapEntry { client });
                        }
                    }
                    Err(error) => errors.push(format!(
                        "{}: could not start debugger {}: {error}",
                        record.manifest.id, debugger.id
                    )),
                }
            }
        }
    }

    pub fn debug_request(
        &self,
        extension_id: &str,
        debugger_id: &str,
        command: &str,
        arguments: Value,
    ) -> Result<Value, String> {
        let mut inner = self
            .inner
            .lock()
            .map_err(|_| "extension host lock was poisoned".to_string())?;
        let entry = inner
            .debug_servers
            .get_mut(&format!("{extension_id}:{debugger_id}"))
            .ok_or_else(|| format!("debugger is not active: {extension_id}/{debugger_id}"))?;
        entry
            .client
            .request(command, arguments, std::time::Duration::from_secs(10))
    }

    pub fn is_debug_active(&self, extension_id: &str, debugger_id: &str) -> bool {
        self.inner
            .lock()
            .map(|inner| {
                inner
                    .debug_servers
                    .contains_key(&format!("{extension_id}:{debugger_id}"))
            })
            .unwrap_or(false)
    }

    pub fn is_lsp_active(&self, extension_id: &str, language: &str) -> bool {
        self.inner
            .lock()
            .map(|inner| {
                inner
                    .lsp_servers
                    .contains_key(&lsp_key(extension_id, language))
            })
            .unwrap_or(false)
    }

    pub fn is_active(&self, extension_id: &str) -> bool {
        self.inner
            .lock()
            .map(|inner| inner.processes.contains_key(extension_id))
            .unwrap_or(false)
    }

    pub fn invoke(&self, extension_id: &str, method: &str, params: Value) -> Result<Value, String> {
        let mut inner = self
            .inner
            .lock()
            .map_err(|_| "extension host lock was poisoned".to_string())?;
        let entry = inner
            .processes
            .get_mut(extension_id)
            .ok_or_else(|| format!("extension process is not active: {extension_id}"))?;
        entry.runtime.invoke(method, params)
    }

    pub fn poll_requests(&self) -> Vec<ExtensionRequest> {
        let Ok(mut inner) = self.inner.lock() else {
            return Vec::new();
        };
        let mut requests = Vec::new();
        for (extension_id, entry) in &mut inner.processes {
            while let Some(request) = entry.runtime.pending_request() {
                requests.push(ExtensionRequest {
                    extension_id: extension_id.clone(),
                    request,
                });
            }
        }
        requests
    }

    pub fn has_granted_capability(&self, extension_id: &str, capability: &str) -> bool {
        self.inner
            .lock()
            .map(|inner| {
                inner
                    .granted_capabilities
                    .get(extension_id)
                    .is_some_and(|capabilities| capabilities.contains(capability))
            })
            .unwrap_or(false)
    }

    pub fn has_capability(&self, extension_id: &str, capability: &str) -> bool {
        self.inner
            .lock()
            .map(|inner| {
                inner
                    .processes
                    .get(extension_id)
                    .is_some_and(|entry| entry.capabilities.contains(capability))
            })
            .unwrap_or(false)
    }

    pub fn respond(
        &self,
        extension_id: &str,
        request: &RpcRequest,
        result: Result<Value, RpcError>,
    ) -> Result<(), String> {
        let inner = self
            .inner
            .lock()
            .map_err(|_| "extension host lock was poisoned".to_string())?;
        let entry = inner
            .processes
            .get(extension_id)
            .ok_or_else(|| format!("extension process is not active: {extension_id}"))?;
        entry.runtime.respond(request, result)
    }
}

impl PartialEq for ExtensionHost {
    fn eq(&self, other: &Self) -> bool {
        Arc::ptr_eq(&self.inner, &other.inner)
    }
}

fn language_for_path(path: &Path) -> Option<&'static str> {
    match path
        .extension()
        .and_then(|extension| extension.to_str())
        .map(str::to_ascii_lowercase)
        .as_deref()
    {
        Some("rs") => Some("rust"),
        Some("go") => Some("go"),
        Some("json") => Some("json"),
        Some("yaml" | "yml") => Some("yaml"),
        Some("md" | "markdown") => Some("markdown"),
        Some("toml") => Some("toml"),
        Some("ts" | "tsx" | "mts" | "cts") => Some("typescript"),
        Some("js" | "jsx" | "mjs" | "cjs") => Some("javascript"),
        _ => None,
    }
}

fn lsp_key(extension_id: &str, language: &str) -> String {
    format!("{extension_id}:{language}")
}

fn resolve_lsp_command(root: &Path, command: &str) -> Result<PathBuf, String> {
    let candidate = Path::new(command);
    if candidate
        .components()
        .any(|component| matches!(component, std::path::Component::ParentDir))
    {
        return Err("LSP command must not contain parent directory segments".to_string());
    }
    let packaged = root.join(candidate);
    if packaged.is_file() {
        return Ok(packaged);
    }
    if candidate.is_absolute() || candidate.components().count() > 1 {
        let path = candidate.to_path_buf();
        if !path.is_file() {
            return Err(format!("LSP command does not exist: {}", path.display()));
        }
        return Ok(path);
    }
    Ok(PathBuf::from(command))
}

pub fn load_grants(path: &Path) -> Result<Vec<ExtensionGrant>, String> {
    let file =
        File::open(path).map_err(|error| format!("could not open extension grants: {error}"))?;
    serde_json::from_reader(file)
        .map_err(|error| format!("could not parse extension grants: {error}"))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::extensions::registry::{ExtensionSource, ExtensionStatus};
    use prumo_extension_sdk::manifest::{
        Capability, CommandContribution, Contributions, DebuggerContribution, EXTENSION_API,
        ExtensionManifest, HostRequirement, LspContribution, Permission, Risk, RuntimeSpec,
    };
    use std::collections::BTreeMap;
    use std::fs;
    #[cfg(unix)]
    use std::os::unix::fs::PermissionsExt;

    #[cfg(unix)]
    #[test]
    fn starts_granted_debugger_contribution() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().join("package");
        fs::create_dir_all(&root).unwrap();
        let adapter = root.join("adapter.sh");
        fs::write(
            &adapter,
            "#!/bin/sh\nbody='{\"seq\":1,\"type\":\"response\",\"request_seq\":1,\"success\":true,\"body\":{}}'\nprintf 'Content-Length: %s\\r\\n\\r\\n%s' \"${#body}\" \"$body\"\nbody='{\"seq\":2,\"type\":\"response\",\"request_seq\":2,\"success\":true,\"body\":{\"threads\":[]}}'\nprintf 'Content-Length: %s\\r\\n\\r\\n%s' \"${#body}\" \"$body\"\nwhile IFS= read -r _; do :; done\n",
        )
        .unwrap();
        let mut permissions = fs::metadata(&adapter).unwrap().permissions();
        permissions.set_mode(0o755);
        fs::set_permissions(&adapter, permissions).unwrap();
        let manifest = ExtensionManifest {
            schema_version: 1,
            id: "com.example.debugger".to_string(),
            name: "Example Debugger".to_string(),
            description: "Debugger test".to_string(),
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
                id: "debugger".to_string(),
                required: true,
                since: "1.0.0".to_string(),
            }],
            permissions: vec![Permission {
                capability: "debugger".to_string(),
                scope: "workspace".to_string(),
                risk: Risk::High,
                justification: "Starts a debug adapter".to_string(),
            }],
            contributions: Contributions {
                debuggers: vec![DebuggerContribution {
                    id: "com.example.debugger.rust".to_string(),
                    title: "Rust".to_string(),
                    command: "adapter.sh".to_string(),
                    args: Vec::new(),
                }],
                ..Contributions::default()
            },
        };
        fs::write(
            root.join("manifest.json"),
            serde_json::to_vec_pretty(&manifest).unwrap(),
        )
        .unwrap();
        let digest = prumo_extension_sdk::package::directory_digest(&root).unwrap();
        let record = ExtensionRecord {
            manifest,
            root,
            source: ExtensionSource::Development,
            status: ExtensionStatus::Deferred,
            manifest_digest: digest.clone(),
        };
        let grant = ExtensionGrant {
            extension_id: "com.example.debugger".to_string(),
            extension_version: "0.1.0".to_string(),
            manifest_digest: digest,
            capabilities: vec!["debugger".to_string()],
            scopes: BTreeMap::from([("debugger".to_string(), "workspace".to_string())]),
            expires_at: None,
        };
        let host = ExtensionHost::new(directory.path().to_path_buf());
        assert!(host.start(&[record], &[grant]).is_empty());
        assert!(host.is_debug_active("com.example.debugger", "com.example.debugger.rust"));
        assert!(host.has_granted_capability("com.example.debugger", "debugger"));
        assert!(
            host.debug_request(
                "com.example.debugger",
                "com.example.debugger.rust",
                "launch",
                serde_json::json!({}),
            )
            .is_ok()
        );
    }

    #[cfg(unix)]
    #[test]
    fn starts_granted_lsp_contribution() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().join("package");
        fs::create_dir_all(&root).unwrap();
        let server = root.join("server.sh");
        fs::write(
            &server,
            "#!/bin/sh\nbody='{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}'\nprintf 'Content-Length: %s\\r\\n\\r\\n%s' \"${#body}\" \"$body\"\nbody='{\"jsonrpc\":\"2.0\",\"id\":2,\"result\":[{\"name\":\"main\",\"kind\":12}]}'\nprintf 'Content-Length: %s\\r\\n\\r\\n%s' \"${#body}\" \"$body\"\nbody='{\"jsonrpc\":\"2.0\",\"id\":3,\"result\":{\"contents\":{\"value\":\"hover\"}}}'\nprintf 'Content-Length: %s\\r\\n\\r\\n%s' \"${#body}\" \"$body\"\nbody='{\"jsonrpc\":\"2.0\",\"id\":4,\"result\":{\"uri\":\"file:///tmp/main.rs\",\"range\":{\"start\":{\"line\":0}}}}'\nprintf 'Content-Length: %s\\r\\n\\r\\n%s' \"${#body}\" \"$body\"\nbody='{\"jsonrpc\":\"2.0\",\"id\":5,\"result\":[{\"label\":\"main\",\"kind\":12,\"detail\":\"fn main()\"}]}'\nprintf 'Content-Length: %s\\r\\n\\r\\n%s' \"${#body}\" \"$body\"\nbody='{\"jsonrpc\":\"2.0\",\"id\":6,\"result\":[{\"title\":\"Fix import\",\"kind\":\"quickfix\",\"edit\":{\"changes\":{}}}]}'\nprintf 'Content-Length: %s\\r\\n\\r\\n%s' \"${#body}\" \"$body\"\nbody='{\"jsonrpc\":\"2.0\",\"id\":7,\"result\":[{\"range\":{\"start\":{\"line\":0,\"character\":0},\"end\":{\"line\":0,\"character\":0}},\"newText\":\"\"}]}'\nprintf 'Content-Length: %s\\r\\n\\r\\n%s' \"${#body}\" \"$body\"\nwhile IFS= read -r _; do :; done\n",
        )
        .unwrap();
        let mut permissions = fs::metadata(&server).unwrap().permissions();
        permissions.set_mode(0o755);
        fs::set_permissions(&server, permissions).unwrap();
        let manifest = ExtensionManifest {
            schema_version: 1,
            id: "com.example.lsp".to_string(),
            name: "Example LSP".to_string(),
            description: "LSP test".to_string(),
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
                id: "editor.lsp".to_string(),
                required: true,
                since: "1.0.0".to_string(),
            }],
            permissions: vec![Permission {
                capability: "editor.lsp".to_string(),
                scope: "workspace".to_string(),
                risk: Risk::High,
                justification: "Starts a language server".to_string(),
            }],
            contributions: Contributions {
                lsp: vec![LspContribution {
                    id: "com.example.lsp.rust".to_string(),
                    language: "rust".to_string(),
                    command: "server.sh".to_string(),
                    args: Vec::new(),
                }],
                ..Contributions::default()
            },
        };
        fs::write(
            root.join("manifest.json"),
            serde_json::to_vec_pretty(&manifest).unwrap(),
        )
        .unwrap();
        let digest = prumo_extension_sdk::package::directory_digest(&root).unwrap();
        let record = ExtensionRecord {
            manifest,
            root,
            source: ExtensionSource::Development,
            status: ExtensionStatus::Deferred,
            manifest_digest: digest.clone(),
        };
        let grant = ExtensionGrant {
            extension_id: "com.example.lsp".to_string(),
            extension_version: "0.1.0".to_string(),
            manifest_digest: digest,
            capabilities: vec!["editor.lsp".to_string()],
            scopes: BTreeMap::from([("editor.lsp".to_string(), "workspace".to_string())]),
            expires_at: None,
        };
        let host = ExtensionHost::new(directory.path().to_path_buf());
        assert!(host.start(&[record], &[grant]).is_empty());
        assert!(host.is_lsp_active("com.example.lsp", "rust"));
        assert_eq!(
            host.active_lsp_for_path(Path::new("main.rs")),
            Some(("com.example.lsp".to_string(), "rust".to_string()))
        );
        assert!(
            host.sync_active_document(Path::new("main.rs"), "rust", "fn main() {}", 0)
                .is_ok()
        );
        assert!(host.notify_lsp_save(Path::new("main.rs"), "rust").is_ok());
        assert!(
            host.close_lsp_document(Path::new("main.rs"), "rust")
                .is_ok()
        );
        assert_eq!(
            host.lsp_workspace_symbols("com.example.lsp", "rust", "main")
                .unwrap()[0]["name"],
            serde_json::Value::String("main".to_string())
        );
        assert_eq!(
            host.lsp_hover("com.example.lsp", "rust", "main.rs", 1, 0)
                .unwrap()["contents"]["value"],
            serde_json::Value::String("hover".to_string())
        );
        assert!(
            host.lsp_definition("com.example.lsp", "rust", "main.rs", 1, 0)
                .is_ok()
        );
        let completion = host
            .lsp_completion("com.example.lsp", "rust", "main.rs", 1, 0)
            .unwrap();
        assert_eq!(completion.items()[0].label, "main");
        let actions = host
            .lsp_code_actions(
                "com.example.lsp",
                "rust",
                "main.rs",
                LspRange {
                    start: prumo_extension_sdk::lsp::LspPosition {
                        line: 0,
                        character: 0,
                    },
                    end: prumo_extension_sdk::lsp::LspPosition {
                        line: 0,
                        character: 1,
                    },
                },
                Vec::new(),
            )
            .unwrap();
        assert_eq!(actions[0].title, "Fix import");
        let formatting = host
            .lsp_formatting("com.example.lsp", "rust", "main.rs", 4, true)
            .unwrap();
        assert_eq!(formatting.len(), 1);
        assert_eq!(formatting[0].new_text, "");
    }

    #[cfg(unix)]
    #[test]
    fn lsp_request_does_not_hold_the_host_registry_lock() {
        use std::thread;
        use std::time::{Duration, Instant};

        let directory = tempfile::tempdir().unwrap();
        let server = directory.path().join("server.sh");
        fs::write(
            &server,
            "#!/bin/sh\nbody='{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}'\nprintf 'Content-Length: %s\\r\\n\\r\\n%s' \"${#body}\" \"$body\"\nsleep 1\nbody='{\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"contents\":{\"value\":\"hover\"}}}'\nprintf 'Content-Length: %s\\r\\n\\r\\n%s' \"${#body}\" \"$body\"\nwhile IFS= read -r line; do :; done\n",
        )
        .unwrap();
        let mut permissions = fs::metadata(&server).unwrap().permissions();
        permissions.set_mode(0o755);
        fs::set_permissions(&server, permissions).unwrap();

        let host = ExtensionHost::new(directory.path().to_path_buf());
        let client =
            LspClient::start(&server, &[], directory.path(), Duration::from_secs(2)).unwrap();
        let actor = LspActor::start(client, "rust".to_string());
        host.inner.lock().unwrap().lsp_servers.insert(
            lsp_key("com.example.lsp", "rust"),
            LspEntry {
                extension_id: "com.example.lsp".to_string(),
                language: "rust".to_string(),
                actor,
            },
        );

        let request_host = host.clone();
        let request = thread::spawn(move || {
            request_host.lsp_hover("com.example.lsp", "rust", "main.rs", 1, 0)
        });
        thread::sleep(Duration::from_millis(100));
        assert!(!request.is_finished());
        let started = Instant::now();
        assert_eq!(
            host.active_lsp_for_path(Path::new("main.rs")),
            Some(("com.example.lsp".to_string(), "rust".to_string()))
        );
        assert!(started.elapsed() < Duration::from_millis(200));
        assert_eq!(
            request.join().unwrap().unwrap()["contents"]["value"],
            "hover"
        );
    }

    #[cfg(unix)]
    #[test]
    fn real_rust_analyzer_actor_smoke() {
        use std::thread;
        use std::time::{Duration, Instant};

        if std::env::var("PRUMO_RUN_RUST_ANALYZER_SMOKE").as_deref() != Ok("1") {
            return;
        }
        let workspace = tempfile::tempdir().unwrap();
        let source_path = workspace.path().join("src/main.rs");
        fs::create_dir_all(source_path.parent().unwrap()).unwrap();
        fs::write(
            workspace.path().join("Cargo.toml"),
            "[package]\nname = \"lsp-smoke\"\nversion = \"0.1.0\"\nedition = \"2024\"\n",
        )
        .unwrap();
        let source = "struct Demo { field: usize }\nfn main() {\n    let value = Demo { field: 1 };\n    value.\n}\n";
        fs::write(&source_path, source).unwrap();

        let package_root =
            PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("extensions/rust-analyzer");
        let manifest: prumo_extension_sdk::manifest::ExtensionManifest =
            serde_json::from_slice(&fs::read(package_root.join("manifest.json")).unwrap()).unwrap();
        let digest = prumo_extension_sdk::package::directory_digest(&package_root).unwrap();
        let grant = ExtensionGrant {
            extension_id: manifest.id.clone(),
            extension_version: manifest.version.clone(),
            manifest_digest: digest,
            capabilities: vec![prumo_extension_sdk::manifest::LSP_CAPABILITY.to_string()],
            scopes: BTreeMap::from([(
                prumo_extension_sdk::manifest::LSP_CAPABILITY.to_string(),
                "workspace".to_string(),
            )]),
            expires_at: None,
        };
        let record = ExtensionRecord {
            manifest,
            root: package_root,
            source: ExtensionSource::Development,
            status: ExtensionStatus::Ready,
            manifest_digest: grant.manifest_digest.clone(),
        };
        let host = ExtensionHost::new(workspace.path().to_path_buf());
        assert!(host.start(&[record], &[grant]).is_empty());
        assert!(
            host.sync_active_document(&source_path, "rust", source, 1)
                .is_ok()
        );

        let started = Instant::now();
        let deadline = Instant::now() + Duration::from_secs(15);
        let mut completion = None;
        while Instant::now() < deadline {
            match host.lsp_completion(
                "com.prumo.rust-analyzer",
                "rust",
                &source_path.to_string_lossy(),
                4,
                10,
            ) {
                Ok(response) => {
                    if response
                        .clone()
                        .items()
                        .iter()
                        .any(|item| item.label == "field")
                    {
                        completion = Some(response);
                        break;
                    }
                    thread::sleep(Duration::from_millis(100));
                }
                Err(error) if Instant::now() < deadline => {
                    eprintln!("rust-analyzer actor retry: {error}");
                    thread::sleep(Duration::from_millis(100));
                }
                Err(error) => panic!("rust-analyzer actor completion failed: {error}"),
            }
        }
        assert!(
            completion.is_some(),
            "rust-analyzer actor did not return field"
        );
        eprintln!(
            "rust-analyzer actor completion latency: {:?}",
            started.elapsed()
        );
        assert!(host.close_lsp_document(&source_path, "rust").is_ok());
    }

    #[cfg(unix)]
    #[test]
    fn starts_only_granted_process_extensions() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path().join("package");
        fs::create_dir_all(&root).unwrap();
        let executable = root.join("extension.sh");
        fs::write(
            &executable,
            "#!/bin/sh\ncount=0\nwhile IFS= read -r line; do\n  count=$((count + 1))\n  printf '{\"jsonrpc\":\"2.0\",\"id\":%s,\"result\":{\"extension_api\":\"prumo.viewer.extensions/v1\"}}\\n' \"$count\"\n  if [ \"$count\" -eq 1 ]; then\n    printf '{\"jsonrpc\":\"2.0\",\"id\":9,\"method\":\"ui/notify\",\"params\":{\"message\":\"hello\"}}\\n'\n  fi\ndone\n",
        )
        .unwrap();
        let mut permissions = fs::metadata(&executable).unwrap().permissions();
        permissions.set_mode(0o755);
        fs::set_permissions(&executable, permissions).unwrap();
        let manifest = ExtensionManifest {
            schema_version: 1,
            id: "com.example.host-process".to_string(),
            name: "Host Process".to_string(),
            description: "Host process test".to_string(),
            version: "0.1.0".to_string(),
            publisher: "Example".to_string(),
            extension_api: EXTENSION_API.to_string(),
            host: HostRequirement {
                min_version: "0.1.0".to_string(),
                max_version: "0.2.0".to_string(),
            },
            runtime: RuntimeSpec {
                kind: RuntimeKind::Process,
                protocol: Some(prumo_extension_sdk::manifest::PROCESS_PROTOCOL.to_string()),
                entrypoint: BTreeMap::from([("linux".to_string(), "extension.sh".to_string())]),
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
                justification: "Handles commands".to_string(),
            }],
            contributions: Contributions {
                commands: vec![CommandContribution {
                    id: "com.example.host-process.command".to_string(),
                    title: "Process Command".to_string(),
                    description: "Process command".to_string(),
                    action: "process".to_string(),
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
        let digest = prumo_extension_sdk::package::directory_digest(&root).unwrap();
        let record = ExtensionRecord {
            manifest,
            root,
            source: ExtensionSource::Development,
            status: ExtensionStatus::Deferred,
            manifest_digest: digest.clone(),
        };
        let grant = ExtensionGrant {
            extension_id: "com.example.host-process".to_string(),
            extension_version: "0.1.0".to_string(),
            manifest_digest: digest,
            capabilities: vec!["ui.command".to_string()],
            scopes: BTreeMap::from([("ui.command".to_string(), "workspace".to_string())]),
            expires_at: None,
        };
        let host = ExtensionHost::new(directory.path().to_path_buf());
        let errors = host.start(&[record], &[grant]);
        assert!(errors.is_empty());
        assert!(host.is_active("com.example.host-process"));
        std::thread::sleep(std::time::Duration::from_millis(50));
        let requests = host.poll_requests();
        assert_eq!(requests.len(), 1);
        assert_eq!(requests[0].request.method, "ui/notify");
        host.respond(
            "com.example.host-process",
            &requests[0].request,
            Ok(serde_json::json!({ "ok": true })),
        )
        .unwrap();
        assert_eq!(
            host.invoke(
                "com.example.host-process",
                "command/invoke",
                serde_json::json!({})
            ),
            Ok(serde_json::json!({"extension_api": "prumo.viewer.extensions/v1"}))
        );
    }
}
