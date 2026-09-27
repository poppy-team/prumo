use crate::state::AppState;
use std::path::{Path, PathBuf};
use std::process::Command;

pub fn validate_name(name: &str) -> Result<(), String> {
    let trimmed = name.trim();
    if trimmed.is_empty() {
        return Err("Name must not be empty".to_string());
    }
    if trimmed == "." || trimmed == ".." {
        return Err("Name is not allowed".to_string());
    }
    if trimmed.contains('/') || trimmed.contains('\\') {
        return Err("Name must not contain path separators".to_string());
    }
    Ok(())
}

pub fn create_file(parent: &Path, name: &str) -> Result<PathBuf, String> {
    validate_name(name)?;
    if !parent.is_dir() {
        return Err("Parent folder does not exist".to_string());
    }
    let path = parent.join(name.trim());
    if path.exists() {
        return Err("A file with this name already exists".to_string());
    }
    std::fs::write(&path, "").map_err(|error| format!("Could not create file: {error}"))?;
    Ok(path)
}

pub fn create_folder(parent: &Path, name: &str) -> Result<PathBuf, String> {
    validate_name(name)?;
    if !parent.is_dir() {
        return Err("Parent folder does not exist".to_string());
    }
    let path = parent.join(name.trim());
    if path.exists() {
        return Err("A folder with this name already exists".to_string());
    }
    std::fs::create_dir(&path).map_err(|error| format!("Could not create folder: {error}"))?;
    Ok(path)
}

pub fn rename_path(path: &Path, new_name: &str) -> Result<PathBuf, String> {
    validate_name(new_name)?;
    if !path.exists() {
        return Err("Path does not exist".to_string());
    }
    let parent = path
        .parent()
        .ok_or_else(|| "Path has no parent folder".to_string())?;
    let next = parent.join(new_name.trim());
    if next.exists() {
        return Err("A file with this name already exists".to_string());
    }
    std::fs::rename(path, &next).map_err(|error| format!("Could not rename: {error}"))?;
    Ok(next)
}

fn duplicate_target(path: &Path) -> Result<PathBuf, String> {
    let parent = path
        .parent()
        .ok_or_else(|| "Path has no parent folder".to_string())?;
    let name = path
        .file_name()
        .and_then(|name| name.to_str())
        .ok_or_else(|| "Invalid file name".to_string())?;
    let candidate = |index: usize| {
        if path.is_dir() {
            if index == 0 {
                parent.join(format!("{name} copy"))
            } else {
                parent.join(format!("{name} copy {index}"))
            }
        } else {
            let stem = path
                .file_stem()
                .and_then(|stem| stem.to_str())
                .unwrap_or(name);
            let suffix = path
                .extension()
                .and_then(|extension| extension.to_str())
                .map(|extension| format!(".{extension}"))
                .unwrap_or_default();
            if index == 0 {
                parent.join(format!("{stem} copy{suffix}"))
            } else {
                parent.join(format!("{stem} copy {index}{suffix}"))
            }
        }
    };
    let mut index = 0;
    loop {
        let next = candidate(index);
        if !next.exists() {
            return Ok(next);
        }
        index += 1;
        if index > 99 {
            return Err("Could not find a free duplicate name".to_string());
        }
    }
}

fn copy_directory(source: &Path, target: &Path) -> Result<(), String> {
    std::fs::create_dir(target).map_err(|error| format!("Could not duplicate: {error}"))?;
    let entries =
        std::fs::read_dir(source).map_err(|error| format!("Could not duplicate: {error}"))?;
    for entry in entries {
        let entry = entry.map_err(|error| format!("Could not duplicate: {error}"))?;
        let from = entry.path();
        let to = target.join(entry.file_name());
        if from.is_dir() {
            copy_directory(&from, &to)?;
        } else {
            std::fs::copy(&from, &to).map_err(|error| format!("Could not duplicate: {error}"))?;
        }
    }
    Ok(())
}

pub fn duplicate_path(path: &Path) -> Result<PathBuf, String> {
    if !path.exists() {
        return Err("Path does not exist".to_string());
    }
    let target = duplicate_target(path)?;
    if path.is_dir() {
        copy_directory(path, &target)?;
    } else {
        std::fs::copy(path, &target).map_err(|error| format!("Could not duplicate: {error}"))?;
    }
    Ok(target)
}

pub fn delete_path(path: &Path) -> Result<(), String> {
    if !path.exists() {
        return Err("Path does not exist".to_string());
    }
    if path.is_dir() {
        std::fs::remove_dir_all(path)
            .map_err(|error| format!("Could not delete folder: {error}"))?;
    } else {
        std::fs::remove_file(path).map_err(|error| format!("Could not delete file: {error}"))?;
    }
    Ok(())
}

