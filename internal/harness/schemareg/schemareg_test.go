package schemareg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAllAndMigrate(t *testing.T) {
	dir, err := repoRootTest()
	if err != nil {
		t.Skip("repo root not found")
	}
	if err := ValidateAll(dir); err != nil {
		t.Fatalf("schemas must parse: %v", err)
	}
	doc := map[string]any{"id": "x"}
	got, err := Migrate("harness-checkpoint", 1, doc)
	if err != nil || got["id"] != "x" {
		t.Fatalf("v1 identity failed: %v", err)
	}
	if _, err := Migrate("harness-checkpoint", 0, doc); err == nil {
		t.Fatal("missing migration path must error")
	}
	if _, err := Migrate("nope", 1, doc); err == nil {
		t.Fatal("unknown contract must error")
	}
}

func repoRootTest() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
