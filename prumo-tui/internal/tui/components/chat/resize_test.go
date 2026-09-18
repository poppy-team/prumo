package chat

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/message"
	"github.com/raillen/prumo-tui/internal/session"
)

func newMessages(t *testing.T, rows int) *messagesCmp {
	t.Helper()
	application := app.New(app.Options{})
	cmp := NewMessagesCmp(application).(*messagesCmp)
	// The opening size settles like any other, so a test that changes only the
	// height is testing what it says it is.
	cmp.SetSize(100, 30)
	cmp.Update(resizeSettledMsg{width: 100})
	for i := 0; i < rows; i++ {
		if _, err := application.Messages.Create(t.Context(), "S1", message.CreateMessageParams{
			Role:  message.User,
			Parts: []message.ContentPart{message.TextContent{Text: "row"}},
		}); err != nil {
			t.Fatalf("cannot seed: %v", err)
		}
	}
	// Selecting a session renders in a command, so the test runs it: without it
	// nothing is drawn and every assertion below would be about an empty view.
	if cmd := cmp.SetSession(session.Session{ID: "S1", Title: "S1"}); cmd != nil {
		cmp.Update(cmd())
	}
	return cmp
}

// A drag emits a size per frame. Every one of them invalidates every rendered
// row, so a client that renders on each one re-renders the transcript dozens of
// times for a frame each — and only the last of those frames is ever seen.
func TestAStormOfSizesSettlesIntoOneRender(t *testing.T) {
	cmp := newMessages(t, 20)

	// The drag: three sizes in a row, each asking for a render it should not get.
	var last tea.Cmd
	for _, width := range []int{60, 80, 120} {
		last = cmp.SetSize(width, 30)
		if cmp.pendingWidth != width {
			t.Fatalf("the width %d was applied without waiting for the terminal to stop", width)
		}
	}

	// The settles that were overtaken do nothing.
	for _, stale := range []int{60, 80} {
		cmp.Update(resizeSettledMsg{width: stale})
		if cmp.pendingWidth == 0 {
			t.Fatalf("a settle for width %d was treated as the last word", stale)
		}
	}

	// The last one renders, and only then is the transcript current.
	if last == nil {
		t.Fatal("the terminal changed size and nothing was scheduled")
	}
	cmp.Update(resizeSettledMsg{width: 120})
	if cmp.pendingWidth != 0 {
		t.Fatal("the last settle did not clear the pending width")
	}
	// Every row that is drawn was rendered for the width the terminal settled
	// on, which the cache records per message.
	for id, cached := range cmp.cachedContent {
		if cached.width != 120 {
			t.Fatalf("message %s is still cached for width %d", id, cached.width)
		}
	}
}

// A height change is not a width change: the rows are already drawn, and
// re-rendering them for one more visible line is work nobody sees.
func TestOnlyAWidthChangeSchedulesARender(t *testing.T) {
	cmp := newMessages(t, 5)

	if cmd := cmp.SetSize(100, 40); cmd != nil {
		t.Fatal("a taller terminal asked for the transcript to be re-rendered")
	}
	if cmp.pendingWidth != 0 {
		t.Fatal("a height change left a width waiting")
	}
}

// Rendering every message to show forty rows is work nobody sees: what is drawn
// is the tail, and the rest arrives when a reader scrolls to it.
func TestALongTranscriptIsRenderedFromItsEnd(t *testing.T) {
	cmp := newMessages(t, 400)

	if cmp.renderedFrom == 0 {
		t.Fatal("the whole conversation was rendered to show a window of it")
	}
	// What is drawn is bounded by the budget, not by the conversation.
	if limit := cmp.renderBudget() + 4; len(cmp.uiMessages) > limit {
		t.Fatalf("drew %d rows for a budget of %d", len(cmp.uiMessages), limit)
	}
	// And the newest message is among them: it is what a reader came for.
	newest := cmp.messages[len(cmp.messages)-1]
	if _, cached := cmp.cachedContent[newest.ID]; !cached {
		t.Fatal("the newest message was not rendered")
	}
}

// Reaching the top of what is drawn is a reader asking for what is above it.
func TestScrollingToTheTopRendersWhatIsAbove(t *testing.T) {
	cmp := newMessages(t, 400)
	before := cmp.renderedFrom

	for i := 0; i < 200 && cmp.renderedFrom > 0; i++ {
		cmp.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	}

	if cmp.renderedFrom >= before {
		t.Fatalf("scrolling to the top left the transcript starting at %d", cmp.renderedFrom)
	}
	// The reader keeps their place: the rows that were added are above them.
	if cmp.viewport.YOffset() == 0 && cmp.renderedFrom > 0 {
		t.Fatal("more was rendered and the view did not move with it")
	}
}
