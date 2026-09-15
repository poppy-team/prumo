package docpublish

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSyntheticDocs lays down a repository big enough for build cost to be
// visible: one index plus count-1 topic documents.
func writeSyntheticDocs(t testing.TB, root string, count int) {
	t.Helper()
	var index strings.Builder
	index.WriteString("# Synthetic documentation\n\n")
	for i := 0; i < count; i++ {
		rel := fmt.Sprintf("docs/topic-%03d.md", i)
		body := fmt.Sprintf("# Topic %03d\n\nBody for topic %03d.\n\n- point one\n- point two\n", i, i)
		if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		index.WriteString(fmt.Sprintf("- [Topic %03d](docs/topic-%03d.md)\n", i, i))
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(index.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestLargeRepositoryIncrementalInvalidationIsBounded asserts the property a
// scale claim actually rests on: a no-op build rewrites nothing, and changing
// one document rewrites a bounded slice of the artifact set rather than the
// whole site (W21.11).
func TestLargeRepositoryIncrementalInvalidationIsBounded(t *testing.T) {
	const documents = 120
	root := t.TempDir()
	writeSyntheticDocs(t, root, documents)

	full, err := BuildGraph(root, DefaultOutDir, Options{Locale: "en"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if full.Pages < documents {
		t.Fatalf("expected at least %d pages, got %d", documents, full.Pages)
	}
	if len(full.Written) != full.Artifacts+1 {
		t.Fatalf("a cold build must write every artifact plus the manifest: %d written of %d",
			len(full.Written), full.Artifacts)
	}

	// A rebuild with unchanged inputs must be silent.
	noop, err := BuildGraph(root, DefaultOutDir, Options{Locale: "en"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(noop.Written) != 0 {
		t.Fatalf("an unchanged rebuild must not rewrite anything, got %v", noop.Written)
	}
	if len(noop.Unchanged) != full.Artifacts+1 {
		t.Fatalf("every artifact must be reported unchanged: %d of %d",
			len(noop.Unchanged), full.Artifacts+1)
	}

	// One edited document must invalidate a bounded slice of a large site.
	changed := filepath.Join(root, "docs", "topic-042.md")
	if err := os.WriteFile(changed, []byte("# Topic 042\n\nRewritten body.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	incremental, err := BuildGraph(root, DefaultOutDir, Options{Locale: "en"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(incremental.Written) == 0 {
		t.Fatal("an edited document must invalidate something")
	}
	if len(incremental.Written) >= full.Artifacts+1 {
		t.Fatalf("an incremental build must rewrite less than a cold build: %d of %d",
			len(incremental.Written), full.Artifacts+1)
	}
	wrote := map[string]bool{}
	for _, rel := range incremental.Written {
		wrote[rel] = true
	}
	if !wrote[filepath.ToSlash(filepath.Join(DefaultOutDir, "markdown", "topic-042.md"))] {
		t.Fatalf("the edited document's own projection must be rewritten: %v", incremental.Written)
	}
}

// BenchmarkBuildLargeRepository reports the cost of a cold build and of an
// incremental rebuild over a large documentation set, so a scale claim in the
// documentation can be reproduced rather than asserted (W21.11).
func BenchmarkBuildLargeRepository(b *testing.B) {
	const documents = 200
	root := b.TempDir()
	writeSyntheticDocs(b, root, documents)

	b.Run("cold", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			removeBuildOutput(b, root)
			if _, err := BuildGraph(root, DefaultOutDir, Options{Locale: "en"}, nil); err != nil {
				b.Fatal(err)
			}
		}
	})

	if _, err := BuildGraph(root, DefaultOutDir, Options{Locale: "en"}, nil); err != nil {
		b.Fatal(err)
	}
	b.Run("unchanged", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if _, err := BuildGraph(root, DefaultOutDir, Options{Locale: "en"}, nil); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func removeBuildOutput(b *testing.B, root string) {
	b.Helper()
	if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(DefaultOutDir))); err != nil {
		b.Fatal(err)
	}
}
