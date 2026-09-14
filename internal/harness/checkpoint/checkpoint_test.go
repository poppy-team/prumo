package checkpoint

import (
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

func TestSaveLoadLatest(t *testing.T) {
	s := New(t.TempDir())
	cp := agent.Checkpoint{ID: "cp1", RunID: "R1", State: agent.NativeAgentState{RunID: "R1", Phase: agent.PhaseCheckpoint}}
	if err := s.Save(cp); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load("cp1")
	if err != nil || got.RunID != "R1" {
		t.Fatalf("load failed: %v %+v", err, got)
	}
	latest, err := s.Latest("R1")
	if err != nil || latest.ID != "cp1" {
		t.Fatalf("latest failed: %v", err)
	}
}

func TestIdempotencyPreventsDuplicate(t *testing.T) {
	s := New(t.TempDir())
	fx := agent.PendingEffect{ID: "fx1", Kind: "edit", Status: agent.EffectPending, IdempotencyKey: "k1", Target: "a.go"}
	ok, err := s.RecordIntent(fx)
	if err != nil || !ok {
		t.Fatalf("first intent must proceed: %v %v", ok, err)
	}
	if err := s.RecordOutcome("fx1", agent.EffectApplied); err != nil {
		t.Fatal(err)
	}
	// Replay with same key but new id must be skipped.
	ok, err = s.RecordIntent(agent.PendingEffect{ID: "fx2", Kind: "edit", IdempotencyKey: "k1"})
	if err != nil || ok {
		t.Fatalf("duplicate side effect must be skipped, got ok=%v err=%v", ok, err)
	}
}
