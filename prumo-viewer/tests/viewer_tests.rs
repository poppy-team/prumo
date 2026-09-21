use std::fs;
use std::path::PathBuf;
use prumo_viewer::client::Event;
use prumo_viewer::config::{resolve_renderer, CliArgs, GraphicsConfig, GraphicsRenderer};
use prumo_viewer::diff::{DiffLineTag, DiffService, FileDiff, MergeResult};
use prumo_viewer::document::DocumentStore;
use prumo_viewer::follow::AgentFollowController;
use prumo_viewer::projection::{FileBadgeKind, RunFileProjection};
use prumo_viewer::workspace::CachedWorkspaceTree;
use tempfile::tempdir;

#[test]
fn test_renderer_selection_policy() {
    let mut config = GraphicsConfig::default();
    assert_eq!(config.renderer, GraphicsRenderer::Glow);

    // Default CLI args: resolves to Glow (OpenGL baseline)
    let cli_default = CliArgs {
        renderer_override: None,
        wgpu_backend_override: None,
        safe_graphics: false,
        socket_path: None,
        workspace_dir: PathBuf::from("."),
    };
    let renderer = resolve_renderer(&config, &cli_default);
    assert_eq!(renderer, eframe::Renderer::Glow);

    // Explicit --renderer wgpu
    let cli_wgpu = CliArgs {
        renderer_override: Some(GraphicsRenderer::Wgpu),
        wgpu_backend_override: None,
        safe_graphics: false,
        socket_path: None,
        workspace_dir: PathBuf::from("."),
    };
    let renderer_wgpu = resolve_renderer(&config, &cli_wgpu);
    assert_eq!(renderer_wgpu, eframe::Renderer::Wgpu);

    // Safe graphics flag forces Glow even if config says Wgpu
    config.renderer = GraphicsRenderer::Wgpu;
    let cli_safe = CliArgs {
        renderer_override: Some(GraphicsRenderer::Wgpu),
        wgpu_backend_override: None,
        safe_graphics: true,
        socket_path: None,
        workspace_dir: PathBuf::from("."),
    };
    let renderer_safe = resolve_renderer(&config, &cli_safe);
    assert_eq!(renderer_safe, eframe::Renderer::Glow);
}

#[test]
fn test_file_diff_computation() {
    let old_text = "fn main() {\n    println!(\"hello\");\n}\n";
    let new_text = "fn main() {\n    println!(\"hello world\");\n    println!(\"added\");\n}\n";

    let diff = FileDiff::compute(old_text, new_text);
    assert!(diff.additions > 0);
    assert!(diff.deletions > 0);
    assert!(diff.lines.iter().any(|l| l.tag == DiffLineTag::Insert && l.text.contains("hello world")));
    assert!(diff.lines.iter().any(|l| l.tag == DiffLineTag::Delete && l.text.contains("hello\");")));
}

#[test]
fn test_three_way_merge_clean_and_conflict() {
    let base = "line 1\nline 2\nline 3\n";

    // 1. Clean merge: only agent changed
    let ours = "line 1\nline 2\nline 3\n";
    let theirs = "line 1\nline 2\nline 3 modified\n";
    let res = DiffService::three_way_merge(base, ours, theirs);
    match res {
        MergeResult::Clean(merged) => {
            assert_eq!(merged, theirs);
        }
        _ => panic!("Expected clean merge when user buffer has no edits"),
    }

    // 2. Clean merge: only human changed
    let ours = "line 1 modified by human\nline 2\nline 3\n";
    let theirs = "line 1\nline 2\nline 3\n";
    let res = DiffService::three_way_merge(base, ours, theirs);
    match res {
        MergeResult::Clean(merged) => {
            assert_eq!(merged, ours);
        }
        _ => panic!("Expected clean merge when agent has no edits"),
    }

    // 3. Concurrent edits -> Conflict detection
    let ours = "line 1 edited by human\nline 2\nline 3\n";
    let theirs = "line 1 edited by agent\nline 2\nline 3\n";
    let res = DiffService::three_way_merge(base, ours, theirs);
    match res {
        MergeResult::Conflict { merged_with_markers, conflicts } => {
            assert!(merged_with_markers.contains("<<<<<<< HUMAN EDITS"));
            assert!(merged_with_markers.contains(">>>>>>> AGENT EDITS"));
            assert_eq!(conflicts.len(), 1);
        }
        _ => panic!("Expected conflict when both human and agent edit the same file"),
    }
}

