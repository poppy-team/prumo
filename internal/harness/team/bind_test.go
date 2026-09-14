package team

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/model"
)

func fakeDeps() RunnerDeps {
	return RunnerDeps{
		Goal:     "nested goal",
		MaxTurns: 2,
		NewProvider: func(_ context.Context, _ Role) (model.Provider, error) {
			return model.NewFake(map[string][]model.ScriptStep{
				"*": {{Kind: "text", Text: "ok"}, {Kind: "complete"}},
			}), nil
		},
	}
}

func TestNestedRunCompletes(t *testing.T) {
	ws := t.TempDir()
	work := RunWork(fakeDeps())
	ev, usage, err := work(context.Background(), Role{Name: "dev", Workspace: ws})
	if err != nil {
		t.Fatal(err)
	}
	if len(ev) == 0 || usage["turns"] == 0 {
		t.Fatalf("bad nested result: %v %v", ev, usage)
	}
	if _, err := os.Stat(filepath.Join(ws, ".prumo", "runs")); err != nil {
		t.Fatalf("child checkpoint missing: %v", err)
	}
}

func TestNestedRunRequiresWorkspace(t *testing.T) {
	work := RunWork(fakeDeps())
	if _, _, err := work(context.Background(), Role{Name: "ghost"}); err == nil {
		t.Fatal("workspace-less role must not nest")
	}
}

func TestNestedRunBudgetEnforced(t *testing.T) {
	work := RunWork(RunnerDeps{
		Goal: "x", MaxTurns: 5,
		NewProvider: func(_ context.Context, _ Role) (model.Provider, error) {
			return model.NewFake(map[string][]model.ScriptStep{
				"*": {{Kind: "tool_call", Tool: &agent.ToolCall{ID: "c", Name: "git.status", IdempotencyKey: "k"}}, {Kind: "complete"}},
			}), nil
		},
	})
	ws := t.TempDir()
	_, _, err := work(context.Background(), Role{Name: "dev", Workspace: ws, Budget: map[string]float64{"tool_calls": 0}})
	if err != nil {
		t.Fatalf("zero limit means untracked, got: %v", err)
	}
	_, _, err = work(context.Background(), Role{Name: "dev2", Workspace: t.TempDir(), Budget: map[string]float64{"tool_calls": 1}})
	// 5 turns x 1 tool each vs limit 1 → must exhaust.
	if err == nil {
		t.Fatal("expected budget exhaustion across turns")
	}
}

func TestChildHandoffValidates(t *testing.T) {
	b, err := ChildHandoff("dev", "rev", agent.NativeAgentState{RunID: "R-dev", ContextManifestID: "ctx"}, "rev1")
	if err != nil || b.Validate() != nil {
		t.Fatal("child handoff must validate")
	}
	if b.Refs["child_role"] != "dev" || !strings.Contains(b.Handoff.Summary, "R-dev") {
		t.Fatalf("bad handoff refs: %+v", b)
	}
}

func TestNestedTeamEndToEnd(t *testing.T) {
	ws := t.TempDir()
	sum, err := Runner{
		Team: Team{ID: "t-nested", Delegation: DelegationSolo, Roles: []Role{{Name: "dev", Workspace: ws}}},
		Work: RunWork(fakeDeps()),
	}.Run(context.Background())
	if err != nil || sum.Status != "complete" {
		t.Fatalf("nested team failed: %+v %v", sum, err)
	}
}
