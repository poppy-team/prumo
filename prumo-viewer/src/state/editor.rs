use super::{
    Diagnostic, DiagnosticSeverity, DocumentConflict, DocumentTab, EditorPopupKind,
    EditorPopupState, EditorReveal,
};
use std::path::{Path, PathBuf};

#[derive(Debug, Clone, PartialEq)]
pub struct EditorState {
    pub editor_viewport_height: f32,
    pub tabs: Vec<DocumentTab>,
    pub active_cursor_position: usize,
    pub active_selection_range: Option<(usize, usize)>,
    pub editor_popup: Option<EditorPopupState>,
    pub editor_popup_generation: u64,
    pub diagnostics: Vec<Diagnostic>,
    pub pending_lsp_save: Option<PathBuf>,
    pub active_tab_index: Option<usize>,
    pub pending_close_tab: Option<usize>,
    pub pending_editor_reveal: Option<EditorReveal>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct EditorCursorState {
    pub position: usize,
    pub selection: Option<(usize, usize)>,
}

impl Default for EditorState {
    fn default() -> Self {
        Self {
            editor_viewport_height: 600.,
            tabs: Vec::new(),
            active_cursor_position: 0,
            active_selection_range: None,
            editor_popup: None,
            editor_popup_generation: 0,
            diagnostics: Vec::new(),
            pending_lsp_save: None,
            active_tab_index: None,
            pending_close_tab: None,
            pending_editor_reveal: None,
        }
    }
}

impl EditorState {
    pub fn active_tab(&self) -> Option<&DocumentTab> {
        self.active_tab_index.and_then(|index| self.tabs.get(index))
    }

    pub fn active_tab_mut(&mut self) -> Option<&mut DocumentTab> {
        self.active_tab_index
            .and_then(|index| self.tabs.get_mut(index))
    }

    pub fn tab_index(&self, path: &Path) -> Option<usize> {
        self.tabs.iter().position(|tab| tab.path == path)
    }

    pub fn begin_editor_popup(
        &mut self,
        kind: EditorPopupKind,
        path: PathBuf,
        revision: u64,
        cursor: usize,
        selection: Option<(usize, usize)>,
        range: Option<(usize, usize)>,
    ) -> u64 {
        self.active_cursor_position = cursor;
        self.active_selection_range = selection;
        self.editor_popup_generation = self.editor_popup_generation.saturating_add(1);
        let request_id = self.editor_popup_generation;
        self.editor_popup = Some(EditorPopupState {
            kind,
            path,
            revision,
            cursor,
            selection,
            range,
            request_id,
            items: Vec::new(),
            selected_index: 0,
            loading: true,
            error: None,
        });
        request_id
    }

    pub fn close_editor_popup(&mut self) {
        if self.editor_popup.is_some() {
            self.editor_popup_generation = self.editor_popup_generation.saturating_add(1);
            self.editor_popup = None;
        }
    }

    pub fn is_editor_popup_request_current(
        &self,
        request_id: u64,
        path: &Path,
        revision: u64,
    ) -> bool {
        self.editor_popup.as_ref().is_some_and(|popup| {
            popup.request_id == request_id
                && popup.path.as_path() == path
                && popup.revision == revision
                && popup.cursor == self.active_cursor_position
                && popup.selection == self.active_selection_range
                && self
                    .active_tab()
                    .is_some_and(|tab| tab.path.as_path() == path && tab.revision == revision)
        })
    }

    pub fn update_active_content(&mut self, content: String) {
        let changed = self
            .active_tab_mut()
            .is_some_and(|tab| tab.content != content);
        if changed {
            if let Some(tab) = self.active_tab_mut() {
                tab.revision = tab.revision.saturating_add(1);
                tab.content = content.clone();
                tab.is_dirty = content != tab.persisted_content;
            }
            self.close_editor_popup();
        }
    }

    pub fn mark_active_saved(&mut self) {
        let saved_path = self.active_tab().map(|tab| tab.path.clone());
        if let Some(tab) = self.active_tab_mut() {
            tab.persisted_content = tab.content.clone();
            tab.is_dirty = false;
            tab.conflict = None;
        }
        self.close_editor_popup();
        if let Some(path) = saved_path {
            self.pending_lsp_save = Some(path);
        }
    }

    pub fn set_active_conflict(&mut self, disk_content: String) {
        if let Some(tab) = self.active_tab_mut() {
            tab.conflict = Some(DocumentConflict::ExternalChange { disk_content });
        }
    }

