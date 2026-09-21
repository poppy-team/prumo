use std::cmp::Ordering;
use std::fs;
use std::path::{Path, PathBuf};
use crate::watcher::WorkspaceEvent;

#[derive(Debug, Clone)]
pub struct FileNode {
    pub path: PathBuf,
    pub name: String,
    pub is_dir: bool,
    pub is_expanded: bool,
    pub children: Vec<FileNode>,
}

impl FileNode {
    pub fn build(dir: &Path, max_depth: usize) -> Self {
        let name = dir
            .file_name()
            .and_then(|s| s.to_str())
            .unwrap_or(".")
            .to_string();

        if !dir.is_dir() {
            return Self {
                path: dir.to_path_buf(),
                name,
                is_dir: false,
                is_expanded: false,
                children: Vec::new(),
            };
        }

        let mut children = Vec::new();
        if max_depth > 0 {
            if let Ok(entries) = fs::read_dir(dir) {
                for entry in entries.flatten() {
                    let path = entry.path();
                    let file_name = entry.file_name().to_string_lossy().to_string();

                    // Filter out heavy common ignore dirs
                    if file_name.starts_with('.') && file_name != ".prumo" && file_name != ".github" {
                        continue;
                    }
                    if file_name == "target" || file_name == "node_modules" || file_name == "vendor" {
                        continue;
                    }

                    if path.is_dir() {
                        children.push(FileNode::build(&path, max_depth - 1));
                    } else {
                        children.push(FileNode {
                            path,
                            name: file_name,
                            is_dir: false,
                            is_expanded: false,
                            children: Vec::new(),
                        });
                    }
                }
            }
        }

        // Sort: directories first, then alphabetical
        children.sort_by(|a, b| match (a.is_dir, b.is_dir) {
            (true, false) => Ordering::Less,
            (false, true) => Ordering::Greater,
            _ => a.name.to_lowercase().cmp(&b.name.to_lowercase()),
        });

        Self {
            path: dir.to_path_buf(),
            name,
            is_dir: true,
            is_expanded: true, // root is expanded by default
            children,
        }
    }

    pub fn toggle_expand(&mut self) {
        if self.is_dir {
            self.is_expanded = !self.is_expanded;
        }
    }
}

#[derive(Debug, Clone)]
pub struct CachedWorkspaceTree {
    pub root_dir: PathBuf,
    pub root_node: FileNode,
    pub is_dirty: bool,
}

impl CachedWorkspaceTree {
    pub fn new(root_dir: impl AsRef<Path>) -> Self {
        let root = root_dir.as_ref().to_path_buf();
        let root_node = FileNode::build(&root, 6);
        Self {
            root_dir: root,
            root_node,
            is_dirty: false,
        }
    }

    pub fn refresh(&mut self) {
        self.root_node = FileNode::build(&self.root_dir, 6);
        self.is_dirty = false;
    }

    pub fn apply_event(&mut self, _event: &WorkspaceEvent) {
        // Mark dirty to re-index on next idle tick or refresh directly
        self.refresh();
    }
}
