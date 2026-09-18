package runtime

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/raillen/prumo-tui/internal/llm/models"
)

// eventLog builds a run log of the size the performance requirement names: a
// short run, a long one, and one large enough that an unbounded fold shows.
func eventLog(size int) *timelineClient {
	client := &timelineClient{status: "complete"}
	for i := 0; i < size; i++ {
		switch i % 4 {
		case 0:
			client.append(Event{Kind: "text_delta", Payload: map[string]any{"text": "token "}})
		case 1:
			client.append(Event{Kind: "reasoning_delta", Payload: map[string]any{"text": "thinking "}})
		case 2:
			client.append(Event{Kind: "tool_call_ready", Payload: map[string]any{"id": fmt.Sprintf("tc-%d", i), "name": "fs.read"}})
		default:
			client.append(Event{Kind: "file.changed", Payload: map[string]any{"path": "internal/x.go", "operation": "modified"}})
		}
	}
	return client
}

// BenchmarkFoldTimeline is what reading a run costs the client: the poll loop
// folds the log into the conversation on every tick, so this is the per-poll
// cost at the sizes the requirement names.
func BenchmarkFoldTimeline(b *testing.B) {
	for _, size := range []int{200, 2000, 20000} {
		b.Run(fmt.Sprintf("events_%d", size), func(b *testing.B) {
			client := eventLog(size)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				r := newTestRunner(client, models.Model{})
				out, err := r.Run(context.Background(), "S1", "say hello")
				if err != nil {
					b.Fatalf("run: %v", err)
				}
				deadline := time.After(60 * time.Second)
				for {
					select {
					case _, ok := <-out:
						if !ok {
							goto done
						}
					case <-deadline:
						b.Fatal("the run never finished")
					}
				}
			done:
			}
		})
	}
}
