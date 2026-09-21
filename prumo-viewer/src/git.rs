use std::path::Path;
use git2::{Repository, StatusOptions};

#[derive(Debug, Clone, Default)]
pub struct GitStatusSnapshot {
    pub branch: String,
    pub is_dirty: bool,
    pub staged_count: usize,
    pub unstaged_count: usize,
    pub untracked_count: usize,
}

pub struct GitService;

impl GitService {
    /// Discovers Git status for the given workspace path.
    /// Invariant: Returns quickly with safe fallback if not a Git repository.
    pub fn discover_status(workspace_root: &Path) -> Option<GitStatusSnapshot> {
        let repo = Repository::discover(workspace_root).ok()?;

        let head = repo.head().ok();
        let branch = head
            .as_ref()
            .and_then(|h| h.shorthand().ok())
            .unwrap_or("HEAD")
            .to_string();

        let mut opts = StatusOptions::new();
        opts.include_untracked(true).recurse_untracked_dirs(false);

        let statuses = repo.statuses(Some(&mut opts)).ok()?;

        let mut staged_count = 0;
        let mut unstaged_count = 0;
        let mut untracked_count = 0;

        for entry in statuses.iter() {
            let s = entry.status();
            if s.is_index_new() || s.is_index_modified() || s.is_index_deleted() || s.is_index_renamed() {
                staged_count += 1;
            }
            if s.is_wt_modified() || s.is_wt_deleted() || s.is_wt_renamed() {
                unstaged_count += 1;
            }
            if s.is_wt_new() {
                untracked_count += 1;
            }
        }

        let is_dirty = staged_count > 0 || unstaged_count > 0;

        Some(GitStatusSnapshot {
            branch,
            is_dirty,
            staged_count,
            unstaged_count,
            untracked_count,
        })
    }
}
