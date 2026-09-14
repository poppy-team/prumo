package handoff

import (
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

func TestBuildValidate(t *testing.T) {
	b, err := Build("native", "codex", agent.NativeAgentState{RunID: "R1", ContextManifestID: "ctx1"}, "rev-abc", "summary", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := b
	bad.Refs = map[string]string{"run": "R1"}
	if err := bad.Validate(); err == nil {
		t.Fatal("expected missing-ref error")
	}
}
