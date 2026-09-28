package dialog

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestQuitDialogRendersButtonsOnOneLine(t *testing.T) {
	cmp := NewQuitCmp()
	cmp.SetSession("test-123", "feature/my-branch")

	view := cmp.View().Content
	if !strings.Contains(view, "Save & Quit") {
		t.Fatalf("expected view to contain 'Save & Quit', got:\n%s", view)
	}
	if !strings.Contains(view, "Quit") {
		t.Fatalf("expected view to contain 'Quit', got:\n%s", view)
	}
	if !strings.Contains(view, "Cancel") {
		t.Fatalf("expected view to contain 'Cancel', got:\n%s", view)
	}
	if !strings.Contains(view, "feature/my-branch") {
		t.Fatalf("expected view to contain session title 'feature/my-branch', got:\n%s", view)
	}

	// Verify that Save & Quit, Quit, and Cancel appear in the view,
	// and that the button line contains all 3 buttons on the same line.
	lines := strings.Split(view, "\n")
	foundButtonsLine := false
	for _, line := range lines {
		if strings.Contains(line, "Save & Quit") && strings.Contains(line, "Quit") && strings.Contains(line, "Cancel") {
			foundButtonsLine = true
			break
		}
	}
	if !foundButtonsLine {
		t.Fatalf("expected all three buttons to be rendered on the exact same line, lines were:\n%s", view)
	}
}

func TestQuitDialogSavesSessionTitleOnSaveAndQuit(t *testing.T) {
	cmp := NewQuitCmp()
	cmp.SetSession("session-abc", "custom-session-title")

	// Press Enter while input is focused -> should emit QuitWithSaveMsg
	_, cmd := cmp.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command from Enter key")
	}

	msg := cmd()
	saveMsg, ok := msg.(QuitWithSaveMsg)
	if !ok {
		t.Fatalf("expected QuitWithSaveMsg, got %T: %+v", msg, msg)
	}
	if saveMsg.SessionID != "session-abc" {
		t.Fatalf("expected session ID 'session-abc', got %q", saveMsg.SessionID)
	}
	if saveMsg.Title != "custom-session-title" {
		t.Fatalf("expected title 'custom-session-title', got %q", saveMsg.Title)
	}
}

func TestQuitDialogNavigationAndKeys(t *testing.T) {
	cmp := NewQuitCmp()
	cmp.SetSession("session-xyz", "my-run")

	// Press Esc -> should emit CloseQuitMsg
	_, cmd := cmp.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected command from Esc key")
	}
	msg := cmd()
	if _, ok := msg.(CloseQuitMsg); !ok {
		t.Fatalf("expected CloseQuitMsg from Esc, got %T", msg)
	}

	// Down arrow moves focus to buttons
	cmp.SetSession("session-xyz", "my-run")
	cmp.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	// Press 'q' on buttons -> returns tea.Quit
	_, cmd = cmp.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatal("expected tea.Quit command from 'q'")
	}
	if cmd() != tea.Quit() {
		t.Fatalf("expected tea.Quit(), got %v", cmd())
	}
}
