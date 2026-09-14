package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/model"
)

func TestInjectConsumedNextTurn(t *testing.T) {
	fake := model.NewFake(map[string][]model.ScriptStep{"*": {{Kind: "text", Text: "hi"}, {Kind: "complete"}}})
	r := NewRunner(Services{Models: fake, Tools: &stubTools{}}, "R-steer", "S")
	r.MaxTurns = 5
	if err := r.Step(context.Background()); err != nil { // prepare
		t.Fatal(err)
	}
	if err := r.Inject(agent.Message{ID: "m-steer", Role: agent.RoleUser, Content: "pivot"}); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range r.Messages {
		if m.ID == "m-steer" {
			found = true
		}
	}
	if !found {
		t.Fatal("injected message missing")
	}
}

func TestInjectRefusedWhenTerminal(t *testing.T) {
	r := NewRunner(Services{}, "R-term", "S")
	r.State.Phase = agent.PhaseComplete
	if err := r.Inject(agent.Message{ID: "m", Role: agent.RoleUser}); err == nil {
		t.Fatal("terminal run must refuse steer")
	}
}

func TestCompactKeepsRecent(t *testing.T) {
	r := NewRunner(Services{}, "R-compact", "S")
	r.CompactKeep = 2
	for i := 0; i < 10; i++ {
		role := agent.RoleUser
		if i%2 == 0 {
			role = agent.RoleTool
		}
		r.Messages = append(r.Messages, agent.Message{ID: string(rune('a' + i)), Role: role, Content: "m"})
	}
	r.mu.Lock()
	r.maybeCompactLocked()
	r.mu.Unlock()
	if len(r.Messages) != 3 || r.Messages[0].Role != agent.RoleSystem {
		t.Fatalf("expected summary + 2 kept, got %d", len(r.Messages))
	}
	if !strings.Contains(r.Messages[0].Content, "compacted 8 messages") {
		t.Fatalf("bad summary: %q", r.Messages[0].Content)
	}
	if r.Messages[2].ID != string(rune('a'+9)) {
		t.Fatal("most recent must be preserved")
	}
}
