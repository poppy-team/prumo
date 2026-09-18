package dialog

import (
	"strings"
	"testing"

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
