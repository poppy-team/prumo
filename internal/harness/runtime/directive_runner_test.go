package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/directive"
)

type dummyToolExecutor struct {
	executedCalls []agent.ToolCall
}

func (d *dummyToolExecutor) Execute(ctx context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	d.executedCalls = append(d.executedCalls, call)
	return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: "ok"}, nil
}

func (d *dummyToolExecutor) KindOf(toolName string) string {
	if strings.HasPrefix(toolName, "edit.") {
		return "side-effecting"
	}
	return "read-only"
}

func (d *dummyToolExecutor) OperationOf(toolName string) string {
	return "modified"
}

func TestRunnerWithDirective_ScopeFirewallBlock(t *testing.T) {
	dir, err := directive.CompileDirective(directive.CompilerInput{
		GoalID:         "g-1",
		TaskID:         "t-1",
		TaskIntent:     "Edit allowed file",
		AllowedPaths:   []string{"src/allowed.go"},
		ForbiddenPaths: []string{"secret/*", "docs/critical.md"},
		HardLimitCents: 100,
	})
	if err != nil {
		t.Fatalf("failed to compile directive: %v", err)
	}

	exec := &dummyToolExecutor{}
	svc := Services{
		Tools: exec,
	}

	runner := NewRunnerWithDirective(svc, dir, "sess-1")
	if len(runner.Messages) == 0 || !strings.Contains(runner.Messages[0].Content, "=== 1. HARD POLICIES ===") {
		t.Errorf("expected initial message seeded with prompt projection, got: %#v", runner.Messages)
	}

	// Queue a tool call targeting forbidden path "docs/critical.md"
	runner.ToolQ = []agent.ToolCall{
		{
			ID:        "call-bad",
			Name:      "edit.patch",
			Arguments: map[string]any{"path": "docs/critical.md"},
		},
	}
	runner.State.Phase = agent.PhaseExecuteTool

	ctx := context.Background()
	err = runner.Step(ctx)
	if err == nil || !strings.Contains(err.Error(), "directive scope violation") {
		t.Fatalf("expected scope violation error, got: %v", err)
	}
	if runner.State.Phase != agent.PhaseFailed {
		t.Errorf("expected PhaseFailed, got %s", runner.State.Phase)
	}
	if len(exec.executedCalls) != 0 {
		t.Errorf("tool executor should NOT have been called, but was called with: %#v", exec.executedCalls)
	}

	// Now try an allowed path
	runnerAllowed := NewRunnerWithDirective(svc, dir, "sess-2")
	runnerAllowed.ToolQ = []agent.ToolCall{
		{
			ID:        "call-good",
			Name:      "edit.patch",
			Arguments: map[string]any{"path": "src/allowed.go"},
		},
	}
	runnerAllowed.State.Phase = agent.PhaseExecuteTool

	err = runnerAllowed.Step(ctx)
	if err != nil {
		t.Fatalf("expected allowed mutation to succeed, got: %v", err)
	}
	if len(exec.executedCalls) != 1 {
		t.Errorf("expected 1 tool call executed, got %d", len(exec.executedCalls))
	}
}
