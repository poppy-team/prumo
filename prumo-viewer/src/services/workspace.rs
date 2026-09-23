use std::collections::HashMap;
use std::fs;
use std::path::{Path, PathBuf};

use crate::state::{AgentFileStatus, AppState, FileKind, FlattenedTreeItem, TreeNode};

const IGNORED_DIRS: &[&str] = &[
    ".git",
    ".prumo",
    "target",
    "node_modules",
    ".cache",
    "dist",
    "build",
    "__pycache__",
    ".venv",
];

pub fn validate_path_boundary(root: &Path, target: &Path) -> Result<PathBuf, String> {
    let canonical_root = root
        .canonicalize()
        .map_err(|error| format!("Invalid workspace root {}: {error}", root.display()))?;
    let canonical_target = if target.exists() {
        target
            .canonicalize()
            .map_err(|error| format!("Invalid target path {}: {error}", target.display()))?
    } else {
        let parent = target.parent().unwrap_or(root);
        let canonical_parent = parent
            .canonicalize()
            .map_err(|error| format!("Invalid parent directory {}: {error}", parent.display()))?;
        if !canonical_parent.starts_with(&canonical_root) {
            return Err(format!(
                "Security violation: path {} escapes workspace boundary {}",
                target.display(),
                root.display()
            ));
        }
        canonical_parent.join(target.file_name().unwrap_or_default())
    };

    if canonical_target.starts_with(&canonical_root) {
        Ok(canonical_target)
    } else {
        Err(format!(
            "Security violation: path {} escapes workspace boundary {}",
            target.display(),
            root.display()
        ))
    }
}

pub fn scan_workspace(root: &Path) -> Result<TreeNode, std::io::Error> {
    let name = root
        .file_name()
        .and_then(|name| name.to_str())
        .unwrap_or("workspace")
        .to_string();
    let mut root_node = TreeNode {
        name,
        path: root.to_path_buf(),
        rel_path: String::new(),
        is_dir: true,
        kind: FileKind::Folder,
        children: Vec::new(),
        expanded: true,
        agent_status: None,
    };

    if root.is_dir() {
        build_subtree(&mut root_node, root, root)?;
    }
    Ok(root_node)
}

pub fn refresh_workspace(state: &mut AppState) -> Result<(), String> {
    let mut tree = scan_workspace(&state.workspace_root)
        .map_err(|error| format!("Could not refresh workspace: {error}"))?;
    let previous = tree_metadata(&state.tree);
    apply_tree_metadata(&mut tree, &previous);
    let candidates = collect_file_paths(&tree);
    state.tree = tree;
    state.flattened_tree = flatten_tree(&state.tree);
    state.file_candidates = candidates;
    state.clear_notice();
    Ok(())
}

pub fn flatten_tree(root: &TreeNode) -> Vec<FlattenedTreeItem> {
    let mut result = Vec::new();
    for child in &root.children {
        append_flattened(child, 0, &mut result);
    }
    result
}

pub fn toggle_folder(node: &mut TreeNode, target_path: &Path) -> bool {
    if node.is_dir && node.path == target_path {
        node.expanded = !node.expanded;
        return true;
    }
    node.children
        .iter_mut()
        .any(|child| child.is_dir && toggle_folder(child, target_path))
}

fn build_subtree(
    node: &mut TreeNode,
    current_dir: &Path,
    root: &Path,
) -> Result<(), std::io::Error> {
    let entries = match fs::read_dir(current_dir) {
        Ok(entries) => entries,
        Err(_) => return Ok(()),
    };
    let mut directories = Vec::new();
    let mut files = Vec::new();

    for entry in entries.flatten() {
        let path = entry.path();
        let file_name = entry.file_name().to_string_lossy().to_string();
        if file_name.starts_with('.') && file_name != ".gitignore" {
            continue;
        }
        if path.is_dir() && IGNORED_DIRS.contains(&file_name.as_str()) {
            continue;
        }
        let rel_path = path
            .strip_prefix(root)
            .unwrap_or(&path)
            .to_string_lossy()
            .to_string();

        if path.is_dir() {
            let mut directory = TreeNode {
                name: file_name,
                path: path.clone(),
                rel_path,
                is_dir: true,
                kind: FileKind::Folder,
                children: Vec::new(),
                expanded: false,
                agent_status: None,
            };
            build_subtree(&mut directory, &path, root)?;
            directories.push(directory);
        } else {
            files.push(TreeNode {
                name: file_name,
                path: path.clone(),
                rel_path,
                is_dir: false,
                kind: FileKind::from_path(&path),
                children: Vec::new(),
                expanded: false,
                agent_status: None,
            });
        }
    }

    directories.sort_by_key(|entry| entry.name.to_lowercase());
    files.sort_by_key(|entry| entry.name.to_lowercase());
    node.children.extend(directories);
    node.children.extend(files);
    Ok(())
}

