package runtime

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/checkpoint"
	"github.com/raillen/prumo/internal/harness/model"
	"github.com/raillen/prumo/internal/harness/perm"
)

// A checkpoint has to carry what a continuation needs, not only where the state
// machine stopped.
//
// Before this, Checkpoint held State alone. The conversation, the ready tool
// queue, the observations already produced and the effect flag lived only on the
// Runner, so a resumed run had a phase and no history behind it. The CLI then
// reported the run as resumed and set its phase to complete, claiming a turn no
// model ever took (GAP-123).

func completingFake() model.Provider {
	return model.NewFake(map[string][]model.ScriptStep{
		"*": {{Kind: "text", Text: "continued"}, {Kind: "complete"}},
	})
}

func TestCheckpointCarriesTheConversation(t *testing.T) {
	store := checkpoint.New(t.TempDir())
	runner := NewRunner(Services{
		Models: completingFake(), Tools: &stubTools{},
		Perms:       perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: store,
	}, "R-conv", "S-1")
	runner.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "hello"}}
	// The runner checkpoints on the way out of a turn, so drive it there.
	if err := runner.RunUntilDone(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	cp, err := store.Latest("R-conv")
	if err != nil {
		t.Fatalf("latest checkpoint: %v", err)
	}
	if len(cp.Messages) == 0 {
		t.Fatal("a checkpoint must carry the conversation, otherwise it cannot be continued")
	}
	if !cp.Resumable() {
		t.Fatal("a checkpoint with a conversation must report itself resumable")
	}
}

func TestCheckpointCarriesToolQueueObservationsAndEffectFlag(t *testing.T) {
	store := checkpoint.New(t.TempDir())
	runner := NewRunner(Services{
		Models: completingFake(), Tools: &stubTools{},
		Perms:       perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: store,
	}, "R-state", "S-1")
	runner.ToolQ = []agent.ToolCall{{ID: "c1", Name: "git.status"}}
	runner.Obs = []agent.Observation{{ToolCallID: "c0", Content: "previous result"}}
	runner.AfterSideEffects = true
	runner.TurnsDone = 3
	if err := runner.persist(); err != nil {
		t.Fatalf("persist: %v", err)
	}

	cp, err := store.Latest("R-state")
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if len(cp.ToolQ) != 1 || cp.ToolQ[0].ID != "c1" {
		t.Fatalf("the ready tool queue must survive, got %+v", cp.ToolQ)
	}
	if len(cp.Obs) != 1 || cp.Obs[0].ToolCallID != "c0" {
		t.Fatalf("observations must survive, got %+v", cp.Obs)
	}
	if !cp.AfterSideEffects {
		t.Fatal("the side-effect flag must survive; a continuation must not fall back after an effect applied")
	}
	if cp.TurnsDone != 3 {
		t.Fatalf("turn count must survive, got %d", cp.TurnsDone)
	}
}

func TestRestoreFromRebuildsARunnableRunner(t *testing.T) {
	store := checkpoint.New(t.TempDir())
	runner := NewRunner(Services{
		Models: completingFake(), Tools: &stubTools{},
		Perms:       perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: store,
	}, "R-restore", "S-1")
	runner.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "hello"}}
	if err := runner.RunUntilDone(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	cp, err := store.Latest("R-restore")
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	// The run above completed, so its saved position carries StopReason
	// "completed" and would short-circuit straight back to Complete. Rewind the
	// position and clear the reason, which is what a checkpoint taken
	// mid-conversation actually looks like.
	cp.State.Phase = agent.PhaseCheckpoint
	cp.State.StopReason = ""

	restored := NewRunner(Services{
		Models: completingFake(), Tools: &stubTools{},
		Perms:       perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: store,
	}, "R-restore", "S-1")
	restored.MaxTurns = 2
	restored.RestoreFrom(cp)

	if len(restored.Messages) != len(cp.Messages) {
		t.Fatalf("restore lost the conversation: %d vs %d", len(restored.Messages), len(cp.Messages))
	}
	if restored.State.Phase != agent.PhaseCheckpoint {
		t.Fatalf("restore must put the runner back at the saved phase, got %s", restored.State.Phase)
	}

	// The restored runner must be runnable: stepping must advance the phase
	// machine without error and without re-losing the history. The exact
	// terminal phase depends on MaxTurns and the stop reason, which is the
	// phase machine's business and not what this test is about.
	if err := restored.Step(context.Background()); err != nil {
		t.Fatalf("a restored runner must be able to continue: %v", err)
	}
	if restored.State.Phase == agent.PhaseCheckpoint {
		t.Fatal("stepping a restored runner must advance past the saved safe point")
	}
	if len(restored.Messages) < len(cp.Messages) {
		t.Fatal("continuing must not discard the restored conversation")
	}
}

func TestCheckpointWithoutConversationIsNotResumable(t *testing.T) {
	// The shape an old checkpoint has: a phase and no history.
	legacy := agent.Checkpoint{ID: "R-legacy-r1", RunID: "R-legacy", State: agent.NativeAgentState{
		RunID: "R-legacy", Phase: agent.PhaseCheckpoint,
	}}
	if legacy.Resumable() {
		t.Fatal("a checkpoint with no conversation must not claim to be resumable")
	}
}

func TestRestoreDoesNotAliasTheCheckpointSlices(t *testing.T) {
	store := checkpoint.New(filepath.Join(t.TempDir(), "checkpoints"))
	runner := NewRunner(Services{
		Models: completingFake(), Tools: &stubTools{},
		Perms:       perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: store,
	}, "R-alias", "S-1")
	runner.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "original"}}
	if err := runner.persist(); err != nil {
		t.Fatalf("persist: %v", err)
	}
	cp, err := store.Latest("R-alias")
	if err != nil {
		t.Fatalf("latest: %v", err)
	}

	restored := NewRunner(Services{
		Models: completingFake(), Tools: &stubTools{},
		Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
	}, "R-alias", "S-1")
	restored.RestoreFrom(cp)
	restored.Messages[0].Content = "mutated"

	if cp.Messages[0].Content != "original" {
		t.Fatal("restore must copy the conversation, not alias the checkpoint's slice")
	}
}
