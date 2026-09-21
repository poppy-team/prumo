use std::collections::HashMap;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DocumentSessionState {
    pub open_tabs: Vec<String>,
    pub active_tab: Option<String>,
}

#[derive(Debug, Clone)]
pub struct DocumentBuffer {
    pub path: PathBuf,
    pub content: String,
    pub saved_content: String,
    pub is_dirty: bool,
    pub is_deleted_on_disk: bool,
    pub cursor_line: usize,
    pub cursor_col: usize,
    pub highlight_range: Option<(usize, usize)>, // (start_line, end_line)
    pub version: u64,
}

impl DocumentBuffer {
    pub fn new(path: PathBuf, content: String) -> Self {
        Self {
            path,
            saved_content: content.clone(),
            content,
            is_dirty: false,
            is_deleted_on_disk: false,
            cursor_line: 1,
            cursor_col: 1,
            highlight_range: None,
            version: 1,
        }
    }

    pub fn lines_count(&self) -> usize {
        if self.content.is_empty() {
            1
        } else {
            self.content.lines().count().max(1)
        }
    }

    pub fn line_str(&self, line_num: usize) -> Option<&str> {
        if line_num == 0 {
            return None;
        }
        self.content.lines().nth(line_num - 1)
    }

    pub fn set_content(&mut self, new_content: String) {
        if new_content != self.content {
            self.content = new_content;
            self.is_dirty = self.content != self.saved_content;
            self.version += 1;
        }
    }

    pub fn mark_saved(&mut self) {
        self.saved_content = self.content.clone();
        self.is_dirty = false;
        self.is_deleted_on_disk = false;
        self.version += 1;
    }

    pub fn revert(&mut self) {
        self.content = self.saved_content.clone();
        self.is_dirty = false;
        self.version += 1;
    }

    pub fn go_to_line(&mut self, line: usize) {
        let max_lines = self.lines_count();
        self.cursor_line = line.clamp(1, max_lines);
        self.cursor_col = 1;
    }

    pub fn highlight_lines(&mut self, start: usize, end: usize) {
        let max_lines = self.lines_count();
        let s = start.clamp(1, max_lines);
        let e = end.clamp(s, max_lines);
        self.highlight_range = Some((s, e));
        self.cursor_line = s;
    }

    pub fn clear_highlight(&mut self) {
        self.highlight_range = None;
    }
}

#[derive(Debug, Default)]
pub struct DocumentStore {
    workspace_root: Option<PathBuf>,
    documents: HashMap<PathBuf, DocumentBuffer>,
    open_tabs: Vec<PathBuf>,
    active_tab: Option<PathBuf>,
    nav_history: Vec<PathBuf>,
    nav_index: usize,
}

impl DocumentStore {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn with_workspace_root(mut self, root: PathBuf) -> Self {
        self.workspace_root = Some(root.canonicalize().unwrap_or(root));
        self
    }

    pub fn set_workspace_root(&mut self, root: PathBuf) {
        self.workspace_root = Some(root.canonicalize().unwrap_or(root));
    }

    /// Validates that candidate path does not escape workspace boundaries (Invariant 11)
    pub fn validate_path(&self, candidate: &Path) -> Result<PathBuf, io::Error> {
        let canonical = candidate.canonicalize().unwrap_or_else(|_| candidate.to_path_buf());
        if let Some(root) = &self.workspace_root {
            if !canonical.starts_with(root) {
                return Err(io::Error::new(
                    io::ErrorKind::PermissionDenied,
                    format!("Path outside workspace rejected: {}", canonical.display()),
                ));
            }
        }
        Ok(canonical)
    }

    pub fn open_file(&mut self, path: &Path) -> Result<&mut DocumentBuffer, io::Error> {
        let canonical = self.validate_path(path)?;

        if !self.documents.contains_key(&canonical) {
            let content = if canonical.exists() {
                fs::read_to_string(&canonical).unwrap_or_else(|_| "<binary or unreadable file>".to_string())
            } else {
                String::new()
            };
            let buf = DocumentBuffer::new(canonical.clone(), content);
            self.documents.insert(canonical.clone(), buf);
        }

        if !self.open_tabs.contains(&canonical) {
            self.open_tabs.push(canonical.clone());
        }

        self.set_active(&canonical);
        Ok(self.documents.get_mut(&canonical).unwrap())
    }

    pub fn set_active(&mut self, path: &Path) {
        let canonical = path.canonicalize().unwrap_or_else(|_| path.to_path_buf());
        if self.open_tabs.contains(&canonical) {
            self.active_tab = Some(canonical.clone());
            if self.nav_history.last() != Some(&canonical) {
                self.nav_history.push(canonical);
                self.nav_index = self.nav_history.len() - 1;
            }
        }
    }

    pub fn close_file(&mut self, path: &Path) {
        let canonical = path.canonicalize().unwrap_or_else(|_| path.to_path_buf());
        if let Some(pos) = self.open_tabs.iter().position(|p| p == &canonical) {
            self.open_tabs.remove(pos);
            if self.active_tab.as_ref() == Some(&canonical) {
                self.active_tab = self.open_tabs.get(pos.saturating_sub(1)).cloned()
                    .or_else(|| self.open_tabs.first().cloned());
            }
        }
    }

