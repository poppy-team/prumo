package checkpoint

import (
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

func TestPruneKeepsNewest(t *testing.T) {
	s := New(t.TempDir())
	for i := 0; i < 5; i++ {
		cp := agent.Checkpoint{ID: string(rune('a' + i)), RunID: "R", State: agent.NativeAgentState{RunID: "R"}}
		if err := s.Save(cp); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := s.Prune(2)
	if err != nil || removed != 3 {
		t.Fatalf("expected 3 pruned, got %d %v", removed, err)
	}
	if _, err := s.Latest("R"); err != nil {
		t.Fatalf("latest must survive prune: %v", err)
	}
	removed, err = s.Prune(5)
	if err != nil || removed != 0 {
		t.Fatalf("nothing to prune: %d %v", removed, err)
	}
}
