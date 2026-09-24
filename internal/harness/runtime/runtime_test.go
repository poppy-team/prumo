package runtime

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/checkpoint"
	"github.com/raillen/prumo/internal/harness/gateway"
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

// The gateway was built with selection, fallback, retry, a circuit breaker and a
// quota filter, and the runner was handed the raw adapter — so a run reached
// exactly one provider and none of that machinery ever executed. Every fix to it
// was a fix to code no run touched (GAP-102).

func TestTheRunnerRoutesThroughTheGatewayWhenGivenOne(t *testing.T) {
	// Selection is deterministic by provider name, so the provider that fails is
	// named to come first: this test is about the fallback happening, not about
	// the order.
	g := gateway.New()
	primary := &recordingProvider{name: "a-failing", events: []agent.ModelEvent{
		{Kind: agent.EventError, Error: "rate limited", Retryable: true, Quota: true},
	}}
	backup := &recordingProvider{name: "b-working", events: []agent.ModelEvent{
		{Kind: agent.EventTextDelta, Text: "answered by the backup"},
		{Kind: agent.EventCompleted, Finished: true},
	}}
	g.Register(primary)
	g.Register(backup)
	g.DeclareTarget(gateway.RouteTarget{Provider: "a-failing", Model: "m1", Privacy: "external"})
	g.DeclareTarget(gateway.RouteTarget{Provider: "b-working", Model: "m1", Privacy: "external"})
	g.Retry = gateway.RetryPolicy{Attempts: 1}

	r := NewRunner(Services{
		Models: g, Tools: &stubTools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: checkpoint.New(t.TempDir()),
	}, "R-route", "S1")
	r.MaxTurns = 1
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatalf("the run must recover by routing to the backup: %v", err)
	}
	if primary.calls == 0 {
		t.Error("the primary was never called: the runner bypassed the gateway")
	}
	if backup.calls == 0 {
		t.Error("the backup was never called: no fallback happened")
	}
}

func TestTheRunnerRefusesTransparentFallbackAfterASideEffect(t *testing.T) {
	// Swapping providers mid-run after an observable effect is not a retry: the
	// next provider has not seen the tool results and re-plans from a different
	// state. The run has to hand off, so the call fails loudly instead.
	g := gateway.New()
	primary := &recordingProvider{name: "a-failing", events: []agent.ModelEvent{
		{Kind: agent.EventError, Error: "boom", Retryable: true},
	}}
	backup := &recordingProvider{name: "b-working", events: []agent.ModelEvent{
		{Kind: agent.EventCompleted, Finished: true},
	}}
	g.Register(primary)
	g.Register(backup)
	g.DeclareTarget(gateway.RouteTarget{Provider: "a-failing", Model: "m1", Privacy: "external"})
	g.DeclareTarget(gateway.RouteTarget{Provider: "b-working", Model: "m1", Privacy: "external"})
	g.Retry = gateway.RetryPolicy{Attempts: 1}

	r := NewRunner(Services{
		Models: g, Tools: &stubTools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: checkpoint.New(t.TempDir()),
	}, "R-effect", "S1")
	r.MaxTurns = 2
	r.AfterSideEffects = true
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}}
	err := r.RunUntilDone(context.Background())
	if err == nil {
		t.Fatal("a failed call after a side effect must not be silently routed elsewhere")
	}
	if backup.calls != 0 {
		t.Error("the backup was called after the run had already had an observable effect")
	}
}

func TestAGatewayWithNothingRegisteredFailsRatherThanSilentlySucceeding(t *testing.T) {
	g := gateway.New()
	r := NewRunner(Services{
		Models: g, Tools: &stubTools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: checkpoint.New(t.TempDir()),
	}, "R-empty", "S1")
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}}
	if err := r.RunUntilDone(context.Background()); err == nil {
		t.Fatal("a gateway with no provider must fail the run, not report an empty answer as success")
	}
}

// recordingProvider replays fixed events and counts its calls.
type recordingProvider struct {
	name   string
	events []agent.ModelEvent
	calls  int
}

