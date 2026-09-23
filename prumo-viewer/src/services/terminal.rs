use portable_pty::{CommandBuilder, MasterPty, PtySize, native_pty_system};
use std::{
    io::{Read, Write},
    path::Path,
    sync::{
        Arc, Mutex,
        mpsc::{self, Receiver, Sender},
    },
    thread,
};

const DEFAULT_ROWS: u16 = 24;
const DEFAULT_COLUMNS: u16 = 120;

#[derive(Debug)]
pub enum TerminalEvent {
    Output(Vec<u8>),
    Reset,
    Exited,
}

struct TerminalSession {
    master: Box<dyn MasterPty + Send>,
    writer: Box<dyn Write + Send>,
    child: Box<dyn portable_pty::Child + Send + Sync>,
}

impl Drop for TerminalSession {
    fn drop(&mut self) {
        let _ = self.child.kill();
    }
}

#[derive(Clone)]
pub struct TerminalRuntime {
    workspace_root: std::path::PathBuf,
    shell: String,
    session: Arc<Mutex<TerminalSession>>,
    sender: Arc<Mutex<Sender<TerminalEvent>>>,
    receiver: Arc<Mutex<Receiver<TerminalEvent>>>,
}

impl PartialEq for TerminalRuntime {
    fn eq(&self, other: &Self) -> bool {
        Arc::ptr_eq(&self.session, &other.session) && Arc::ptr_eq(&self.receiver, &other.receiver)
    }
}

impl TerminalRuntime {
    pub fn new(workspace_root: &Path, shell: Option<String>) -> Result<Self, String> {
        let (sender, receiver) = mpsc::channel();
        let session = spawn_session(
            workspace_root,
            shell.as_deref().unwrap_or(default_shell().as_str()),
            &sender,
        )?;
        Ok(Self {
            workspace_root: workspace_root.to_path_buf(),
            shell: shell.unwrap_or_else(default_shell),
            session: Arc::new(Mutex::new(session)),
            sender: Arc::new(Mutex::new(sender)),
            receiver: Arc::new(Mutex::new(receiver)),
        })
    }

    pub fn write(&self, data: &str) -> Result<(), String> {
        let mut session = self
            .session
            .lock()
            .map_err(|_| "terminal session lock is poisoned".to_string())?;
        session
            .writer
            .write_all(data.as_bytes())
            .and_then(|()| session.writer.flush())
            .map_err(|error| format!("terminal write failed: {error}"))
    }

    pub fn resize(&self, rows: u16, columns: u16) -> Result<(), String> {
        let session = self
            .session
            .lock()
            .map_err(|_| "terminal session lock is poisoned".to_string())?;
        session
            .master
            .resize(PtySize {
                rows: rows.max(2_u16),
                cols: columns.max(20_u16),
                pixel_width: 0,
                pixel_height: 0,
            })
            .map_err(|error| format!("terminal resize failed: {error}"))
    }

    pub fn restart(&self) -> Result<(), String> {
        let (sender, receiver) = mpsc::channel();
        let next_session = spawn_session(&self.workspace_root, &self.shell, &sender)?;
        let mut session = self
            .session
            .lock()
            .map_err(|_| "terminal session lock is poisoned".to_string())?;
        let mut stored_sender = self
            .sender
            .lock()
            .map_err(|_| "terminal sender lock is poisoned".to_string())?;
        let mut stored_receiver = self
            .receiver
            .lock()
            .map_err(|_| "terminal receiver lock is poisoned".to_string())?;
        *session = next_session;
        *stored_sender = sender;
        *stored_receiver = receiver;
        let _ = stored_sender.send(TerminalEvent::Reset);
        Ok(())
    }

    pub fn drain(&self) -> Vec<TerminalEvent> {
        let Ok(receiver) = self.receiver.lock() else {
            return Vec::new();
        };
        let mut events = Vec::new();
        while let Ok(event) = receiver.try_recv() {
            events.push(event);
        }
        events
    }
}

fn spawn_session(
    workspace_root: &Path,
    shell: &str,
    sender: &Sender<TerminalEvent>,
) -> Result<TerminalSession, String> {
    let pair = native_pty_system()
        .openpty(PtySize {
            rows: DEFAULT_ROWS,
            cols: DEFAULT_COLUMNS,
            pixel_width: 0,
            pixel_height: 0,
        })
        .map_err(|error| format!("failed to open PTY: {error}"))?;
    let mut command = CommandBuilder::new(shell);
    command.cwd(workspace_root);
    command.env("TERM", "xterm-256color");
    let child = pair
        .slave
        .spawn_command(command)
        .map_err(|error| format!("failed to start terminal shell: {error}"))?;
    let mut reader = pair
        .master
        .try_clone_reader()
        .map_err(|error| format!("failed to read terminal output: {error}"))?;
    let writer = pair
        .master
        .take_writer()
        .map_err(|error| format!("failed to write terminal input: {error}"))?;
    let sender = sender.clone();
    thread::spawn(move || {
        let mut buffer = [0_u8; 8192];
        loop {
            match reader.read(&mut buffer) {
                Ok(0) => break,
                Ok(read) => {
                    if sender
                        .send(TerminalEvent::Output(buffer[..read].to_vec()))
                        .is_err()
                    {
                        break;
                    }
                }
                Err(_) => break,
            }
        }
        let _ = sender.send(TerminalEvent::Exited);
    });
    Ok(TerminalSession {
        master: pair.master,
        writer,
        child,
    })
}

fn default_shell() -> String {
    #[cfg(unix)]
    {
        std::env::var("SHELL").unwrap_or_else(|_| "/bin/sh".to_string())
    }
    #[cfg(windows)]
    {
        std::env::var("COMSPEC").unwrap_or_else(|_| "powershell.exe".to_string())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::Duration;
    use tempfile::tempdir;

    #[test]
    fn pty_round_trips_shell_output() {
        let directory = tempdir().unwrap();
        let runtime = TerminalRuntime::new(directory.path(), Some("/bin/sh".to_string())).unwrap();
        runtime
            .write("printf '__prumo_terminal_ok__\\n'\r")
            .unwrap();
        let deadline = std::time::Instant::now() + Duration::from_secs(5);
        let mut output = String::new();
        while std::time::Instant::now() < deadline && !output.contains("__prumo_terminal_ok__") {
            thread::sleep(Duration::from_millis(20));
            for event in runtime.drain() {
                if let TerminalEvent::Output(bytes) = event {
                    output.push_str(&String::from_utf8_lossy(&bytes));
                }
            }
        }
        assert!(
            output.contains("__prumo_terminal_ok__"),
            "output was {output:?}"
        );

        runtime.restart().unwrap();
        runtime.write("printf '__prumo_restart_ok__\\n'\r").unwrap();
        let deadline = std::time::Instant::now() + Duration::from_secs(5);
        let mut restarted_output = String::new();
        while std::time::Instant::now() < deadline
            && !restarted_output.contains("__prumo_restart_ok__")
        {
            thread::sleep(Duration::from_millis(20));
            for event in runtime.drain() {
                if let TerminalEvent::Output(bytes) = event {
                    restarted_output.push_str(&String::from_utf8_lossy(&bytes));
                }
            }
        }
        assert!(
            restarted_output.contains("__prumo_restart_ok__"),
            "restarted output was {restarted_output:?}"
        );
    }
}
