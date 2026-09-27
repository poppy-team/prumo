use std::fs;
use std::path::{Path, PathBuf};

use crate::services::workspace::validate_path_boundary;
use crate::state::{
    AppState, ConflictResolution, DocumentConflict, DocumentTab, FileKind, NoticeTone,
};

pub const MAX_FILE_SIZE_BYTES: u64 = 8 * 1024 * 1024;

pub fn file_uri_to_path(uri: &str) -> Result<PathBuf, String> {
    let encoded = uri
        .strip_prefix("file://")
        .ok_or_else(|| "LSP URI is not a file URI".to_string())?;
    let (authority, encoded_path) = if let Some(path) = encoded.strip_prefix('/') {
        (None, format!("/{path}"))
    } else if let Some((authority, path)) = encoded.split_once('/') {
        (Some(authority), format!("/{path}"))
    } else {
        (Some(encoded), "/".to_string())
    };
    let path = decode_uri_component(&encoded_path)?;
    let path = if path.as_bytes().first() == Some(&b'/')
        && path.as_bytes().get(1).is_some_and(u8::is_ascii_alphabetic)
        && path.as_bytes().get(2) == Some(&b':')
    {
        path[1..].to_string()
    } else {
        path
    };
    let path = match authority {
        None | Some("localhost") => path,
        Some(authority) => format!("//{authority}{path}"),
    };
    Ok(PathBuf::from(path))
}

fn decode_uri_component(value: &str) -> Result<String, String> {
    let mut bytes = Vec::with_capacity(value.len());
    let mut characters = value.chars();
    while let Some(character) = characters.next() {
        if character == '%' {
            let high = characters.next().and_then(|value| value.to_digit(16));
            let low = characters.next().and_then(|value| value.to_digit(16));
            let (Some(high), Some(low)) = (high, low) else {
                return Err("LSP URI contains invalid percent encoding".to_string());
            };
            bytes.push((high * 16 + low) as u8);
        } else {
            let mut buffer = [0; 4];
            bytes.extend_from_slice(character.encode_utf8(&mut buffer).as_bytes());
        }
    }
    String::from_utf8(bytes).map_err(|_| "LSP URI is not valid UTF-8".to_string())
}

pub fn utf16_position_to_line_character(content: &str, position: usize) -> (usize, usize) {
    let mut consumed = 0;
    let mut line = 1;
    let mut character = 0;
    let mut characters = content.chars().peekable();
    while let Some(value) = characters.next() {
        if consumed >= position {
            break;
        }
        if value == '\r' && characters.peek() == Some(&'\n') {
            continue;
        }
        let width = value.len_utf16();
        if value == '\n' {
            line += 1;
            character = 0;
        } else {
            character += width;
        }
        consumed += width;
    }
    (line, character)
}

pub fn line_character_to_utf16_position(
    content: &str,
    line: usize,
    character: usize,
) -> Option<usize> {
    if line == 0 {
        return None;
    }
    let mut current_line = 1;
    let mut line_character = 0;
    let mut position = 0;
    let mut characters = content.chars().peekable();
    while let Some(value) = characters.next() {
        if current_line == line && line_character >= character {
            return Some(position);
        }
        if value == '\r' && characters.peek() == Some(&'\n') {
            continue;
        }
        position += value.len_utf16();
        if value == '\n' {
            current_line += 1;
            line_character = 0;
        } else {
            line_character += value.len_utf16();
        }
    }
    (current_line == line && line_character >= character).then_some(position)
}

pub fn utf16_position_to_byte(
    content: &str,
    line: usize,
    character: usize,
) -> Result<usize, String> {
    if line == 0 {
        return Err("LSP line must be one-based".to_string());
    }
    let mut current_line = 1;
    let mut consumed = 0;
    let mut characters = content.char_indices().peekable();
    while let Some((offset, value)) = characters.next() {
        if current_line == line && consumed >= character {
            return Ok(offset);
        }
        if value == '\r' && characters.peek().is_some_and(|(_, next)| *next == '\n') {
            if current_line == line && consumed >= character {
                return Ok(offset);
            }
            continue;
        }
        if value == '\n' {
            if current_line == line && consumed >= character {
                return Ok(offset);
            }
            current_line += 1;
            consumed = 0;
        } else {
            consumed += value.len_utf16();
        }
    }
    if current_line == line && consumed >= character {
        Ok(content.len())
    } else {
        Err(format!(
            "LSP position is outside the document: {line}:{character}"
        ))
    }
}

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
        revision: 0,
        conflict: None,
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
    state.close_editor_popup();
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
        if let Ok(disk_content) = fs::read_to_string(&path)
            && disk_content != persisted_content
        {
            state.set_active_conflict(disk_content);
        }
        state.show_notice(NoticeTone::Error, error.clone());
        return Err(error);
    }

    state.mark_active_saved();
    let saved = state.active_tab().map(|tab| tab.rel_path.clone());
    if let Some(saved) = saved {
        state.follow_conflict_paths.retain(|path| path != &saved);
        state.agent_line_decorations.clear();
    }
    state.show_notice(NoticeTone::Success, "File saved");
    Ok(())
}

