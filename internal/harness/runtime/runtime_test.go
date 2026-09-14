package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/checkpoint"
	"github.com/raillen/prumo/internal/harness/model"
	"github.com/raillen/prumo/internal/harness/perm"
)

type stubTools struct {
	results map[string]agent.ToolResult
	kinds   map[string]string
	calls   []string
}

func (s *stubTools) Execute(_ context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	s.calls = append(s.calls, call.Name)
	if r, ok := s.results[call.Name]; ok {
		r.ToolCallID = call.ID
		return r, nil
	}
	return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: "ok"}, nil
}
func (s *stubTools) KindOf(name string) string { return s.kinds[name] }

func TestSimpleCompletion(t *testing.T) {
	fake := model.NewFake(map[string][]model.ScriptStep{"*": []model.ScriptStep{{Kind: "text", Text: "hi"}, {Kind: "complete"}}})
	r := NewRunner(Services{Models: fake, Tools: &stubTools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}), Checkpoints: checkpoint.New(t.TempDir())}, "R1", "S1")
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "hello"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.State.Phase != agent.PhaseComplete {
		t.Fatalf("expected complete, got %s (%s)", r.State.Phase, r.State.StopReason)
	}
}

func TestToolTrajectory(t *testing.T) {
	fake := model.NewFake(map[string][]model.ScriptStep{
		"*": {
			{Kind: "tool_call", Tool: &agent.ToolCall{ID: "c1", TurnID: "turn-1", Name: "fs.read", IdempotencyKey: "k1"}},
			{Kind: "complete"},
		},
	})
	tools := &stubTools{results: map[string]agent.ToolResult{"fs.read": {ExitCode: 0, Output: "file contents"}}}
	// Second turn completes: swap script after first request by keying on nothing;
	// instead allow max 1 tool turn then complete via MaxTurns.
	r := NewRunner(Services{Models: fake, Tools: tools, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}), Checkpoints: checkpoint.New(t.TempDir())}, "R2", "S1")
	r.MaxTurns = 1
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "read"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(tools.calls) != 1 || tools.calls[0] != "fs.read" {
		t.Fatalf("expected fs.read execution, got %v", tools.calls)
	}
}

func TestPermissionDenial(t *testing.T) {
	fake := model.NewFake(map[string][]model.ScriptStep{"*": []model.ScriptStep{{Kind: "tool_call", Tool: &agent.ToolCall{ID: "c1", Name: "edit.patch", IdempotencyKey: "k"}}, {Kind: "complete"}}})
	r := NewRunner(Services{
		Models: fake, Tools: &stubTools{kinds: map[string]string{"edit.patch": "destructive"}},
		Perms:       perm.New(perm.Policy{DefaultAction: agent.PermissionDeny}),
		Checkpoints: checkpoint.New(t.TempDir()),
	}, "R3", "S1")
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}}
	err := r.RunUntilDone(context.Background())
	if err == nil {
		t.Fatal("expected permission denial error")
	}
	if r.State.Phase != agent.PhaseFailed {
		t.Fatalf("expected failed, got %s", r.State.Phase)
	}
}

func TestPermissionResume(t *testing.T) {
	fake := model.NewFake(map[string][]model.ScriptStep{"*": []model.ScriptStep{{Kind: "tool_call", Tool: &agent.ToolCall{ID: "c1", Name: "edit.patch", IdempotencyKey: "k"}}, {Kind: "complete"}}})
	engine := perm.New(perm.Policy{DefaultAction: agent.PermissionAsk})
	r := NewRunner(Services{Models: fake, Tools: &stubTools{}, Perms: engine, Checkpoints: checkpoint.New(t.TempDir())}, "R4", "S1")
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.State.Phase != agent.PhaseYield {
		t.Fatalf("expected yield on ask, got %s", r.State.Phase)
	}
	// Approver allows; re-run with allow policy resumes to execute.
	r.Svc.Perms = perm.New(perm.Policy{DefaultAction: agent.PermissionAllow})
	r.State.Phase = agent.PhaseExecuteTool
	r.MaxTurns = 1
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestBudgetExhaustion(t *testing.T) {
	fake := model.NewFake(map[string][]model.ScriptStep{"*": []model.ScriptStep{{Kind: "usage", Usage: &agent.Usage{InputTokens: 1000000, OutputTokens: 0}}, {Kind: "complete"}}})
	r := NewRunner(Services{
		Models: fake, Tools: &stubTools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints:   checkpoint.New(t.TempDir()),
		ConsumeBudget: func(u agent.Usage) error { return errors.New("hard budget exhausted") },
	}, "R5", "S1")
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}}
	err := r.RunUntilDone(context.Background())
	if err == nil {
		t.Fatal("expected budget error")
	}
}

func TestCheckpointResume(t *testing.T) {
	dir := t.TempDir()
	fake := model.NewFake(map[string][]model.ScriptStep{"*": []model.ScriptStep{{Kind: "text", Text: "hi"}, {Kind: "complete"}}})
	r := NewRunner(Services{Models: fake, Tools: &stubTools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}), Checkpoints: checkpoint.New(dir)}, "R6", "S1")
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Simulate kill: reload latest checkpoint and resume in a new Runner.
	store := checkpoint.New(dir)
	cp, err := store.Latest("R6")
	if err != nil {
		t.Fatal(err)
	}
	r2 := NewRunner(Services{Models: fake, Tools: &stubTools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}), Checkpoints: store}, "R6", "S1")
	r2.State = cp.State
	if r2.State.Phase != agent.PhaseComplete && r2.State.Phase != agent.PhaseYield && r2.State.Phase != agent.PhaseCheckpoint {
		t.Fatalf("resumed state should be terminal-ish, got %s", r2.State.Phase)
	}
}
