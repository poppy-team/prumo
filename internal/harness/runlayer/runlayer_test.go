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
func (s stubBase) KindOf(string) string { return s.kind }

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
	eng.Approve("p1", "user")
	eng.Deny("p2", "user", "nope")
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
	rec, err := WriteEvidence(filepath.Join(t.TempDir(), "ev.json"), "R1", "complete", "completed",
		map[string]float64{"tokens": 5}, []ToolReport{{Name: "test.run"}})
	if err != nil || rec.Type != "harness_run" {
		t.Fatalf("evidence failed: %+v %v", rec, err)
	}
}

func TestBridge(t *testing.T) {
	evs := []agent.AgentEvent{{ID: "e1", RunID: "R1", Kind: "run.started"}}
	if err := BridgeToObservability(filepath.Join(t.TempDir(), "obs.jsonl"), evs); err != nil {
		t.Fatal(err)
	}
}
