//go:build ignore

package tui

import (
	"context"
	"fmt"
	"testing"
	"time"

	prumo "github.com/raillen/prumo/sdk/prumo"
)

// The redraw budget (H10 criterion 5) is measured here and nowhere else. The
// method is ADR 009's: record a distribution on reference hardware, and do not
// invent a threshold until one exists. These benchmarks are the distribution.
//
// The property under test is not "the view is fast" but "the view costs what is
// *new*". A 200-row timeline makes the render cost visible; the poll benchmark
// shows that a tick with a full log and an advanced cursor returns no rows,
// which is what stops a long run from getting quadratically expensive.

const redrawRows = 200

func fillTimeline(timeline *Timeline, n int) {
	for i := 0; i < n; i++ {
		timeline.Append(ev(fmt.Sprintf("e%d", i), "text_delta", map[string]any{"text": fmt.Sprintf("line %d", i)}))
	}
}

// redrawModel returns a run panel holding n events, which is the state whose
// frame cost the criterion names.
func redrawModel(t testing.TB, n int) *Model {
	t.Helper()
	ops := &stubOps{}
	model := newTestModel(t, ops)
	session, err := StartSession(context.Background(), ops, StartConfig{Goal: "redraw baseline"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	model = update(t, model, sessionMsg{session: session})
	fillTimeline(model.Timeline, n)
	return model
}

func BenchmarkViewWith200Events(b *testing.B) {
	model := redrawModel(b, redrawRows)
	if model.Timeline.Len() != redrawRows {
		b.Fatalf("timeline has %d rows, want %d", model.Timeline.Len(), redrawRows)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.View()
	}
}

func BenchmarkTimelineAppend200(b *testing.B) {
	timeline := NewTimeline(DefaultTimelineCapacity)
	fillTimeline(timeline, redrawRows)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timeline.Append(ev("new", "text_delta", map[string]any{"text": "line"}))
	}
}

func BenchmarkSessionPoll200(b *testing.B) {
	events := make([]prumo.Event, 0, redrawRows)
	for i := 0; i < redrawRows; i++ {
		events = append(events, ev(fmt.Sprintf("e%d", i), "text_delta", map[string]any{"text": "line"}))
	}
	ops := &stubOps{events: events}
	session, err := StartSession(context.Background(), ops, StartConfig{Goal: "redraw baseline"})
	if err != nil {
		b.Fatalf("start: %v", err)
	}
	if _, err := session.Poll(context.Background()); err != nil {
		b.Fatalf("prime: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := session.Poll(context.Background()); err != nil {
			b.Fatalf("poll: %v", err)
		}
	}
}

// TestRedrawBaseline200 reports the numbers the ADR cites, so a run of the
// suite states the baseline instead of only asserting that it exists.
func TestRedrawBaseline200(t *testing.T) {
	model := redrawModel(t, redrawRows)
	const frames = 200
	start := time.Now()
	for i := 0; i < frames; i++ {
		_ = model.View()
	}
	elapsed := time.Since(start)
	t.Logf("redraw baseline: %d rows, %d frames in %s (%.3f ms/frame)",
		redrawRows, frames, elapsed, float64(elapsed.Microseconds())/float64(frames)/1000)
}
