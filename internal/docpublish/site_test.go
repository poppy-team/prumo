package docpublish

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"README.md":                       "# Prumo\n\nEntry point.\n",
		"docs/product/vision.md":          "# Vision\n\n## Problem\n\nThe product solves coordination.\n\n## Outcome\n\nA provider-neutral harness.\n",
		"docs/reference/cli.md":           "# CLI reference\n\nSee [vision](../product/vision.md) and [gone](../missing/nope.md).\n",
		"docs/agents/instruction-ir.json": `{"version":1,"rules":[{"id":"a","text":"x"}]}`,
		"AGENTS.md":                       "# AGENTS.md\n\nAgent only.\n",
	}
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestLoadBuildsDeterministicGraph(t *testing.T) {
	root := fixture(t)
	graph, err := Load(root, "0.6.0", "en")
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Pages) != 4 {
		t.Fatalf("expected 4 pages, got %d (%v)", len(graph.Pages), graph.Routes())
	}
	human := graph.Human()
	if len(human) != 3 {
		t.Fatalf("agent-only surfaces must be excluded from the human site: %d pages", len(human))
	}
	routes := graph.Routes()
	want := []string{"/", "/product/vision", "/reference/cli"}
	for _, w := range want {
		found := false
		for _, r := range routes {
			if r == w {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing route %s in %v", w, routes)
		}
	}
	for _, p := range graph.Pages {
		if p.Source == "AGENTS.md" && p.Audience != AudienceAgent {
			t.Fatal("AGENTS.md must be an agent-only surface")
		}
		if p.Source == "docs/product/vision.md" && p.Title != "Vision" {
			t.Fatalf("unexpected title %q", p.Title)
		}
	}
}

func TestDoctorFindsDeadInternalLinks(t *testing.T) {
	root := fixture(t)
	graph, err := Load(root, "0.6.0", "en")
	if err != nil {
		t.Fatal(err)
	}
	findings := graph.Doctor()
	kinds := map[string]int{}
	for _, f := range findings {
		kinds[f.Kind]++
	}
	if kinds["dead-internal-link"] != 1 {
		t.Fatalf("expected exactly one dead link, got %#v", findings)
	}
}

func TestRenderersProjectOneGraph(t *testing.T) {
	root := fixture(t)
	graph, err := Load(root, "0.6.0", "en")
	if err != nil {
		t.Fatal(err)
	}
	for name, renderer := range Renderers() {
		artifacts, err := renderer.Render(graph)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(artifacts) == 0 {
			t.Fatalf("%s produced no artifacts", name)
		}
		for _, a := range artifacts {
			if a.Path == "" || strings.Contains(a.Path, "..") {
				t.Fatalf("%s produced an unsafe path %q", name, a.Path)
			}
		}
	}

	llms := Renderers()["llms"]
	artifacts, err := llms.Render(graph)
	if err != nil {
		t.Fatal(err)
	}
	index := artifactByPath(t, artifacts, "llms.txt")
	if !strings.Contains(index, "/product/vision") || !strings.Contains(index, "AGENTS.md") {
		t.Fatalf("llms.txt must index human and agent surfaces:\n%s", index)
	}

	starlight := Renderers()["starlight"]
	artifacts, err = starlight.Render(graph)
	if err != nil {
		t.Fatal(err)
	}
	page := artifactByPath(t, artifacts, "starlight/content/docs/product/vision.mdx")
	for _, want := range []string{"title: \"Vision\"", "slug: \"/product/vision\"", "version: \"0.6.0\""} {
		if !strings.Contains(page, want) {
			t.Fatalf("starlight page missing %q:\n%s", want, page)
		}
	}
	manifest := artifactByPath(t, artifacts, "starlight/site-manifest.json")
	if !strings.Contains(manifest, "\"versions\"") || !strings.Contains(manifest, "\"locales\"") {
		t.Fatalf("site manifest must declare locales and versions:\n%s", manifest)
	}
}

func TestSearchIndexChunksByHeadingAndQueryRanks(t *testing.T) {
	root := fixture(t)
	result, err := BuildGraph(root, "", Options{Version: "0.6.0", Locale: "en"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Pages != 4 || result.Artifacts == 0 {
		t.Fatalf("unexpected build result: %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(DefaultOutDir), "search-index.json")); err != nil {
		t.Fatal(err)
	}

	hits, err := Query(root, "provider-neutral harness", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 || hits[0].Route != "/product/vision" {
		t.Fatalf("expected the vision page to rank first: %#v", hits)
	}
	if hits[0].Heading != "Outcome" {
		t.Fatalf("expected the Outcome chunk, got %#v", hits[0])
	}

	none, err := Query(root, "zzzz-not-present", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(none) != 0 {
		t.Fatalf("expected no hits, got %#v", none)
	}
}

func TestBuildRejectsUnknownRendererAndIsIdempotent(t *testing.T) {
	root := fixture(t)
	if _, err := BuildGraph(root, "", Options{Version: "0.6.0", Locale: "en"}, []string{"not-a-renderer"}); err == nil {
		t.Fatal("unknown renderer must be an error, never a silent skip")
	}
	first, err := BuildGraph(root, "", Options{Version: "0.6.0", Locale: "en"}, []string{"markdown"})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Written) == 0 {
		t.Fatalf("expected writes on the first build: %#v", first)
	}
	second, err := BuildGraph(root, "", Options{Version: "0.6.0", Locale: "en"}, []string{"markdown"})
	if err != nil {
		t.Fatal(err)
	}
	// Idempotence is now exact: a rebuild with unchanged inputs writes nothing
	// at all, manifest included, so the written set is a truthful change signal.
	if len(second.Written) != 0 || len(second.Unchanged) != first.Artifacts+1 {
		t.Fatalf("expected an idempotent second build, got %#v", second)
	}
}

func artifactByPath(t *testing.T, artifacts []Artifact, path string) string {
	t.Helper()
	for _, a := range artifacts {
		if a.Path == path {
			return a.Content
		}
	}
	t.Fatalf("artifact %s not found", path)
	return ""
}
