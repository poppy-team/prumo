package chat

import (
	"strings"
	"testing"

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
