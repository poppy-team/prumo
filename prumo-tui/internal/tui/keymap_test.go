package tui

import (
	"context"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/session"
	"github.com/raillen/prumo-tui/internal/tui/components/chat"
)

func ctrl(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Mod: tea.ModCtrl} }

// The interaction contract reserves two chords and states what they do. A
// client that rebinds them is not following the map a user learned elsewhere.
func TestReservedChordsMatchTheInteractionContract(t *testing.T) {
	if !slices.Contains(keys.Cancel.Keys(), "ctrl+c") {
		t.Errorf("ctrl+c is the cancel chord; this client binds %v", keys.Cancel.Keys())
	}
	if !slices.Contains(keys.Quit.Keys(), "ctrl+q") {
		t.Errorf("ctrl+q is the quit chord; this client binds %v", keys.Quit.Keys())
	}
}

// Ctrl+C cancels the active operation and never ends the session silently. With
// nothing running it opens the confirmation rather than doing nothing, since a
// chord that answers with silence reads as a broken key.
func TestCtrlCAsksInsteadOfExiting(t *testing.T) {
	updated, cmd := shell(t, 100, 30).Update(ctrl('c'))

	model, ok := updated.(appModel)
	if !ok {
		t.Fatalf("the shell changed shape: %T", updated)
	}
	if !model.showQuit {
		t.Fatal("ctrl+c with nothing in flight did not open the quit confirmation")
	}
	if cmd != nil {
		if _, quits := cmd().(tea.QuitMsg); quits {
			t.Fatal("ctrl+c ended the client instead of asking")
		}
	}
}

// With a run in flight the same chord goes to the run, which the daemon owns:
// the client asks for the stop and stays open.
func TestCtrlCCancelsTheRunInFlight(t *testing.T) {
	harness := &stubHarness{status: "running"}
	application := app.New(app.Options{Client: harness})
	model, _ := New(application).Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Selecting the session is what tells the shell which run a chord acts on.
	model, _ = model.Update(chat.SessionSelectedMsg(session.Session{ID: "S1", Title: "S1"}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := application.CoderAgent.Run(ctx, "S1", "say hello"); err != nil {
		t.Fatalf("cannot start a run: %v", err)
	}

	updated, _ := model.Update(ctrl('c'))

	if !slices.Contains(harness.stops, "S1") {
		t.Fatalf("ctrl+c did not ask the daemon to stop the run: %v", harness.stops)
	}
	if shell, ok := updated.(appModel); ok && shell.showQuit {
		t.Fatal("ctrl+c cancelled the run and opened the quit prompt at the same time")
	}
}

// Every action the client offers is reachable from the palette, and every entry
// says what it does: a command whose label is all a user gets is a guess.
func TestThePaletteExplainsEveryCommand(t *testing.T) {
	model, ok := New(app.New(app.Options{Client: &stubHarness{}})).(*appModel)
	if !ok {
		t.Fatal("New did not return the shell")
	}
	if len(model.commands) == 0 {
		t.Fatal("the palette offers no command at all")
	}
	for _, command := range model.commands {
		if command.Title == "" || command.Description == "" {
			t.Errorf("command %q explains nothing: %+v", command.ID, command)
		}
		if command.Handler == nil {
			t.Errorf("command %q has no handler, so selecting it does nothing", command.ID)
		}
	}
}

// Focus has to be visible without colour, so the marker is a glyph: the
// composer's border is heavy while it holds the keyboard and light while a
// layer is over it.
func TestTheComposerShowsWhetherItHoldsTheKeyboard(t *testing.T) {
	focused := plain(shell(t, 100, 30).View().Content)
	if !strings.Contains(focused, "━") {
		t.Fatal("the focused composer drew no marker that survives a terminal without colour")
	}

	opened, cmd := shell(t, 100, 30).Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	withDialog := plain(drive(t, opened, cmd, 4).View().Content)
	if strings.Contains(withDialog, "━") {
		t.Fatal("a dialog holds the keyboard and the composer still claims it")
	}
}

// Navigation is complete from the keyboard: every surface has a chord, which is
// what "a new action that can only be clicked is a defect" asks for.
func TestEverySurfaceHasAChord(t *testing.T) {
	for name, binding := range map[string][]string{
		"palette":  keys.Commands.Keys(),
		"models":   keys.Models.Keys(),
		"sessions": keys.SwitchSession.Keys(),
		"logs":     keys.Logs.Keys(),
		"themes":   keys.SwitchTheme.Keys(),
		"files":    keys.Filepicker.Keys(),
		"help":     keys.Help.Keys(),
	} {
		if len(binding) == 0 {
			t.Errorf("%s has no keyboard route", name)
		}
	}
}
