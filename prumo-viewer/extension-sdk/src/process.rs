use crate::manifest::{EXTENSION_API, ExtensionManifest, RuntimeKind};
use crate::rpc::{RpcError, RpcRequest, RpcResponse, read_message, write_message};
use serde_json::{Value, json};
use std::io::BufReader;
use std::path::{Path, PathBuf};
use std::process::{Child, ChildStdin, Command, Stdio};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Arc, Mutex, mpsc};
use std::thread;
use std::time::{Duration, Instant};

const START_TIMEOUT: Duration = Duration::from_secs(3);
const INVOKE_TIMEOUT: Duration = Duration::from_secs(5);

#[derive(Debug)]
pub enum ProcessEvent {
    Request(RpcRequest),
    Response(RpcResponse),
    Error(String),
    Closed,
}

pub struct ProcessRuntime {
    stdin: Arc<Mutex<Option<ChildStdin>>>,
    child: Arc<Mutex<Option<Child>>>,
    responses: mpsc::Receiver<ProcessEvent>,
    requests: mpsc::Receiver<ProcessEvent>,
    pending_requests: Vec<RpcRequest>,
    next_id: AtomicU64,
}

impl ProcessRuntime {
    pub fn start(
        manifest: &ExtensionManifest,
        root: &Path,
        workspace_root: &Path,
        granted_capabilities: &[String],
    ) -> Result<Self, String> {
        if manifest.runtime.kind != RuntimeKind::Process {
            return Err("extension runtime is not process-based".to_string());
        }
        let entrypoint = resolve_entrypoint(manifest, root)?;
        let mut command = Command::new(entrypoint);
        command
            .current_dir(root)
            .env_clear()
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::null());
        let mut child = command
            .spawn()
            .map_err(|error| format!("could not start extension process: {error}"))?;
        let stdin = child
            .stdin
            .take()
            .ok_or_else(|| "extension stdin was not piped".to_string())?;
        let stdout = child
            .stdout
            .take()
            .ok_or_else(|| "extension stdout was not piped".to_string())?;
        let (response_sender, responses) = mpsc::channel();
        let (request_sender, requests) = mpsc::channel();
        thread::spawn(move || read_process_events(stdout, response_sender, request_sender));
        let mut runtime = Self {
            stdin: Arc::new(Mutex::new(Some(stdin))),
            child: Arc::new(Mutex::new(Some(child))),
            responses,
            requests,
            pending_requests: Vec::new(),
            next_id: AtomicU64::new(1),
        };
        let mut params = json!({
            "extension_api": EXTENSION_API,
            "extension_id": manifest.id,
            "extension_version": manifest.version,
            "host_api": EXTENSION_API,
            "granted_capabilities": granted_capabilities,
        });
        if granted_capabilities
            .iter()
            .any(|capability| capability == "workspace.read")
        {
            params["workspace_root"] = json!(workspace_root);
        }
        let result = runtime.invoke_with_timeout("initialize", params, START_TIMEOUT)?;
        let returned_api = result
            .get("extension_api")
            .and_then(Value::as_str)
            .unwrap_or_default();
        if returned_api != EXTENSION_API {
            runtime.stop();
            return Err("extension returned an incompatible API version".to_string());
        }
        Ok(runtime)
    }

    pub fn invoke(&mut self, method: &str, params: Value) -> Result<Value, String> {
        self.invoke_with_timeout(method, params, INVOKE_TIMEOUT)
    }

    pub fn pending_request(&mut self) -> Option<RpcRequest> {
        while let Ok(ProcessEvent::Request(request)) = self.requests.try_recv() {
            self.pending_requests.push(request);
        }
        self.pending_requests.pop()
    }

    pub fn respond(
        &self,
        request: &RpcRequest,
        result: Result<Value, RpcError>,
    ) -> Result<(), String> {
        let response = match result {
            Ok(result) => RpcResponse {
                jsonrpc: "2.0".to_string(),
                id: request.id,
                result: Some(result),
                error: None,
            },
            Err(error) => RpcResponse {
                jsonrpc: "2.0".to_string(),
                id: request.id,
                result: None,
                error: Some(error),
            },
        };
        self.write(&response)
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

    fn invoke_with_timeout(
        &mut self,
        method: &str,
        params: Value,
        timeout: Duration,
    ) -> Result<Value, String> {
        let id = self.next_id.fetch_add(1, Ordering::Relaxed);
        let request = RpcRequest {
            jsonrpc: "2.0".to_string(),
            id,
            method: method.to_string(),
            params,
        };
        self.write(&request)?;
        wait_for_response(&mut self.responses, id, timeout)
    }

    fn write(&self, value: &impl serde::Serialize) -> Result<(), String> {
        let mut stdin = self
            .stdin
            .lock()
            .map_err(|_| "extension stdin lock was poisoned".to_string())?;
        let stdin = stdin
            .as_mut()
            .ok_or_else(|| "extension process is closed".to_string())?;
        write_message(stdin, value)
    }
}

impl Drop for ProcessRuntime {
    fn drop(&mut self) {
        self.stop();
    }
}

