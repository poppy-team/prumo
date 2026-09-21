use std::path::{Path, PathBuf};
use std::sync::mpsc::{channel, Receiver, Sender};
use std::time::Duration;
use notify::{Config, EventKind, RecommendedWatcher, RecursiveMode, Watcher};

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum WorkspaceEvent {
    Created(PathBuf),
    Modified(PathBuf),
    Deleted(PathBuf),
}

pub struct WorkspaceWatcher {
    _watcher: Option<RecommendedWatcher>,
    rx: Receiver<WorkspaceEvent>,
}

impl WorkspaceWatcher {
    pub fn new(root: &Path) -> Result<Self, String> {
        let (tx, rx) = channel();
        let event_tx: Sender<WorkspaceEvent> = tx;

        let mut watcher = RecommendedWatcher::new(
            move |res: Result<notify::Event, notify::Error>| {
                if let Ok(ev) = res {
                    let kind = match ev.kind {
                        EventKind::Create(_) => {
                            ev.paths.first().map(|p| WorkspaceEvent::Created(p.clone()))
                        }
                        EventKind::Modify(_) => {
                            ev.paths.first().map(|p| WorkspaceEvent::Modified(p.clone()))
                        }
                        EventKind::Remove(_) => {
                            ev.paths.first().map(|p| WorkspaceEvent::Deleted(p.clone()))
                        }
                        _ => None,
                    };

                    if let Some(event) = kind {
                        let _ = event_tx.send(event);
                    }
                }
            },
            Config::default().with_poll_interval(Duration::from_millis(500)),
        )
        .map_err(|e| format!("failed to initialize file watcher: {}", e))?;

        if root.exists() {
            watcher
                .watch(root, RecursiveMode::Recursive)
                .map_err(|e| format!("failed to watch path {}: {}", root.display(), e))?;
        }

        Ok(Self {
            _watcher: Some(watcher),
            rx,
        })
    }

    pub fn poll_events(&self) -> Vec<WorkspaceEvent> {
        let mut events = Vec::new();
        while let Ok(ev) = self.rx.try_recv() {
            events.push(ev);
        }
        events
    }
}
