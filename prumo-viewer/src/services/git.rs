use std::path::Path;
use std::process::Command;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GitFile {
    pub path: String,
    pub staged_kind: Option<char>,
    pub unstaged_kind: Option<char>,
}

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct GitStatus {
    pub is_repo: bool,
    pub branch: String,
    pub ahead: usize,
    pub behind: usize,
    pub files: Vec<GitFile>,
}

pub fn git_available() -> bool {
    Command::new("git")
        .arg("--version")
        .output()
        .map(|output| output.status.success())
        .unwrap_or(false)
}

fn git_output(root: &Path, args: &[&str]) -> Result<String, String> {
    let output = Command::new("git")
        .arg("-C")
        .arg(root)
        .args(args)
        .output()
        .map_err(|error| format!("Git is not available: {error}"))?;
    if output.status.success() {
        Ok(String::from_utf8_lossy(&output.stdout).to_string())
    } else {
        Err(String::from_utf8_lossy(&output.stderr).trim().to_string())
    }
}

fn parse_ahead_behind(header: &str) -> (usize, usize) {
    let mut ahead = 0;
    let mut behind = 0;
    for part in header.split(',') {
        let part = part.trim().trim_matches(&['[', ']'][..]);
        if let Some(count) = part.strip_prefix("ahead ") {
            ahead = count.parse().unwrap_or(0);
        }
        if let Some(count) = part.strip_prefix("behind ") {
            behind = count.parse().unwrap_or(0);
        }
    }
    (ahead, behind)
}

fn display_path(raw: &str) -> String {
    if let Some((_, new)) = raw.split_once(" -> ") {
        new.trim_matches('"').to_string()
    } else {
        raw.trim_matches('"').to_string()
    }
}

pub fn git_status(root: &Path) -> GitStatus {
    let empty = GitStatus::default();
    let Ok(output) = git_output(root, &["status", "--porcelain=v1", "-b", "-uall"]) else {
        return empty;
    };
    let mut lines = output.lines();
    let Some(header) = lines.next() else {
        return empty;
    };
    let header = header.strip_prefix("## ").unwrap_or(header);
    let branch = header
        .split(' ')
        .next()
        .unwrap_or("")
        .split("...")
        .next()
        .unwrap_or("")
        .to_string();
    let (ahead, behind) = parse_ahead_behind(header);
    let mut files = Vec::new();
    for line in lines {
        if line.len() < 4 {
            continue;
        }
        let staged = line.chars().next().unwrap_or(' ');
        let unstaged = line.chars().nth(1).unwrap_or(' ');
        files.push(GitFile {
            path: display_path(line[3..].trim()),
            staged_kind: (staged != ' ' && staged != '?').then_some(staged),
            unstaged_kind: (unstaged != ' ').then_some(unstaged),
        });
    }
    GitStatus {
        is_repo: true,
        branch,
        ahead,
        behind,
        files,
    }
}

pub fn parse_unified_diff(text: &str) -> Vec<(String, char)> {
    let mut lines = Vec::new();
    for line in text.lines() {
        if let Some(content) = line.strip_prefix('+') {
            if line.starts_with("+++") {
                continue;
            }
            lines.push((content.to_string(), '+'));
        } else if let Some(content) = line.strip_prefix('-') {
            if line.starts_with("---") {
                continue;
            }
            lines.push((content.to_string(), '-'));
        } else if line.starts_with("@@") {
            lines.push((line.to_string(), ' '));
        } else if let Some(content) = line.strip_prefix(' ') {
            lines.push((content.to_string(), ' '));
        } else if line.is_empty() {
            lines.push((String::new(), ' '));
        }
    }
    lines
}

pub fn validate_git_path(root: &Path, path: &str) -> Result<String, String> {
    if path.trim().is_empty() || Path::new(path).is_absolute() {
        return Err("Git path must be relative to the workspace".to_string());
    }
    let candidate = root.join(path);
    crate::services::workspace::validate_path_boundary(root, &candidate)?;
    Ok(path.to_string())
}

