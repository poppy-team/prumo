package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/model"
	harnessprotocol "github.com/raillen/prumo/internal/harness/protocol"
)

// The daemon built every run with NewRunner, so the scope firewall that guards
// tool calls was exercised by exactly one test and by nothing else. A production
// run had no scope: the model could reach anything the tools could reach, and
// the only limit was the permission engine — which asks a human, and a daemon
// run has no human watching it (GAP-127).

func TestADaemonRunIsGovernedByAScope(t *testing.T) {
	s := &Server{limits: DefaultLimits()}
	dirIR, err := s.directiveFor("R-1", "fix the bug", t.TempDir(), s.limits.RunBudgetCents)
	if err != nil {
		t.Fatalf("a run could not be given a scope: %v", err)
	}
	if dirIR.WorkspaceRoot == "" {
		t.Fatal("the directive names no workspace, so it can scope nothing")
	}
	if len(dirIR.MutationBoundaries.AllowedPaths) == 0 {
		t.Fatal("the directive allows no mutation at all; a run that may not edit is not a run")
	}
	// Inside the workspace is allowed.
	if !dirIR.CanMutatePath(filepath.Join(dirIR.WorkspaceRoot, "main.go")) {
		t.Fatal("a path inside the workspace was refused; the firewall refuses its own tools' paths")
	}
	// Outside it is not.
	if dirIR.CanMutatePath("/etc/passwd") {
		t.Fatal("a path outside the workspace was allowed for mutation")
	}
	if dirIR.CanMutatePath("../outside.go") {
		t.Fatal("a parent-relative path escaped the workspace")
	}
}

func TestARunWithNoWorkspaceRefusesToStart(t *testing.T) {
	// A run with no boundary is a run with every boundary removed, so it is not
	// started with an empty scope and hoped for.
	s := &Server{limits: DefaultLimits()}
	if _, err := s.directiveFor("R-1", "goal", "", s.limits.RunBudgetCents); err == nil {
		t.Fatal("a run with no workspace was given a scope anyway")
	}
}

func TestARunWithNoCostCeilingRefusesToStart(t *testing.T) {
	s := &Server{limits: DefaultLimits()}
	_, err := s.directiveFor("R-1", "goal", t.TempDir(), 0)
	if err == nil {
		t.Fatal("a run with no cost ceiling was allowed to start")
	}
	if !strings.Contains(err.Error(), "ceiling") {
		t.Fatalf("the refusal does not say what is missing: %v", err)
	}
}

func TestTheFirewallStopsAProductionToolCall(t *testing.T) {
	// The point of the change, end to end: a model that asks to write outside
	// the workspace is stopped by the daemon, not only by a test that calls the
	// runtime directly.
	//
	// The target is a sibling temp directory rather than /etc: the permission
	// engine denies /etc by prefix, so a test aimed there proves the policy
	// works and says nothing about the firewall. A sibling is outside the
	// workspace and permitted by policy, which is exactly the case the
	// permission engine cannot catch and the firewall exists for.
	workspace := t.TempDir()
	outside := filepath.Join(t.TempDir(), "prumo-owned")
	var executed []string
	tools := &recordingTools{onExecute: func(call agent.ToolCall) {
		executed = append(executed, call.Name)
	}}
	provider := model.NewFake(map[string][]model.ScriptStep{
		"*": {
			{Kind: "tool_call", Tool: &agent.ToolCall{
				ID: "c1", Name: "edit.create",
				Arguments: map[string]any{"path": outside},
			}},
			{Kind: "complete"},
		},
	})

	s := New(filepath.Join(t.TempDir(), "d.sock"), t.TempDir(), Deps{
		NewProvider: func(string, string, string, string) (model.Provider, error) { return provider, nil },
		Tools:       tools,
		Limits:      DefaultLimits(),
	})
	runID, stopReason := s.runOnceForTest(context.Background(), workspace, "write outside the workspace")

	// Whatever the run reported, the tool must not have run.
	for _, name := range executed {
		t.Fatalf("a tool executed despite being outside the workspace: %s", name)
	}
	// The reason is read from the status the caller itself polls, because that
	// is where a caller learns what happened — not only in an event stream the
	// caller may not have subscribed to.
	if !strings.Contains(stopReason, "scope firewall") {
		t.Fatalf("the run did not report a scope violation; stop reason was %q", stopReason)
	}
	if !strings.Contains(stopReason, outside) {
		t.Fatalf("the stop reason does not name the refused path: %q", stopReason)
	}
	_ = runID
	if _, err := os.Stat(outside); err == nil {
		t.Fatal("the file outside the workspace exists; the tool ran")
	}
}

