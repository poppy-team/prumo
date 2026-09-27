use notify::{Config, Event, RecommendedWatcher, RecursiveMode, Watcher};
use std::path::Path;
use std::sync::mpsc::{self, Receiver};
use std::thread;
use std::time::Duration;

fn is_noise(path: &Path) -> bool {
    path.components().any(|part| {
        matches!(
            part.as_os_str().to_str(),
            Some(".git" | "target" | "node_modules" | ".prumo" | ".venv" | "dist" | "build")
        )
    })
}

pub fn start_workspace_watcher(root: &Path) -> Option<Receiver<()>> {
    let (notify_sender, notify_receiver) = mpsc::channel();
    let mut watcher = RecommendedWatcher::new(
        move |result: Result<Event, notify::Error>| {
            if let Ok(event) = result {
                let _ = notify_sender.send(event);
            }
        },
        Config::default(),
    )
    .ok()?;
    watcher.watch(root, RecursiveMode::Recursive).ok()?;
    let (dirty_sender, dirty_receiver) = mpsc::channel();
    thread::spawn(move || {
        let _watcher = watcher;
        let mut pending = false;
        loop {
            match notify_receiver.recv_timeout(Duration::from_millis(400)) {
                Ok(event) => {
                    if event.paths.iter().any(|path| !is_noise(path)) {
                        pending = true;
                    }
                }
                Err(mpsc::RecvTimeoutError::Timeout) => {
                    if pending {
                        pending = false;
                        if dirty_sender.send(()).is_err() {
                            break;
                        }
                    }
                }
                Err(mpsc::RecvTimeoutError::Disconnected) => break,
            }
        }
    });
    Some(dirty_receiver)
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    #[test]
    fn watcher_signals_real_file_changes() {
        let directory = tempdir().unwrap();
        let root = directory.path().to_path_buf();
        let Some(receiver) = start_workspace_watcher(&root) else {
            return;
        };
        std::fs::write(root.join("watched.txt"), "hello\n").unwrap();
        let deadline = std::time::Instant::now() + Duration::from_secs(5);
        let mut signaled = false;
        while std::time::Instant::now() < deadline && !signaled {
            if receiver.recv_timeout(Duration::from_millis(200)).is_ok() {
                signaled = true;
            }
        }
        assert!(signaled, "watcher did not report the new file");
    }

    #[test]
    fn noise_paths_are_ignored() {
        assert!(is_noise(Path::new("/work/target/debug/app")));
        assert!(is_noise(Path::new("/work/.git/HEAD")));
        assert!(!is_noise(Path::new("/work/src/main.rs")));
    }
}
