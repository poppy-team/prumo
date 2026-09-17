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
	results    map[string]agent.ToolResult
	kinds      map[string]string
	operations map[string]string
	calls      []string
}

func (s *stubTools) Execute(_ context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	s.calls = append(s.calls, call.Name)
	if r, ok := s.results[call.Name]; ok {
		r.ToolCallID = call.ID
		return r, nil
	}
	return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: "ok"}, nil
}
func (s *stubTools) OperationOf(name string) string { return s.operations[name] }
func (s *stubTools) KindOf(name string) string      { return s.kinds[name] }

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
	tools := &stubTools{}
	dir := t.TempDir()
	r := NewRunner(Services{Models: fake, Tools: tools, Perms: engine, Checkpoints: checkpoint.New(dir)}, "R4", "S1")
	r.MaxTurns = 1
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.State.Phase != agent.PhaseYield {
		t.Fatalf("expected yield on ask, got %s", r.State.Phase)
	}
	// A run waiting for approval must be recoverable, and must carry the call
	// it stopped on: a client cannot answer a request it cannot name.
	if len(r.State.PendingPerms) != 1 || len(r.State.PendingTools) != 1 {
		t.Fatalf("pending state not carried: perms=%v tools=%v", r.State.PendingPerms, r.State.PendingTools)
	}
	cp, err := checkpoint.New(dir).Latest("R4")
	if err != nil {
		t.Fatalf("a yield for approval must leave a checkpoint: %v", err)
	}
	if len(cp.State.PendingPerms) != 1 || len(cp.State.PendingTools) != 1 {
		t.Fatalf("checkpoint lost the pending state: %+v", cp.State)
	}
	// Nothing decided yet: the pending request has no resolution on the engine.
	if _, decided := engine.Resolution("perm-c1"); decided {
		t.Fatal("no decision should exist before the client answers")
	}
	// A request id that is not pending must be refused, not silently accepted.
	if err := r.ResolvePermission("perm-other", true, "operator", ""); err == nil {
		t.Fatal("unknown request id must be refused")
	}
	// Answer by the real path — no Phase hack — and the turn finishes.
	if err := r.ResolvePermission(r.State.PendingPerms[0], true, "operator", ""); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(tools.calls) != 1 || tools.calls[0] != "edit.patch" {
		t.Fatalf("approved tool did not execute: %v (phase %s)", tools.calls, r.State.Phase)
	}
	if r.State.Phase != agent.PhaseComplete {
		t.Fatalf("phase after approval = %s, want complete", r.State.Phase)
	}
}

func TestResolvePermissionRejectionFailsTheRun(t *testing.T) {
	fake := model.NewFake(map[string][]model.ScriptStep{"*": []model.ScriptStep{{Kind: "tool_call", Tool: &agent.ToolCall{ID: "c1", Name: "edit.delete", IdempotencyKey: "k"}}, {Kind: "complete"}}})
	tools := &stubTools{}
	r := NewRunner(Services{
		Models: fake, Tools: tools,
		Perms:       perm.New(perm.Policy{DefaultAction: agent.PermissionAsk}),
		Checkpoints: checkpoint.New(t.TempDir()),
	}, "R7", "S1")
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.ResolvePermission(r.State.PendingPerms[0], false, "operator", "outside the workspace"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	if err := r.RunUntilDone(context.Background()); err == nil {
		t.Fatal("a rejected permission must fail the run")
	}
	if r.State.Phase != agent.PhaseFailed {
		t.Fatalf("phase after rejection = %s, want failed", r.State.Phase)
	}
	if len(tools.calls) != 0 {
		t.Fatalf("a rejected tool must not execute: %v", tools.calls)
	}
}

func TestResolvePermissionRequiresAPendingRequest(t *testing.T) {
	r := NewRunner(Services{Perms: perm.New(perm.Policy{})}, "R8", "S1")
	if err := r.ResolvePermission("perm-c1", true, "operator", ""); err == nil {
		t.Fatal("answering with nothing pending must be refused")
	}
}

// TestFileChangeIsReportedAsAnEvent is the vocabulary a client needs to show
// what a run touched: the change is a fact about the run, so it belongs on the
// timeline rather than only in the daemon's private side-effect journal.
func TestFileChangeIsReportedAsAnEvent(t *testing.T) {
	fake := model.NewFake(map[string][]model.ScriptStep{"*": {
		{Kind: "tool_call", Tool: &agent.ToolCall{
			ID: "c1", Name: "edit.patch",
			Arguments: map[string]any{"path": "internal/x.go"}, IdempotencyKey: "k",
		}},
		{Kind: "complete"},
	}})
	tools := &stubTools{
		kinds:      map[string]string{"edit.patch": "side-effecting"},
		operations: map[string]string{"edit.patch": "modified"},
	}
	var events []agent.AgentEvent
	r := NewRunner(Services{
		Models: fake, Tools: tools,
		Perms:       perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: checkpoint.New(t.TempDir()),
		Events:      func(ev agent.AgentEvent) { events = append(events, ev) },
	}, "R-file", "S1")
	r.MaxTurns = 1
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "patch it"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatal(err)
	}

	var changed *agent.AgentEvent
	kinds := make([]string, 0, len(events))
	for i := range events {
		kinds = append(kinds, events[i].Kind)
		if events[i].Kind == "file.changed" {
			changed = &events[i]
		}
	}
	if changed == nil {
		t.Fatalf("no file.changed event among %v", kinds)
	}
	if changed.Payload["path"] != "internal/x.go" {
		t.Fatalf("the event does not name the file it reports: %v", changed.Payload)
	}
	if changed.Payload["operation"] != "modified" {
		t.Fatalf("the event does not name the operation: %v", changed.Payload)
	}
}

// TestToolWithoutFileOperationReportsNoChange keeps the event honest: a tool
// that cannot name a change reports none. Decorating every call would make the
// event say nothing.
func TestToolWithoutFileOperationReportsNoChange(t *testing.T) {
	fake := model.NewFake(map[string][]model.ScriptStep{"*": {
		{Kind: "tool_call", Tool: &agent.ToolCall{
			ID: "c1", Name: "process.exec",
			Arguments: map[string]any{"command": "true"}, IdempotencyKey: "k",
		}},
		{Kind: "complete"},
	}})
	tools := &stubTools{kinds: map[string]string{"process.exec": "side-effecting"}}
	var events []agent.AgentEvent
	r := NewRunner(Services{
		Models: fake, Tools: tools,
		Perms:       perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: checkpoint.New(t.TempDir()),
		Events:      func(ev agent.AgentEvent) { events = append(events, ev) },
	}, "R-exec", "S1")
	r.MaxTurns = 1
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "run it"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, ev := range events {
		if ev.Kind == "file.changed" {
			t.Fatalf("a command that cannot name its file change must not claim one: %v", ev.Payload)
		}
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
