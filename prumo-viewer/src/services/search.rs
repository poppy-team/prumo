use crate::state::QuickOpenMatch;
use regex::RegexBuilder;
use std::path::{Path, PathBuf};

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct FileMatch {
    pub line: usize,
    pub start: usize,
    pub end: usize,
    pub preview: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TextMatch {
    pub path: PathBuf,
    pub rel_path: String,
    pub line: usize,
    pub start: usize,
    pub end: usize,
    pub preview: String,
}

fn search_pattern(query: &str, match_case: bool, use_regex: bool) -> Result<regex::Regex, String> {
    if query.is_empty() {
        return Err("Empty search".to_string());
    }
    let pattern = if use_regex {
        query.to_string()
    } else {
        regex::escape(query)
    };
    RegexBuilder::new(&pattern)
        .case_insensitive(!match_case)
        .build()
        .map_err(|error| format!("Invalid pattern: {error}"))
}

pub fn find_in_text(content: &str, query: &str, match_case: bool) -> Vec<FileMatch> {
    let Ok(pattern) = search_pattern(query, match_case, false) else {
        return Vec::new();
    };
    let mut matches = Vec::new();
    let mut offset = 0;
    for (line_number, line) in content.split('\n').enumerate() {
        for found in pattern.find_iter(line) {
            matches.push(FileMatch {
                line: line_number + 1,
                start: offset + found.start(),
                end: offset + found.end(),
                preview: line.trim().to_string(),
            });
        }
        offset += line.len() + 1;
    }
    matches
}

fn content_is_binary(sample: &[u8]) -> bool {
    sample.iter().take(8192).any(|byte| *byte == 0)
}

pub fn search_workspace_text(
    root: &Path,
    query: &str,
    match_case: bool,
    use_regex: bool,
    limit: usize,
) -> Result<Vec<TextMatch>, String> {
    let pattern = search_pattern(query, match_case, use_regex)?;
    let walker = ignore::WalkBuilder::new(root)
        .hidden(false)
        .git_ignore(true)
        .git_exclude(true)
        .max_filesize(Some(2 * 1024 * 1024))
        .build();
    let mut matches = Vec::new();
    for entry in walker {
        let entry = entry.map_err(|error| format!("Search failed: {error}"))?;
        if !entry
            .file_type()
            .map(|kind| kind.is_file())
            .unwrap_or(false)
        {
            continue;
        }
        let path = entry.path();
        let content = match std::fs::read(path) {
            Ok(content) => content,
            Err(_) => continue,
        };
        if content_is_binary(&content) {
            continue;
        }
        let text = String::from_utf8_lossy(&content);
        let mut line_offset = 0;
        for (line_number, line) in text.split('\n').enumerate() {
            if matches.len() >= limit {
                break;
            }
            if let Some(found) = pattern.find(line) {
                matches.push(TextMatch {
                    path: path.to_path_buf(),
                    rel_path: path
                        .strip_prefix(root)
                        .map(|relative| relative.display().to_string())
                        .unwrap_or_else(|_| path.display().to_string()),
                    line: line_number + 1,
                    start: line_offset + found.start(),
                    end: line_offset + found.end(),
                    preview: line.trim().chars().take(160).collect(),
                });
            }
            line_offset += line.len() + 1;
        }
        if matches.len() >= limit {
            break;
        }
    }
    Ok(matches)
}

pub fn replace_workspace_text(
    root: &Path,
    query: &str,
    replacement: &str,
    match_case: bool,
    use_regex: bool,
) -> Result<(usize, usize), String> {
    let pattern = search_pattern(query, match_case, use_regex)?;
    let walker = ignore::WalkBuilder::new(root)
        .hidden(false)
        .git_ignore(true)
        .git_exclude(true)
        .max_filesize(Some(2 * 1024 * 1024))
        .build();
    let mut changed_files = 0;
    let mut replaced_matches = 0;
    for entry in walker {
        let entry = entry.map_err(|error| format!("Search failed: {error}"))?;
        if !entry
            .file_type()
            .map(|kind| kind.is_file())
            .unwrap_or(false)
        {
            continue;
        }
        let path = entry.path();
        let content = match std::fs::read(path) {
            Ok(content) => content,
            Err(_) => continue,
        };
        if content_is_binary(&content) {
            continue;
        }
        let text = String::from_utf8_lossy(&content);
        let count = pattern.find_iter(&text).count();
        if count == 0 {
            continue;
        }
        let next = pattern.replace_all(&text, replacement).into_owned();
        std::fs::write(path, next).map_err(|error| format!("Replace failed: {error}"))?;
        changed_files += 1;
        replaced_matches += count;
    }
    Ok((replaced_matches, changed_files))
}

/// Performs fuzzy matching across candidate file paths with scoring.
pub fn fuzzy_search(
    root: &Path,
    query: &str,
    candidates: &[String],
    limit: usize,
) -> Vec<QuickOpenMatch> {
    let clean_query = query.trim().to_lowercase();

    if clean_query.is_empty() {
        return candidates
            .iter()
            .take(limit)
            .map(|rel| QuickOpenMatch {
                path: root.join(rel),
                rel_path: rel.clone(),
                score: 0,
            })
            .collect();
    }

    let mut matches: Vec<QuickOpenMatch> = Vec::new();

    for rel in candidates {
        if let Some(score) = score_match(&clean_query, rel) {
            matches.push(QuickOpenMatch {
                path: root.join(rel),
                rel_path: rel.clone(),
                score,
            });
        }
    }

    // Sort descending by score, then ascending by length
    matches.sort_by(|a, b| {
        b.score
            .cmp(&a.score)
            .then_with(|| a.rel_path.len().cmp(&b.rel_path.len()))
            .then_with(|| a.rel_path.cmp(&b.rel_path))
    });

    matches.truncate(limit);
    matches
}

fn score_match(query: &str, candidate: &str) -> Option<i32> {
    let cand_lower = candidate.to_lowercase();
    let file_name = candidate
        .rsplit('/')
        .next()
        .unwrap_or(candidate)
        .to_lowercase();

    let mut score = 0;
    let mut query_chars = query.chars().peekable();
    let mut last_match_idx: Option<usize> = None;
    let mut matched_in_filename = false;

    // Check if filename contains query as direct substring
    if file_name.contains(query) {
        score += 50;
        matched_in_filename = true;
    }

    // Sequence match
    let mut cand_chars = cand_lower.char_indices();
    while let Some(&qc) = query_chars.peek() {
        let mut found = false;
        for (idx, cc) in cand_chars.by_ref() {
            if qc == cc {
                found = true;
                query_chars.next();

                // Consecutive match bonus
                if let Some(last) = last_match_idx
                    && idx == last + 1
                {
                    score += 15;
                }

                // Word boundary bonus (after slash or underscore or dot)
                if idx == 0
                    || cand_lower.as_bytes().get(idx.saturating_sub(1)).copied() == Some(b'/')
                    || cand_lower.as_bytes().get(idx.saturating_sub(1)).copied() == Some(b'_')
                    || cand_lower.as_bytes().get(idx.saturating_sub(1)).copied() == Some(b'-')
                {
                    score += 20;
                }

                last_match_idx = Some(idx);
                score += 5;
                break;
            }
        }

        if !found {
            return None; // Could not match full query sequence
        }
    }

    // Bonus for matching inside filename
    if matched_in_filename {
        score += 30;
    }

    // Penalty for length
    score -= (candidate.len() / 5) as i32;

    Some(score)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::path::PathBuf;

    #[test]
    fn test_fuzzy_matching() {
        let root = PathBuf::from("/workspace");
        let candidates = vec![
            "src/main.rs".to_string(),
            "src/services/workspace.rs".to_string(),
            "src/services/document.rs".to_string(),
            "README.md".to_string(),
            "Cargo.toml".to_string(),
        ];

        // "main" matches "src/main.rs" best
        let results = fuzzy_search(&root, "main", &candidates, 10);
        assert!(!results.is_empty());
        assert_eq!(results[0].rel_path, "src/main.rs");

        // "doc" matches "src/services/document.rs"
        let doc_results = fuzzy_search(&root, "doc", &candidates, 10);
        assert!(!doc_results.is_empty());
        assert_eq!(doc_results[0].rel_path, "src/services/document.rs");

        // Empty query returns all candidates
        let empty_results = fuzzy_search(&root, "", &candidates, 10);
        assert_eq!(empty_results.len(), 5);

        // Non-matching query returns nothing
        let none_results = fuzzy_search(&root, "xyz123", &candidates, 10);
        assert!(none_results.is_empty());
    }

    #[test]
    fn find_in_text_reports_line_byte_ranges() {
        let content = "fn main() {}\nfn helper() {}\n";
        let matches = find_in_text(content, "fn", false);
        assert_eq!(matches.len(), 2);
        assert_eq!(matches[0].line, 1);
        assert_eq!(matches[1].line, 2);
        assert_eq!(&content[matches[0].start..matches[0].end], "fn");
        let cased = find_in_text(content, "FN", true);
        assert!(cased.is_empty());
        let insensitive = find_in_text(content, "FN", false);
        assert_eq!(insensitive.len(), 2);
    }

    #[test]
    fn workspace_text_search_and_replace_round_trip() {
        let directory = tempfile::tempdir().unwrap();
        let root = directory.path();
        std::fs::write(root.join("one.txt"), "hello world\nhello again\n").unwrap();
        std::fs::write(root.join("two.txt"), "nothing here\n").unwrap();
        let matches = search_workspace_text(root, "hello", false, false, 100).unwrap();
        assert_eq!(matches.len(), 2);
        assert_eq!(matches[0].line, 1);
        assert_eq!((matches[0].start, matches[0].end), (0, 5));
        let (replaced, files) = replace_workspace_text(root, "hello", "hi", false, false).unwrap();
        assert_eq!((replaced, files), (2, 1));
        let after = std::fs::read_to_string(root.join("one.txt")).unwrap();
        assert_eq!(after, "hi world\nhi again\n");
    }
}
