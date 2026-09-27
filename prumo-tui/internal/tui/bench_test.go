package tui

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// The costs below are the client's own, separate from the latency of whatever
// serves the run: a redraw that stalls is a client defect even when the model
// answered instantly.

// BenchmarkRenderConversation is the cost of one frame with the transcript
// already folded.
//
// The sizes are in messages rather than events deliberately: the event counts
// the requirement names are benchmarked where they are folded (the runtime
// package), and a hundred events make one message. Two thousand messages is
// already a conversation longer than any single run.
func BenchmarkRenderConversation(b *testing.B) {
	for _, rows := range []int{200, 2000} {
		b.Run(fmt.Sprintf("messages_%d", rows), func(b *testing.B) {
			model := conversation(b, rows, 120, 40)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = model.View().Content
			}
		})
	}
}

// BenchmarkResizeStorm is a terminal being dragged: every width is a relayout,
// and a client that recomputes the whole transcript per frame shows up here.
func BenchmarkResizeStorm(b *testing.B) {
	model := conversation(b, 2000, 120, 40)
	widths := []int{40, 200, 80, 120}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		updated, cmd := model.Update(tea.WindowSizeMsg{Width: widths[i%len(widths)], Height: 40})
		model = drive(b, updated, cmd, 2)
	}
}
