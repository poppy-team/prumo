package handoff

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The Bundle is a serialisation format, not a message in flight. Build assembles
// one, Validate checks the four refs, and the CLI prints it and discards it.
// There is no LoadBundle and no ParseBundle in the repository: nothing reads a
// bundle back (GAP-135).
//
// This test is the reminder. A dispatch would need a reader, and the day one
// appears this test is the signal that the package comment and the gap register
// have to change — rather than the code quietly becoming a protocol that the
// documentation still calls a format.

func TestThereIsNoBundleReader(t *testing.T) {
	repo := filepath.Clean(filepath.Join("..", "..", ".."))
	root, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	// A reader would be a function that turns bytes back into a Bundle. Its
	// absence is the point, so the search is for the shape of one.
	reader := regexp.MustCompile(`func\s+\w*\(?[^)]*\)?\s*(Bundle|agent\.Handoff)\s*\{`)
	deserialiser := regexp.MustCompile(`json\.Unmarshal\([^)]*,\s*&?(bundle|h)\b`)
	var found []string
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if strings.HasPrefix(rel, "testdata/") || strings.HasSuffix(rel, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		text := string(data)
		if reader.MatchString(text) && deserialiser.MatchString(text) {
			found = append(found, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) > 0 {
		t.Fatalf("something now reads a handoff bundle back (%v); the package is a protocol and the documentation is out of date", found)
	}
}