    pub fn replace_diagnostics(&mut self, path: &Path, diagnostics: Vec<Diagnostic>) {
        self.diagnostics
            .retain(|diagnostic| diagnostic.path != path);
        self.diagnostics.extend(diagnostics);
        self.diagnostics.sort_by(|left, right| {
            left.path
                .cmp(&right.path)
                .then(left.line.cmp(&right.line))
                .then(left.column.cmp(&right.column))
        });
    }

    pub fn clear_diagnostics(&mut self, path: &Path) {
        self.diagnostics
            .retain(|diagnostic| diagnostic.path != path);
    }

    pub fn diagnostic_counts(&self) -> (usize, usize) {
        self.diagnostics
            .iter()
            .fold((0, 0), |(errors, warnings), diagnostic| {
                match diagnostic.severity {
                    DiagnosticSeverity::Error => (errors + 1, warnings),
                    DiagnosticSeverity::Warning => (errors, warnings + 1),
                    DiagnosticSeverity::Information | DiagnosticSeverity::Hint => {
                        (errors, warnings)
                    }
                }
            })
    }

    pub fn remove_tab_at(&mut self, index: usize) -> Option<PathBuf> {
        if index >= self.tabs.len() {
            return None;
        }
        let removed_path = self.tabs[index].path.clone();
        self.tabs.remove(index);
        self.clear_diagnostics(&removed_path);
        if self
            .editor_popup
            .as_ref()
            .is_some_and(|popup| popup.path.as_path() == removed_path.as_path())
        {
            self.close_editor_popup();
        }
        if self.pending_lsp_save.as_deref() == Some(removed_path.as_path()) {
            self.pending_lsp_save = None;
        }
        if self.tabs.is_empty() {
            self.active_tab_index = None;
            self.active_cursor_position = 0;
            self.active_selection_range = None;
        } else if let Some(active_index) = self.active_tab_index {
            self.active_tab_index = Some(match active_index.cmp(&index) {
                std::cmp::Ordering::Less => active_index,
                std::cmp::Ordering::Equal => index.min(self.tabs.len() - 1),
                std::cmp::Ordering::Greater => active_index - 1,
            });
        } else {
            self.active_tab_index = Some(index.min(self.tabs.len() - 1));
        }
        self.pending_close_tab = None;
        Some(removed_path)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::state::FileKind;

    fn tab(path: &str, content: &str) -> DocumentTab {
        DocumentTab {
            path: PathBuf::from(path),
            rel_path: path.to_string(),
            title: path.rsplit('/').next().unwrap_or(path).to_string(),
            is_dirty: false,
            content: content.to_string(),
            persisted_content: content.to_string(),
            revision: 0,
            conflict: None,
            is_agent_modified: false,
            kind: FileKind::Rust,
        }
    }

    #[test]
    fn popup_and_content_updates_stay_inside_editor_state() {
        let mut state = EditorState::default();
        let path = PathBuf::from("/tmp/main.rs");
        state.tabs.push(tab("/tmp/main.rs", "fn main() {}"));
        state.active_tab_index = Some(0);
        let request_id = state.begin_editor_popup(
            EditorPopupKind::Completion,
            path.clone(),
            0,
            12,
            None,
            Some((12, 12)),
        );
        assert!(state.is_editor_popup_request_current(request_id, &path, 0));
        state.active_selection_range = Some((0, 1));
        assert!(!state.is_editor_popup_request_current(request_id, &path, 0));
        state.update_active_content("fn main() { changed }".to_string());
        assert!(state.editor_popup.is_none());
        assert_eq!(state.active_tab().unwrap().revision, 1);
    }

    #[test]
    fn removing_last_tab_resets_editor_selection() {
        let mut state = EditorState::default();
        state.tabs.push(tab("/tmp/main.rs", "fn main() {}"));
        state.active_tab_index = Some(0);
        state.active_cursor_position = 8;
        state.active_selection_range = Some((2, 8));
        state.pending_close_tab = Some(0);
        assert_eq!(state.remove_tab_at(0), Some(PathBuf::from("/tmp/main.rs")));
        assert!(state.tabs.is_empty());
        assert_eq!(state.active_tab_index, None);
        assert_eq!(state.active_cursor_position, 0);
        assert_eq!(state.active_selection_range, None);
        assert_eq!(state.pending_close_tab, None);
    }
}