pub fn move_to_trash(root: &Path, path: &Path) -> Result<(), String> {
    let canonical_root = root
        .canonicalize()
        .map_err(|error| format!("Invalid workspace root {}: {error}", root.display()))?;
    let target = crate::services::workspace::validate_path_boundary(root, path)?;
    if target == canonical_root {
        return Err("The workspace root cannot be moved to trash".to_string());
    }
    if !target.exists() {
        return Err("Path does not exist".to_string());
    }
    let output = trash_command(&target)
        .output()
        .map_err(|error| format!("Could not open the system trash: {error}"))?;
    if !output.status.success() {
        let detail = String::from_utf8_lossy(&output.stderr).trim().to_string();
        return Err(if detail.is_empty() {
            "The system trash rejected the item".to_string()
        } else {
            format!("The system trash rejected the item: {detail}")
        });
    }
    if target.exists() {
        return Err("The system trash did not move the item".to_string());
    }
    Ok(())
}

#[cfg(target_os = "linux")]
fn trash_command(path: &Path) -> Command {
    let mut command = Command::new("gio");
    command.args(["trash", "--"]).arg(path);
    command
}

#[cfg(target_os = "macos")]
fn trash_command(path: &Path) -> Command {
    let mut command = Command::new("osascript");
    command.args([
        "-e",
        "on run argv",
        "-e",
        "tell application \"Finder\" to delete POSIX file (item 1 of argv)",
        "-e",
        "end run",
    ]);
    command.arg(path);
    command
}

#[cfg(target_os = "windows")]
fn trash_command(path: &Path) -> Command {
    let script = r#"
$path = $args[0]
if (Test-Path -LiteralPath $path -PathType Container) {
    [Microsoft.VisualBasic.FileIO.FileSystem]::DeleteDirectory(
        $path,
        [Microsoft.VisualBasic.FileIO.UIOption]::OnlyErrorDialogs,
        [Microsoft.VisualBasic.FileIO.RecycleOption]::SendToRecycleBin
    )
} else {
    [Microsoft.VisualBasic.FileIO.FileSystem]::DeleteFile(
        $path,
        [Microsoft.VisualBasic.FileIO.UIOption]::OnlyErrorDialogs,
        [Microsoft.VisualBasic.FileIO.RecycleOption]::SendToRecycleBin
    )
}
"#;
    let mut command = Command::new("powershell.exe");
    command.args(["-NoProfile", "-NonInteractive", "-Command", script]);
    command.arg(path);
    command
}

#[cfg(not(any(target_os = "linux", target_os = "macos", target_os = "windows")))]
fn trash_command(_path: &Path) -> Command {
    Command::new("prumo-trash-unsupported")
}

pub fn retarget_tabs(state: &mut AppState, previous: &Path, next: &Path) {
    let workspace_root = state.workspace_root.clone();
    for tab in state.editor.tabs.iter_mut() {
        if tab.path == previous {
            tab.path = next.to_path_buf();
            tab.title = next
                .file_name()
                .and_then(|name| name.to_str())
                .unwrap_or(&tab.title)
                .to_string();
            tab.rel_path = next
                .strip_prefix(&workspace_root)
                .map(|relative| relative.display().to_string())
                .unwrap_or_else(|_| tab.rel_path.clone());
        }
    }
    if state.selected_path.as_deref() == Some(previous) {
        state.selected_path = Some(next.to_path_buf());
    }
}

pub fn close_tabs_for_path(state: &mut AppState, removed: &Path) {
    let mut index = 0;
    while index < state.tabs.len() {
        if state.tabs[index].path == removed || state.tabs[index].path.starts_with(removed) {
            state.remove_tab(index);
        } else {
            index += 1;
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    #[test]
    fn file_operations_round_trip() {
        let directory = tempdir().unwrap();
        let root = directory.path();
        let file = create_file(root, "note.txt").unwrap();
        assert!(file.is_file());
        assert!(create_file(root, "note.txt").is_err());
        assert!(create_file(root, "a/b").is_err());
        let renamed = rename_path(&file, "renamed.txt").unwrap();
        assert!(renamed.is_file());
        assert!(!file.exists());
        let copy = duplicate_path(&renamed).unwrap();
        assert!(copy.is_file());
        assert_ne!(copy, renamed);
        delete_path(&renamed).unwrap();
        assert!(!renamed.exists());
        assert!(delete_path(&renamed).is_err());
    }

    #[test]
    fn trash_rejects_workspace_root_and_external_paths() {
        let workspace = tempdir().unwrap();
        let outside = tempdir().unwrap();
        assert!(move_to_trash(workspace.path(), workspace.path()).is_err());
        assert!(move_to_trash(workspace.path(), outside.path()).is_err());
    }

    #[cfg(target_os = "linux")]
    #[test]
    fn system_trash_command_uses_gio() {
        let command = trash_command(Path::new("/tmp/prumo-item"));
        assert_eq!(command.get_program(), "gio");
        assert_eq!(
            command.get_args().collect::<Vec<_>>(),
            vec!["trash", "--", "/tmp/prumo-item"]
        );
    }

    #[test]
    fn folder_operations_round_trip() {
        let directory = tempdir().unwrap();
        let root = directory.path();
        let folder = create_folder(root, "docs").unwrap();
        create_file(&folder, "inside.txt").unwrap();
        let copy = duplicate_path(&folder).unwrap();
        assert!(copy.join("inside.txt").is_file());
        delete_path(&folder).unwrap();
        assert!(!folder.exists());
        assert!(copy.exists());
    }
}