#[test]
fn test_document_store_and_buffer_invariants() {
    let dir = tempdir().unwrap();
    let file_path = dir.path().join("test.txt");
    fs::write(&file_path, "initial content\nsecond line\n").unwrap();

    let mut store = DocumentStore::new();
    let doc = store.open_file(&file_path).unwrap();
    assert_eq!(doc.content, "initial content\nsecond line\n");
    assert_eq!(doc.lines_count(), 2);
    assert!(!doc.is_dirty);

    // Human makes an edit
    doc.set_content("initial content\nsecond line modified\n".to_string());
    assert!(doc.is_dirty);
    assert_eq!(doc.version, 2);

    // External change occurred on disk
    fs::write(&file_path, "external update from agent\n").unwrap();

    // Critical Invariant: reload_if_clean MUST NOT overwrite dirty human edits!
    let reloaded = store.reload_if_clean(&file_path).unwrap();
    assert!(!reloaded, "Dirty buffer must NOT be reloaded from disk!");
    let current_doc = store.get_document(&file_path).unwrap();
    assert!(current_doc.is_dirty);
    assert_eq!(current_doc.content, "initial content\nsecond line modified\n");

    // Once saved, clean document can be safely reloaded
    store.save_active().unwrap();
    assert!(!store.active_document().unwrap().is_dirty);
}

#[test]
fn test_run_file_projection_badges() {
    let dir = tempdir().unwrap();
    let root = dir.path();
    let file_a = root.join("src/main.rs");
    let file_b = root.join("src/lib.rs");

    let mut proj = RunFileProjection::new("run-123");

    // 1. Agent modifies file_a
    let ev1 = Event {
        id: "ev-1".to_string(),
        run_id: "run-123".to_string(),
        kind: "tool_call_ready".to_string(),
        payload: serde_json::json!({
            "name": "replace_file_content",
            "path": "src/main.rs",
            "line": 15,
        }),
    };
    proj.process_event(&ev1, root);
    assert_eq!(proj.currently_working_file, Some(file_a.clone()));
    assert_eq!(proj.get_badge(&file_a), Some(FileBadgeKind::Working));

    // 2. Tool completed -> badge becomes Modified
    let ev2 = Event {
        id: "ev-2".to_string(),
        run_id: "run-123".to_string(),
        kind: "tool_result".to_string(),
        payload: serde_json::json!({}),
    };
    proj.process_event(&ev2, root);
    assert_eq!(proj.currently_working_file, None);
    assert_eq!(proj.get_badge(&file_a), Some(FileBadgeKind::Modified));

    // 3. Agent creates file_b
    let ev3 = Event {
        id: "ev-3".to_string(),
        run_id: "run-123".to_string(),
        kind: "tool_call_ready".to_string(),
        payload: serde_json::json!({
            "name": "write_to_file",
            "path": "src/lib.rs",
        }),
    };
    proj.process_event(&ev3, root);
    assert_eq!(proj.currently_working_file, Some(file_b.clone()));

    let ev4 = Event {
        id: "ev-4".to_string(),
        run_id: "run-123".to_string(),
        kind: "tool_result".to_string(),
        payload: serde_json::json!({}),
    };
    proj.process_event(&ev4, root);
    assert_eq!(proj.get_badge(&file_b), Some(FileBadgeKind::Created));

    // 4. Verification passes on file_b
    let ev5 = Event {
        id: "ev-5".to_string(),
        run_id: "run-123".to_string(),
        kind: "verification.passed".to_string(),
        payload: serde_json::json!({ "path": "src/lib.rs" }),
    };
    proj.process_event(&ev5, root);
    assert_eq!(proj.get_badge(&file_b), Some(FileBadgeKind::Verified));
}

