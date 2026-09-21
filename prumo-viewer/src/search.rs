use std::fs;
use std::path::{Path, PathBuf};
use nucleo_matcher::{Config, Matcher, Utf32Str};
use walkdir::WalkDir;

pub struct QuickOpenMatcher {
    matcher: Matcher,
}

impl Default for QuickOpenMatcher {
    fn default() -> Self {
        Self {
            matcher: Matcher::new(Config::DEFAULT),
        }
    }
}

impl QuickOpenMatcher {
    pub fn new() -> Self {
        Self::default()
    }

    /// Fuzzy-matches workspace files against a query and returns top matches sorted by score.
    pub fn match_files(&mut self, query: &str, files: &[String]) -> Vec<(String, u16)> {
        if query.trim().is_empty() {
            return files.iter().take(20).map(|f| (f.clone(), 1u16)).collect();
        }

        let mut results = Vec::new();
        let mut query_buf = Vec::new();
        let needle = Utf32Str::new(query, &mut query_buf);

        for file in files {
            let mut haystack_buf = Vec::new();
            let haystack = Utf32Str::new(file, &mut haystack_buf);
            if let Some(score) = self.matcher.fuzzy_match(haystack, needle) {
                results.push((file.clone(), score));
            }
        }

        results.sort_by(|a, b| b.1.cmp(&a.1));
        results.truncate(30);
        results
    }
}

#[derive(Debug, Clone)]
pub struct SearchMatch {
    pub path: PathBuf,
    pub line_number: usize,
    pub line_text: String,
}

pub struct WorkspaceSearcher;

impl WorkspaceSearcher {
    /// Searches for a text pattern across readable text files in the workspace.
    pub fn search(root: &Path, query: &str, max_results: usize) -> Vec<SearchMatch> {
        let mut results = Vec::new();
        if query.trim().is_empty() {
            return results;
        }

        let query_lower = query.to_lowercase();

        for entry in WalkDir::new(root).into_iter().filter_map(|e| e.ok()) {
            let path = entry.path();
            if entry.file_type().is_dir() {
                let name = entry.file_name().to_string_lossy();
                if name.starts_with('.') && name != ".prumo" {
                    continue;
                }
                if name == "target" || name == "node_modules" || name == "vendor" {
                    continue;
                }
            }

            if entry.file_type().is_file() {
                let ext = path.extension().and_then(|s| s.to_str()).unwrap_or("");
                // Only scan code / text files
                if matches!(ext, "rs" | "go" | "ts" | "js" | "json" | "md" | "toml" | "yaml" | "yml" | "sh" | "c" | "h" | "cpp") {
                    if let Ok(content) = fs::read_to_string(path) {
                        for (idx, line) in content.lines().enumerate() {
                            if line.to_lowercase().contains(&query_lower) {
                                results.push(SearchMatch {
                                    path: path.to_path_buf(),
                                    line_number: idx + 1,
                                    line_text: line.trim().to_string(),
                                });
                                if results.len() >= max_results {
                                    return results;
                                }
                            }
                        }
                    }
                }
            }
        }

        results
    }
}
