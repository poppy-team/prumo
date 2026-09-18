package dialog

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/raillen/prumo-tui/internal/runtime"
)

// A model nobody declared must not read as a model without features: the two are
// different facts, and only one of them is knowable.
func TestThePickerSaysWhenNothingWasDeclared(t *testing.T) {
	d := NewModelDialogCmp().(*modelDialogCmp)
	d.models = []string{"gpt-4o-mini"}
	d.info = map[string]runtime.ModelCapabilities{}

	frame := plainFrame(d.View().Content)
	if !strings.Contains(frame, "not declared") {
		t.Fatalf("an undeclared model is drawn as if something were known:\n%s", frame)
	}
}

// What a model declares is shown beside it, so a user does not have to know by
// heart which models see.
func TestThePickerShowsWhatEachModelDeclares(t *testing.T) {
	d := NewModelDialogCmp().(*modelDialogCmp)
	d.models = []string{"vision-model", "plain-model"}
	d.info = map[string]runtime.ModelCapabilities{
		"vision-model": {Declared: true, Features: []string{"text", "vision", "tools"}},
		"plain-model":  {Declared: true, Features: []string{"text"}},
	}

	frame := plainFrame(d.View().Content)
	if !strings.Contains(frame, "[text vision tools]") {
		t.Fatalf("the declared features are missing:\n%s", frame)
	}
	if !strings.Contains(frame, "[text]") {
		t.Fatalf("a model that declares only text lost its line:\n%s", frame)
	}
}

// A declaration with nothing in it is its own case: the file named the model and
// claimed nothing about it.
func TestAnEmptyDeclarationIsNotSilence(t *testing.T) {
	d := NewModelDialogCmp().(*modelDialogCmp)
	d.models = []string{"mystery"}
	d.info = map[string]runtime.ModelCapabilities{"mystery": {Declared: true}}

	if frame := plainFrame(d.View().Content); !strings.Contains(frame, "nothing claimed") {
		t.Fatalf("an empty declaration read as an undeclared model:\n%s", frame)
	}
}

func TestFreeModelsPrioritizedAndBadged(t *testing.T) {
	d := NewModelDialogCmp().(*modelDialogCmp)
	d.Update(ModelsLoadedMsg{
		Models: []string{"opencode/gpt-5", "opencode/nemotron-3.5-lightning-free", "opencode/claude-sonnet-4"},
		Info:   map[string]runtime.ModelCapabilities{},
	})

	if len(d.filtered) != 3 {
		t.Fatalf("expected 3 models, got %d", len(d.filtered))
	}
	// Free model must come first
	if d.filtered[0] != "opencode/nemotron-3.5-lightning-free" {
		t.Fatalf("expected free model at index 0, got %q", d.filtered[0])
	}

	frame := plainFrame(d.View().Content)
	if !strings.Contains(frame, "[FREE]") {
		t.Fatalf("expected [FREE] badge in view:\n%s", frame)
	}
	if !strings.Contains(frame, "1 free") {
		t.Fatalf("expected '1 free' count in hint:\n%s", frame)
	}
}

func TestModelDialogSearchFilter(t *testing.T) {
	d := NewModelDialogCmp().(*modelDialogCmp)
	d.Update(ModelsLoadedMsg{
		Models: []string{
			"opencode/nemotron-3.5-lightning-free",
			"opencode/claude-sonnet-4-5",
			"opencode/gemini-3-flash",
		},
		Info: map[string]runtime.ModelCapabilities{},
	})

	d.search.SetValue("sonnet")
	d.applyFilter()

	if len(d.filtered) != 1 || d.filtered[0] != "opencode/claude-sonnet-4-5" {
		t.Fatalf("expected 1 filtered model 'opencode/claude-sonnet-4-5', got %v", d.filtered)
	}

	frame := plainFrame(d.View().Content)
	if !strings.Contains(frame, "opencode/claude-sonnet-4-5") {
		t.Fatalf("expected search match to be rendered:\n%s", frame)
	}
	if strings.Contains(frame, "gemini-3-flash") {
		t.Fatalf("unmatched model should not be rendered:\n%s", frame)
	}
}

func TestModelDialogNavigationAndEnterSelection(t *testing.T) {
	d := NewModelDialogCmp()
	d.Update(ModelsLoadedMsg{
		Models: []string{
			"opencode/nemotron-3.5-lightning-free",
			"opencode/mimo-v2.5-free",
		},
		Info: map[string]runtime.ModelCapabilities{},
	})

	// Down arrow moves cursor to mimo
	d.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	cmp := d.(*modelDialogCmp)
	if cmp.cursor != 1 {
		t.Fatalf("expected cursor at 1, got %d", cmp.cursor)
	}

	// Enter selects item at cursor
	_, cmd := d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected selection command on Enter")
	}
	msg := cmd()
	selected, ok := msg.(ModelSelectedMsg)
	if !ok {
		t.Fatalf("expected ModelSelectedMsg, got %T", msg)
	}
	if selected.Model != "opencode/mimo-v2.5-free" {
		t.Fatalf("expected 'opencode/mimo-v2.5-free', got %q", selected.Model)
	}
}

// plainFrame strips the styling, which is what a reader sees on a terminal that
// cannot draw colour.
func plainFrame(frame string) string {
	var out strings.Builder
	inEscape := false
	for _, r := range frame {
		switch {
		case r == 0x1b:
			inEscape = true
		case inEscape && (r == 'm' || r == 'K'):
			inEscape = false
		case inEscape:
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}
