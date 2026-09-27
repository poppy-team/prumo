use serde_json::{Value, json};
use std::io::{BufRead, BufReader, Read, Write};
use std::path::Path;
use std::process::{Child, ChildStdin, Command, Stdio};
use std::sync::atomic::{AtomicI64, Ordering};
use std::sync::{Arc, Mutex, mpsc};
use std::thread;
use std::time::{Duration, Instant};

const MAX_DAP_MESSAGE_BYTES: usize = 8 * 1024 * 1024;

pub struct DapClient {
    stdin: Arc<Mutex<Option<ChildStdin>>>,
    child: Arc<Mutex<Option<Child>>>,
    responses: mpsc::Receiver<Result<Value, String>>,
    next_id: AtomicI64,
}

impl DapClient {
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
            .map_err(|error| format!("could not start debug adapter: {error}"))?;
        let stdin = child
            .stdin
            .take()
            .ok_or_else(|| "debug adapter stdin was not piped".to_string())?;
        let stdout = child
            .stdout
            .take()
            .ok_or_else(|| "debug adapter stdout was not piped".to_string())?;
        let (sender, responses) = mpsc::channel();
        thread::spawn(move || read_responses(stdout, sender));
        let mut client = Self {
            stdin: Arc::new(Mutex::new(Some(stdin))),
            child: Arc::new(Mutex::new(Some(child))),
            responses,
            next_id: AtomicI64::new(1),
        };
        client.request(
            "initialize",
            json!({
                "clientID": "prumo-viewer",
                "adapterID": "prumo",
                "linesStartAt1": true,
                "columnsStartAt1": true,
            }),
            timeout,
        )?;
        Ok(client)
    }

    pub fn launch(&mut self, timeout: Duration) -> Result<Value, String> {
        self.request("launch", json!({}), timeout)
    }

    pub fn request(
        &mut self,
        command: &str,
        arguments: Value,
        timeout: Duration,
    ) -> Result<Value, String> {
        let id = self.next_id.fetch_add(1, Ordering::Relaxed);
        self.write(&json!({
            "seq": id,
            "type": "request",
            "command": command,
            "arguments": arguments,
        }))?;
        let deadline = Instant::now() + timeout;
        loop {
            let remaining = deadline.saturating_duration_since(Instant::now());
            if remaining.is_zero() {
                return Err("debug adapter request timed out".to_string());
            }
            match self.responses.recv_timeout(remaining) {
                Ok(Ok(value)) => {
                    if value.get("request_seq").and_then(Value::as_i64) != Some(id)
                        || value.get("type").and_then(Value::as_str) != Some("response")
                    {
                        continue;
                    }
                    if value.get("success").and_then(Value::as_bool) == Some(false) {
                        return Err(value
                            .get("message")
                            .and_then(Value::as_str)
                            .unwrap_or("debug adapter request failed")
                            .to_string());
                    }
                    return Ok(value.get("body").cloned().unwrap_or(Value::Null));
                }
                Ok(Err(error)) => return Err(error),
                Err(mpsc::RecvTimeoutError::Timeout) => {
                    return Err("debug adapter request timed out".to_string());
                }
                Err(mpsc::RecvTimeoutError::Disconnected) => {
                    return Err("debug adapter response channel closed".to_string());
                }
            }
        }
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

    fn write(&self, value: &Value) -> Result<(), String> {
        let body = serde_json::to_vec(value).map_err(|error| error.to_string())?;
        if body.len() > MAX_DAP_MESSAGE_BYTES {
            return Err("debug adapter message exceeds maximum size".to_string());
        }
        let mut framed = format!("Content-Length: {}\r\n\r\n", body.len()).into_bytes();
        framed.extend_from_slice(&body);
        let mut stdin = self
            .stdin
            .lock()
            .map_err(|_| "debug adapter stdin lock was poisoned".to_string())?;
        stdin
            .as_mut()
            .ok_or_else(|| "debug adapter is closed".to_string())?
            .write_all(&framed)
            .map_err(|error| error.to_string())
    }
}

impl Drop for DapClient {
    fn drop(&mut self) {
        self.stop();
    }
}

fn read_responses(stdout: std::process::ChildStdout, sender: mpsc::Sender<Result<Value, String>>) {
    let mut reader = BufReader::new(stdout);
    loop {
        let mut content_length = None;
        loop {
            let mut line = String::new();
            match reader.read_line(&mut line) {
                Ok(0) => {
                    let _ = sender.send(Err("debug adapter stream closed".to_string()));
                    return;
                }
                Ok(_) => {
                    let line = line.trim_end_matches(['\r', '\n']);
                    if line.is_empty() {
                        break;
                    }
                    if let Some(value) = line.strip_prefix("Content-Length:") {
                        content_length = value.trim().parse::<usize>().ok();
                    }
                }
                Err(error) => {
                    let _ = sender.send(Err(error.to_string()));
                    return;
                }
            }
        }
        let Some(length) = content_length else {
            let _ = sender.send(Err(
                "debug adapter message has no Content-Length".to_string()
            ));
            return;
        };
        if length > MAX_DAP_MESSAGE_BYTES {
            let _ = sender.send(Err("debug adapter message exceeds maximum size".to_string()));
            return;
        }
        let mut body = vec![0; length];
        if reader.read_exact(&mut body).is_err() {
            let _ = sender.send(Err("debug adapter body was truncated".to_string()));
            return;
        }
        match serde_json::from_slice(&body) {
            Ok(value) => {
                if sender.send(Ok(value)).is_err() {
                    return;
                }
            }
            Err(error) => {
                let _ = sender.send(Err(error.to_string()));
                return;
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rejects_oversized_debug_messages() {
        let value = json!({ "data": "x".repeat(MAX_DAP_MESSAGE_BYTES) });
        let client = DapClient {
            stdin: Arc::new(Mutex::new(None)),
            child: Arc::new(Mutex::new(None)),
            responses: mpsc::channel().1,
            next_id: AtomicI64::new(1),
        };
        assert!(client.write(&value).is_err());
    }
}
