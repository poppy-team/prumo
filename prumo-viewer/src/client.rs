use std::io::{BufRead, BufReader, Write};
use std::os::unix::net::UnixStream;
use std::path::{Path, PathBuf};
use std::time::Duration;
use serde::{Deserialize, Serialize};
use tokio::sync::mpsc;

pub const PROTOCOL_VERSION: &str = "0.4.0";

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RunStatus {
    pub run_id: String,
    pub status: String,
    #[serde(default)]
    pub phase: String,
    #[serde(default)]
    pub stop_reason: String,
    #[serde(default)]
    pub active: bool,
    #[serde(default)]
    pub pending_permissions: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Event {
    pub id: String,
    pub run_id: String,
    pub kind: String,
    #[serde(default)]
    pub payload: serde_json::Value,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct StartRequest {
    pub goal: String,
    #[serde(default)]
    pub provider: String,
    #[serde(default)]
    pub model: String,
    #[serde(default)]
    pub base_url: String,
    #[serde(default)]
    pub max_turns: usize,
    #[serde(default)]
    pub run_id: String,
    #[serde(default)]
    pub workspace: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DiffResponse {
    pub path: String,
    pub kind: String,
    pub content: String,
}

#[derive(Debug, Clone)]
pub struct PrumoClient {
    socket_path: PathBuf,
    timeout: Duration,
}

impl PrumoClient {
    pub fn new(socket_path: impl AsRef<Path>) -> Self {
        Self {
            socket_path: socket_path.as_ref().to_path_buf(),
            timeout: Duration::from_secs(10),
        }
    }

    pub fn socket_path(&self) -> &Path {
        &self.socket_path
    }

    pub fn is_alive(&self) -> bool {
        if !self.socket_path.exists() {
            return false;
        }
        match UnixStream::connect(&self.socket_path) {
            Ok(stream) => {
                let _ = stream.set_read_timeout(Some(Duration::from_millis(500)));
                let _ = stream.set_write_timeout(Some(Duration::from_millis(500)));
                true
            }
            Err(_) => false,
        }
    }

    fn call(&self, req: &serde_json::Value) -> Result<serde_json::Value, String> {
        let mut stream = UnixStream::connect(&self.socket_path)
            .map_err(|e| format!("connect to {}: {}", self.socket_path.display(), e))?;

        stream
            .set_read_timeout(Some(self.timeout))
            .map_err(|e| e.to_string())?;
        stream
            .set_write_timeout(Some(self.timeout))
            .map_err(|e| e.to_string())?;

        let mut data = serde_json::to_vec(req).map_err(|e| e.to_string())?;
        data.push(b'\n');
        stream.write_all(&data).map_err(|e| e.to_string())?;
        stream.flush().map_err(|e| e.to_string())?;

        let mut reader = BufReader::new(stream);
        let mut line = String::new();
        reader.read_line(&mut line).map_err(|e| e.to_string())?;

        if line.trim().is_empty() {
            return Err("empty response from daemon".to_string());
        }

        let resp: serde_json::Value = serde_json::from_str(&line)
            .map_err(|e| format!("invalid json response: {} (raw: {})", e, line.trim()))?;

        if let Some(ok) = resp.get("ok").and_then(|v| v.as_bool()) {
            if !ok {
                let err_msg = resp
                    .get("error")
                    .and_then(|v| v.as_str())
                    .unwrap_or("unknown daemon error");
                return Err(err_msg.to_string());
            }
        }

        Ok(resp)
    }

    pub fn list(&self) -> Result<Vec<RunStatus>, String> {
        let req = serde_json::json!({ "op": "list" });
        let resp = self.call(&req)?;
        let runs_val = resp.get("runs").cloned().unwrap_or(serde_json::Value::Array(vec![]));
        serde_json::from_value(runs_val).map_err(|e| e.to_string())
    }

    pub fn status(&self, run_id: &str) -> Result<RunStatus, String> {
        let req = serde_json::json!({ "op": "status", "run_id": run_id });
        let resp = self.call(&req)?;
        serde_json::from_value(resp).map_err(|e| e.to_string())
    }

    pub fn events(&self, run_id: &str) -> Result<Vec<Event>, String> {
        let req = serde_json::json!({ "op": "events", "run_id": run_id });
        let resp = self.call(&req)?;
        let evs_val = resp.get("events").cloned().unwrap_or(serde_json::Value::Array(vec![]));
        serde_json::from_value(evs_val).map_err(|e| e.to_string())
    }

    pub fn start(&self, req: StartRequest) -> Result<String, String> {
        let payload = serde_json::json!({
            "op": "start",
            "goal": req.goal,
            "provider": req.provider,
            "model": req.model,
            "base_url": req.base_url,
            "max_turns": if req.max_turns == 0 { 10 } else { req.max_turns },
            "run_id": req.run_id,
            "workspace": req.workspace,
        });
        let resp = self.call(&payload)?;
        resp.get("run_id")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string())
            .ok_or_else(|| "missing run_id in response".to_string())
    }

    pub fn cancel(&self, run_id: &str) -> Result<(), String> {
        let req = serde_json::json!({ "op": "cancel", "run_id": run_id });
        self.call(&req)?;
        Ok(())
    }

    pub fn steer(&self, run_id: &str, message: &str) -> Result<(), String> {
        let req = serde_json::json!({ "op": "steer", "run_id": run_id, "message": message });
        self.call(&req)?;
        Ok(())
    }

    pub fn approve(&self, request_id: &str) -> Result<(), String> {
        let req = serde_json::json!({ "op": "approve", "request_id": request_id });
        self.call(&req)?;
        Ok(())
    }

    pub fn deny(&self, request_id: &str, reason: &str) -> Result<(), String> {
        let req = serde_json::json!({ "op": "deny", "request_id": request_id, "reason": reason });
        self.call(&req)?;
        Ok(())
    }

    pub fn diff(&self, run_id: &str, path: &str) -> Result<DiffResponse, String> {
        let req = serde_json::json!({ "op": "diff", "run_id": run_id, "path": path });
        let resp = self.call(&req)?;
        serde_json::from_value(resp).map_err(|e| e.to_string())
    }

    pub fn subscribe(
        &self,
        run_id: String,
        from: usize,
    ) -> Result<mpsc::UnboundedReceiver<Event>, String> {
        let socket_path = self.socket_path.clone();
        let (tx, rx) = mpsc::unbounded_channel();

        std::thread::spawn(move || {
            let stream = match UnixStream::connect(&socket_path) {
                Ok(s) => s,
                Err(e) => {
                    eprintln!("[PrumoClient::subscribe] connect error: {}", e);
                    return;
                }
            };

            let mut reader = BufReader::new(match stream.try_clone() {
                Ok(s) => s,
                Err(_) => return,
            });
            let mut writer = stream;

            let req = serde_json::json!({
                "op": "subscribe",
                "run_id": run_id,
                "from": from,
            });
            let mut data = serde_json::to_vec(&req).unwrap_or_default();
            data.push(b'\n');
            if writer.write_all(&data).is_err() || writer.flush().is_err() {
                return;
            }

            let mut ack_line = String::new();
            if reader.read_line(&mut ack_line).is_err() {
                return;
            }

            let mut line = String::new();
            while reader.read_line(&mut line).is_ok() {
                if line.trim().is_empty() {
                    line.clear();
                    continue;
                }

                if let Ok(msg) = serde_json::from_str::<serde_json::Value>(&line) {
                    if msg.get("op").and_then(|v| v.as_str()) == Some("event") {
                        if let Some(ev_val) = msg.get("event") {
                            if let Ok(ev) = serde_json::from_value::<Event>(ev_val.clone()) {
                                let is_terminal = ev.kind == "run.finished"
                                    || ev.kind == "run.completed"
                                    || ev.kind == "run.failed"
                                    || ev.kind == "run.cancelled";

                                if tx.send(ev).is_err() || is_terminal {
                                    break;
                                }
                            }
                        }
                    }
                }
                line.clear();
            }
        });

        Ok(rx)
    }
}