    /// Rename preserves buffer content and open tab position (Invariant 5)
    pub fn rename_file(&mut self, old_path: &Path, new_path: &Path) {
        let old_can = old_path.canonicalize().unwrap_or_else(|_| old_path.to_path_buf());
        let new_can = new_path.canonicalize().unwrap_or_else(|_| new_path.to_path_buf());

        if let Some(mut doc) = self.documents.remove(&old_can) {
            doc.path = new_can.clone();
            self.documents.insert(new_can.clone(), doc);
        }

        if let Some(pos) = self.open_tabs.iter().position(|p| p == &old_can) {
            self.open_tabs[pos] = new_can.clone();
        }

        if self.active_tab.as_ref() == Some(&old_can) {
            self.active_tab = Some(new_can.clone());
        }

        for item in &mut self.nav_history {
            if *item == old_can {
                *item = new_can.clone();
            }
        }
    }

    /// Deleting an open file produces an explicit deleted state (Invariant 6)
    pub fn mark_deleted(&mut self, path: &Path) {
        let canonical = path.canonicalize().unwrap_or_else(|_| path.to_path_buf());
        if let Some(doc) = self.documents.get_mut(&canonical) {
            doc.is_deleted_on_disk = true;
        }
    }

    pub fn active_document(&self) -> Option<&DocumentBuffer> {
        let path = self.active_tab.as_ref()?;
        self.documents.get(path)
    }

    pub fn active_document_mut(&mut self) -> Option<&mut DocumentBuffer> {
        let path = self.active_tab.clone()?;
        self.documents.get_mut(&path)
    }

    pub fn get_document(&self, path: &Path) -> Option<&DocumentBuffer> {
        let canonical = path.canonicalize().unwrap_or_else(|_| path.to_path_buf());
        self.documents.get(&canonical)
    }

    pub fn get_document_mut(&mut self, path: &Path) -> Option<&mut DocumentBuffer> {
        let canonical = path.canonicalize().unwrap_or_else(|_| path.to_path_buf());
        self.documents.get_mut(&canonical)
    }

    pub fn open_tabs(&self) -> &[PathBuf] {
        &self.open_tabs
    }

    pub fn active_tab(&self) -> Option<&PathBuf> {
        self.active_tab.as_ref()
    }

    pub fn save_active(&mut self) -> Result<(), io::Error> {
        if let Some(path) = self.active_tab.clone() {
            self.save_file(&path)?;
        }
        Ok(())
    }

    pub fn save_file(&mut self, path: &Path) -> Result<(), io::Error> {
        let canonical = path.canonicalize().unwrap_or_else(|_| path.to_path_buf());
        if let Some(doc) = self.documents.get_mut(&canonical) {
            if doc.is_dirty || doc.is_deleted_on_disk {
                if let Some(parent) = doc.path.parent() {
                    fs::create_dir_all(parent)?;
                }
                fs::write(&doc.path, &doc.content)?;
                doc.mark_saved();
            }
        }
        Ok(())
    }

    /// External modification check: reload from disk ONLY if document is clean.
    /// If document has unsaved human edits, do NOT silently overwrite!
    pub fn reload_if_clean(&mut self, path: &Path) -> Result<bool, io::Error> {
        let canonical = path.canonicalize().unwrap_or_else(|_| path.to_path_buf());
        if let Some(doc) = self.documents.get_mut(&canonical) {
            if !doc.is_dirty && canonical.exists() {
                let fresh = fs::read_to_string(&canonical)?;
                if fresh != doc.content {
                    doc.content = fresh.clone();
                    doc.saved_content = fresh;
                    doc.is_deleted_on_disk = false;
                    doc.version += 1;
                    return Ok(true);
                }
            }
        }
        Ok(false)
    }

    /// Preserves tab state across restarts (Invariant 14)
    pub fn save_session(&self, file_path: &Path) -> Result<(), io::Error> {
        if let Some(parent) = file_path.parent() {
            fs::create_dir_all(parent)?;
        }
        let state = DocumentSessionState {
            open_tabs: self.open_tabs.iter().map(|p| p.to_string_lossy().to_string()).collect(),
            active_tab: self.active_tab.as_ref().map(|p| p.to_string_lossy().to_string()),
        };
        let data = serde_json::to_string_pretty(&state)
            .map_err(|e| io::Error::new(io::ErrorKind::InvalidData, e))?;
        fs::write(file_path, data)
    }

    /// Restores tab state from saved session (Invariant 14)
    pub fn load_session(&mut self, file_path: &Path) -> Result<(), io::Error> {
        if !file_path.exists() {
            return Ok(());
        }
        let data = fs::read_to_string(file_path)?;
        let state: DocumentSessionState = serde_json::from_str(&data)
            .map_err(|e| io::Error::new(io::ErrorKind::InvalidData, e))?;

        for tab_str in state.open_tabs {
            let path = PathBuf::from(tab_str);
            if path.exists() {
                let _ = self.open_file(&path);
            }
        }

        if let Some(active_str) = state.active_tab {
            let path = PathBuf::from(active_str);
            self.set_active(&path);
        }

        Ok(())
    }
}