pub fn git_file_diff(root: &Path, path: &str, staged: bool) -> Result<Vec<(String, char)>, String> {
    let path = validate_git_path(root, path)?;
    let mut args = vec!["diff", "--no-color", "--"];
    if staged {
        args.insert(1, "--cached");
    }
    let mut full_args: Vec<&str> = args;
    full_args.push(&path);
    let output = git_output(root, &full_args)?;
    let lines = parse_unified_diff(&output);
    if !lines.is_empty() || !is_untracked_file(root, &path)? {
        return Ok(lines);
    }
    let content = std::fs::read_to_string(root.join(&path))
        .map_err(|error| format!("Could not read untracked file: {error}"))?;
    Ok(content
        .lines()
        .map(|line| (line.to_string(), '+'))
        .collect())
}

fn is_untracked_file(root: &Path, path: &str) -> Result<bool, String> {
    let output = git_output(root, &["ls-files", "--", path])?;
    Ok(output.trim().is_empty() && root.join(path).is_file())
}

pub fn git_changed_lines(
    root: &Path,
    path: &str,
    staged: bool,
) -> Result<Vec<(usize, char)>, String> {
    let path = validate_git_path(root, path)?;
    let mut args = vec!["diff", "--no-color", "--unified=0"];
    if staged {
        args.insert(1, "--cached");
    }
    args.extend(["--", &path]);
    let output = git_output(root, &args)?;
    let mut changed = Vec::new();
    for line in output.lines() {
        let Some(header) = line.strip_prefix("@@") else {
            continue;
        };
        let fields: Vec<&str> = header.split_whitespace().take(2).collect();
        if fields.len() < 2 {
            continue;
        }
        let Some(new_start) = parse_diff_start(fields[1]) else {
            continue;
        };
        let new_count = fields[1]
            .split_once(',')
            .map_or(1, |(_, count)| count.parse::<usize>().unwrap_or(1));
        let old_count = fields[0]
            .split_once(',')
            .map_or(1, |(_, count)| count.parse::<usize>().unwrap_or(1));
        let tag = if old_count == 0 {
            '+'
        } else if new_count == 0 {
            '-'
        } else {
            '~'
        };
        let count = new_count.max(usize::from(old_count > 0 && new_count == 0));
        for offset in 0..count {
            changed.push((new_start.max(1).saturating_add(offset), tag));
            if changed.len() >= 5000 {
                return Ok(changed);
            }
        }
    }
    if changed.is_empty() && is_untracked_file(root, &path)? {
        let content = std::fs::read_to_string(root.join(&path))
            .map_err(|error| format!("Could not read untracked file: {error}"))?;
        return Ok(content
            .lines()
            .enumerate()
            .take(5000)
            .map(|(line, _)| (line + 1, '+'))
            .collect());
    }
    Ok(changed)
}

fn parse_diff_start(value: &str) -> Option<usize> {
    let value = value.strip_prefix('+').unwrap_or(value);
    value
        .split_once(',')
        .map_or(value, |(start, _)| start)
        .parse()
        .ok()
}

pub fn git_stage(root: &Path, path: &str) -> Result<(), String> {
    let path = validate_git_path(root, path)?;
    git_output(root, &["add", "--", &path]).map(|_| ())
}

pub fn git_unstage(root: &Path, path: &str) -> Result<(), String> {
    let path = validate_git_path(root, path)?;
    git_output(root, &["restore", "--staged", "--", &path]).map(|_| ())
}

pub fn git_discard(root: &Path, path: &str, tracked: bool) -> Result<(), String> {
    let path = validate_git_path(root, path)?;
    if tracked {
        git_output(root, &["checkout", "--", &path]).map(|_| ())
    } else {
        let absolute = root.join(&path);
        if absolute.is_dir() {
            std::fs::remove_dir_all(&absolute).map_err(|error| format!("Could not delete: {error}"))
        } else {
            std::fs::remove_file(&absolute).map_err(|error| format!("Could not delete: {error}"))
        }
    }
}

pub fn git_commit(root: &Path, message: &str) -> Result<String, String> {
    let message = message.trim();
    if message.is_empty() {
        return Err("Commit message must not be empty".to_string());
    }
    git_output(root, &["commit", "-m", message])
}

pub fn git_push(root: &Path) -> Result<String, String> {
    git_output(root, &["push"])
}

