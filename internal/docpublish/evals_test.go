package docpublish

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type reconstructionSource struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type reconstructionAuthority struct {
	Path string `json:"path"`
	Role string `json:"role"`
}

type reconstructionCase struct {
	ID                      string                    `json:"id"`
	Kind                    string                    `json:"kind"`
	Title                   string                    `json:"title"`
	Sources                 []reconstructionSource    `json:"sources"`
	AuthorityMap            []reconstructionAuthority `json:"authority_map"`
	ExpectPages             int                       `json:"expect_pages"`
	ExpectRoutes            []string                  `json:"expect_routes"`
	ExpectTitle             string                    `json:"expect_title"`
	ExpectAudience          map[string]string         `json:"expect_audience"`
	ExpectHumanPages        []string                  `json:"expect_human_pages"`
	ExpectHistorical        []string                  `json:"expect_historical"`
	ExpectCurrentPages      []string                  `json:"expect_current_pages"`
	ExpectMarkdownRoundtrip *bool                     `json:"expect_markdown_roundtrip"`
	ExpectIdempotent        *bool                     `json:"expect_idempotent"`
}

// TestReconstructionEvalCorpus reconstructs documentation from its canonical
// sources and pins the properties that make the reconstruction trustworthy:
// the projection is lossless, routes are derived from paths rather than titles,
// and separation rules hold (W13.4).
func TestReconstructionEvalCorpus(t *testing.T) {
	cases := loadReconstructionCases(t)
	if len(cases) < 3 {
		t.Fatalf("reconstruction corpus is unexpectedly small: %d cases", len(cases))
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			if tc.Kind != "reconstruction" {
				t.Fatalf("case %s: unknown kind %q", tc.ID, tc.Kind)
			}
			root := writeReconstructionFixture(t, tc)
			graph, err := LoadWithOptions(root, Options{Locale: "en"})
			if err != nil {
				t.Fatal(err)
			}
			if len(graph.Pages) != tc.ExpectPages {
				t.Fatalf("case %s (%s): pages = %d, want %d", tc.ID, tc.Title, len(graph.Pages), tc.ExpectPages)
			}
			routes := make([]string, 0, len(graph.Pages))
			for _, page := range graph.Pages {
				routes = append(routes, page.Route)
			}
			if len(tc.ExpectRoutes) > 0 && strings.Join(routes, ",") != strings.Join(tc.ExpectRoutes, ",") {
				t.Fatalf("case %s (%s): routes = %v, want %v", tc.ID, tc.Title, routes, tc.ExpectRoutes)
			}
			if tc.ExpectTitle != "" {
				found := false
				for _, page := range graph.Pages {
					if page.Title == tc.ExpectTitle {
						found = true
					}
				}
				if !found {
					t.Errorf("case %s (%s): no page is titled %q", tc.ID, tc.Title, tc.ExpectTitle)
				}
			}
			for source, audience := range tc.ExpectAudience {
				page, ok := pageBySource(graph, source)
				if !ok {
					t.Errorf("case %s (%s): no page reconstructed from %s", tc.ID, tc.Title, source)
					continue
				}
				if string(page.Audience) != audience {
					t.Errorf("case %s (%s): %s audience = %s, want %s", tc.ID, tc.Title, source, page.Audience, audience)
				}
			}
			if len(tc.ExpectHumanPages) > 0 {
				human := []string{}
				for _, page := range graph.Human() {
					human = append(human, page.Route)
				}
				sort.Strings(human)
				if strings.Join(human, ",") != strings.Join(tc.ExpectHumanPages, ",") {
					t.Errorf("case %s (%s): human pages = %v, want %v", tc.ID, tc.Title, human, tc.ExpectHumanPages)
				}
			}
			for _, source := range tc.ExpectHistorical {
				page, ok := pageBySource(graph, source)
				if !ok || !page.Historical {
					t.Errorf("case %s (%s): %s must be classified historical", tc.ID, tc.Title, source)
				}
			}
			if len(tc.ExpectCurrentPages) > 0 {
				current := []string{}
				for _, page := range graph.Current() {
					current = append(current, page.Route)
				}
				sort.Strings(current)
				if strings.Join(current, ",") != strings.Join(tc.ExpectCurrentPages, ",") {
					t.Errorf("case %s (%s): AI-retrievable pages = %v, want %v", tc.ID, tc.Title, current, tc.ExpectCurrentPages)
				}
			}
			if tc.ExpectMarkdownRoundtrip != nil && *tc.ExpectMarkdownRoundtrip {
				assertMarkdownRoundTrip(t, tc, graph)
			}
			if tc.ExpectIdempotent != nil && *tc.ExpectIdempotent {
				assertReconstructionIsIdempotent(t, tc, root)
			}
		})
	}
}

