package chat

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/tui/components/dialog"
)

func newEditor(t *testing.T) *editorCmp {
	t.Helper()
	return NewEditorCmp(app.New(app.Options{})).(*editorCmp)
}

// A file is named in the goal, not attached to it: what a model can see is the
// harness's decision, made from the model's own capabilities, and the transcript
// keeps saying which file was meant.
func TestPickingAFileWritesItsPathIntoTheGoal(t *testing.T) {
	editor := newEditor(t)

	editor.Update(dialog.ReferenceSelectedMsg{Path: "shot.png"})
	if got := editor.textarea.Value(); !strings.Contains(got, "shot.png") {
		t.Fatalf("the picked path did not reach the goal: %q", got)
	}

	// A second pick is separated from the first, so two references read as two.
	editor.Update(dialog.ReferenceSelectedMsg{Path: "logs/error.png"})
	got := editor.textarea.Value()
	if !strings.Contains(got, "shot.png logs/error.png") {
		t.Fatalf("the references ran together: %q", got)
	}
}

// What is typed stays: picking a file adds to the goal rather than replacing it.
func TestPickingAFileKeepsWhatWasTyped(t *testing.T) {
	editor := newEditor(t)
	editor.textarea.SetValue("o que está errado aqui?")

	editor.Update(dialog.ReferenceSelectedMsg{Path: "shot.png"})

	got := editor.textarea.Value()
	if !strings.HasPrefix(got, "o que está errado aqui?") || !strings.Contains(got, "shot.png") {
		t.Fatalf("the goal was replaced instead of extended: %q", got)
	}
}

// Entering a directory navigates; only a file becomes a reference.
func TestADirectoryIsNotAReference(t *testing.T) {
	editor := newEditor(t)
	editor.Update(dialog.ReferenceSelectedMsg{Path: "internal/"})

	if got := editor.textarea.Value(); got != "internal/" {
		t.Fatalf("the editor changed something it was not asked to: %q", got)
	}
}

func TestEnterSendsMessage(t *testing.T) {
	editor := newEditor(t)
	editor.textarea.SetValue("hello world")

	_, cmd := editor.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected send command on Enter, got nil")
	}
	if got := editor.textarea.Value(); got != "" {
		t.Fatalf("expected editor value to be reset after send, got %q", got)
	}
}

func TestAltEnterInsertsNewline(t *testing.T) {
	editor := newEditor(t)
	editor.textarea.SetValue("line1")

	_, cmd := editor.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModAlt})
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(SendMsg); ok {
			t.Fatal("Alt+Enter should not send message")
		}
	}
	if got := editor.textarea.Value(); got != "line1\n" {
		t.Fatalf("expected 'line1\\n', got %q", got)
	}
}

func TestShiftEnterInsertsNewline(t *testing.T) {
	editor := newEditor(t)
	editor.textarea.SetValue("line1")

	_, cmd := editor.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift})
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(SendMsg); ok {
			t.Fatal("Shift+Enter should not send message")
		}
	}
	if got := editor.textarea.Value(); got != "line1\n" {
		t.Fatalf("expected 'line1\\n', got %q", got)
	}
}