pub fn git_pull(root: &Path) -> Result<String, String> {
    git_output(root, &["pull", "--ff-only"])
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::path::PathBuf;
    use tempfile::tempdir;

    fn init_repo() -> Option<(tempfile::TempDir, PathBuf)> {
        if !git_available() {
            return None;
        }
        let directory = tempdir().unwrap();
        let root = directory.path().to_path_buf();
        git_output(&root, &["init"]).ok()?;
        git_output(&root, &["config", "user.email", "test@prumo.dev"]).ok()?;
        git_output(&root, &["config", "user.name", "Prumo Test"]).ok()?;
        std::fs::write(root.join("tracked.txt"), "one\n").unwrap();
        git_output(&root, &["add", "."]).ok()?;
        git_output(&root, &["commit", "-m", "initial"]).ok()?;
        Some((directory, root))
    }

    #[test]
    fn git_status_stage_commit_round_trip() {
        let Some((_directory, root)) = init_repo() else {
            return;
        };
        let status = git_status(&root);
        assert!(status.is_repo);
        assert!(status.files.is_empty());
        std::fs::write(root.join("tracked.txt"), "one\ntwo\n").unwrap();
        std::fs::write(root.join("new.txt"), "new\n").unwrap();
        let status = git_status(&root);
        assert_eq!(status.files.len(), 2);
        git_stage(&root, "tracked.txt").unwrap();
        git_stage(&root, "new.txt").unwrap();
        let status = git_status(&root);
        assert!(status.files.iter().all(|file| file.staged_kind.is_some()));
        let message = git_commit(&root, "second").unwrap();
        assert!(message.contains("second") || message.contains("1 file"));
        assert!(git_status(&root).files.is_empty());
    }

    #[test]
    fn git_diff_unstage_discard_round_trip() {
        let Some((_directory, root)) = init_repo() else {
            return;
        };
        std::fs::write(root.join("tracked.txt"), "one\ntwo\n").unwrap();
        let diff = git_file_diff(&root, "tracked.txt", false).unwrap();
        assert!(diff.iter().any(|(line, tag)| *tag == '+' && line == "two"));
        git_stage(&root, "tracked.txt").unwrap();
        git_unstage(&root, "tracked.txt").unwrap();
        assert!(
            git_status(&root)
                .files
                .iter()
                .any(|file| { file.path == "tracked.txt" && file.unstaged_kind.is_some() })
        );
        git_discard(&root, "tracked.txt", true).unwrap();
        assert!(git_status(&root).files.is_empty());
    }

    #[test]
    fn reports_changed_lines_for_editor_preview() {
        let Some((_directory, root)) = init_repo() else {
            return;
        };
        std::fs::write(root.join("tracked.txt"), "one\ntwo\nthree\n").unwrap();
        let lines = git_changed_lines(&root, "tracked.txt", false).unwrap();
        assert!(lines.iter().any(|(line, _)| *line >= 2));
    }

    #[test]
    fn previews_deleted_lines_with_removal_tag() {
        let Some((_directory, root)) = init_repo() else {
            return;
        };
        std::fs::remove_file(root.join("tracked.txt")).unwrap();
        let lines = git_changed_lines(&root, "tracked.txt", false).unwrap();
        assert!(lines.iter().any(|(line, tag)| *line == 1 && *tag == '-'));
    }

    #[test]
    fn previews_untracked_files_as_additions() {
        let Some((_directory, root)) = init_repo() else {
            return;
        };
        std::fs::write(root.join("new.txt"), "one\ntwo\n").unwrap();
        let lines = git_changed_lines(&root, "new.txt", false).unwrap();
        assert_eq!(lines, vec![(1, '+'), (2, '+')]);
        let diff = git_file_diff(&root, "new.txt", false).unwrap();
        assert!(diff.iter().all(|(_, tag)| *tag == '+'));
        assert!(diff.iter().any(|(line, _)| line == "two"));
    }

    #[test]
    fn rejects_paths_outside_workspace() {
        let directory = tempdir().unwrap();
        assert!(validate_git_path(directory.path(), "../outside").is_err());
    }

    #[test]
    fn non_repository_reports_empty_status() {
        let directory = tempdir().unwrap();
        let status = git_status(directory.path());
        assert!(!status.is_repo);
    }
}
