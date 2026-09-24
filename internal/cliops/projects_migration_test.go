package cliops

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The snapshot was taken after the write it was meant to protect. Its purpose
// is to be the state the migration started from; taken afterwards it is a
// snapshot of the migration's own result, and rolling back restores the
// migrated file with the version that caused the migration gone (GAP-145).

func TestTheMigrationSnapshotPrecedesTheWrite(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "prumo.json")
	original := `{"version":2,"protocol":null}`
	if err := os.WriteFile(manifest, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New(root)

	report, err := s.migrateV2(root, false)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, _ := report["snapshot"].(string)
	if snapshot == "" {
		t.Fatal("the migration reported no snapshot at all")
	}
	if _, err := os.Stat(snapshot); err != nil {
		t.Fatalf("the snapshot named in the report does not exist: %v", err)
	}

	// The manifest has moved on...
	migrated, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(migrated), `"version": 3`) {
		t.Fatalf("the migration did not happen: %s", migrated)
	}

	// ...and what the snapshot holds is the version it started from. If the
	// snapshot were taken afterwards it would contain version 3 too, and the
	// snapshot would be a copy of the result rather than a way back.
	archive, err := zip.OpenReader(snapshot)
	if err != nil {
		t.Fatalf("the snapshot is not a readable archive: %v", err)
	}
	defer archive.Close()
	var held string
	for _, file := range archive.File {
		if filepath.Base(file.Name) != "prumo.json" {
			continue
		}
		reader, openErr := file.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		data, readErr := io.ReadAll(reader)
		reader.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		held = string(data)
	}
	if held == "" {
		t.Fatalf("the snapshot holds no prumo.json; it cannot roll the migration back")
	}
	if !strings.Contains(held, `"version":2`) {
		t.Fatalf("the snapshot holds %q, which is the migrated state, not the pre-migration one", held)
	}
}

func TestAMigrationThatCannotBeSnapshottedDoesNotStart(t *testing.T) {
	root := t.TempDir()
	// prumo.json inside a directory that cannot be created: the snapshot path
	// lives under .prumo/snapshots, and making that impossible must stop the
	// migration rather than follow through without a way back.
	blocked := filepath.Join(root, ".prumo")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "prumo.json")
	if err := os.WriteFile(manifest, []byte(`{"version":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New(root)
	if _, err := s.migrateV2(root, false); err == nil {
		t.Fatal("the migration ran with no snapshot available")
	}
	after, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), `"version":2`) {
		t.Fatalf("the manifest was not left alone: %s", after)
	}
	if strings.Contains(string(after), `"version": 3`) {
		t.Fatalf("the manifest was migrated even though the snapshot failed: %s", after)
	}
}
