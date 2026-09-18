package dialog

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/raillen/prumo-tui/internal/onboard"
)

func options() []onboard.Option {
	return []onboard.Option{
		{ID: "install-opencode", Label: "install it", Detail: "runs npm install -g opencode-ai", Automatic: true},
		{ID: "api-key", Label: "use a key", Detail: "export PRUMO_MODEL_API_KEY"},
		{ID: "later", Label: "not now", Detail: "the fake provider keeps runs deterministic"},
	}
}

// Every option says what it will do, in the dialog and not only in the code: a
// choice made blind is a choice the person cannot make.
func TestEveryOptionIsShownWithItsConsequence(t *testing.T) {
	view := NewOnboardDialogCmp(options()).View().Content

	for _, want := range []string{"install it", "runs npm install -g opencode-ai", "use a key", "export PRUMO_MODEL_API_KEY", "not now", "the fake provider keeps runs deterministic"} {
		if !strings.Contains(view, want) {
			t.Errorf("the dialog lost %q", want)
		}
	}
}

func TestEnterChoosesTheHighlightedOption(t *testing.T) {
	cmp := NewOnboardDialogCmp(options())

	_, cmd := cmp.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd = cmp.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	msg := runCmd(cmd)
	choice, ok := msg.(OnboardChoiceMsg)
	if !ok {
		t.Fatalf("enter produced %T", msg)
	}
	if choice.ID != "api-key" {
		t.Fatalf("enter chose %q", choice.ID)
	}
}

// Dismissing is an answer too, and it is the one that lets the person get to
// work: a dialog that cannot be left blocks what it offered to help with.
func TestEscapeIsNotNow(t *testing.T) {
	cmp := NewOnboardDialogCmp(options())

	_, cmd := cmp.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	choice, ok := runCmd(cmd).(OnboardChoiceMsg)
	if !ok || choice.ID != "later" {
		t.Fatalf("escape produced %+v", choice)
	}
}

// Moving past the end stops at the end: a cursor that wraps makes the same key
// mean two things.
func TestTheCursorStopsAtTheEnds(t *testing.T) {
	cmp := NewOnboardDialogCmp(options())

	for i := 0; i < 5; i++ {
		cmp.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if _, cmd := cmp.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); runCmd(cmd).(OnboardChoiceMsg).ID != "later" {
		t.Fatal("the cursor ran past the last option")
	}
	for i := 0; i < 5; i++ {
		cmp.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	}
	if _, cmd := cmp.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); runCmd(cmd).(OnboardChoiceMsg).ID != "install-opencode" {
		t.Fatal("the cursor ran past the first option")
	}
}

func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}
