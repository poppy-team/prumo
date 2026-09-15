package docpublish

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/raillen/prumo/internal/harness/doccompile"
)

// BuildResult reports what a build produced.
type BuildResult struct {
	OutDir    string   `json:"out_dir"`
	Renderers []string `json:"renderers"`
	Pages     int      `json:"pages"`
	Artifacts int      `json:"artifacts"`
	Written   []string `json:"written"`
	Unchanged []string `json:"unchanged"`
}

// BuildGraph loads the graph and writes every selected renderer deterministically
// under outDir. Renderer names are validated: an unknown renderer is an error,
// never a silent skip (W18.1).
func BuildGraph(root, outDir string, opts Options, selected []string) (BuildResult, error) {
	if outDir == "" {
		outDir = DefaultOutDir
	}
	graph, err := LoadWithOptions(root, opts)
	if err != nil {
		return BuildResult{}, err
	}
	registry := Renderers()
	names := selected
	if len(names) == 0 {
		names = make([]string, 0, len(registry))
		for name := range registry {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	result := BuildResult{OutDir: outDir, Renderers: names, Pages: len(graph.Pages),
		Written: []string{}, Unchanged: []string{}}
	for _, name := range names {
		renderer, ok := registry[name]
		if !ok {
			return BuildResult{}, fmt.Errorf("unknown documentation renderer: %s", name)
		}
		artifacts, err := renderer.Render(graph)
		if err != nil {
			return BuildResult{}, err
		}
		for _, artifact := range artifacts {
			rel := filepath.ToSlash(filepath.Join(outDir, artifact.Path))
			abs := filepath.Join(root, filepath.FromSlash(rel))
			result.Artifacts++
			if existing, err := os.ReadFile(abs); err == nil && string(existing) == artifact.Content {
				result.Unchanged = append(result.Unchanged, rel)
				continue
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return BuildResult{}, err
			}
			if _, err := doccompile.Write(abs, artifact.Content); err != nil {
				return BuildResult{}, err
			}
			result.Written = append(result.Written, rel)
		}
	}
	data, err := json.MarshalIndent(map[string]any{
		"version": graph.Version, "locale": graph.Locale, "versioned_routes": graph.VersionedRoutes,
		"pages": len(graph.Pages), "renderers": names, "routes": graph.Routes(),
	}, "", "  ")
	if err != nil {
		return BuildResult{}, err
	}
	manifestRel := filepath.ToSlash(filepath.Join(outDir, "manifest.json"))
	manifestContent := string(append(data, '\n'))
	manifestAbs := filepath.Join(root, filepath.FromSlash(manifestRel))
	// The manifest is compared like any other artifact: writing it
	// unconditionally would make every build over a large repository report
	// churn it did not produce, and the written set is the signal callers use
	// to decide what actually changed (W21.11).
	if existing, err := os.ReadFile(manifestAbs); err == nil && string(existing) == manifestContent {
		result.Unchanged = append(result.Unchanged, manifestRel)
	} else {
		if _, err := doccompile.Write(manifestAbs, manifestContent); err != nil {
			return BuildResult{}, err
		}
		result.Written = append(result.Written, manifestRel)
	}
	sort.Strings(result.Written)
	sort.Strings(result.Unchanged)
	return result, nil
}

// SearchResult is one ranked query hit.
type SearchResult struct {
	Route   string `json:"route"`
	Title   string `json:"title"`
	Heading string `json:"heading,omitempty"`
	Source  string `json:"source"`
	Score   int    `json:"score"`
	Excerpt string `json:"excerpt"`
}

// Query ranks search entries for a term. Scoring is deterministic term overlap:
// no embeddings and no model are required for the read surface to work.
func Query(root, term string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}
	indexPath := filepath.Join(root, filepath.FromSlash(DefaultOutDir), "search-index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, err
	}
	var index struct {
		Entries []SearchEntry `json:"entries"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}
	terms := strings.Fields(strings.ToLower(term))
	if len(terms) == 0 {
		return nil, nil
	}
	results := []SearchResult{}
	for _, entry := range index.Entries {
		haystack := strings.ToLower(entry.Title + " " + entry.Heading + " " + entry.Text)
		score := 0
		for _, t := range terms {
			score += strings.Count(haystack, t)
		}
		if score == 0 {
			continue
		}
		results = append(results, SearchResult{Route: entry.Route, Title: entry.Title, Heading: entry.Heading,
			Source: entry.Source, Score: score, Excerpt: excerpt(entry.Text, terms)})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].Route != results[j].Route {
			return results[i].Route < results[j].Route
		}
		return results[i].Heading < results[j].Heading
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func excerpt(text string, terms []string) string {
	lower := strings.ToLower(text)
	at := -1
	for _, t := range terms {
		if i := strings.Index(lower, t); i >= 0 && (at < 0 || i < at) {
			at = i
		}
	}
	if at < 0 {
		at = 0
	}
	start := at - 120
	if start < 0 {
		start = 0
	}
	end := start + 240
	if end > len(text) {
		end = len(text)
	}
	out := strings.TrimSpace(text[start:end])
	if start > 0 {
		out = "…" + out
	}
	if end < len(text) {
		out += "…"
	}
	return out
}
