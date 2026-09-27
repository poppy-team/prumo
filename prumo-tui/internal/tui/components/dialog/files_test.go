package dialog

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestFilesDialogListAndDiff(t *testing.T) {
	d := NewFilesDialogCmp()
	d.SetFiles([]ChangedFile{
		{Path: "main.go", Operation: "modified"},
		{Path: "README.md", Operation: "created"},
	})

	// Initial view shows file list
	view := d.View().Content
	if !strings.Contains(view, "main.go") || !strings.Contains(view, "modified") {
		t.Fatalf("expected main.go in view, got: %s", view)
	}

	// Press Enter to view diff
	_, cmd := d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command on Enter")
	}
	msg := cmd()
	viewDiffMsg, ok := msg.(ViewDiffMsg)
	if !ok || viewDiffMsg.Path != "main.go" {
		t.Fatalf("expected ViewDiffMsg for main.go, got: %v", msg)
	}

	// Supply DiffLoadedMsg
	d.SetDiff(DiffLoadedMsg{
		Path:    "main.go",
		Kind:    "patch",
		Content: "--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-old\n+new",
	})

	diffView := d.View().Content
	if !strings.Contains(diffView, "Diff: main.go") {
		t.Fatalf("expected diff title in view, got: %s", diffView)
	}

	// Press Escape to return to file list
	d.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	listView := d.View().Content
	if !strings.Contains(listView, "Files this run changed") {
		t.Fatalf("expected return to file list, got: %s", listView)
	}

	// Escape from file list closes dialog
	_, closeCmd := d.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if closeCmd == nil {
		t.Fatal("expected close command on Escape from list")
	}
	closeMsg := closeCmd()
	if _, ok := closeMsg.(CloseFilesDialogMsg); !ok {
		t.Fatalf("expected CloseFilesDialogMsg, got: %v", closeMsg)
	}
}

func TestFilesDialogDiffError(t *testing.T) {
	d := NewFilesDialogCmp()
	d.SetFiles([]ChangedFile{{Path: "missing.go", Operation: "deleted"}})

	d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	d.SetDiff(DiffLoadedMsg{
		Path: "missing.go",
		Err:  errors.New("failed to read diff"),
	})

	diffView := d.View().Content
	if !strings.Contains(diffView, "Error loading diff") {
		t.Fatalf("expected error in view, got: %s", diffView)
	}
}