// assertMarkdownRoundTrip is the reconstruction property that matters most: the
// projected Markdown must be the canonical Markdown, byte for byte.
func assertMarkdownRoundTrip(t *testing.T, tc reconstructionCase, graph Graph) {
	t.Helper()
	artifacts, err := Renderers()["markdown"].Render(graph)
	if err != nil {
		t.Fatal(err)
	}
	bySource := map[string]string{}
	for _, source := range tc.Sources {
		bySource[source.Path] = source.Content
	}
	if len(artifacts) != len(graph.Human()) {
		t.Fatalf("case %s: %d artifacts for %d human pages", tc.ID, len(artifacts), len(graph.Human()))
	}
	for _, page := range graph.Human() {
		want, ok := bySource[page.Source]
		if !ok {
			continue
		}
		if page.Markdown != want {
			t.Errorf("case %s (%s): reconstructed Markdown differs from the source %s", tc.ID, tc.Title, page.Source)
		}
	}
}

// assertReconstructionIsIdempotent re-renders the graph and requires the exact
// same artifacts, so a rebuild cannot silently reshuffle routes or content.
func assertReconstructionIsIdempotent(t *testing.T, tc reconstructionCase, root string) {
	t.Helper()
	first, err := BuildGraph(root, DefaultOutDir, Options{Locale: "en"}, []string{"markdown"})
	if err != nil {
		t.Fatal(err)
	}
	firstContent := readArtifacts(t, root, first)
	second, err := BuildGraph(root, DefaultOutDir, Options{Locale: "en"}, []string{"markdown"})
	if err != nil {
		t.Fatal(err)
	}
	secondContent := readArtifacts(t, root, second)
	if len(firstContent) != len(secondContent) {
		t.Fatalf("case %s: rebuild produced %d artifacts then %d", tc.ID, len(firstContent), len(secondContent))
	}
	for path, content := range firstContent {
		if secondContent[path] != content {
			t.Errorf("case %s: rebuild changed %s", tc.ID, path)
		}
	}
	if len(second.Written) != 0 {
		t.Errorf("case %s: a rebuild with unchanged inputs must write nothing, rewrote %v", tc.ID, second.Written)
	}
}

func readArtifacts(t *testing.T, root string, result BuildResult) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, rel := range append(append([]string{}, result.Written...), result.Unchanged...) {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		out[rel] = string(data)
	}
	return out
}

func pageBySource(graph Graph, source string) (Page, bool) {
	for _, page := range graph.Pages {
		if page.Source == source {
			return page, true
		}
	}
	return Page{}, false
}

func writeReconstructionFixture(t *testing.T, tc reconstructionCase) string {
	t.Helper()
	root := t.TempDir()
	for _, source := range tc.Sources {
		path := filepath.Join(root, filepath.FromSlash(source.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source.Content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	documents := tc.AuthorityMap
	if len(documents) == 0 {
		documents = []reconstructionAuthority{{Path: "docs/**", Role: "canonical"}, {Path: "*.md", Role: "canonical"}}
	}
	data, err := json.MarshalIndent(map[string]any{
		"version": 1, "current_version": "0.6.0", "documents": documents,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "docs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AUTHORITY_MAP.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func loadReconstructionCases(t *testing.T) []reconstructionCase {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "evals", "reconstruction", "cases", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("reconstruction corpus has no cases")
	}
	cases := make([]reconstructionCase, 0, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var tc reconstructionCase
		if err := json.Unmarshal(data, &tc); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if tc.ID == "" || tc.Kind == "" {
			t.Fatalf("%s: a reconstruction case requires id and kind", f)
		}
		cases = append(cases, tc)
	}
	return cases
}
