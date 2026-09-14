package team

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeFastForwardAndAdd(t *testing.T) {
	base := t.TempDir()
	overlay := t.TempDir()
	writeFile(t, filepath.Join(base, "keep.txt"), "same")
	writeFile(t, filepath.Join(base, "edit.txt"), "v1")
	writeFile(t, filepath.Join(overlay, "keep.txt"), "same")
	writeFile(t, filepath.Join(overlay, "edit.txt"), "v2")
	writeFile(t, filepath.Join(overlay, "new.txt"), "new")
	baseline := SnapshotWorkspace(base)
	res, err := MergeWorkspaces(base, overlay, baseline)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %v", res.Conflicts)
	}
	data, _ := os.ReadFile(filepath.Join(base, "edit.txt"))
	if string(data) != "v2" {
		t.Fatalf("fast-forward failed: %q", data)
	}
	if _, err := os.Stat(filepath.Join(base, "new.txt")); err != nil {
		t.Fatal("added file missing")
	}
}

func TestMergeConflictLeavesBase(t *testing.T) {
	base := t.TempDir()
	overlay := t.TempDir()
	writeFile(t, filepath.Join(base, "f.txt"), "v1")
	baseline := SnapshotWorkspace(base)
	writeFile(t, filepath.Join(base, "f.txt"), "v1-base-changed")
	writeFile(t, filepath.Join(overlay, "f.txt"), "v1-overlay-changed")
	res, err := MergeWorkspaces(base, overlay, baseline)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Conflicts) != 1 || res.Conflicts[0] != "f.txt" {
		t.Fatalf("expected one conflict: %+v", res)
	}
	data, _ := os.ReadFile(filepath.Join(base, "f.txt"))
	if string(data) != "v1-base-changed" {
		t.Fatalf("conflict must leave base untouched: %q", data)
	}
}
