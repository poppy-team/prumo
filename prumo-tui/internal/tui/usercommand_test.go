package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/commands"
	"github.com/raillen/prumo-tui/internal/tui/components/chat"
	"github.com/raillen/prumo-tui/internal/tui/components/dialog"
)

// writeUserCommand puts a prompt in the client's own command directory.
func writeUserCommand(t *testing.T, name, body string) {
	t.Helper()
	dir := commands.Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("cannot create the command directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("cannot write the command: %v", err)
	}
}

func userCommandIn(t *testing.T, model *appModel, id string) dialog.Command {
	t.Helper()
	for _, command := range model.commands {
		if command.ID == id {
			return command
		}
	}
	t.Fatalf("%s is not in the palette: %+v", id, model.commands)
	return dialog.Command{}
}

// The palette is the command surface: what a user writes becomes an entry, with
// what it does written under it.
func TestAUserCommandReachesThePalette(t *testing.T) {
	t.Setenv("PRUMO_TUI_CONFIG_DIR", t.TempDir())
	writeUserCommand(t, "review.md", "---\ndescription: Look for problems\n---\nReview the working tree.\n")

	model, ok := New(app.New(app.Options{Client: &stubHarness{}})).(*appModel)
	if !ok {
		t.Fatal("New did not return the shell")
	}

	command := userCommandIn(t, model, "user:review")
	if command.Title != "review" || command.Description != "Look for problems" {
		t.Fatalf("the entry does not explain itself: %+v", command)
	}
}

// A prompt with nothing to fill in is sent as it was written.
func TestACommandWithoutArgumentsIsSentDirectly(t *testing.T) {
	t.Setenv("PRUMO_TUI_CONFIG_DIR", t.TempDir())
	writeUserCommand(t, "notes.md", "Summarise the open questions.\n")

	model, _ := New(app.New(app.Options{Client: &stubHarness{}})).(*appModel)
	cmd := userCommandIn(t, model, "user:notes").Handler(dialog.Command{})
	if cmd == nil {
		t.Fatal("selecting the command did nothing")
	}
	sent, ok := cmd().(chat.SendMsg)
	if !ok {
		t.Fatalf("the command produced %T", cmd())
	}
	if !strings.Contains(sent.Text, "Summarise the open questions") {
		t.Fatalf("the prompt was altered: %q", sent.Text)
	}
}

// A prompt that declares arguments asks for them first.
func TestACommandWithArgumentsAsksForThem(t *testing.T) {
	t.Setenv("PRUMO_TUI_CONFIG_DIR", t.TempDir())
	writeUserCommand(t, "review.md", "---\nargs: [area]\n---\nReview the {{area}} area.\n")

	model, _ := New(app.New(app.Options{Client: &stubHarness{}})).(*appModel)
	cmd := userCommandIn(t, model, "user:review").Handler(dialog.Command{})
	ask, ok := cmd().(dialog.ShowMultiArgumentsDialogMsg)
	if !ok {
		t.Fatalf("the command produced %T instead of asking", cmd())
	}
	if len(ask.ArgNames) != 1 || ask.ArgNames[0] != "area" {
		t.Fatalf("the dialog was not told what to ask for: %+v", ask)
	}
	if !strings.Contains(ask.Content, "{{area}}") {
		t.Fatalf("the dialog was not given the prompt to fill: %q", ask.Content)
	}
}

// Submitting the dialog sends the prompt with the arguments in it, which is what
// the dialog existed for and never did.
func TestSubmittingTheArgumentsSendsTheFilledPrompt(t *testing.T) {
	t.Setenv("PRUMO_TUI_CONFIG_DIR", t.TempDir())
	writeUserCommand(t, "review.md", "---\nargs: [area]\n---\nReview the {{area}} area.\n")

	model, _ := New(app.New(app.Options{Client: &stubHarness{}})).(*appModel)

	_, cmd := model.Update(dialog.CloseMultiArgumentsDialogMsg{
		Submit:    true,
		CommandID: "user:review",
		Args:      map[string]string{"area": "parser"},
	})
	if cmd == nil {
		t.Fatal("submitting the arguments sent nothing")
	}
	sent, ok := cmd().(chat.SendMsg)
	if !ok {
		t.Fatalf("submitting produced %T", cmd())
	}
	if !strings.Contains(sent.Text, "Review the parser area") {
		t.Fatalf("the prompt was not filled in: %q", sent.Text)
	}
}

// A dialog that was dismissed runs nothing: the command is a prompt, and a
// prompt nobody finished is not a message.
func TestADismissedDialogSendsNothing(t *testing.T) {
	t.Setenv("PRUMO_TUI_CONFIG_DIR", t.TempDir())

	model, _ := New(app.New(app.Options{Client: &stubHarness{}})).(*appModel)
	_, cmd := model.Update(dialog.CloseMultiArgumentsDialogMsg{Submit: false, CommandID: "user:review"})
	if cmd != nil {
		if _, sends := cmd().(chat.SendMsg); sends {
			t.Fatal("a dismissed dialog sent a message")
		}
	}
}
