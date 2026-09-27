package runlayer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/perm"
)

type stubBase struct{ kind string }

func (s stubBase) Execute(_ context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	return agent.ToolResult{ToolCallID: call.ID, Output: "ok"}, nil
}
func (s stubBase) OperationOf(string) string { return "" }
func (s stubBase) KindOf(string) string      { return s.kind }

func TestBudgetHardFail(t *testing.T) {
	tr := NewTracker(10, 0, 1)
	if err := tr.ConsumeUsage(agent.Usage{InputTokens: 6, OutputTokens: 5}); err == nil {
		t.Fatal("11 tokens over limit 10 must fail")
	}
	tools := &CountingTools{Base: stubBase{}, Tracker: NewTracker(0, 0, 1)}
	if _, err := tools.Execute(context.Background(), agent.ToolCall{ID: "c1", Name: "fs.read"}); err != nil {
		t.Fatal(err)
	}
	if _, err := tools.Execute(context.Background(), agent.ToolCall{ID: "c2", Name: "fs.read"}); err == nil {
		t.Fatal("second tool call over limit must fail")
	}
	if len(tools.ReportsCopy()) != 1 || tools.ReportsCopy()[0].Name != "fs.read" {
		t.Fatalf("reports wrong: %+v", tools.ReportsCopy())
	}
	if err := tr.Save(filepath.Join(t.TempDir(), "budget.json")); err != nil {
		t.Fatal(err)
	}
	if got := tr.Snapshot()["tokens"]; got != 0 {
		t.Fatalf("failed consume must not record: %v", got)
	}
}

func TestStrictGate(t *testing.T) {
	gate := StrictGate()
	if err := gate([]ToolReport{{Name: "test.run", ExitCode: 0}}); err != nil {
		t.Fatal("passing tests must open the gate")
	}
	if err := gate([]ToolReport{{Name: "test.run", ExitCode: 1}}); err == nil {
		t.Fatal("failing tests must close the gate")
	}
	if err := gate(nil); err == nil {
		t.Fatal("no test evidence must close the gate")
	}
}

func TestDumpPermissions(t *testing.T) {
	eng := perm.New(perm.Policy{DefaultAction: agent.PermissionAllow})
	if _, err := eng.Approve("p1", "fp-p1", "user"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if _, err := eng.Deny("p2", "fp-p2", "user", "nope"); err != nil {
		t.Fatalf("deny: %v", err)
	}
	path := filepath.Join(t.TempDir(), "perms.jsonl")
	if err := DumpPermissions(path, eng); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if strings.Count(string(data), "\n") != 2 {
		t.Fatalf("expected 2 resolutions:\n%s", data)
	}
}

func TestWriteEvidence(t *testing.T) {
	rec, err := WriteEvidence(filepath.Join(t.TempDir(), "ev.json"), "R1", "P00-G01", "complete", "completed",
		map[string]float64{"tokens": 5}, []ToolReport{{Name: "test.run"}})
	if err != nil {
		t.Fatalf("evidence write failed: %v", err)
	}
	if rec.Type != "harness_run" {
		t.Fatalf("unexpected type %q", rec.Type)
	}
	if rec.Producer == "" || rec.Timestamp == "" || rec.Status == "" || rec.GoalID != "P00-G01" {
		t.Fatalf("record is missing a required field: %+v", rec)
	}
	if rec.Status != "passed" {
		t.Fatalf("a completed run with no failing reports must be passed, got %q", rec.Status)
	}
}

func TestBridge(t *testing.T) {
	evs := []agent.AgentEvent{{ID: "e1", RunID: "R1", Kind: "run.started"}}
	if err := BridgeToObservability(filepath.Join(t.TempDir(), "obs.jsonl"), evs); err != nil {
		t.Fatal(err)
	}
}

func TestCachedTokensCountAgainstTheTokenCeiling(t *testing.T) {
	// The token meter counted only input and output, so a run whose entire prompt
	// was cached reported a token total near zero and walked through a token
	// ceiling it had in fact filled. The provider served every one of those
	// tokens; whether it was cheap in money is the cost meter's business, and
	// conflating the two made the capacity limit decorative (GAP-130).
	tr := NewTracker(1000, 0, 0)
	if err := tr.ConsumeUsage(agent.Usage{InputTokens: 10, OutputTokens: 10}); err != nil {
		t.Fatal(err)
	}
	// 900 cache reads plus 81 cache writes: 981 tokens of provider work, on top
	// of the 20 already counted, so the run crosses a 1000-token ceiling.
	err := tr.ConsumeUsage(agent.Usage{CacheReadTokens: 900, CacheWriteTokens: 81})
	if err == nil {
		t.Fatal("a fully cached run must still count against the token ceiling")
	}
}

func TestACachedRunCanExhaustTheTokenCeilingOnItsOwn(t *testing.T) {
	tr := NewTracker(1000, 0, 0)
	if err := tr.ConsumeUsage(agent.Usage{CacheReadTokens: 1000}); err != nil {
		t.Fatalf("a run that is exactly at the limit must be allowed: %v", err)
	}
	if err := tr.ConsumeUsage(agent.Usage{CacheReadTokens: 1}); err == nil {
		t.Fatal("the token past the ceiling must be refused")
	}
}

func TestTotalTokensCountsEveryDimensionExactlyOnce(t *testing.T) {
	u := agent.Usage{InputTokens: 10, OutputTokens: 20, CacheReadTokens: 30, CacheWriteTokens: 40}
	if got, want := u.TotalTokens(), 100; got != want {
		t.Fatalf("total = %d, want %d", got, want)
	}
	// Reasoning is a breakdown of output, not a fifth dimension: counting it
	// again is how the same token ends up billed twice.
	u.ReasoningTokens = 15
	if got, want := u.TotalTokens(), 100; got != want {
		t.Fatalf("reasoning changed the total to %d; it must stay %d", got, want)
	}
}