#[test]
fn test_agent_follow_controller_invariants() {
    let dir = tempdir().unwrap();
    let root = dir.path();
    let file = root.join("code.go");
    fs::write(&file, "package main\n\nfunc main() {\n}\n").unwrap();

    let mut store = DocumentStore::new();
    let mut follow = AgentFollowController::new();
    assert!(follow.enabled);

    // Event targets line 3
    let ev = Event {
        id: "ev-1".to_string(),
        run_id: "run-1".to_string(),
        kind: "tool_call_ready".to_string(),
        payload: serde_json::json!({
            "name": "replace_file_content",
            "path": "code.go",
            "line": 3,
            "end_line": 4,
        }),
    };

    // 1. With follow enabled: opens file, sets cursor and highlight range
    follow.handle_event(&ev, root, &mut store);
    assert_eq!(follow.last_followed_path, Some(file.clone()));
    let doc = store.active_document().unwrap();
    assert_eq!(doc.cursor_line, 3);
    assert_eq!(doc.highlight_range, Some((3, 4)));

    // 2. Invariant: While user is actively typing, follow must NOT steal focus
    follow.notify_user_edit_started();
    let ev2 = Event {
        id: "ev-2".to_string(),
        run_id: "run-1".to_string(),
        kind: "tool_call_ready".to_string(),
        payload: serde_json::json!({
            "name": "replace_file_content",
            "path": "other.go",
            "line": 10,
        }),
    };
    follow.handle_event(&ev2, root, &mut store);
    // Active document is still code.go, not other.go!
    assert_eq!(store.active_tab().unwrap(), &file.canonicalize().unwrap());

    // 3. Invariant: When follow is disabled, never touch navigation
    follow.notify_user_edit_idle();
    follow.enabled = false;
    follow.handle_event(&ev2, root, &mut store);
    assert_eq!(store.active_tab().unwrap(), &file.canonicalize().unwrap());
}

#[test]
fn test_cached_workspace_tree() {
    let dir = tempdir().unwrap();
    let root = dir.path();
    fs::create_dir_all(root.join("src")).unwrap();
    fs::create_dir_all(root.join(".git")).unwrap();
    fs::create_dir_all(root.join("target")).unwrap();
    fs::write(root.join("src/main.rs"), "fn main() {}").unwrap();
    fs::write(root.join("target/debug_file"), "binary").unwrap();

    let tree = CachedWorkspaceTree::new(root);
    assert_eq!(tree.root_node.children.len(), 1, "Ignored dirs like .git and target should be filtered");
    let src_node = &tree.root_node.children[0];
    assert_eq!(src_node.name, "src");
    assert!(src_node.is_dir);
    assert_eq!(src_node.children.len(), 1);
    assert_eq!(src_node.children[0].name, "main.rs");

    let all = tree.all_files();
    assert_eq!(all, vec!["src/main.rs"]);
}

#[test]
fn test_quick_open_fuzzy_matching() {
    use prumo_viewer::search::QuickOpenMatcher;

    let mut matcher = QuickOpenMatcher::new();
    let files = vec![
        "src/client.rs".to_string(),
        "src/main.rs".to_string(),
        "src/config.rs".to_string(),
        "docs/architecture.md".to_string(),
    ];

    // 1. Query "cli" matches "src/client.rs" as top candidate
    let matches = matcher.match_files("cli", &files);
    assert!(!matches.is_empty());
    assert_eq!(matches[0].0, "src/client.rs");

    // 2. Query "arch" matches "docs/architecture.md"
    let matches_arch = matcher.match_files("arch", &files);
    assert!(!matches_arch.is_empty());
    assert_eq!(matches_arch[0].0, "docs/architecture.md");

    // 3. Empty query returns initial files
    let matches_empty = matcher.match_files("", &files);
    assert_eq!(matches_empty.len(), 4);
}

#[test]
fn test_workspace_searcher() {
    use prumo_viewer::search::WorkspaceSearcher;

    let dir = tempdir().unwrap();
    let root = dir.path();
    fs::create_dir_all(root.join("src")).unwrap();
    fs::write(
        root.join("src/lib.rs"),
        "pub fn calculate_magic() -> i32 {\n    42 // the magic answer\n}\n",
    )
    .unwrap();
    fs::write(root.join("src/other.rs"), "pub fn irrelevant() {}\n").unwrap();

    let results = WorkspaceSearcher::search(root, "magic answer", 10);
    assert_eq!(results.len(), 1);
    assert_eq!(results[0].line_number, 2);
    assert!(results[0].line_text.contains("the magic answer"));
    assert_eq!(
        results[0].path.file_name().unwrap().to_str().unwrap(),
        "lib.rs"
    );
}