fn append_flattened(node: &TreeNode, depth: usize, output: &mut Vec<FlattenedTreeItem>) {
    output.push(FlattenedTreeItem {
        path: node.path.clone(),
        rel_path: node.rel_path.clone(),
        name: node.name.clone(),
        is_dir: node.is_dir,
        depth,
        expanded: node.expanded,
        kind: node.kind,
        agent_status: node.agent_status,
    });
    if node.is_dir && node.expanded {
        for child in &node.children {
            append_flattened(child, depth + 1, output);
        }
    }
}

fn collect_file_paths(node: &TreeNode) -> Vec<String> {
    let mut files = Vec::new();
    collect_file_paths_into(node, &mut files);
    files.sort();
    files
}

fn collect_file_paths_into(node: &TreeNode, output: &mut Vec<String>) {
    if !node.is_dir && !node.rel_path.is_empty() {
        output.push(node.rel_path.clone());
    }
    for child in &node.children {
        collect_file_paths_into(child, output);
    }
}

fn tree_metadata(node: &TreeNode) -> HashMap<PathBuf, (bool, Option<AgentFileStatus>)> {
    let mut metadata = HashMap::new();
    collect_tree_metadata(node, &mut metadata);
    metadata
}

fn collect_tree_metadata(
    node: &TreeNode,
    output: &mut HashMap<PathBuf, (bool, Option<AgentFileStatus>)>,
) {
    output.insert(node.path.clone(), (node.expanded, node.agent_status));
    for child in &node.children {
        collect_tree_metadata(child, output);
    }
}

fn apply_tree_metadata(
    node: &mut TreeNode,
    metadata: &HashMap<PathBuf, (bool, Option<AgentFileStatus>)>,
) {
    if let Some((expanded, agent_status)) = metadata.get(&node.path) {
        node.expanded = *expanded;
        node.agent_status = *agent_status;
    }
    for child in &mut node.children {
        apply_tree_metadata(child, metadata);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    #[test]
    fn validates_path_boundary() {
        let root_directory = tempdir().unwrap();
        let root = root_directory.path();
        let inside = root.join("src/main.rs");
        fs::create_dir_all(inside.parent().unwrap()).unwrap();
        fs::write(&inside, "fn main() {}").unwrap();
        let outside_directory = tempdir().unwrap();
        let outside = outside_directory.path().join("secret.txt");
        fs::write(&outside, "secret").unwrap();

        assert!(validate_path_boundary(root, &inside).is_ok());
        assert!(validate_path_boundary(root, &outside).is_err());
    }

    #[test]
    fn scans_and_flattens_tree() {
        let directory = tempdir().unwrap();
        let root = directory.path();
        fs::create_dir_all(root.join("src")).unwrap();
        fs::create_dir_all(root.join(".git")).unwrap();
        fs::write(root.join("src/lib.rs"), "// lib").unwrap();
        fs::write(root.join("README.md"), "# Readme").unwrap();

        let mut tree = scan_workspace(root).unwrap();
        assert_eq!(tree.children.len(), 2);
        assert_eq!(flatten_tree(&tree).len(), 2);
        assert!(toggle_folder(&mut tree, &root.join("src")));
        assert_eq!(flatten_tree(&tree).len(), 3);
    }

    #[test]
    fn refresh_preserves_expanded_directories() {
        let directory = tempdir().unwrap();
        let root = directory.path();
        fs::create_dir_all(root.join("src")).unwrap();
        fs::write(root.join("src/lib.rs"), "// lib").unwrap();
        let mut state = AppState::new(root.to_path_buf());
        refresh_workspace(&mut state).unwrap();
        assert!(toggle_folder(&mut state.tree, &root.join("src")));

        fs::write(root.join("src/new.rs"), "// new").unwrap();
        refresh_workspace(&mut state).unwrap();
        assert!(
            state
                .flattened_tree
                .iter()
                .any(|item| item.rel_path == "src/new.rs")
        );
    }
}