func TestTheRunIDIsTheOneTheCallerChose(t *testing.T) {
	// The runner used to take its run id from the directive's task id, so every
	// event carried an id the caller was not watching.
	workspace := t.TempDir()
	provider := model.NewFake(map[string][]model.ScriptStep{
		"*": {{Kind: "text", Text: "ok"}, {Kind: "complete"}},
	})
	s := New(filepath.Join(t.TempDir(), "d.sock"), t.TempDir(), Deps{
		NewProvider: func(string, string, string, string) (model.Provider, error) { return provider, nil },
		Tools:       &recordingTools{},
		Limits:      DefaultLimits(),
	})
	runID, _ := s.runOnceForTest(context.Background(), workspace, "say ok")
	_ = time.Now()
	for _, ev := range s.eventsForTest(runID) {
		if strings.Contains(ev, ":task") {
			t.Fatalf("an event carried the directive's task id as the run id: %s", ev)
		}
	}
}

// recordingTools records the calls that reached it, so a test can prove a tool
// did not run rather than infer it from the run's status.
type recordingTools struct {
	onExecute func(call agent.ToolCall)
	mu        sync.Mutex
	executed  []string
}

func (r *recordingTools) Execute(_ context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	r.mu.Lock()
	r.executed = append(r.executed, call.Name)
	r.mu.Unlock()
	if r.onExecute != nil {
		r.onExecute(call)
	}
	return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: "done"}, nil
}

func (r *recordingTools) KindOf(name string) string {
	if strings.HasPrefix(name, "edit.") {
		return "side-effecting"
	}
	return "read-only"
}

func (r *recordingTools) OperationOf(string) string { return "modified" }

// runOnceForTest drives the daemon's real op path and waits for the run to stop.
// It does not reach into execute: the point is that the production entry point
// is the one under test.
func (s *Server) runOnceForTest(ctx context.Context, workspace, goal string) (string, string) {
	res := s.dispatch(map[string]any{
		"op": "start", "protocol_version": harnessprotocol.Version,
		"run_id": "R-scope", "goal": goal,
		"provider": "fake", "model": "fake-default", "workspace": workspace, "max_turns": 3,
	})
	if res["ok"] != true {
		return "", fmt.Sprintf("start failed: %v", res)
	}
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		status := s.dispatch(map[string]any{
			"op": "status", "protocol_version": harnessprotocol.Version, "run_id": "R-scope",
		})
		if phase, _ := status["phase"].(string); phase == "complete" || phase == "failed" {
			// A run that stopped is a run that stopped. Whether stopping is the
			// right outcome is what the caller decides, so the helper returns the
			// reason and lets the caller judge it.
			reason, _ := status["stop_reason"].(string)
			return "R-scope", reason
		}
		select {
		case <-ctx.Done():
			return "R-scope", ctx.Err().Error()
		case <-time.After(50 * time.Millisecond):
		}
	}
	return "R-scope", "the run did not stop"
}

// eventsForTest reads the run's events through the same file the status op
// reads, so the test sees what a client would see.
func (s *Server) eventsForTest(runID string) []string {
	res := s.dispatch(map[string]any{
		"op": "events", "protocol_version": harnessprotocol.Version, "run_id": runID,
	})
	raw, _ := res["events"].([]map[string]any)
	out := make([]string, 0, len(raw))
	for _, ev := range raw {
		out = append(out, fmt.Sprintf("%v %v %v", ev["kind"], ev["run_id"], ev["payload"]))
	}
	return out
}
