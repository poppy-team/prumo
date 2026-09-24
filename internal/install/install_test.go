package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifestRoundTripAndIdempotency(t *testing.T) {
	home := t.TempDir()
	manifest := Manifest{PrumoVersion: "0.4.0-dev", BinaryPath: "/tmp/prumo", Connectors: map[string]string{"opencode": "installed"}, CreatedPaths: []string{"/tmp/b", "/tmp/a", "/tmp/a"}}
	if err := SaveManifest(home, manifest); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := SaveManifest(home, manifest); err != nil {
		t.Fatalf("second save: %v", err)
	}
	got, err := LoadManifest(home)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got.CreatedPaths) != 2 || got.CreatedPaths[0] != "/tmp/a" {
		t.Fatalf("unexpected paths: %#v", got.CreatedPaths)
	}
}

func TestRemoveManagedPathsDoesNotTouchUnmanagedProject(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(filepath.Join(project, ".ai"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	projectFile := filepath.Join(project, ".ai", "goal.json")
	if err := os.WriteFile(projectFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	managed := filepath.Join(home, "cache", "entry")
	if err := os.MkdirAll(filepath.Dir(managed), 0755); err != nil {
		t.Fatalf("mkdir managed: %v", err)
	}
	if err := os.WriteFile(managed, []byte("cache"), 0644); err != nil {
		t.Fatalf("write managed: %v", err)
	}
	removed, leftovers := RemoveManagedPaths(home, []string{managed})
	if len(removed) != 1 || len(leftovers) != 0 {
		t.Fatalf("remove result: %v %v", removed, leftovers)
	}
	if _, err := os.Stat(projectFile); err != nil {
		t.Fatalf("project data changed: %v", err)
	}
}

func TestHomeDirExplicitWins(t *testing.T) {
	home, err := HomeDir("./test-home")
	if err != nil {
		t.Fatalf("home: %v", err)
	}
	if filepath.Base(home) != "test-home" {
		t.Fatalf("unexpected home: %s", home)
	}
}
