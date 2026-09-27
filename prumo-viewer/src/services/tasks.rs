use crate::services::workspace::validate_path_boundary;
use prumo_extension_sdk::manifest::TaskContribution;
use std::io::Read;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::thread;
use std::time::{Duration, Instant};

const MAX_OUTPUT_BYTES: usize = 1024 * 1024;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TaskResult {
    pub exit_code: Option<i32>,
    pub stdout: String,
    pub stderr: String,
    pub timed_out: bool,
}

pub fn run_task(
    root: &Path,
    task: &TaskContribution,
    timeout: Duration,
) -> Result<TaskResult, String> {
    let working_directory = task
        .working_directory
        .as_deref()
        .map(|directory| validate_path_boundary(root, &root.join(directory)))
        .transpose()?
        .unwrap_or_else(|| root.to_path_buf());
    let command = if task.command.contains('/') || task.command.contains('\\') {
        let candidate = PathBuf::from(&task.command);
        let path = if candidate.is_absolute() {
            candidate
        } else {
            validate_path_boundary(root, &root.join(&task.command))?
        };
        if !path.is_file() {
            return Err(format!("task command does not exist: {}", path.display()));
        }
        path
    } else {
        PathBuf::from(&task.command)
    };
    let mut process = Command::new(command)
        .args(&task.args)
        .current_dir(working_directory)
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|error| format!("could not start task: {error}"))?;
    let stdout = process
        .stdout
        .take()
        .ok_or_else(|| "task stdout was not piped".to_string())?;
    let stderr = process
        .stderr
        .take()
        .ok_or_else(|| "task stderr was not piped".to_string())?;
    let stdout_reader = thread::spawn(move || read_bounded(stdout));
    let stderr_reader = thread::spawn(move || read_bounded(stderr));
    let deadline = Instant::now() + timeout;
    let mut timed_out = false;
    let status = loop {
        if let Some(status) = process
            .try_wait()
            .map_err(|error| format!("could not wait for task: {error}"))?
        {
            break status;
        }
        if Instant::now() >= deadline {
            let _ = process.kill();
            timed_out = true;
            break process
                .wait()
                .map_err(|error| format!("could not reap task: {error}"))?;
        }
        thread::sleep(Duration::from_millis(10));
    };
    let stdout = stdout_reader
        .join()
        .map_err(|_| "task stdout reader panicked".to_string())??;
    let stderr = stderr_reader
        .join()
        .map_err(|_| "task stderr reader panicked".to_string())??;
    Ok(TaskResult {
        exit_code: status.code(),
        stdout: String::from_utf8_lossy(&stdout).into_owned(),
        stderr: String::from_utf8_lossy(&stderr).into_owned(),
        timed_out,
    })
}

fn read_bounded(mut reader: impl Read) -> Result<Vec<u8>, String> {
    let mut output = Vec::with_capacity(8192);
    let mut buffer = [0; 8192];
    loop {
        let read = reader
            .read(&mut buffer)
            .map_err(|error| format!("could not read task output: {error}"))?;
        if read == 0 {
            break;
        }
        let remaining = MAX_OUTPUT_BYTES.saturating_sub(output.len());
        output.extend_from_slice(&buffer[..read.min(remaining)]);
    }
    Ok(output)
}

#[cfg(test)]
mod tests {
    use super::*;
    use prumo_extension_sdk::manifest::TaskContribution;

    #[test]
    fn rejects_task_escape() {
        let directory = tempfile::tempdir().unwrap();
        let task = TaskContribution {
            id: "com.example.task".to_string(),
            title: "Task".to_string(),
            command: "echo".to_string(),
            args: Vec::new(),
            working_directory: Some("../outside".to_string()),
        };
        assert!(run_task(directory.path(), &task, Duration::from_secs(1)).is_err());
    }

    #[cfg(unix)]
    #[test]
    fn runs_task_and_captures_bounded_output() {
        let directory = tempfile::tempdir().unwrap();
        let task = TaskContribution {
            id: "com.example.task".to_string(),
            title: "Task".to_string(),
            command: "/bin/sh".to_string(),
            args: vec!["-c".to_string(), "printf task".to_string()],
            working_directory: None,
        };
        let result = run_task(directory.path(), &task, Duration::from_secs(1)).unwrap();
        assert_eq!(result.exit_code, Some(0));
        assert_eq!(result.stdout, "task");
        assert!(!result.timed_out);
    }
}
