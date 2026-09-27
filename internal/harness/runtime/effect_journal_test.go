package runtime

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/checkpoint"
	"github.com/raillen/prumo/internal/harness/model"
	"github.com/raillen/prumo/internal/harness/perm"
)

// The effect journal was fully implemented and used only by its own tests. No
// production runner wrote an intent or an outcome, so a run replayed after a
// crash repeated whatever it had already done (GAP-126). These tests drive the
// real execute path, not the store in isolation.

type recordingTools struct {
	mu    sync.Mutex
	calls []agent.ToolCall
	kind  string
}

func (s *recordingTools) Execute(_ context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, call)
	return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: "ok"}, nil
}
func (s *recordingTools) KindOf(string) string      { return s.kind }
func (s *recordingTools) OperationOf(string) string { return "modified" }

func deletingModel() model.Provider {
	return model.NewFake(map[string][]model.ScriptStep{"*": {
		{Kind: "tool_call", Tool: &agent.ToolCall{
			ID: "c1", TurnID: "t1", Name: "edit.delete",
			Arguments: map[string]any{"path": "docs/keep.md"},
		}},
		{Kind: "complete"},
	}})
}

func newJournalledRunner(t *testing.T, store *checkpoint.Store, tools *recordingTools) *Runner {
	t.Helper()
	r := NewRunner(Services{
		Models:        deletingModel(),
		Tools:         tools,
		Perms:         perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints:   store,
		EffectJournal: store,
	}, "R-journal", "S1")
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "delete it"}}
	return r
}

func TestRunnerJournalsTheIntentAndOutcome(t *testing.T) {
	store := checkpoint.New(t.TempDir())
	tools := &recordingTools{kind: "side-effecting"}
	r := newJournalledRunner(t, store, tools)

	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	effects := store.Effects()
	if len(effects) != 1 {
		t.Fatalf("the journal must hold one effect, got %d", len(effects))
	}
	got := effects[0]
	if got.Status != agent.EffectApplied {
		t.Fatalf("a successful call must be recorded applied, got %s", got.Status)
	}
	if got.ID != "fx-c1" || got.IdempotencyKey != "fx-c1" {
		t.Fatalf("the effect must be identified by the tool call: %+v", got)
	}
	if got.Target != "docs/keep.md" {
		t.Fatalf("the journal must record what was touched, got %q", got.Target)
	}
	if got.ObservableEffect != "modified" {
		t.Fatalf("the journal must record the observable effect, got %q", got.ObservableEffect)
	}
}

func TestRunnerJournalsAFailedCallAsNotApplied(t *testing.T) {
	store := checkpoint.New(t.TempDir())
	tools := &failingTools{}
	r := NewRunner(Services{
		Models:        deletingModel(),
		Tools:         tools,
		Perms:         perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints:   store,
		EffectJournal: store,
	}, "R-journal", "S1")
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "delete it"}}

	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	effects := store.Effects()
	if len(effects) != 1 {
		t.Fatalf("one effect expected, got %d", len(effects))
	}
	if effects[0].Status != agent.EffectFailed {
		t.Fatalf("a failing call must be recorded as not applied, got %s", effects[0].Status)
	}
}

func TestReplayedRunDoesNotRepeatAnAppliedEffect(t *testing.T) {
	// The property the journal exists for: a run that stops and resumes must
	// not apply the same effect twice.
	dir := t.TempDir()
	store := checkpoint.New(dir)
	tools := &recordingTools{kind: "side-effecting"}

	first := newJournalledRunner(t, store, tools)
	if err := first.RunUntilDone(context.Background()); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if len(tools.calls) != 1 {
		t.Fatalf("first run made %d calls, want 1", len(tools.calls))
	}

	// A second runner over the same store stands in for a restart: the journal
	// is what crosses the process boundary.
	second := &recordingTools{kind: "side-effecting"}
	restored := newJournalledRunner(t, store, second)
	if err := restored.RunUntilDone(context.Background()); err != nil {
		t.Fatalf("replayed run: %v", err)
	}
	if len(second.calls) != 0 {
		t.Fatalf("an effect already recorded as applied must not be repeated; %d calls", len(second.calls))
	}
}

