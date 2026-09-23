use std::fs;
use std::path::Path;

use crate::services::workspace::validate_path_boundary;
use crate::state::{AppState, DocumentTab, FileKind, NoticeTone};

pub const MAX_FILE_SIZE_BYTES: u64 = 8 * 1024 * 1024;

pub fn open_file(root: &Path, file_path: &Path) -> Result<DocumentTab, String> {
    let canonical_path = validate_path_boundary(root, file_path)?;
    let metadata = fs::metadata(&canonical_path).map_err(|error| {
        format!(
            "Failed to read metadata for {}: {error}",
            canonical_path.display()
        )
    })?;

    if metadata.len() > MAX_FILE_SIZE_BYTES {
        return Err(format!(
            "File exceeds the 8 MB safety limit (size: {} bytes)",
            metadata.len()
        ));
    }

    let content = fs::read_to_string(&canonical_path).map_err(|error| {
        format!(
            "Failed to read {} as UTF-8: {error}",
            canonical_path.display()
        )
    })?;
    let rel_path = canonical_path
        .strip_prefix(root)
        .unwrap_or(&canonical_path)
        .to_string_lossy()
        .to_string();
    let title = canonical_path
        .file_name()
        .and_then(|name| name.to_str())
        .unwrap_or("untitled")
        .to_string();
    let kind = FileKind::from_path(&canonical_path);

    Ok(DocumentTab {
        path: canonical_path,
        rel_path,
        title,
        is_dirty: false,
        persisted_content: content.clone(),
        content,
        is_agent_modified: false,
        kind,
    })
}

pub fn save_file(
    root: &Path,
    file_path: &Path,
    content: &str,
    persisted_content: &str,
) -> Result<(), String> {
    let canonical_path = validate_path_boundary(root, file_path)?;
    let disk_content = fs::read_to_string(&canonical_path).map_err(|error| {
        format!(
            "Failed to read {} before saving: {error}",
            canonical_path.display()
        )
    })?;

    if disk_content != persisted_content && disk_content != content {
        return Err(format!(
            "{} changed on disk after it was opened. Save cancelled to avoid overwriting another edit.",
            canonical_path.display()
        ));
    }
    if disk_content == content {
        return Ok(());
    }

    fs::write(&canonical_path, content)
        .map_err(|error| format!("Failed to write {}: {error}", canonical_path.display()))
}

pub fn activate_path(state: &mut AppState, path: &Path) -> Result<(), String> {
    if state.activate_path(path) {
        state.clear_notice();
        return Ok(());
    }

    let tab = open_file(&state.workspace_root, path).map_err(|error| {
        let message = format!("Could not open {}: {error}", path.display());
        state.show_notice(NoticeTone::Error, message.clone());
        message
    })?;
    state.tabs.push(tab);
    state.active_tab_index = Some(state.tabs.len() - 1);
    state.selected_path = Some(path.to_path_buf());
    state.clear_notice();
    Ok(())
}

pub fn save_active_tab(state: &mut AppState) -> Result<(), String> {
    let (workspace_root, path, content, persisted_content) = {
        let tab = state
            .active_tab()
            .ok_or_else(|| "There is no active file to save".to_string())?;
        (
            state.workspace_root.clone(),
            tab.path.clone(),
            tab.content.clone(),
            tab.persisted_content.clone(),
        )
    };

    if let Err(error) = save_file(&workspace_root, &path, &content, &persisted_content) {
        state.show_notice(NoticeTone::Error, error.clone());
        return Err(error);
    }

    state.mark_active_saved();
    state.show_notice(NoticeTone::Success, "File saved");
    Ok(())
}

pub fn request_close_tab(state: &mut AppState, index: usize) {
    let Some(tab) = state.tabs.get(index) else {
        return;
    };
    if tab.is_dirty {
        state.pending_close_tab = Some(index);
        return;
    }
    state.remove_tab(index);
}

pub fn discard_pending_tab(state: &mut AppState) {
    if let Some(index) = state.pending_close_tab {
        state.remove_tab(index);
    }
}

pub fn save_and_close_pending_tab(state: &mut AppState) {
    if state.pending_close_tab.is_none() {
        return;
    }
    if save_active_tab(state).is_ok()
        && let Some(index) = state.pending_close_tab
    {
        state.remove_tab(index);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::path::PathBuf;
    use tempfile::tempdir;

    #[test]
    fn opens_and_saves_document() {
        let directory = tempdir().unwrap();
        let root = directory.path();
        let file_path = root.join("test.rs");
        fs::write(&file_path, "fn hello() {}").unwrap();

        let tab = open_file(root, &file_path).unwrap();
        assert_eq!(tab.title, "test.rs");
        assert_eq!(tab.content, "fn hello() {}");
        assert_eq!(tab.kind, FileKind::Rust);
        assert!(!tab.is_dirty);

        let new_content = "fn hello_world() {}";
        save_file(root, &file_path, new_content, &tab.persisted_content).unwrap();
        assert_eq!(fs::read_to_string(&file_path).unwrap(), new_content);
    }

    #[test]
    fn rejects_external_change_before_save() {
        let directory = tempdir().unwrap();
        let root = directory.path();
        let file_path = root.join("test.txt");
        fs::write(&file_path, "base").unwrap();
        let tab = open_file(root, &file_path).unwrap();
        fs::write(&file_path, "agent change").unwrap();

        let error =
            save_file(root, &file_path, "human change", &tab.persisted_content).unwrap_err();
        assert!(error.contains("changed on disk"));
        assert_eq!(fs::read_to_string(&file_path).unwrap(), "agent change");
    }

    #[test]
    fn rejects_path_outside_workspace() {
        let root_directory = tempdir().unwrap();
        let other_directory = tempdir().unwrap();
        let secret = other_directory.path().join("secret.key");
        fs::write(&secret, "supersecret").unwrap();

        let error = open_file(root_directory.path(), &secret).unwrap_err();
        assert!(error.contains("Security violation"));
    }

    #[test]
    fn dirty_tab_requires_confirmation() {
        let directory = tempdir().unwrap();
        let root = directory.path();
        let file_path = root.join("test.txt");
        fs::write(&file_path, "base").unwrap();
        let mut state = AppState::new(root.to_path_buf());
        activate_path(&mut state, &file_path).unwrap();
        state.update_active_content("edited".to_string());

        request_close_tab(&mut state, 0);
        assert_eq!(state.tabs.len(), 1);
        assert_eq!(state.pending_close_tab, Some(0));

        discard_pending_tab(&mut state);
        assert!(state.tabs.is_empty());
        assert_eq!(state.active_tab_index, None);
    }

    #[test]
    fn save_button_and_global_save_share_buffer() {
        let directory = tempdir().unwrap();
        let root = directory.path();
        let file_path = root.join("test.txt");
        fs::write(&file_path, "base").unwrap();
        let mut state = AppState::new(PathBuf::from(root));
        activate_path(&mut state, &file_path).unwrap();
        state.update_active_content("edited".to_string());

        save_active_tab(&mut state).unwrap();
        assert!(!state.active_tab().unwrap().is_dirty);
        assert_eq!(fs::read_to_string(&file_path).unwrap(), "edited");
    }
}
