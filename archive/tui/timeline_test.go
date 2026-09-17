package tui

import (
	"strings"
	"testing"

	prumo "github.com/raillen/prumo/sdk/prumo"
)

func ev(id, kind string, payload map[string]any) prumo.Event {
	return prumo.Event{ID: id, RunID: "R1", Kind: kind, Payload: payload}
}

func TestTimelineBoundsAndKeepsStableIndices(t *testing.T) {
	timeline := NewTimeline(3)
	for i := 0; i < 10; i++ {
		timeline.Append(ev("e"+string(rune('a'+i)), "run.started", nil))
	}
	entries := timeline.Entries()
	if len(entries) != 3 {
		t.Fatalf("capacity 3 kept %d rows", len(entries))
	}
	// Numbering continues past the window: a reader must be able to tell that
	// rows were dropped, not be shown a timeline that silently restarts at 1.
	if entries[0].Index != 8 || entries[2].Index != 10 {
		t.Fatalf("indices not continuous past the window: %d..%d", entries[0].Index, entries[2].Index)
	}
	if timeline.Dropped() != 7 {
		t.Fatalf("dropped = %d, want 7", timeline.Dropped())
	}
	if timeline.Len() != 3 {
		t.Fatalf("Len = %d, want 3", timeline.Len())
	}
}

func TestTimelineSeverityVocabulary(t *testing.T) {
	cases := []struct {
		kind string
		want Severity
	}{
		{"run.finished", SeveritySuccess},
		{"permission_denied", SeverityError},
		{"permission_wait", SeverityWarning},
		{"error", SeverityError},
		{"warning", SeverityWarning},
		{"text_delta", SeverityInfo},
		{"something.the.daemon.invented", SeverityInfo},
	}
	for _, tc := range cases {
		if got := severityOf(tc.kind); got != tc.want {
			t.Errorf("severityOf(%q) = %q, want %q", tc.kind, got, tc.want)
		}
	}
}

func TestTimelineDetailIsDeterministic(t *testing.T) {
	payload := map[string]any{"tokens": float64(1200), "level": "L1", "pressure": "healthy"}
	first := detailOf(ev("a", "context.compiled", payload))
	second := detailOf(ev("b", "context.compiled", payload))
	if first != second {
		t.Fatalf("same payload rendered differently:\n  %s\n  %s", first, second)
	}
	// Keys are sorted, so the order is a property of the data rather than of
	// Go's map iteration.
	if !strings.HasPrefix(first, "level=L1") {
		t.Fatalf("keys are not sorted: %s", first)
	}
	if !strings.Contains(first, "tokens=1200") {
		t.Fatalf("integer payload should render without a decimal point: %s", first)
	}
}

func TestTimelineTitlePrefersHumanField(t *testing.T) {
	// A multi-line model delta must not grow the row to fill the screen.
	entry := func() Entry {
		tl := NewTimeline(4)
		tl.Append(ev("e", "text_delta", map[string]any{"text": "first line\nsecond line\nthird"}))
		last, _ := tl.Last()
		return last
	}()
	if entry.Title != "first line" {
		t.Fatalf("title = %q, want the first line only", entry.Title)
	}
	if strings.Contains(entry.Detail, "text=") {
		t.Fatalf("detail repeats the title: %q", entry.Detail)
	}
}

func TestTimelineReportsContextCompiled(t *testing.T) {
	tl := NewTimeline(4)
	tl.Append(ev("e", "context.compiled", map[string]any{"tokens": float64(42)}))
	last, ok := tl.Last()
	if !ok {
		t.Fatal("no last entry")
	}
	if !strings.Contains(last.Title, "42") {
		t.Fatalf("compiled context title should carry the token count: %q", last.Title)
	}
}
