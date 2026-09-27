package runtime

import (
	"context"
	"errors"
	"strings"
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
	// The policy's own evaluation is on the engine, but no human has answered:
	// the decision must still be the policy's ask, by no one but "policy".
	if res, decided := engine.Resolution("perm-c1", ""); !decided || res.Decision != agent.PermissionAsk || res.Actor != "policy" {
		t.Fatalf("no human decision should exist before the client answers: %+v", res)
	}
	// A request id that is not pending must be refused, not silently accepted.
	if err := r.ResolvePermission("perm-other", "some-fingerprint", true, "operator", ""); err == nil {
		t.Fatal("unknown request id must be refused")
	}
	// An answer must quote the fingerprint of what is actually pending.
	pendingID := r.State.PendingPerms[0]
	if err := r.ResolvePermission(pendingID, "", true, "operator", ""); err == nil {
		t.Fatal("an answer without a fingerprint must be refused")
	}
	if err := r.ResolvePermission(pendingID, "wrong-fingerprint", true, "operator", ""); err == nil {
		t.Fatal("an answer quoting the wrong fingerprint must be refused")
	}
	// Answer by the real path — no Phase hack — and the turn finishes.
	fingerprint := r.PendingFingerprint(pendingID)
	if fingerprint == "" {
		t.Fatal("a pending request must expose the fingerprint an approver is shown")
	}
	if err := r.ResolvePermission(pendingID, fingerprint, true, "operator", ""); err != nil {
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
	pendingID := r.State.PendingPerms[0]
	if err := r.ResolvePermission(pendingID, r.PendingFingerprint(pendingID), false, "operator", "outside the workspace"); err != nil {
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
	if err := r.ResolvePermission("perm-c1", "fp", true, "operator", ""); err == nil {
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

// A budget that is checked only after the usage arrives bounds nothing: the
// call is already paid for. The real tracker refuses the call instead, so this
// uses it rather than a stub hook that no longer exists.
func TestBudgetStopsARunBeforeTheCallItCannotAfford(t *testing.T) {
	fake := model.NewFake(map[string][]model.ScriptStep{"*": []model.ScriptStep{{Kind: "usage", Usage: &agent.Usage{InputTokens: 1000000, OutputTokens: 0}}, {Kind: "complete"}}})
	budget := &tinyBudget{limit: 10} // one call cannot fit
	r := NewRunner(Services{
		Models: fake, Tools: &stubTools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints:   checkpoint.New(t.TempDir()),
		ReserveBudget: budget.reserve, BudgetExhausted: budget.exhausted,
	}, "R5", "S1")
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}}
	err := r.RunUntilDone(context.Background())
	if err == nil {
		t.Fatal("a run that cannot afford its first call must fail")
	}
	if !strings.Contains(err.Error(), "budget") {
		t.Fatalf("the failure must name the budget, got %v", err)
	}
	if r.State.Phase != agent.PhaseFailed {
		t.Fatalf("phase = %s, want failed", r.State.Phase)
	}
}

// The allowance is a reservation, not a cap: what a call does not use goes back,
// so a run is stopped by the real spend rather than by the pessimism of the
// estimate.
func TestBudgetReleasesWhatACallDidNotUse(t *testing.T) {
	budget := &tinyBudget{limit: 100_000}
	release, err := budget.reserve(50_000)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	release(10, 0)
	if err := budget.exhausted(); err != nil {
		t.Fatalf("an unused reservation must be released, not held: %v", err)
	}
}

// tinyBudget is the reservation protocol in miniature. The real tracker lives
// in runlayer, which imports this package, so exercising it here would be an
// import cycle; what belongs to this test is that the runtime uses the protocol
// correctly, not how the tracker implements it.
type tinyBudget struct {
	limit   float64
	spent   float64
	held    float64
	failNow bool
}

func (b *tinyBudget) reserve(tokens float64) (func(float64, float64), error) {
	if b.failNow || b.spent+b.held+tokens > b.limit {
		b.failNow = true
		return nil, errors.New("hard budget exhausted for tokens")
	}
	b.held += tokens
	released := false
	return func(actual, _ float64) {
		if released {
			return
		}
		released = true
		b.held -= tokens
		b.spent += actual
	}, nil
}

func (b *tinyBudget) exhausted() error {
	if b.failNow {
		return errors.New("hard budget exhausted for tokens")
	}
	return nil
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

// A provider failure that the runtime swallows is worse than one it reports: the
// loop advances as though the model finished, so a partial reply stays in the
// transcript and the run is recorded as a success. The runtime used to fail only
// on non-retryable errors, on the assumption that something above would retry the
// rest — but the gateway is not in this path (GAP-102), so a 429, a 500 or a
// stream cut mid-answer was swallowed whole (GAP-131).

func TestARetryableProviderErrorFailsTheTurnInsteadOfBeingSwallowed(t *testing.T) {
	fake := model.NewFake(map[string][]model.ScriptStep{"*": {
		{Kind: "text", Text: "half an answer"},
		{Kind: "error", Error: "overloaded", Retryable: true},
	}})
	r := NewRunner(Services{
		Models: fake, Tools: &stubTools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: checkpoint.New(t.TempDir()),
	}, "R-err", "S1")
	r.MaxTurns = 2
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}}
	err := r.RunUntilDone(context.Background())
	if err == nil {
		t.Fatal("a run whose model call failed must not finish as a success")
	}
	if r.State.Phase != agent.PhaseFailed {
		t.Fatalf("phase = %s, want failed", r.State.Phase)
	}
	if !Retryable(err) {
		t.Errorf("the error must still be marked retryable so a caller can tell the provider's fault from ours: %v", err)
	}
	if !strings.Contains(err.Error(), "overloaded") {
		t.Errorf("the failure must name what the provider said, got %v", err)
	}
}

func TestANonRetryableProviderErrorIsNotMarkedRetryable(t *testing.T) {
	fake := model.NewFake(map[string][]model.ScriptStep{"*": {
		{Kind: "error", Error: "invalid tool call", Retryable: false},
	}})
	r := NewRunner(Services{
		Models: fake, Tools: &stubTools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: checkpoint.New(t.TempDir()),
	}, "R-err2", "S1")
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}}
	err := r.RunUntilDone(context.Background())
	if err == nil {
		t.Fatal("a failed model call must fail the run")
	}
	if Retryable(err) {
		t.Errorf("a provider error that is not retryable must not claim to be: %v", err)
	}
}

func TestAPartialReplyIsNotKeptAsThoughItWereTheAnswer(t *testing.T) {
	// The turn failed, so the half-answer must not be sitting in the transcript
	// labelled as the model's completed reply. Anything that reads the messages
	// later would otherwise present it as the result.
	fake := model.NewFake(map[string][]model.ScriptStep{"*": {
		{Kind: "text", Text: "half an answer"},
		{Kind: "error", Error: "overloaded", Retryable: true},
	}})
	r := NewRunner(Services{
		Models: fake, Tools: &stubTools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: checkpoint.New(t.TempDir()),
	}, "R-err3", "S1")
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}}
	if err := r.RunUntilDone(context.Background()); err == nil {
		t.Fatal("the run must fail")
	}
	for _, m := range r.Messages {
		if m.Role == agent.RoleAgent && m.Content == "half an answer" {
			t.Error("a failed turn's partial text must not be recorded as a finished agent reply")
		}
	}
}