#[test]
fn test_workspace_boundary_rejection() {
    let ws_dir = tempdir().unwrap();
    let outside_dir = tempdir().unwrap();

    let inside_file = ws_dir.path().join("inside.txt");
    fs::write(&inside_file, "inside content").unwrap();

    let outside_file = outside_dir.path().join("outside.txt");
    fs::write(&outside_file, "secret outside content").unwrap();

    let mut store = DocumentStore::new();
    store.set_workspace_root(ws_dir.path().to_path_buf());

    // Inside file opens successfully
    assert!(store.open_file(&inside_file).is_ok());

    // Outside file must be rejected by workspace boundary check
    let err = store.open_file(&outside_file);
    assert!(err.is_err());
    let err_msg = err.unwrap_err().to_string();
    assert!(
        err_msg.contains("Path outside workspace rejected"),
        "Unexpected error: {}",
        err_msg
    );
}

#[test]
fn test_tab_preservation_on_rename() {
    let dir = tempdir().unwrap();
    let old_file = dir.path().join("old_name.txt");
    let new_file = dir.path().join("new_name.txt");
    fs::write(&old_file, "persisted content across rename").unwrap();

    let mut store = DocumentStore::new();
    store.set_workspace_root(dir.path().to_path_buf());
    let doc = store.open_file(&old_file).unwrap();
    doc.set_content("persisted content with dirty edits".to_string());
    assert!(doc.is_dirty);

    // Simulate disk rename + store rename
    fs::rename(&old_file, &new_file).unwrap();
    store.rename_file(&old_file, &new_file);

    // Assert old path is gone from tabs, new path is active and retains dirty content
    assert_eq!(store.open_tabs().len(), 1);
    let active_doc = store.active_document().unwrap();
    assert_eq!(active_doc.path, new_file.canonicalize().unwrap());
    assert!(active_doc.is_dirty);
    assert_eq!(active_doc.content, "persisted content with dirty edits");
}

#[test]
fn test_deleted_file_explicit_state() {
    let dir = tempdir().unwrap();
    let file = dir.path().join("doomed.txt");
    fs::write(&file, "hello").unwrap();

    let mut store = DocumentStore::new();
    store.open_file(&file).unwrap();

    let doc = store.active_document().unwrap();
    assert!(!doc.is_deleted_on_disk);

    // Mark deleted
    store.mark_deleted(&file);
    let doc_after = store.active_document().unwrap();
    assert!(doc_after.is_deleted_on_disk);
}

#[test]
fn test_large_diff_truncation() {
    // Generate over 8,000 lines
    let mut old_text = String::new();
    let mut new_text = String::new();
    for i in 0..8500 {
        old_text.push_str(&format!("old line {}\n", i));
        new_text.push_str(&format!("new line {}\n", i));
    }

    let diff = FileDiff::compute(&old_text, &new_text);
    assert!(diff.is_truncated);
    assert!(diff.lines.len() <= 8001);
}

#[test]
fn test_session_persistence() {
    let dir = tempdir().unwrap();
    let f1 = dir.path().join("doc1.txt");
    let f2 = dir.path().join("doc2.txt");
    fs::write(&f1, "1").unwrap();
    fs::write(&f2, "2").unwrap();

    let mut store = DocumentStore::new();
    store.open_file(&f1).unwrap();
    store.open_file(&f2).unwrap();

    let session_file = dir.path().join(".prumo/session.json");
    store.save_session(&session_file).unwrap();
    assert!(session_file.exists());

    let mut new_store = DocumentStore::new();
    let _ = new_store.load_session(&session_file);
    assert_eq!(new_store.open_tabs().len(), 2);
    assert_eq!(
        new_store.active_tab().unwrap(),
        &f2.canonicalize().unwrap()
    );
}

#[test]
fn test_git_status_discovery() {
    use git2::Repository;
    use prumo_viewer::git::GitService;

    let dir = tempdir().unwrap();
    let _repo = Repository::init(dir.path()).unwrap();

    // In an empty repo, HEAD may not be set yet, status should still succeed safely
    let status_opt = GitService::discover_status(dir.path());
    assert!(status_opt.is_some());
    let status = status_opt.unwrap();
    assert_eq!(status.unstaged_count, 0);

    // Add untracked file
    fs::write(dir.path().join("untracked.txt"), "hello").unwrap();
    let status2 = GitService::discover_status(dir.path()).unwrap();
    assert_eq!(status2.untracked_count, 1);
}