pub fn resolve_active_conflict(
    state: &mut AppState,
    resolution: ConflictResolution,
) -> Result<(), String> {
    let Some(tab) = state.active_tab_mut() else {
        return Err("There is no active file to resolve".to_string());
    };
    let Some(conflict) = tab.conflict.clone() else {
        return Err("The active file has no external-change conflict".to_string());
    };
    let DocumentConflict::ExternalChange { disk_content } = conflict;
    match resolution {
        ConflictResolution::Reload => {
            tab.content = disk_content.clone();
            tab.persisted_content = disk_content;
            tab.is_dirty = false;
            tab.conflict = None;
            tab.revision = tab.revision.saturating_add(1);
        }
        ConflictResolution::KeepMine => {
            tab.persisted_content = disk_content.clone();
            tab.is_dirty = tab.content != disk_content;
            tab.conflict = None;
            tab.revision = tab.revision.saturating_add(1);
        }
        ConflictResolution::ThreeWayMerge => {
            let base = &tab.persisted_content;
            let mine = &tab.content;
            let theirs = &disk_content;
            let outcome = crate::services::merge::three_way_merge(base, mine, theirs);
            let is_clean = outcome.is_clean();
            tab.content = outcome.content;
            tab.persisted_content = disk_content.clone();
            tab.is_dirty = !is_clean || tab.content != disk_content;
            tab.conflict = None;
            tab.revision = tab.revision.saturating_add(1);
        }
    }
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
    fn converts_utf16_cursor_positions_and_file_uris() {
        assert_eq!(utf16_position_to_line_character("one\ntwo", 5), (2, 1));
        assert_eq!(
            file_uri_to_path("file:///tmp/a%20file.rs").unwrap(),
            PathBuf::from("/tmp/a file.rs")
        );
        assert_eq!(
            file_uri_to_path("file:///C:/tmp/main.rs").unwrap(),
            PathBuf::from("C:/tmp/main.rs")
        );
        assert_eq!(
            file_uri_to_path("file://server/share/main.rs").unwrap(),
            PathBuf::from("//server/share/main.rs")
        );
    }

    #[test]
    fn converts_lsp_positions_to_byte_offsets() {
        let content = "á\nsecond";
        assert_eq!(utf16_position_to_byte(content, 1, 0).unwrap(), 0);
        assert_eq!(utf16_position_to_byte(content, 1, 1).unwrap(), 2);
        assert_eq!(utf16_position_to_byte(content, 2, 0).unwrap(), 3);
        assert_eq!(utf16_position_to_byte(content, 2, 3).unwrap(), 6);
        assert!(utf16_position_to_byte(content, 2, 99).is_err());
        assert_eq!(line_character_to_utf16_position(content, 2, 0), Some(2));
        assert_eq!(line_character_to_utf16_position(content, 2, 3), Some(5));
        let crlf = "one\r\ntwo";
        assert_eq!(utf16_position_to_line_character(crlf, 4), (2, 0));
        assert_eq!(utf16_position_to_byte(crlf, 1, 3).unwrap(), 3);
        assert_eq!(line_character_to_utf16_position(crlf, 2, 0), Some(4));
    }

    #[test]
    fn resolves_external_change_before_saving() {
        let directory = tempdir().unwrap();
        let root = directory.path();
        let file_path = root.join("test.txt");
        fs::write(&file_path, "base").unwrap();
        let mut state = AppState::new(PathBuf::from(root));
        activate_path(&mut state, &file_path).unwrap();
        state.update_active_content("human".to_string());
        fs::write(&file_path, "agent").unwrap();

        assert!(save_active_tab(&mut state).is_err());
        assert!(state.active_tab().unwrap().conflict.is_some());
        resolve_active_conflict(&mut state, ConflictResolution::KeepMine).unwrap();
        assert!(state.active_tab().unwrap().is_dirty);
        save_active_tab(&mut state).unwrap();
        assert_eq!(fs::read_to_string(&file_path).unwrap(), "human");
    }

    #[test]
    fn resolves_conflict_with_three_way_merge() {
        let directory = tempdir().unwrap();
        let root = directory.path();
        let file_path = root.join("test.txt");
        fs::write(&file_path, "header\nbase\nfooter\n").unwrap();
        let mut state = AppState::new(PathBuf::from(root));
        activate_path(&mut state, &file_path).unwrap();
        state.update_active_content("header\nbase\nfooter\nhuman addition\n".to_string());
        fs::write(&file_path, "agent addition\nheader\nbase\nfooter\n").unwrap();

        assert!(save_active_tab(&mut state).is_err());
        assert!(state.active_tab().unwrap().conflict.is_some());
        resolve_active_conflict(&mut state, ConflictResolution::ThreeWayMerge).unwrap();
        assert!(state.active_tab().unwrap().conflict.is_none());
        let merged = &state.active_tab().unwrap().content;
        assert!(merged.contains("agent addition"));
        assert!(merged.contains("human addition"));
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
