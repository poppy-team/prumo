use crate::state::QuickOpenMatch;
use std::path::Path;

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
}