func TestReplayedRunSkipsAnUnknownOutcomeRatherThanRepeatingIt(t *testing.T) {
	// A pending entry means the process died between the intent and the
	// outcome. The default policy for a side-effecting tool is skip, so the
	// effect is not guessed at.
	store := checkpoint.New(t.TempDir())
	if _, err := store.RecordIntent(agent.PendingEffect{
		ID: "fx-c1", Kind: "side-effecting", IdempotencyKey: "fx-c1",
		Target: "docs/keep.md", RecoveryPolicy: "skip",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	tools := &recordingTools{kind: "side-effecting"}
	r := newJournalledRunner(t, store, tools)
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(tools.calls) != 0 {
		t.Fatalf("an effect of unknown outcome must not be repeated; %d calls", len(tools.calls))
	}
}

func TestReplayedRunFailsClosedWithoutARecoveryPolicy(t *testing.T) {
	// No policy is not permission to repeat. The run must stop and say why.
	store := checkpoint.New(t.TempDir())
	if _, err := store.RecordIntent(agent.PendingEffect{
		ID: "fx-c1", Kind: "side-effecting", IdempotencyKey: "fx-c1",
		Target: "docs/keep.md",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	tools := &recordingTools{kind: "side-effecting"}
	r := newJournalledRunner(t, store, tools)
	err := r.RunUntilDone(context.Background())
	if err == nil {
		t.Fatal("an effect of unknown outcome with no policy must fail the run")
	}
	// RecordIntent stamps the default "fail" onto a new effect, so the refusal
	// names the policy rather than its absence. Either wording is a refusal.
	if !strings.Contains(err.Error(), "recovery policy") {
		t.Fatalf("the failure must say why, got %v", err)
	}
	if len(tools.calls) != 0 {
		t.Fatalf("nothing may run after a fail-closed refusal; %d calls", len(tools.calls))
	}
}

func TestReadOnlyToolsAreRetriedOnReplay(t *testing.T) {
	// Repeating a read is harmless, so an unknown outcome is not a reason to
	// stop the run. The policy is the one the runtime itself writes for a read,
	// standing in for a previous life of this run that died before recording the
	// outcome.
	store := checkpoint.New(t.TempDir())
	if _, err := store.RecordIntent(agent.PendingEffect{
		ID: "fx-c1", Kind: "read-only", IdempotencyKey: "fx-c1", RecoveryPolicy: "retry",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	tools := &recordingTools{kind: "read-only"}

	r := NewRunner(Services{
		Models:        readingModel(),
		Tools:         tools,
		Perms:         perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints:   store,
		EffectJournal: store,
	}, "R-read", "S1")
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "read it"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatalf("a read must not fail the run: %v", err)
	}
	if len(tools.calls) != 1 {
		t.Fatalf("a read must be retried, got %d calls", len(tools.calls))
	}
	// The effect is now recorded applied, so a further replay stops repeating it.
	effects := store.Effects()
	if len(effects) != 1 || effects[0].Status != agent.EffectApplied {
		t.Fatalf("the retried read must be recorded applied: %+v", effects)
	}
}

// A fresh read records retry, which is what makes the test above possible: the
// policy has to come from somewhere, and the tool kind is what says whether
// repeating is safe.
func TestReadOnlyEffectIsRecordedAsRetryable(t *testing.T) {
	store := checkpoint.New(t.TempDir())
	tools := &recordingTools{kind: "read-only"}
	r := NewRunner(Services{
		Models:        readingModel(),
		Tools:         tools,
		Perms:         perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints:   store,
		EffectJournal: store,
	}, "R-read2", "S1")
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "read it"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	effects := store.Effects()
	if len(effects) != 1 {
		t.Fatalf("one effect expected, got %d", len(effects))
	}
	if effects[0].RecoveryPolicy != "retry" {
		t.Fatalf("a read must be recorded retryable, got %q", effects[0].RecoveryPolicy)
	}
}

// A side-effecting call is recorded skip, so a replay waits for a person rather
// than writing twice.
func TestSideEffectIsRecordedAsSkippable(t *testing.T) {
	store := checkpoint.New(t.TempDir())
	tools := &recordingTools{kind: "side-effecting"}
	r := newJournalledRunner(t, store, tools)
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	effects := store.Effects()
	if len(effects) != 1 {
		t.Fatalf("one effect expected, got %d", len(effects))
	}
	if effects[0].RecoveryPolicy != "skip" {
		t.Fatalf("a side effect must be recorded as skipped on replay, got %q", effects[0].RecoveryPolicy)
	}
}

// A policy recorded with the effect is the one that governs a replay. The
// runtime computing retry for a read must not override a deliberate skip.
func TestStoredRecoveryPolicyWinsOverTheCurrentToolKind(t *testing.T) {
	store := checkpoint.New(t.TempDir())
	if _, err := store.RecordIntent(agent.PendingEffect{
		ID: "fx-c1", Kind: "read-only", IdempotencyKey: "fx-c1", RecoveryPolicy: "skip",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	tools := &recordingTools{kind: "read-only"}
	r := NewRunner(Services{
		Models:        readingModel(),
		Tools:         tools,
		Perms:         perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Checkpoints:   store,
		EffectJournal: store,
	}, "R-read", "S1")
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "read it"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(tools.calls) != 0 {
		t.Fatalf("a recorded skip must not be overridden on replay; %d calls", len(tools.calls))
	}
}

func readingModel() model.Provider {
	return model.NewFake(map[string][]model.ScriptStep{"*": {
		{Kind: "tool_call", Tool: &agent.ToolCall{
			ID: "c1", TurnID: "t1", Name: "fs.read",
			Arguments: map[string]any{"path": "README.md"},
		}},
		{Kind: "complete"},
	}})
}

type failingTools struct{}

func (failingTools) Execute(_ context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: "nope"}, nil
}
func (failingTools) KindOf(string) string      { return "side-effecting" }
func (failingTools) OperationOf(string) string { return "modified" }
