package completions

import (
	"testing"
)

func TestProcessNullTerminatedOutput(t *testing.T) {
	input := []byte("foo.go\x00bar/baz.txt\x00.hidden\x00")
	got := processNullTerminatedOutput(input)

	if len(got) != 2 {
		t.Fatalf("expected 2 matches, got %d: %v", len(got), got)
	}
	if got[0] != "foo.go" {
		t.Errorf("expected foo.go, got %s", got[0])
	}
	if got[1] != "bar/baz.txt" {
		t.Errorf("expected bar/baz.txt, got %s", got[1])
	}
}

func TestProcessNullTerminatedOutputEmpty(t *testing.T) {
	got := processNullTerminatedOutput([]byte{})
	if len(got) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(got))
	}
}

func TestGetFilesContextGroup(t *testing.T) {
	cg := NewFileAndFolderContextGroup()
	if cg.GetId() != "file" {
		t.Errorf("expected id 'file', got '%s'", cg.GetId())
	}
	entry := cg.GetEntry()
	if entry.DisplayValue() != "Files & Folders" {
		t.Errorf("unexpected entry title: %s", entry.DisplayValue())
	}

	// Calling GetChildEntries should not fail even in directories with broken symlinks or errors
	entries, err := cg.GetChildEntries("")
	if err != nil {
		t.Fatalf("GetChildEntries failed: %v", err)
	}
	if len(entries) == 0 {
		t.Log("GetChildEntries returned 0 entries, which is acceptable in some test environments")
	}
}