func (p *recordingProvider) Name() string { return p.name }
func (p *recordingProvider) Capabilities() model.Capabilities {
	return model.Capabilities{Streaming: true}
}
func (p *recordingProvider) Models(context.Context) ([]string, error) {
	return []string{"m1"}, nil
}
func (p *recordingProvider) Health(context.Context) (string, error) { return "healthy", nil }
func (p *recordingProvider) Stream(_ context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	p.calls++
	ch := make(chan agent.ModelEvent, len(p.events))
	for _, ev := range p.events {
		ev.RequestID = req.RequestID
		ch <- ev
	}
	close(ch)
	return ch, nil
}

// Every model request in a run carried the same turn and the same request id,
// because TurnID was set once when the runner was built and never moved. Since a
// tool call's idempotency key is derived from the request id, two different turns
// calling the same tool collided on it, and every observation in the run carried
// the same turn, so nothing could say which turn produced what (GAP-117).

func TestEachTurnGetsItsOwnIdentity(t *testing.T) {
	// Observed at the provider rather than through a runtime event, because the
	// request id is what the provider sees and that is what has to be distinct.
	recorder := &recordingRequestProvider{steps: []agent.ModelEvent{
		{Kind: agent.EventToolCallReady, ToolCall: &agent.ToolCall{
			ID: "c1", TurnID: "turn-1", Name: "fs.read",
			Arguments: map[string]any{"path": "README.md"},
		}},
		{Kind: agent.EventCompleted, Finished: true},
	}}
	r := NewRunner(Services{
		Models: recorder, Tools: &stubTools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: checkpoint.New(t.TempDir()),
	}, "R-turns", "S1")
	r.MaxTurns = 3
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}}
	_ = r.RunUntilDone(context.Background())

	if r.TurnsDone < 2 {
		t.Fatalf("the run took %d turn(s); this test needs at least two to observe a repeated id", r.TurnsDone)
	}
	if len(recorder.requestIDs) < 2 {
		t.Fatalf("saw %d model requests, want at least 2: %v", len(recorder.requestIDs), recorder.requestIDs)
	}
	seen := map[string]bool{}
	for i, id := range recorder.requestIDs {
		if seen[id] {
			t.Fatalf("request id %q was reused on turn %d: %v", id, i+1, recorder.requestIDs)
		}
		seen[id] = true
	}
	for i, turn := range recorder.turnIDs {
		if seen[turn] {
			t.Fatalf("turn id %q was reused on request %d: %v", turn, i+1, recorder.turnIDs)
		}
		seen[turn] = true
	}
}

// recordingRequestProvider records the identity of every request it is handed.
type recordingRequestProvider struct {
	mu         sync.Mutex
	steps      []agent.ModelEvent
	requestIDs []string
	turnIDs    []string
	systems    []string
	calls      int
}

func (p *recordingRequestProvider) Name() string { return "recorder" }
func (p *recordingRequestProvider) Capabilities() model.Capabilities {
	return model.Capabilities{Streaming: true, ToolCalls: true, Usage: true}
}
func (p *recordingRequestProvider) Models(context.Context) ([]string, error) {
	return []string{"m"}, nil
}
func (p *recordingRequestProvider) Health(context.Context) (string, error) { return "healthy", nil }
func (p *recordingRequestProvider) Stream(_ context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	p.mu.Lock()
	p.requestIDs = append(p.requestIDs, req.RequestID)
	p.turnIDs = append(p.turnIDs, req.TurnID)
	for _, m := range req.Messages {
		if m.Role == agent.RoleSystem {
			p.systems = append(p.systems, m.Content)
		}
	}
	p.calls++
	ch := make(chan agent.ModelEvent, 4)
	for _, step := range p.steps {
		step.RequestID = req.RequestID
		ch <- step
	}
	close(ch)
	p.mu.Unlock()
	return ch, nil
}

func TestTheTurnAdvancesWhenATurnCompletes(t *testing.T) {
	r := NewRunner(Services{
		Models: model.NewFake(map[string][]model.ScriptStep{"*": {{Kind: "complete"}}}),
		Tools:  &stubTools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: checkpoint.New(t.TempDir()),
	}, "R-adv", "S1")
	if r.State.TurnID != "turn-1" {
		t.Fatalf("a new run starts at turn-1, got %q", r.State.TurnID)
	}
	r.TurnsDone = 1
	r.advanceTurn()
	if r.State.TurnID != "turn-2" {
		t.Fatalf("turn = %q, want turn-2", r.State.TurnID)
	}
}

