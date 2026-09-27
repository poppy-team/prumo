use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::collections::HashMap;
use std::fmt::{Display, Formatter};
use std::path::{Path, PathBuf};
use std::time::Duration;

const MAX_RESPONSE_BYTES: u64 = 4 * 1024 * 1024;
const CLIENT_TIMEOUT: Duration = Duration::from_secs(2);

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ClientError {
    #[allow(dead_code)]
    UnsupportedPlatform,
    Connect {
        path: PathBuf,
        message: String,
    },
    Io(String),
    InvalidResponse(String),
    Remote(String),
}

impl Display for ClientError {
    fn fmt(&self, formatter: &mut Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::UnsupportedPlatform => {
                write!(formatter, "local Prumo protocol requires Unix sockets or TCP")
            }
            Self::Connect { path, message } => {
                write!(formatter, "cannot connect to {}: {message}", path.display())
            }
            Self::Io(message) => write!(formatter, "Prumo protocol I/O failed: {message}"),
            Self::InvalidResponse(message) => {
                write!(formatter, "invalid Prumo protocol response: {message}")
            }
            Self::Remote(message) => write!(formatter, "Prumo daemon error: {message}"),
        }
    }
}

impl std::error::Error for ClientError {}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct DaemonRun {
    pub run_id: String,
    pub status: String,
    #[serde(default)]
    pub phase: String,
    #[serde(default)]
    pub pending_permissions: Vec<String>,
    #[serde(default)]
    pub permission_fingerprints: HashMap<String, String>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct DaemonDiff {
    pub path: String,
    pub kind: String,
    pub content: String,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct DaemonEvent {
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub run_id: String,
    pub kind: String,
    #[serde(default)]
    pub payload: Value,
    #[serde(default)]
    pub created_at: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DaemonModels {
    pub provider: String,
    pub models: Vec<String>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DaemonSnapshot {
    pub protocol_version: String,
    pub selected_run: Option<DaemonRun>,
    pub runs: Vec<DaemonRun>,
    pub events: Vec<DaemonEvent>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PrumoClient {
    socket_path: PathBuf,
    timeout: Duration,
}

impl PrumoClient {
    pub fn new(custom_socket: Option<PathBuf>, workspace_root: &Path) -> Self {
        let socket_path = custom_socket
            .or_else(|| std::env::var("PRUMO_SOCKET").ok().map(PathBuf::from))
            .unwrap_or_else(|| {
                workspace_root
                    .join(".prumo")
                    .join("runtime")
                    .join("harness")
                    .join("agentd.sock")
            });

        Self {
            socket_path,
            timeout: CLIENT_TIMEOUT,
        }
    }

    #[cfg(test)]
    pub fn socket_path(&self) -> &Path {
        &self.socket_path
    }

    pub fn protocol_version(&self) -> Result<String, ClientError> {
        let response = self.call(serde_json::json!({ "op": "protocol" }))?;
        response
            .get("version")
            .and_then(Value::as_str)
            .map(str::to_string)
            .ok_or_else(|| ClientError::InvalidResponse("protocol version is missing".to_string()))
    }

    pub fn snapshot(&self) -> Result<DaemonSnapshot, ClientError> {
        let protocol_version = self.protocol_version()?;
        let list_response = self.call(serde_json::json!({ "op": "list" }))?;
        let runs: Vec<DaemonRun> = serde_json::from_value(
            list_response
                .get("runs")
                .cloned()
                .ok_or_else(|| ClientError::InvalidResponse("run list is missing".to_string()))?,
        )
        .map_err(|error| ClientError::InvalidResponse(error.to_string()))?;

        let selected_run = select_run(&runs);
        let events =
            if let Some(run) = selected_run.as_ref() {
                let response = self.call(serde_json::json!({
                    "op": "events",
                    "run_id": run.run_id,
                }))?;
                serde_json::from_value(response.get("events").cloned().ok_or_else(|| {
                    ClientError::InvalidResponse("event list is missing".to_string())
                })?)
                .map_err(|error| ClientError::InvalidResponse(error.to_string()))?
            } else {
                Vec::new()
            };

        Ok(DaemonSnapshot {
            protocol_version,
            selected_run,
            runs,
            events,
        })
    }

    pub fn start_goal_with_options(
        &self,
        goal: &str,
        workspace_root: &Path,
        provider: Option<&str>,
        model: Option<&str>,
        api_key: Option<&str>,
        base_url: Option<&str>,
    ) -> Result<String, ClientError> {
        let mut request = serde_json::json!({
            "op": "start",
            "goal": goal,
            "workspace": workspace_root.to_string_lossy(),
        });
        if let Some(provider) = provider.filter(|value| !value.trim().is_empty()) {
            request["provider"] = Value::String(provider.to_string());
        }
        if let Some(model) = model.filter(|value| !value.trim().is_empty()) {
            request["model"] = Value::String(model.to_string());
        }
        if let Some(api_key) = api_key.filter(|value| !value.trim().is_empty()) {
            request["api_key"] = Value::String(api_key.to_string());
        }
        if let Some(base_url) = base_url.filter(|value| !value.trim().is_empty()) {
            request["base_url"] = Value::String(base_url.to_string());
        }
        let response = self.call(request)?;
        response
            .get("run_id")
            .and_then(Value::as_str)
            .map(str::to_string)
            .ok_or_else(|| ClientError::InvalidResponse("run id is missing".to_string()))
    }

    pub fn cancel(&self, run_id: &str) -> Result<(), ClientError> {
        self.call(serde_json::json!({ "op": "cancel", "run_id": run_id }))?;
        Ok(())
    }

    pub fn steer(&self, run_id: &str, message: &str) -> Result<(), ClientError> {
        self.call(serde_json::json!({
            "op": "steer",
            "run_id": run_id,
            "message": message,
        }))?;
        Ok(())
    }

    #[allow(dead_code)]
    pub fn models(&self, provider: &str) -> Result<DaemonModels, ClientError> {
        self.models_with_options(provider, None, None)
    }

    pub fn models_with_options(
        &self,
        provider: &str,
        api_key: Option<&str>,
        base_url: Option<&str>,
    ) -> Result<DaemonModels, ClientError> {
        let mut request = serde_json::json!({
            "op": "models",
            "provider": provider,
        });
        if let Some(api_key) = api_key.filter(|value| !value.trim().is_empty()) {
            request["api_key"] = Value::String(api_key.to_string());
        }
        if let Some(base_url) = base_url.filter(|value| !value.trim().is_empty()) {
            request["base_url"] = Value::String(base_url.to_string());
        }
        let response = self.call(request)?;
        let provider = response
            .get("provider")
            .and_then(Value::as_str)
            .unwrap_or(provider)
            .to_string();
        let models = response
            .get("models")
            .and_then(Value::as_array)
            .ok_or_else(|| ClientError::InvalidResponse("model list is missing".to_string()))?
            .iter()
            .filter_map(Value::as_str)
            .map(str::to_string)
            .collect();
        Ok(DaemonModels { provider, models })
    }

    pub fn approve(
        &self,
        run_id: &str,
        request_id: &str,
        fingerprint: Option<&str>,
    ) -> Result<(), ClientError> {
        self.permission(run_id, request_id, fingerprint, true)
    }

    pub fn deny(
        &self,
        run_id: &str,
        request_id: &str,
        fingerprint: Option<&str>,
    ) -> Result<(), ClientError> {
        self.permission(run_id, request_id, fingerprint, false)
    }

    fn permission(
        &self,
        run_id: &str,
        request_id: &str,
        fingerprint: Option<&str>,
        approved: bool,
    ) -> Result<(), ClientError> {
        let mut request = serde_json::json!({
            "op": if approved { "approve" } else { "deny" },
            "run_id": run_id,
            "request_id": request_id,
        });
        if let Some(fp) = fingerprint.filter(|s| !s.is_empty()) {
            request["fingerprint"] = Value::String(fp.to_string());
        }
        self.call(request)?;
        Ok(())
    }

    pub fn diff(&self, run_id: &str, path: &str) -> Result<DaemonDiff, ClientError> {
        let response = self.call(serde_json::json!({
            "op": "diff",
            "run_id": run_id,
            "path": path,
        }))?;
        let path = response
            .get("path")
            .and_then(Value::as_str)
            .unwrap_or(path)
            .to_string();
        let kind = response
            .get("kind")
            .and_then(Value::as_str)
            .unwrap_or("modified")
            .to_string();
        let content = response
            .get("content")
            .and_then(Value::as_str)
            .ok_or_else(|| ClientError::InvalidResponse("diff content is missing".to_string()))?
            .to_string();
        Ok(DaemonDiff { path, kind, content })
    }

    pub fn subscribe(
        &self,
        run_id: &str,
        from: usize,
    ) -> Result<std::sync::mpsc::Receiver<DaemonEvent>, ClientError> {
        use std::io::{BufRead, BufReader, Write};

        let mut stream =
            connect_stream(&self.socket_path, None).map_err(|error| ClientError::Connect {
                path: self.socket_path.clone(),
                message: error.to_string(),
            })?;

        let req = serde_json::json!({
            "op": "subscribe",
            "run_id": run_id,
            "from": from,
        });
        let mut payload = serde_json::to_vec(&req)
            .map_err(|error| ClientError::InvalidResponse(error.to_string()))?;
        payload.push(b'\n');
        stream
            .write_all(&payload)
            .map_err(|error| ClientError::Io(error.to_string()))?;

        let mut reader = BufReader::new(stream);
        let mut ack_line = String::new();
        reader
            .read_line(&mut ack_line)
            .map_err(|error| ClientError::Io(error.to_string()))?;

        let ack: Value = serde_json::from_str(&ack_line)
            .map_err(|error| ClientError::InvalidResponse(error.to_string()))?;

        if ack.get("op").and_then(Value::as_str) != Some("subscribed")
            && ack.get("ok").and_then(Value::as_bool) != Some(true)
        {
            let error_msg = ack
                .get("error")
                .and_then(Value::as_str)
                .unwrap_or("subscribe failed")
                .to_string();
            return Err(ClientError::Remote(error_msg));
        }

        let (sender, receiver) = std::sync::mpsc::channel();
        std::thread::spawn(move || {
            let mut line = String::new();
            while let Ok(n) = reader.read_line(&mut line) {
                if n == 0 {
                    break;
                }
                if let Ok(msg) = serde_json::from_str::<Value>(&line)
                    && msg.get("op").and_then(Value::as_str) == Some("event")
                        && let Some(event_val) = msg.get("event")
                            && let Ok(event) = serde_json::from_value::<DaemonEvent>(event_val.clone())
                                && sender.send(event).is_err() {
                                    break;
                                }
                line.clear();
            }
        });

        Ok(receiver)
    }

    fn call(&self, request: Value) -> Result<Value, ClientError> {
        let response = self.exchange(&request)?;
        if response.get("ok").and_then(Value::as_bool) != Some(true) {
            let message = response
                .get("error")
                .and_then(Value::as_str)
                .unwrap_or("unknown daemon error")
                .to_string();
            return Err(ClientError::Remote(message));
        }
        Ok(response)
    }

    fn exchange(&self, request: &Value) -> Result<Value, ClientError> {
        use std::io::{BufRead, BufReader, Read, Write};

        let mut stream =
            connect_stream(&self.socket_path, Some(self.timeout)).map_err(|error| ClientError::Connect {
                path: self.socket_path.clone(),
                message: error.to_string(),
            })?;
        stream
            .set_read_timeout(Some(self.timeout))
            .map_err(|error| ClientError::Io(error.to_string()))?;
        stream
            .set_write_timeout(Some(self.timeout))
            .map_err(|error| ClientError::Io(error.to_string()))?;

        let mut payload = serde_json::to_vec(request)
            .map_err(|error| ClientError::InvalidResponse(error.to_string()))?;
        payload.push(b'\n');
        stream
            .write_all(&payload)
            .map_err(|error| ClientError::Io(error.to_string()))?;

        let mut line = String::new();
        BufReader::new(stream)
            .take(MAX_RESPONSE_BYTES)
            .read_line(&mut line)
            .map_err(|error| ClientError::Io(error.to_string()))?;
        if line.trim().is_empty() {
            return Err(ClientError::InvalidResponse(
                "daemon returned an empty response".to_string(),
            ));
        }
        serde_json::from_str(&line).map_err(|error| ClientError::InvalidResponse(error.to_string()))
    }
}

enum ProtocolStream {
    #[cfg(unix)]
    Unix(std::os::unix::net::UnixStream),
    Tcp(std::net::TcpStream),
}

impl std::io::Read for ProtocolStream {
    fn read(&mut self, buf: &mut [u8]) -> std::io::Result<usize> {
        match self {
            #[cfg(unix)]
            Self::Unix(stream) => stream.read(buf),
            Self::Tcp(stream) => stream.read(buf),
        }
    }
}

impl std::io::Write for ProtocolStream {
    fn write(&mut self, buf: &[u8]) -> std::io::Result<usize> {
        match self {
            #[cfg(unix)]
            Self::Unix(stream) => stream.write(buf),
            Self::Tcp(stream) => stream.write(buf),
        }
    }

    fn flush(&mut self) -> std::io::Result<()> {
        match self {
            #[cfg(unix)]
            Self::Unix(stream) => stream.flush(),
            Self::Tcp(stream) => stream.flush(),
        }
    }
}

impl ProtocolStream {
    fn set_read_timeout(&self, timeout: Option<Duration>) -> std::io::Result<()> {
        match self {
            #[cfg(unix)]
            Self::Unix(stream) => stream.set_read_timeout(timeout),
            Self::Tcp(stream) => stream.set_read_timeout(timeout),
        }
    }

    fn set_write_timeout(&self, timeout: Option<Duration>) -> std::io::Result<()> {
        match self {
            #[cfg(unix)]
            Self::Unix(stream) => stream.set_write_timeout(timeout),
            Self::Tcp(stream) => stream.set_write_timeout(timeout),
        }
    }
}

fn connect_stream(path: &Path, timeout: Option<Duration>) -> Result<ProtocolStream, std::io::Error> {
    let path_str = path.to_string_lossy();
    if path_str.contains(':') && !path_str.starts_with('/') {
        let addr = path_str.trim_start_matches("tcp://");
        let stream = if let Some(t) = timeout {
            let addrs: Vec<std::net::SocketAddr> = std::net::ToSocketAddrs::to_socket_addrs(addr)?.collect();
            if let Some(first) = addrs.first() {
                std::net::TcpStream::connect_timeout(first, t)?
            } else {
                return Err(std::io::Error::new(std::io::ErrorKind::AddrNotAvailable, "no socket addresses"));
            }
        } else {
            std::net::TcpStream::connect(addr)?
        };
        return Ok(ProtocolStream::Tcp(stream));
    }

    #[cfg(unix)]
    {
        let stream = std::os::unix::net::UnixStream::connect(path)?;
        Ok(ProtocolStream::Unix(stream))
    }

    #[cfg(not(unix))]
    {
        let addr = if path_str.contains(':') {
            path_str.trim_start_matches("tcp://")
        } else {
            "127.0.0.1:9099"
        };
        let stream = if let Some(t) = timeout {
            let addrs: Vec<std::net::SocketAddr> = std::net::ToSocketAddrs::to_socket_addrs(addr)?.collect();
            if let Some(first) = addrs.first() {
                std::net::TcpStream::connect_timeout(first, t)?
            } else {
                return Err(std::io::Error::new(std::io::ErrorKind::AddrNotAvailable, "no socket addresses"));
            }
        } else {
            std::net::TcpStream::connect(addr)?
        };
        Ok(ProtocolStream::Tcp(stream))
    }
}

fn select_run(runs: &[DaemonRun]) -> Option<DaemonRun> {
    runs.iter()
        .find(|run| {
            matches!(
                run.status.as_str(),
                "running" | "awaiting_approval" | "yielded"
            )
        })
        .or_else(|| runs.last())
        .cloned()
}

#[cfg(all(test, unix))]
mod tests {
    use super::*;
    use serde_json::json;
    use std::fs;
    use std::io::{BufRead, BufReader, Write};
    use std::os::unix::net::UnixListener;
    use std::sync::mpsc;
    use std::thread;
    use tempfile::tempdir;

    #[test]
    fn uses_workspace_socket_by_default() {
        let root = PathBuf::from("/workspace");
        let client = PrumoClient::new(None, &root);
        assert_eq!(
            client.socket_path(),
            root.join(".prumo/runtime/harness/agentd.sock")
        );
    }

    #[test]
    fn snapshot_reads_protocol_runs_and_events() {
        let directory = tempdir().unwrap();
        let socket_path = directory.path().join("agentd.sock");
        let listener = UnixListener::bind(&socket_path).unwrap();
        let (operation_sender, operation_receiver) = mpsc::channel();
        let responses = [
            json!({"ok": true, "version": "0.4.0", "min_compatible": "0.1.0"}),
            json!({"ok": true, "runs": [{"run_id": "R-1", "status": "running", "phase": "execute_tool"}]}),
            json!({"ok": true, "events": [{"id": "event-1", "run_id": "R-1", "kind": "run.started", "payload": {"goal": "inspect workspace"}, "created_at": "2026-09-23T12:00:00Z"}]}),
        ];

        let server = thread::spawn(move || {
            for response in responses {
                let (mut stream, _) = listener.accept().unwrap();
                let mut line = String::new();
                BufReader::new(&stream).read_line(&mut line).unwrap();
                let request: Value = serde_json::from_str(&line).unwrap();
                operation_sender
                    .send(request["op"].as_str().unwrap().to_string())
                    .unwrap();
                serde_json::to_writer(&mut stream, &response).unwrap();
                stream.write_all(b"\n").unwrap();
            }
        });

        let client = PrumoClient::new(Some(socket_path), directory.path());
        let snapshot = client.snapshot().unwrap();
        server.join().unwrap();

        assert_eq!(snapshot.protocol_version, "0.4.0");
        assert_eq!(snapshot.selected_run.unwrap().run_id, "R-1");
        assert_eq!(snapshot.events[0].kind, "run.started");
        assert_eq!(operation_receiver.recv().unwrap(), "protocol".to_string());
        assert_eq!(operation_receiver.recv().unwrap(), "list".to_string());
        assert_eq!(operation_receiver.recv().unwrap(), "events".to_string());
    }

    #[test]
    fn runner_controls_use_published_protocol_operations() {
        let directory = tempdir().unwrap();
        let socket_path = directory.path().join("agentd.sock");
        let listener = UnixListener::bind(&socket_path).unwrap();
        let (operation_sender, operation_receiver) = mpsc::channel();
        let responses = [
            json!({"ok": true, "provider": "opencode", "models": ["model-a", "model-b"]}),
            json!({"ok": true, "cancelled": true}),
            json!({"ok": true, "steered": true}),
        ];
        let server = thread::spawn(move || {
            for response in responses {
                let (mut stream, _) = listener.accept().unwrap();
                let mut line = String::new();
                BufReader::new(&mut stream).read_line(&mut line).unwrap();
                let request: Value = serde_json::from_str(&line).unwrap();
                operation_sender
                    .send(request["op"].as_str().unwrap().to_string())
                    .unwrap();
                serde_json::to_writer(&mut stream, &response).unwrap();
                stream.write_all(b"\n").unwrap();
            }
        });
        let client = PrumoClient::new(Some(socket_path), directory.path());
        let models = client.models("opencode").unwrap();
        client.cancel("R-1").unwrap();
        client.steer("R-1", "continue tests").unwrap();
        server.join().unwrap();

        assert_eq!(models.models, vec!["model-a", "model-b"]);
        assert_eq!(operation_receiver.recv().unwrap(), "models");
        assert_eq!(operation_receiver.recv().unwrap(), "cancel");
        assert_eq!(operation_receiver.recv().unwrap(), "steer");
    }

    #[test]
    fn snapshot_surfaces_connection_failures() {
        let directory = tempdir().unwrap();
        let socket_path = directory.path().join("missing.sock");
        let client = PrumoClient::new(Some(socket_path.clone()), directory.path());
        let error = client.snapshot().unwrap_err();
        assert!(matches!(error, ClientError::Connect { .. }));
        assert!(!socket_path.exists());
        assert!(!fs::metadata(socket_path).is_ok());
    }

    #[test]
    fn handles_approval_with_fingerprint_and_file_diff() {
        let directory = tempdir().unwrap();
        let socket_path = directory.path().join("agentd.sock");
        let listener = UnixListener::bind(&socket_path).unwrap();
        let (request_sender, request_receiver) = mpsc::channel();
        let responses = [
            json!({"ok": true, "run_id": "R-1", "request_id": "req-1", "approved": true}),
            json!({"ok": true, "path": "src/main.rs", "kind": "modified", "content": "--- a/src/main.rs\n+++ b/src/main.rs\n@@ -1 +1,2 @@\n-old\n+new"}),
        ];
        let server = thread::spawn(move || {
            for response in responses {
                let (mut stream, _) = listener.accept().unwrap();
                let mut line = String::new();
                BufReader::new(&mut stream).read_line(&mut line).unwrap();
                let request: Value = serde_json::from_str(&line).unwrap();
                request_sender.send(request).unwrap();
                serde_json::to_writer(&mut stream, &response).unwrap();
                stream.write_all(b"\n").unwrap();
            }
        });

        let client = PrumoClient::new(Some(socket_path), directory.path());
        client.approve("R-1", "req-1", Some("fp-sha256-1234")).unwrap();
        let diff = client.diff("R-1", "src/main.rs").unwrap();
        server.join().unwrap();

        let req1 = request_receiver.recv().unwrap();
        assert_eq!(req1["op"], "approve");
        assert_eq!(req1["fingerprint"], "fp-sha256-1234");

        let req2 = request_receiver.recv().unwrap();
        assert_eq!(req2["op"], "diff");
        assert_eq!(req2["path"], "src/main.rs");
        assert_eq!(diff.kind, "modified");
        assert!(diff.content.contains("+new"));
    }

    #[test]
    #[cfg(unix)]
    fn handles_push_streaming_events() {
        let directory = tempdir().unwrap();
        let socket_path = directory.path().join("agentd_stream.sock");
        let listener = UnixListener::bind(&socket_path).unwrap();
        let server = thread::spawn(move || {
            let (mut stream, _) = listener.accept().unwrap();
            let mut line = String::new();
            BufReader::new(&mut stream).read_line(&mut line).unwrap();
            let req: Value = serde_json::from_str(&line).unwrap();
            assert_eq!(req["op"], "subscribe");
            assert_eq!(req["run_id"], "R-STREAM");
            assert_eq!(req["from"], 0);

            // send ack
            let ack = json!({"op": "subscribed", "run_id": "R-STREAM", "from": 0});
            serde_json::to_writer(&mut stream, &ack).unwrap();
            stream.write_all(b"\n").unwrap();
            stream.flush().unwrap();

            // stream an event
            let event = json!({
                "op": "event",
                "run_id": "R-STREAM",
                "event": {
                    "id": "ev-1",
                    "run_id": "R-STREAM",
                    "kind": "tool_call_ready",
                    "payload": {"name": "read_file"},
                    "created_at": "2026-09-26T12:00:00Z"
                }
            });
            serde_json::to_writer(&mut stream, &event).unwrap();
            stream.write_all(b"\n").unwrap();
            stream.flush().unwrap();
        });

        let client = PrumoClient::new(Some(socket_path), directory.path());
        let rx = client.subscribe("R-STREAM", 0).unwrap();
        let event = rx.recv_timeout(Duration::from_secs(2)).unwrap();
        assert_eq!(event.id, "ev-1");
        assert_eq!(event.kind, "tool_call_ready");
        server.join().unwrap();
    }
}
