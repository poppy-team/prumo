use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::fmt::{Display, Formatter};
use std::path::{Path, PathBuf};
use std::time::Duration;

const MAX_RESPONSE_BYTES: u64 = 4 * 1024 * 1024;
const CLIENT_TIMEOUT: Duration = Duration::from_secs(2);

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ClientError {
    #[cfg(not(unix))]
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
            #[cfg(not(unix))]
            Self::UnsupportedPlatform => {
                write!(formatter, "local Prumo protocol requires Unix sockets")
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

    pub fn models(&self, provider: &str) -> Result<DaemonModels, ClientError> {
        let response = self.call(serde_json::json!({
            "op": "models",
            "provider": provider,
        }))?;
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

    pub fn approve(&self, run_id: &str, request_id: &str) -> Result<(), ClientError> {
        self.permission(run_id, request_id, true)
    }

    pub fn deny(&self, run_id: &str, request_id: &str) -> Result<(), ClientError> {
        self.permission(run_id, request_id, false)
    }

    fn permission(
        &self,
        run_id: &str,
        request_id: &str,
        approved: bool,
    ) -> Result<(), ClientError> {
        self.call(serde_json::json!({
            "op": if approved { "approve" } else { "deny" },
            "run_id": run_id,
            "request_id": request_id,
        }))?;
        Ok(())
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

    #[cfg(unix)]
    fn exchange(&self, request: &Value) -> Result<Value, ClientError> {
        use std::io::{BufRead, BufReader, Read, Write};
        use std::os::unix::net::UnixStream;

        let mut stream =
            UnixStream::connect(&self.socket_path).map_err(|error| ClientError::Connect {
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

    #[cfg(not(unix))]
    fn exchange(&self, _request: &Value) -> Result<Value, ClientError> {
        Err(ClientError::UnsupportedPlatform)
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
}
