package runtime

import (
	"context"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/model"
)

func BenchmarkRunUntilDone(b *testing.B) {
	fake := model.NewFake(map[string][]model.ScriptStep{"*": {{Kind: "text", Text: "hi"}, {Kind: "complete"}}})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewRunner(Services{Models: fake, Tools: &stubTools{}}, "R-bench", "S")
		r.Messages = []agent.Message{{ID: "m", Role: agent.RoleUser, Content: "hi"}}
		if err := r.RunUntilDone(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}
