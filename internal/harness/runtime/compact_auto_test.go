package runtime

import (
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

func TestAutoCompactOnBudget(t *testing.T) {
	r := NewRunner(Services{}, "R-auto", "S")
	r.CompactBudget = 10 // tiny: any real conversation exceeds it
	for i := 0; i < 30; i++ {
		r.Messages = append(r.Messages, agent.Message{ID: string(rune('a' + i)), Role: agent.RoleTool, Content: strings.Repeat("x", 100)})
	}
	r.mu.Lock()
	r.maybeCompactLocked()
	n := len(r.Messages)
	r.mu.Unlock()
	if n >= 30 || r.Messages[0].Role != agent.RoleSystem {
		t.Fatalf("budget must trigger compaction, got %d messages", n)
	}
}