fn resolve_entrypoint(manifest: &ExtensionManifest, root: &Path) -> Result<PathBuf, String> {
    let platform = current_platform();
    let relative = manifest
        .runtime
        .entrypoint
        .get(platform)
        .ok_or_else(|| format!("extension has no entrypoint for {platform}"))?;
    let root = root
        .canonicalize()
        .map_err(|error| format!("could not resolve extension root: {error}"))?;
    let entrypoint = root
        .join(relative)
        .canonicalize()
        .map_err(|error| format!("could not resolve extension entrypoint: {error}"))?;
    if !entrypoint.starts_with(&root) {
        return Err("extension entrypoint escapes package root".to_string());
    }
    Ok(entrypoint)
}

fn current_platform() -> &'static str {
    if cfg!(target_os = "linux") {
        "linux"
    } else if cfg!(target_os = "macos") {
        "macos"
    } else if cfg!(target_os = "windows") {
        "windows"
    } else {
        "unsupported"
    }
}

fn read_process_events(
    stdout: std::process::ChildStdout,
    response_sender: mpsc::Sender<ProcessEvent>,
    request_sender: mpsc::Sender<ProcessEvent>,
) {
    let mut reader = BufReader::new(stdout);
    loop {
        match read_message(&mut reader) {
            Ok(Some(value)) => {
                if value.get("method").is_some() {
                    match serde_json::from_value(value) {
                        Ok(request) => {
                            if request_sender.send(ProcessEvent::Request(request)).is_err() {
                                return;
                            }
                        }
                        Err(error) => {
                            let _ = response_sender.send(ProcessEvent::Error(error.to_string()));
                            return;
                        }
                    }
                } else {
                    match serde_json::from_value(value) {
                        Ok(response) => {
                            if response_sender
                                .send(ProcessEvent::Response(response))
                                .is_err()
                            {
                                return;
                            }
                        }
                        Err(error) => {
                            let _ = response_sender.send(ProcessEvent::Error(error.to_string()));
                            return;
                        }
                    }
                }
            }
            Ok(None) => {
                let _ = response_sender.send(ProcessEvent::Closed);
                return;
            }
            Err(error) => {
                let _ = response_sender.send(ProcessEvent::Error(error));
                return;
            }
        }
    }
}

fn wait_for_response(
    responses: &mut mpsc::Receiver<ProcessEvent>,
    id: u64,
    timeout: Duration,
) -> Result<Value, String> {
    let deadline = Instant::now() + timeout;
    loop {
        let remaining = deadline.saturating_duration_since(Instant::now());
        if remaining.is_zero() {
            return Err("extension request timed out".to_string());
        }
        match responses.recv_timeout(remaining) {
            Ok(ProcessEvent::Response(response)) if response.id == id => {
                if let Some(error) = response.error {
                    return Err(error.message);
                }
                return response
                    .result
                    .ok_or_else(|| "extension response did not contain a result".to_string());
            }
            Ok(ProcessEvent::Response(_)) => {}
            Ok(ProcessEvent::Request(_)) => {}
            Ok(ProcessEvent::Error(error)) => return Err(error),
            Ok(ProcessEvent::Closed) => return Err("extension process closed".to_string()),
            Err(mpsc::RecvTimeoutError::Timeout) => {
                return Err("extension request timed out".to_string());
            }
            Err(mpsc::RecvTimeoutError::Disconnected) => {
                return Err("extension event channel closed".to_string());
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::manifest::{HostRequirement, RuntimeSpec};
    use std::collections::BTreeMap;
    use std::fs;
    #[cfg(unix)]
    use std::os::unix::fs::PermissionsExt;

    #[cfg(unix)]
    #[test]
    fn starts_and_invokes_process_extension() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path();
        let executable = root.join("extension.sh");
        fs::write(
            &executable,
            "#!/bin/sh\ncount=0\nwhile IFS= read -r line; do\n  count=$((count + 1))\n  printf '{\"jsonrpc\":\"2.0\",\"id\":%s,\"result\":{\"extension_api\":\"prumo.viewer.extensions/v1\",\"ok\":true}}\\n' \"$count\"\ndone\n",
        )
        .unwrap();
        let mut permissions = fs::metadata(&executable).unwrap().permissions();
        permissions.set_mode(0o755);
        fs::set_permissions(&executable, permissions).unwrap();
        let manifest = ExtensionManifest {
            schema_version: 1,
            id: "com.example.process".to_string(),
            name: "Process".to_string(),
            description: "Process test".to_string(),
            version: "0.1.0".to_string(),
            publisher: "Example".to_string(),
            extension_api: EXTENSION_API.to_string(),
            host: HostRequirement {
                min_version: "0.1.0".to_string(),
                max_version: "0.2.0".to_string(),
            },
            runtime: RuntimeSpec {
                kind: RuntimeKind::Process,
                protocol: Some(crate::manifest::PROCESS_PROTOCOL.to_string()),
                entrypoint: BTreeMap::from([("linux".to_string(), "extension.sh".to_string())]),
            },
            capabilities: Vec::new(),
            permissions: Vec::new(),
            contributions: Default::default(),
        };
        let mut runtime = ProcessRuntime::start(&manifest, root, root, &[]).unwrap();
        let result = runtime.invoke("command/invoke", json!({})).unwrap();
        assert_eq!(result["ok"], true);
        runtime.stop();
    }
}
