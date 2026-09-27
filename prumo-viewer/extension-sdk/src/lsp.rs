use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::collections::HashMap;
use std::io::{BufRead, BufReader, Write};
use std::path::Path;
use std::process::{Child, ChildStdin, Command, Stdio};
use std::sync::atomic::{AtomicBool, AtomicI64, Ordering};
use std::sync::{Arc, Mutex, mpsc};
use std::thread;
use std::time::{Duration, Instant};

pub const LSP_MAX_MESSAGE_BYTES: usize = 8 * 1024 * 1024;
const LSP_MAX_PENDING_NOTIFICATIONS: usize = 1000;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct LspError {
    pub code: i64,
    pub message: String,
    #[serde(default)]
    pub data: Option<Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct LspPosition {
    pub line: u32,
    pub character: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct LspRange {
    pub start: LspPosition,
    pub end: LspPosition,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct LspDiagnostic {
    pub range: LspRange,
    #[serde(default)]
    pub severity: Option<u8>,
    #[serde(default)]
    pub code: Option<Value>,
    #[serde(default)]
    pub source: Option<String>,
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct PublishDiagnosticsParams {
    pub uri: String,
    #[serde(default)]
    pub diagnostics: Vec<LspDiagnostic>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct LspTextEdit {
    pub range: LspRange,
    #[serde(rename = "newText")]
    pub new_text: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct LspWorkspaceEdit {
    #[serde(default)]
    pub changes: Option<HashMap<String, Vec<LspTextEdit>>>,
    #[serde(rename = "documentChanges", default)]
    pub document_changes: Option<Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct LspInsertReplaceEdit {
    pub insert: LspRange,
    pub replace: LspRange,
    #[serde(rename = "newText")]
    pub new_text: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(untagged)]
pub enum LspCompletionTextEdit {
    Text(LspTextEdit),
    InsertReplace(LspInsertReplaceEdit),
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct LspCompletionItem {
    pub label: String,
    #[serde(default)]
    pub kind: Option<i64>,
    #[serde(default)]
    pub detail: Option<String>,
    #[serde(rename = "insertText", default)]
    pub insert_text: Option<String>,
    #[serde(rename = "insertTextFormat", default)]
    pub insert_text_format: Option<i64>,
    #[serde(rename = "textEdit", default)]
    pub text_edit: Option<LspCompletionTextEdit>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct LspCompletionList {
    #[serde(rename = "isIncomplete", default)]
    pub is_incomplete: bool,
    pub items: Vec<LspCompletionItem>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(untagged)]
pub enum LspCompletionResponse {
    Items(Vec<LspCompletionItem>),
    List(LspCompletionList),
}

impl LspCompletionResponse {
    pub fn items(self) -> Vec<LspCompletionItem> {
        match self {
            Self::Items(items) => items,
            Self::List(list) => list.items,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct LspCodeAction {
    pub title: String,
    #[serde(default)]
    pub kind: Option<String>,
    #[serde(default)]
    pub edit: Option<LspWorkspaceEdit>,
    #[serde(default)]
    pub command: Option<Value>,
    #[serde(rename = "isPreferred", default)]
    pub is_preferred: Option<bool>,
}

#[derive(Clone)]
pub struct LspCancellation {
    inner: Arc<LspCancellationInner>,
}

struct LspCancellationInner {
    cancelled: AtomicBool,
    request: Mutex<Option<LspCancellationRequest>>,
}

struct LspCancellationRequest {
    id: i64,
    stdin: Arc<Mutex<Option<ChildStdin>>>,
}

impl LspCancellation {
    pub fn new() -> Self {
        Self {
            inner: Arc::new(LspCancellationInner {
                cancelled: AtomicBool::new(false),
                request: Mutex::new(None),
            }),
        }
    }

    pub fn cancel(&self) {
        if self.inner.cancelled.swap(true, Ordering::AcqRel) {
            return;
        }
        if let Some(request) = self.take_request() {
            let _ = write_lsp_value(
                &request.stdin,
                &json!({
                    "jsonrpc": "2.0",
                    "method": "$/cancelRequest",
                    "params": { "id": request.id },
                }),
            );
        }
    }

    fn is_cancelled(&self) -> bool {
        self.inner.cancelled.load(Ordering::Acquire)
    }

    fn register(&self, id: i64, stdin: Arc<Mutex<Option<ChildStdin>>>) -> bool {
        if self.is_cancelled() {
            return false;
        }
        let Ok(mut request) = self.inner.request.lock() else {
            return false;
        };
        *request = Some(LspCancellationRequest { id, stdin });
        drop(request);
        if self.is_cancelled() {
            self.send_cancel_request();
            false
        } else {
            true
        }
    }

    fn clear(&self) {
        if let Ok(mut request) = self.inner.request.lock() {
            request.take();
        }
    }

    fn send_cancel_request(&self) {
        if let Some(request) = self.take_request() {
            let _ = write_lsp_value(
                &request.stdin,
                &json!({
                    "jsonrpc": "2.0",
                    "method": "$/cancelRequest",
                    "params": { "id": request.id },
                }),
            );
        }
    }

    fn take_request(&self) -> Option<LspCancellationRequest> {
        self.inner
            .request
            .lock()
            .ok()
            .and_then(|mut request| request.take())
    }
}

impl Default for LspCancellation {
    fn default() -> Self {
        Self::new()
    }
}

pub fn file_path_to_uri(path: &str) -> String {
    let is_unc = path.starts_with("\\\\") || path.starts_with("//");
    let normalized = path.replace('\\', "/");
    let normalized = if is_unc {
        normalized.trim_start_matches('/').to_string()
    } else if normalized.starts_with('/') {
        normalized
    } else {
        format!("/{normalized}")
    };
    let mut uri = String::from("file://");
    for byte in normalized.bytes() {
        if matches!(
            byte,
            b'A'..=b'Z'
                | b'a'..=b'z'
                | b'0'..=b'9'
                | b'-'
                | b'.'
                | b'_'
                | b'~'
                | b'/'
                | b':'
        ) {
            uri.push(byte as char);
        } else {
            uri.push_str(&format!("%{byte:02X}"));
        }
    }
    uri
}

pub fn frame_message(body: &[u8]) -> Result<Vec<u8>, String> {
    if body.len() > LSP_MAX_MESSAGE_BYTES {
        return Err("LSP message exceeds maximum size".to_string());
    }
    let mut framed = format!("Content-Length: {}\r\n\r\n", body.len()).into_bytes();
    framed.extend_from_slice(body);
    Ok(framed)
}

pub fn read_framed_message<R: BufRead>(reader: &mut R) -> Result<Vec<u8>, String> {
    let mut content_length = None;
    loop {
        let mut line = String::new();
        let read = reader
            .read_line(&mut line)
            .map_err(|error| error.to_string())?;
        if read == 0 {
            return Err("LSP stream closed while reading headers".to_string());
        }
        let line = line.trim_end_matches(['\r', '\n']);
        if line.is_empty() {
            break;
        }
        if let Some(value) = line.strip_prefix("Content-Length:") {
            content_length = Some(
                value
                    .trim()
                    .parse::<usize>()
                    .map_err(|_| "invalid LSP Content-Length".to_string())?,
            );
        }
    }
    let length = content_length.ok_or_else(|| "LSP message has no Content-Length".to_string())?;
    if length > LSP_MAX_MESSAGE_BYTES {
        return Err("LSP message exceeds maximum size".to_string());
    }
    let mut body = vec![0; length];
    reader
        .read_exact(&mut body)
        .map_err(|error| error.to_string())?;
    Ok(body)
}

pub struct LspClient {
    stdin: Arc<Mutex<Option<ChildStdin>>>,
    child: Arc<Mutex<Option<Child>>>,
    responses: mpsc::Receiver<Result<Value, String>>,
    notifications: Arc<Mutex<Vec<Value>>>,
    next_id: AtomicI64,
    initialized: bool,
}

impl LspClient {
    pub fn start(
        program: impl AsRef<std::ffi::OsStr>,
        arguments: &[String],
        root: &Path,
        timeout: Duration,
    ) -> Result<Self, String> {
        let mut command = Command::new(program);
        command
            .args(arguments)
            .current_dir(root)
            .env_clear()
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::null());
        if let Some(path) = std::env::var_os("PATH") {
            command.env("PATH", path);
        }
        let mut child = command
            .spawn()
            .map_err(|error| format!("could not start language server: {error}"))?;
        let stdin = child
            .stdin
            .take()
            .ok_or_else(|| "language server stdin was not piped".to_string())?;
        let stdout = child
            .stdout
            .take()
            .ok_or_else(|| "language server stdout was not piped".to_string())?;
        let (sender, responses) = mpsc::channel();
        let notifications = Arc::new(Mutex::new(Vec::new()));
        let reader_notifications = Arc::clone(&notifications);
        let reader_stdin = Arc::new(Mutex::new(Some(stdin)));
        let client_stdin = Arc::clone(&reader_stdin);
        thread::spawn(move || {
            read_lsp_responses(stdout, sender, reader_notifications, reader_stdin)
        });
        let mut client = Self {
            stdin: client_stdin,
            child: Arc::new(Mutex::new(Some(child))),
            responses,
            notifications,
            next_id: AtomicI64::new(1),
            initialized: false,
        };
        let root_uri = file_path_to_uri(&root.to_string_lossy());
        client.call(
            "initialize",
            json!({
                "processId": null,
                "rootUri": root_uri,
                "capabilities": {},
            }),
            timeout,
            None,
        )?;
        client.notify("initialized", json!({}))?;
        client.initialized = true;
        Ok(client)
    }

    pub fn workspace_symbols(&mut self, query: &str, timeout: Duration) -> Result<Value, String> {
        self.ensure_initialized(timeout)?;
        self.call("workspace/symbol", json!({ "query": query }), timeout, None)
    }

    pub fn workspace_symbols_with_cancellation(
        &mut self,
        query: &str,
        timeout: Duration,
        cancellation: &LspCancellation,
    ) -> Result<Value, String> {
        self.ensure_initialized(timeout)?;
        self.call(
            "workspace/symbol",
            json!({ "query": query }),
            timeout,
            Some(cancellation),
        )
    }

    pub fn hover(
        &mut self,
        file: &str,
        line: usize,
        character: usize,
        timeout: Duration,
    ) -> Result<Value, String> {
        self.ensure_initialized(timeout)?;
        self.call(
            "textDocument/hover",
            json!({
                "textDocument": { "uri": file_path_to_uri(file) },
                "position": { "line": line.saturating_sub(1), "character": character },
            }),
            timeout,
            None,
        )
    }

    pub fn hover_with_cancellation(
        &mut self,
        file: &str,
        line: usize,
        character: usize,
        timeout: Duration,
        cancellation: &LspCancellation,
    ) -> Result<Value, String> {
        self.ensure_initialized(timeout)?;
        self.call(
            "textDocument/hover",
            json!({
                "textDocument": { "uri": file_path_to_uri(file) },
                "position": { "line": line.saturating_sub(1), "character": character },
            }),
            timeout,
            Some(cancellation),
        )
    }

    pub fn definition(
        &mut self,
        file: &str,
        line: usize,
        character: usize,
        timeout: Duration,
    ) -> Result<Value, String> {
        self.ensure_initialized(timeout)?;
        self.call(
            "textDocument/definition",
            json!({
                "textDocument": { "uri": file_path_to_uri(file) },
                "position": { "line": line.saturating_sub(1), "character": character },
            }),
            timeout,
            None,
        )
    }

    pub fn definition_with_cancellation(
        &mut self,
        file: &str,
        line: usize,
        character: usize,
        timeout: Duration,
        cancellation: &LspCancellation,
    ) -> Result<Value, String> {
        self.ensure_initialized(timeout)?;
        self.call(
            "textDocument/definition",
            json!({
                "textDocument": { "uri": file_path_to_uri(file) },
                "position": { "line": line.saturating_sub(1), "character": character },
            }),
            timeout,
            Some(cancellation),
        )
    }

    pub fn completion(
        &mut self,
        file: &str,
        line: usize,
        character: usize,
        timeout: Duration,
    ) -> Result<LspCompletionResponse, String> {
        self.ensure_initialized(timeout)?;
        let value = self.call(
            "textDocument/completion",
            json!({
                "textDocument": { "uri": file_path_to_uri(file) },
                "position": { "line": line.saturating_sub(1), "character": character },
                "context": { "triggerKind": 1 },
            }),
            timeout,
            None,
        )?;
        if value.is_null() {
            return Ok(LspCompletionResponse::Items(Vec::new()));
        }
        serde_json::from_value(value).map_err(|error| error.to_string())
    }

    pub fn completion_with_cancellation(
        &mut self,
        file: &str,
        line: usize,
        character: usize,
        timeout: Duration,
        cancellation: &LspCancellation,
    ) -> Result<LspCompletionResponse, String> {
        self.ensure_initialized(timeout)?;
        let value = self.call(
            "textDocument/completion",
            json!({
                "textDocument": { "uri": file_path_to_uri(file) },
                "position": { "line": line.saturating_sub(1), "character": character },
                "context": { "triggerKind": 1 },
            }),
            timeout,
            Some(cancellation),
        )?;
        if value.is_null() {
            return Ok(LspCompletionResponse::Items(Vec::new()));
        }
        serde_json::from_value(value).map_err(|error| error.to_string())
    }

    pub fn formatting(
        &mut self,
        file: &str,
        tab_size: u32,
        insert_spaces: bool,
        timeout: Duration,
    ) -> Result<Vec<LspTextEdit>, String> {
        self.ensure_initialized(timeout)?;
        let value = self.call(
            "textDocument/formatting",
            json!({
                "textDocument": { "uri": file_path_to_uri(file) },
                "options": { "tabSize": tab_size, "insertSpaces": insert_spaces },
            }),
            timeout,
            None,
        )?;
        if value.is_null() {
            return Ok(Vec::new());
        }
        serde_json::from_value(value).map_err(|error| error.to_string())
    }

    pub fn formatting_with_cancellation(
        &mut self,
        file: &str,
        tab_size: u32,
        insert_spaces: bool,
        timeout: Duration,
        cancellation: &LspCancellation,
    ) -> Result<Vec<LspTextEdit>, String> {
        self.ensure_initialized(timeout)?;
        let value = self.call(
            "textDocument/formatting",
            json!({
                "textDocument": { "uri": file_path_to_uri(file) },
                "options": { "tabSize": tab_size, "insertSpaces": insert_spaces },
            }),
            timeout,
            Some(cancellation),
        )?;
        if value.is_null() {
            return Ok(Vec::new());
        }
        serde_json::from_value(value).map_err(|error| error.to_string())
    }

    pub fn code_actions(
        &mut self,
        file: &str,
        range: LspRange,
        diagnostics: Vec<Value>,
        timeout: Duration,
    ) -> Result<Vec<LspCodeAction>, String> {
        self.ensure_initialized(timeout)?;
        let value = self.call(
            "textDocument/codeAction",
            json!({
                "textDocument": { "uri": file_path_to_uri(file) },
                "range": range,
                "context": { "diagnostics": diagnostics },
            }),
            timeout,
            None,
        )?;
        if value.is_null() {
            return Ok(Vec::new());
        }
        serde_json::from_value(value).map_err(|error| error.to_string())
    }

    pub fn code_actions_with_cancellation(
        &mut self,
        file: &str,
        range: LspRange,
        diagnostics: Vec<Value>,
        timeout: Duration,
        cancellation: &LspCancellation,
    ) -> Result<Vec<LspCodeAction>, String> {
        self.ensure_initialized(timeout)?;
        let value = self.call(
            "textDocument/codeAction",
            json!({
                "textDocument": { "uri": file_path_to_uri(file) },
                "range": range,
                "context": { "diagnostics": diagnostics },
            }),
            timeout,
            Some(cancellation),
        )?;
        if value.is_null() {
            return Ok(Vec::new());
        }
        serde_json::from_value(value).map_err(|error| error.to_string())
    }

    pub fn did_open(
        &self,
        file: &str,
        language: &str,
        text: &str,
        version: i64,
    ) -> Result<(), String> {
        self.notify(
            "textDocument/didOpen",
            json!({
                "textDocument": {
                    "uri": file_path_to_uri(file),
                    "languageId": language,
                    "version": version,
                    "text": text,
                }
            }),
        )
    }

    pub fn did_change(&self, file: &str, version: i64, text: &str) -> Result<(), String> {
        self.notify(
            "textDocument/didChange",
            json!({
                "textDocument": { "uri": file_path_to_uri(file), "version": version },
                "contentChanges": [{ "text": text }],
            }),
        )
    }

    pub fn did_save(&self, file: &str) -> Result<(), String> {
        self.notify(
            "textDocument/didSave",
            json!({ "textDocument": { "uri": file_path_to_uri(file) } }),
        )
    }

    pub fn did_close(&self, file: &str) -> Result<(), String> {
        self.notify(
            "textDocument/didClose",
            json!({ "textDocument": { "uri": file_path_to_uri(file) } }),
        )
    }

    pub fn drain_notifications(&self) -> Vec<Value> {
        self.notifications
            .lock()
            .map(|mut notifications| std::mem::take(&mut *notifications))
            .unwrap_or_default()
    }

    pub fn stop(&self) {
        if let Some(mut child) = self.child.lock().ok().and_then(|mut child| child.take()) {
            let _ = child.kill();
            let _ = child.wait();
        }
        if let Ok(mut stdin) = self.stdin.lock() {
            stdin.take();
        }
    }

    fn ensure_initialized(&mut self, timeout: Duration) -> Result<(), String> {
        if self.initialized {
            Ok(())
        } else {
            self.call("initialize", json!({}), timeout, None)
                .map(|_| ())
        }
    }

    fn call(
        &mut self,
        method: &str,
        params: Value,
        timeout: Duration,
        cancellation: Option<&LspCancellation>,
    ) -> Result<Value, String> {
        let id = self.next_id.fetch_add(1, Ordering::Relaxed);
        let result = (|| {
            if cancellation.is_some_and(|cancellation| cancellation.is_cancelled()) {
                return Err("LSP request cancelled".to_string());
            }
            let request = json!({
                "jsonrpc": "2.0",
                "id": id,
                "method": method,
                "params": params,
            });
            self.write(&request)?;
            if let Some(cancellation) = cancellation
                && !cancellation.register(id, Arc::clone(&self.stdin))
            {
                return Err("LSP request cancelled".to_string());
            }
            let deadline = Instant::now() + timeout;
            loop {
                if cancellation.is_some_and(|cancellation| cancellation.is_cancelled()) {
                    if let Some(cancellation) = cancellation {
                        cancellation.send_cancel_request();
                    }
                    return Err("LSP request cancelled".to_string());
                }
                let remaining = deadline.saturating_duration_since(Instant::now());
                if remaining.is_zero() {
                    return Err("LSP request timed out".to_string());
                }
                let wait = if cancellation.is_some() {
                    remaining.min(Duration::from_millis(20))
                } else {
                    remaining
                };
                match self.responses.recv_timeout(wait) {
                    Ok(Ok(value)) => {
                        if value.get("id").and_then(Value::as_i64) != Some(id) {
                            continue;
                        }
                        if let Some(error) = value.get("error") {
                            let error: LspError = serde_json::from_value(error.clone())
                                .map_err(|error| error.to_string())?;
                            return Err(error.message);
                        }
                        return value
                            .get("result")
                            .cloned()
                            .ok_or_else(|| "LSP response has no result".to_string());
                    }
                    Ok(Err(error)) => return Err(error),
                    Err(mpsc::RecvTimeoutError::Timeout) => {}
                    Err(mpsc::RecvTimeoutError::Disconnected) => {
                        return Err("LSP response channel closed".to_string());
                    }
                }
            }
        })();
        if let Some(cancellation) = cancellation {
            cancellation.clear();
        }
        result
    }

    fn notify(&self, method: &str, params: Value) -> Result<(), String> {
        self.write(&json!({
            "jsonrpc": "2.0",
            "method": method,
            "params": params,
        }))
    }

    fn write(&self, value: &Value) -> Result<(), String> {
        write_lsp_value(&self.stdin, value)
    }
}

fn write_lsp_value(stdin: &Arc<Mutex<Option<ChildStdin>>>, value: &Value) -> Result<(), String> {
    let body = serde_json::to_vec(value).map_err(|error| error.to_string())?;
    let framed = frame_message(&body)?;
    let mut stdin = stdin
        .lock()
        .map_err(|_| "LSP stdin lock was poisoned".to_string())?;
    let stdin = stdin
        .as_mut()
        .ok_or_else(|| "language server is closed".to_string())?;
    stdin.write_all(&framed).map_err(|error| error.to_string())
}

impl Drop for LspClient {
    fn drop(&mut self) {
        self.stop();
    }
}

fn server_request_result(method: &str) -> Value {
    match method {
        "workspace/configuration" => Value::Array(Vec::new()),
        "workspace/applyEdit" => json!({
            "applied": false,
            "failureReason": "remote workspace edits are not supported by the viewer",
        }),
        _ => Value::Null,
    }
}

fn read_lsp_responses(
    stdout: std::process::ChildStdout,
    sender: mpsc::Sender<Result<Value, String>>,
    notifications: Arc<Mutex<Vec<Value>>>,
    stdin: Arc<Mutex<Option<ChildStdin>>>,
) {
    let mut reader = BufReader::new(stdout);
    loop {
        match read_framed_message(&mut reader) {
            Ok(body) => match serde_json::from_slice::<Value>(&body) {
                Ok(value) => {
                    if value.get("method").is_some() {
                        if let Some(id) = value.get("id") {
                            let response = json!({
                                "jsonrpc": "2.0",
                                "id": id,
                                "result": server_request_result(
                                    value.get("method").and_then(Value::as_str).unwrap_or_default(),
                                ),
                            });
                            if let Err(error) = write_lsp_value(&stdin, &response) {
                                let _ = sender.send(Err(error));
                                return;
                            }
                        } else if let Ok(mut notifications) = notifications.lock()
                            && notifications.len() < LSP_MAX_PENDING_NOTIFICATIONS
                        {
                            notifications.push(value);
                        }
                    } else if value.get("id").is_some() {
                        if sender.send(Ok(value)).is_err() {
                            return;
                        }
                    } else if let Ok(mut notifications) = notifications.lock()
                        && notifications.len() < LSP_MAX_PENDING_NOTIFICATIONS
                    {
                        notifications.push(value);
                    }
                }
                Err(error) => {
                    let _ = sender.send(Err(error.to_string()));
                    return;
                }
            },
            Err(error) => {
                let _ = sender.send(Err(error));
                return;
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Cursor;

    #[test]
    fn encodes_file_paths_as_lsp_uris() {
        assert_eq!(
            file_path_to_uri("/tmp/a file.rs"),
            "file:///tmp/a%20file.rs"
        );
        assert_eq!(
            file_path_to_uri("C:\\tmp\\main.rs"),
            "file:///C:/tmp/main.rs"
        );
        assert_eq!(
            file_path_to_uri("\\\\server\\share\\main.rs"),
            "file://server/share/main.rs"
        );
    }

    #[test]
    fn frames_and_reads_lsp_messages() {
        let framed = frame_message(br#"{"jsonrpc":"2.0"}"#).unwrap();
        let mut input = Cursor::new(framed);
        let body = read_framed_message(&mut input).unwrap();
        assert_eq!(body, br#"{"jsonrpc":"2.0"}"#);
    }

    #[test]
    fn parses_completion_and_code_action_responses() {
        let completion: LspCompletionResponse = serde_json::from_value(json!([{
            "label": "main",
            "kind": 12,
            "detail": "fn main()",
            "textEdit": {
                "range": {
                    "start": { "line": 0, "character": 0 },
                    "end": { "line": 0, "character": 0 }
                },
                "newText": "main"
            }
        }]))
        .unwrap();
        assert_eq!(completion.items()[0].label, "main");

        let insert_replace: LspCompletionResponse = serde_json::from_value(json!({
            "isIncomplete": false,
            "items": [{
                "label": "value",
                "textEdit": {
                    "insert": {
                        "start": { "line": 0, "character": 0 },
                        "end": { "line": 0, "character": 1 }
                    },
                    "replace": {
                        "start": { "line": 0, "character": 0 },
                        "end": { "line": 0, "character": 4 }
                    },
                    "newText": "value"
                }
            }]
        }))
        .unwrap();
        let insert_replace_items = insert_replace.items();
        let Some(LspCompletionTextEdit::InsertReplace(edit)) = insert_replace_items
            .first()
            .and_then(|item| item.text_edit.as_ref())
        else {
            panic!("expected insert/replace edit");
        };
        assert_eq!(edit.replace.end.character, 4);

        let actions: Vec<LspCodeAction> = serde_json::from_value(json!([{
            "title": "Fix import",
            "kind": "quickfix",
            "edit": {
                "changes": {
                    "file:///tmp/main.rs": [{
                        "range": {
                            "start": { "line": 0, "character": 0 },
                            "end": { "line": 0, "character": 1 }
                        },
                        "newText": "m"
                    }]
                }
            }
        }]))
        .unwrap();
        assert_eq!(actions[0].title, "Fix import");
        assert_eq!(
            actions[0].edit.as_ref().unwrap().changes.as_ref().unwrap()["file:///tmp/main.rs"][0]
                .new_text,
            "m"
        );
    }

    #[test]
    fn parses_publish_diagnostics_params() {
        let params: PublishDiagnosticsParams = serde_json::from_value(json!({
            "uri": "file:///tmp/main.rs",
            "diagnostics": [{
                "range": {
                    "start": { "line": 0, "character": 2 },
                    "end": { "line": 0, "character": 5 }
                },
                "severity": 1,
                "source": "rust-analyzer",
                "message": "Example"
            }]
        }))
        .unwrap();
        assert_eq!(params.diagnostics[0].range.start.line, 0);
        assert_eq!(params.diagnostics[0].severity, Some(1));
    }

    #[cfg(unix)]
    #[test]
    fn collects_server_notifications_separately_from_responses() {
        use std::fs;
        use std::os::unix::fs::PermissionsExt;
        use std::time::{Duration, Instant};

        let directory = tempfile::tempdir().unwrap();
        let server = directory.path().join("server.sh");
        fs::write(
            &server,
            "#!/bin/sh\nbody='{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}'\nprintf 'Content-Length: %s\\r\\n\\r\\n%s' \"${#body}\" \"$body\"\nbody='{\"jsonrpc\":\"2.0\",\"id\":99,\"method\":\"workspace/configuration\",\"params\":{\"items\":[]}}'\nprintf 'Content-Length: %s\\r\\n\\r\\n%s' \"${#body}\" \"$body\"\nbody='{\"jsonrpc\":\"2.0\",\"method\":\"textDocument/publishDiagnostics\",\"params\":{\"uri\":\"file:///tmp/main.rs\",\"diagnostics\":[]}}'\nprintf 'Content-Length: %s\\r\\n\\r\\n%s' \"${#body}\" \"$body\"\nwhile IFS= read -r line; do :; done\n",
        )
        .unwrap();
        let mut permissions = fs::metadata(&server).unwrap().permissions();
        permissions.set_mode(0o755);
        fs::set_permissions(&server, permissions).unwrap();

        let client =
            LspClient::start(&server, &[], directory.path(), Duration::from_secs(2)).unwrap();
        let deadline = Instant::now() + Duration::from_secs(2);
        let mut notifications = Vec::new();
        while Instant::now() < deadline && notifications.is_empty() {
            notifications.extend(client.drain_notifications());
            std::thread::sleep(Duration::from_millis(10));
        }
        assert_eq!(
            notifications[0]["method"],
            "textDocument/publishDiagnostics"
        );
    }

    #[cfg(unix)]
    #[test]
    fn cancels_an_in_flight_lsp_request() {
        use std::fs;
        use std::os::unix::fs::PermissionsExt;
        use std::thread;
        use std::time::{Duration, Instant};

        let directory = tempfile::tempdir().unwrap();
        let server = directory.path().join("server.sh");
        fs::write(
            &server,
            "#!/bin/sh\nbody='{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}'\nprintf 'Content-Length: %s\\r\\n\\r\\n%s' \"${#body}\" \"$body\"\nsleep 5\nbody='{\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{}}'\nprintf 'Content-Length: %s\\r\\n\\r\\n%s' \"${#body}\" \"$body\"\nwhile IFS= read -r line; do :; done\n",
        )
        .unwrap();
        let mut permissions = fs::metadata(&server).unwrap().permissions();
        permissions.set_mode(0o755);
        fs::set_permissions(&server, permissions).unwrap();

        let mut client =
            LspClient::start(&server, &[], directory.path(), Duration::from_secs(2)).unwrap();
        let cancellation = LspCancellation::new();
        let cancellation_for_thread = cancellation.clone();
        let request = thread::spawn(move || {
            client.hover_with_cancellation(
                "main.rs",
                1,
                0,
                Duration::from_secs(2),
                &cancellation_for_thread,
            )
        });
        thread::sleep(Duration::from_millis(50));
        let started = Instant::now();
        cancellation.cancel();
        let result = request.join().unwrap();
        assert!(result.unwrap_err().contains("cancelled"));
        assert!(started.elapsed() < Duration::from_secs(1));
    }

    #[test]
    fn rejects_unbounded_lsp_message() {
        let body = vec![b'x'; LSP_MAX_MESSAGE_BYTES + 1];
        assert!(frame_message(&body).is_err());
    }
}
