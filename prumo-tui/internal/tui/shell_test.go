package tui

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/message"
	"github.com/raillen/prumo-tui/internal/session"
	"github.com/raillen/prumo-tui/internal/tui/components/chat"
)

// ansiPattern matches the escape sequences a terminal interprets. Removing them
// leaves what a reader sees on a terminal that cannot draw colour, which is the
// frame the accessibility evidence is about.
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

func plain(rendered string) string { return ansiPattern.ReplaceAllString(rendered, "") }

// shell builds the client over a stub harness: the shell is the subject here,
// not the daemon behind it.
func shell(t testing.TB, width, height int) tea.Model {
	t.Helper()
	application := app.New(app.Options{Client: &stubHarness{}})
	model, _ := New(application).Update(tea.WindowSizeMsg{Width: width, Height: height})
	return model
}

// drive runs a command tree the way the program's event loop does, bounded so a
// command that renews itself — the spinner's tick, for one — cannot loop the
// test forever.
func drive(t testing.TB, model tea.Model, cmd tea.Cmd, depth int) tea.Model {
	t.Helper()
	if cmd == nil || depth <= 0 {
		return model
	}
	msg := cmd()
	if msg == nil {
		return model
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, inner := range batch {
			model = drive(t, model, inner, depth-1)
		}
		return model
	}
	updated, next := model.Update(msg)
	return drive(t, updated, next, depth-1)
}

// conversation seeds a session and selects it, which is how a transcript
// reaches the screen: the store is filled first, then the selection loads it.
func conversation(t testing.TB, rows, width, height int) tea.Model {
	t.Helper()
	application := app.New(app.Options{Client: &stubHarness{}})
	ctx := context.Background()
	for i := 0; i < rows; i++ {
		role := message.User
		if i%2 == 1 {
			role = message.Assistant
		}
		if _, err := application.Messages.Create(ctx, "S1", message.CreateMessageParams{
			Role:  role,
			Parts: []message.ContentPart{message.TextContent{Text: fmt.Sprintf("row %d of the conversation", i)}},
		}); err != nil {
			t.Fatalf("cannot seed the conversation: %v", err)
		}
	}
	model, _ := New(application).Update(tea.WindowSizeMsg{Width: width, Height: height})
	selected, cmd := model.Update(chat.SessionSelectedMsg(session.Session{ID: "S1", Title: "S1"}))
	return drive(t, selected, cmd, 4)
}

// A frame is a function of state, not of when it was drawn. Golden frames and
// every comparison built on them depend on this, and so does a user who resizes
// a terminal and expects the same view back.
func TestRenderingIsDeterministic(t *testing.T) {
	const width, height = 100, 30
	first := conversation(t, 6, width, height).View().Content
	second := conversation(t, 6, width, height).View().Content
	if first != second {
		t.Fatalf("the same state drew two different frames:\n%q\n%q", first, second)
	}
}

// The interaction contract declares three size classes and forbids horizontal
// scrolling: content reflows or truncates, but a line never runs past the
// terminal it was drawn for.
func TestEveryWidthClassStaysInsideItsTerminal(t *testing.T) {
	// Compact is under 80, standard 80–119, wide from 120; 40 is below any of
	// them on purpose, because a terminal can be smaller than the spec's floor.
	for _, width := range []int{40, 60, 79, 80, 119, 120, 200} {
		width := width
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			frame := plain(shell(t, width, 30).View().Content)
			if strings.TrimSpace(frame) == "" {
				t.Fatal("the client drew nothing")
			}
			checkWidth(t, frame, width)
		})
	}
}

// A resize is a sequence of layouts, not one: a storm is what exposes a frame
// that was computed for the previous size.
func TestResizeStormKeepsEveryFrameInside(t *testing.T) {
	model := conversation(t, 8, 120, 30)
	for _, width := range []int{40, 200, 79, 121, 60, 80, 300, 45} {
		updated, cmd := model.Update(tea.WindowSizeMsg{Width: width, Height: 24})
		model = drive(t, updated, cmd, 3)
		checkWidth(t, plain(model.View().Content), width)
	}
}

func checkWidth(t *testing.T, frame string, width int) {
	t.Helper()
	for i, line := range strings.Split(frame, "\n") {
		if got := lipgloss.Width(line); got > width {
			t.Fatalf("line %d is %d columns wide in a %d-column terminal: %q", i, got, width, line)
		}
	}
}

// Colour is decoration: with the escape sequences gone the frame still says
// what happened, in words. A state whose only signal is a colour is a defect,
// because a terminal without colour is a supported terminal.
func TestTheFrameCarriesItsMeaningWithoutColour(t *testing.T) {
	frame := plain(conversation(t, 4, 100, 30).View().Content)
	for _, want := range []string{"row 0 of the conversation", "row 3 of the conversation"} {
		if !strings.Contains(frame, want) {
			t.Errorf("the transcript lost %q without colour", want)
		}
	}

	// The shell names itself and its model in words, so a reader knows where
	// they are without a single colour.
	empty := plain(shell(t, 100, 30).View().Content)
	for _, want := range []string{"Prumo", "fake"} {
		if !strings.Contains(empty, want) {
			t.Errorf("the shell lost %q without colour", want)
		}
	}
}

// A conversation that is still loading must say so in text, since a spinner is
// motion and motion is the first thing a reduced-motion terminal drops.
func TestLoadingIsStatedInWords(t *testing.T) {
	application := app.New(app.Options{Client: &stubHarness{}})
	model, _ := New(application).Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	selected, _ := model.Update(chat.SessionSelectedMsg(session.Session{ID: "S1", Title: "S1"}))
	frame := plain(selected.View().Content)
	if !strings.Contains(frame, "Loading") {
		t.Fatalf("the transcript never said it was loading: %q", frame)
	}
}
