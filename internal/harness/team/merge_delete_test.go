package team

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeDeletePropagation(t *testing.T) {
	base := t.TempDir()
	overlay := t.TempDir()
	writeFile(t, filepath.Join(base, "gone.txt"), "bye")
	writeFile(t, filepath.Join(base, "keep.txt"), "same")
	writeFile(t, filepath.Join(overlay, "keep.txt"), "same")
	baseline := SnapshotWorkspace(base)
	res, err := MergeWorkspaces(base, overlay, baseline)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Deleted) != 1 || res.Deleted[0] != "D gone.txt" {
		t.Fatalf("clean delete must propagate: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(base, "gone.txt")); !os.IsNotExist(err) {
		t.Fatal("deleted file must be gone from base")
	}
}

func TestMergeDeleteConflict(t *testing.T) {
	base := t.TempDir()
	overlay := t.TempDir()
	writeFile(t, filepath.Join(base, "f.txt"), "v1")
	baseline := SnapshotWorkspace(base)
	writeFile(t, filepath.Join(base, "f.txt"), "v1-edited-in-base")
	res, err := MergeWorkspaces(base, overlay, baseline)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Conflicts) != 1 || res.Conflicts[0] != "f.txt" {
		t.Fatalf("resurrect-vs-edit must conflict: %+v", res)
	}
	data, _ := os.ReadFile(filepath.Join(base, "f.txt"))
	if string(data) != "v1-edited-in-base" {
		t.Fatal("conflict must leave base untouched")
	}
}
