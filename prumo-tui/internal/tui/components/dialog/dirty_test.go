package dialog

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestDirtyDialogKeyActions(t *testing.T) {
	d := NewDirtyDialog()
	d.SetDirty("add user auth", "/fake/workspace", "2 uncommitted changes")

	// Esc triggers abort
	_, abortCmd := d.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if abortCmd == nil {
		t.Fatal("expected command on Esc abort")
	}

	// 'p' triggers proceed and emits CloseDirtyDialogMsg with action proceed
	_, proceedCmd := d.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
	if proceedCmd == nil {
		t.Fatal("expected command on 'p' proceed")
	}

	// Verify view rendering
	view := d.View().Content
	if !strings.Contains(view, "Uncommitted Workspace Changes") {
		t.Fatalf("expected title in view, got: %s", view)
	}
	if !strings.Contains(view, "[s] Stash") || !strings.Contains(view, "[c] Commit") {
		t.Fatalf("expected action buttons in view, got: %s", view)
	}
}

func TestCheckDirtyStatus(t *testing.T) {
	tmpDir := t.TempDir()

	// Init git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Clean status initially
	dirty, summary, err := CheckDirtyStatus(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dirty {
		t.Fatalf("expected clean repo, got dirty: %s", summary)
	}

	// Create untracked file -> dirty
	if err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	dirty, summary, err = CheckDirtyStatus(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dirty {
		t.Fatal("expected dirty repo after creating file")
	}
	if !strings.Contains(summary, "file.txt") {
		t.Fatalf("expected summary to mention file.txt, got: %s", summary)
	}
}

func TestDirtyDialogProceedEmitsSendMsg(t *testing.T) {
	d := NewDirtyDialog()
	d.SetDirty("refactor tests", "/tmp", "1 uncommitted change")

	// Press 'p'
	_, cmd := d.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
	if cmd == nil {
		t.Fatal("expected batch command on 'p'")
	}
}

func TestDirtyDialogKeyNavigation(t *testing.T) {
	d := NewDirtyDialog()
	d.SetDirty("goal", "/tmp", "summary")

	// Arrow right moves selected button
	d.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	// Arrow left moves back
	d.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	// Tab moves forward
	d.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	bindings := d.BindingKeys()
	if len(bindings) != 4 {
		t.Fatalf("expected 4 bindings, got %d", len(bindings))
	}
}