func TestAResumedRunContinuesTheTurnSequenceRatherThanReusingIt(t *testing.T) {
	// A resumed run with five turns already done goes on to turn six. Reusing an
	// identifier a previous life spent is the collision the advance prevents.
	r := NewRunner(Services{
		Models: model.NewFake(map[string][]model.ScriptStep{"*": {{Kind: "complete"}}}),
		Tools:  &stubTools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: checkpoint.New(t.TempDir()),
	}, "R-res", "S1")
	r.RestoreFrom(agent.Checkpoint{
		State:     agent.NativeAgentState{RunID: "R-res", TurnID: "turn-5", Revision: 5},
		TurnsDone: 5,
	})
	r.TurnsDone++
	r.advanceTurn()
	if r.State.TurnID != "turn-7" {
		t.Fatalf("turn = %q, want turn-7", r.State.TurnID)
	}
}

// The run compiled a context, wrote it to disk, stored its id in state — and then
// nothing read either. The model request is built from the conversation, so the
// files the run judged relevant, the disclosure level and the pressure never
// reached the model that was supposed to act on them. A run that compiles a
// context and does not deliver it has done the work and kept the answer (GAP-128).

func TestTheCompiledContextReachesTheModel(t *testing.T) {
	recorder := &recordingRequestProvider{steps: []agent.ModelEvent{{Kind: agent.EventCompleted, Finished: true}}}
	r := NewRunner(Services{
		Models: recorder, Tools: &stubTools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: checkpoint.New(t.TempDir()),
		ContextManifest: func(context.Context, agent.NativeAgentState) (string, error) {
			return "Read ENTRYPOINT.md, then docs/PRUMO.md.", nil
		},
	}, "R-ctx", "S1")
	r.MaxTurns = 1
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "what should I do?"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(recorder.systems) == 0 {
		t.Fatal("the model received no context at all")
	}
	joined := strings.Join(recorder.systems, "\n")
	if !strings.Contains(joined, "ENTRYPOINT.md") {
		t.Fatalf("the context did not reach the model; it saw: %q", joined)
	}
}

func TestTheContextIsDeliveredOnceNotEveryTurn(t *testing.T) {
	// It is context for the whole run. Re-adding it each turn would grow the
	// conversation with a copy of itself until compaction decided to summarise it.
	recorder := &recordingRequestProvider{steps: []agent.ModelEvent{
		{Kind: agent.EventToolCallReady, ToolCall: &agent.ToolCall{
			ID: "c1", TurnID: "turn-1", Name: "fs.read",
			Arguments: map[string]any{"path": "README.md"},
		}},
		{Kind: agent.EventCompleted, Finished: true},
	}}
	r := NewRunner(Services{
		Models: recorder, Tools: &stubTools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: checkpoint.New(t.TempDir()),
		ContextManifest: func(context.Context, agent.NativeAgentState) (string, error) {
			return "read this first", nil
		},
	}, "R-ctx2", "S1")
	r.MaxTurns = 3
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}}
	_ = r.RunUntilDone(context.Background())

	// The context belongs in every request — it is part of the conversation, and
	// a model that saw it once and not on the next turn would be worse. What must
	// not happen is a second copy accumulating each turn.
	seenInConversation := 0
	for _, m := range r.Messages {
		if m.ID == contextManifestMessageID {
			seenInConversation++
		}
	}
	if seenInConversation != 1 {
		t.Fatalf("the manifest is in the conversation %d times, want exactly 1", seenInConversation)
	}
	// And within any single request, at most one.
	counted := 0
	for _, content := range recorder.systems {
		if strings.Count(content, "read this first") > 1 {
			counted++
		}
	}
	if counted > 0 {
		t.Fatalf("%d request(s) carried the manifest more than once", counted)
	}
}

func TestAnEmptyManifestAddsNothing(t *testing.T) {
	recorder := &recordingRequestProvider{steps: []agent.ModelEvent{{Kind: agent.EventCompleted, Finished: true}}}
	r := NewRunner(Services{
		Models: recorder, Tools: &stubTools{}, Perms: perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints: checkpoint.New(t.TempDir()),
		ContextManifest: func(context.Context, agent.NativeAgentState) (string, error) {
			return "   ", nil
		},
	}, "R-ctx3", "S1")
	r.MaxTurns = 1
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(recorder.systems) != 0 {
		t.Fatalf("an empty manifest added %d system messages", len(recorder.systems))
	}
}
