package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/model"
	"github.com/raillen/prumo/internal/harness/perm"
)

// A tool that failed put the reason in Error and left Output empty, and only
// Output was recorded. The model was handed an empty result for a call that had
// failed, which reads as a call that succeeded and returned nothing (GAP-116).

type resultTools struct {
	result agent.ToolResult
	events []agent.AgentEvent
}

func (s *resultTools) Execute(_ context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	res := s.result
	res.ToolCallID = call.ID
	return res, nil
}
func (s *resultTools) KindOf(string) string      { return "read-only" }
func (s *resultTools) OperationOf(string) string { return "" }

func runWithResult(t *testing.T, res agent.ToolResult) (*Runner, *resultTools) {
	t.Helper()
	tools := &resultTools{result: res}
	provider := &requestRecorder{
		inner: model.NewFake(map[string][]model.ScriptStep{"*": {
			{Kind: "tool_call", Tool: &agent.ToolCall{ID: "c1", TurnID: "t1", Name: "fs.read", Arguments: map[string]any{"path": "README.md"}}},
			{Kind: "complete"},
		}}),
	}
	r := NewRunner(Services{
		Models: provider,
		Tools:  tools,
		Perms:  perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Events: func(ev agent.AgentEvent) { tools.events = append(tools.events, ev) },
	}, "R-result", "S1")
	r.MaxTurns = 3
	r.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "read it"}}
	if err := r.RunUntilDone(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	return r, tools
}

func TestAFailedToolIsRecordedAsFailed(t *testing.T) {
	r, _ := runWithResult(t, agent.ToolResult{ExitCode: 1, Error: "no such file: README.md"})
	if len(r.Obs) != 1 {
		t.Fatalf("one observation expected, got %d", len(r.Obs))
	}
	obs := r.Obs[0]
	if obs.OK {
		t.Fatal("a tool that reported an error must not be recorded as ok")
	}
	if obs.Error != "no such file: README.md" {
		t.Fatalf("the reason must survive to the conversation, got %q", obs.Error)
	}
	if obs.ExitCode != 1 {
		t.Fatalf("the exit code is a fact the model can act on, got %d", obs.ExitCode)
	}
}

func TestANonZeroExitWithNoMessageStillSaysItFailed(t *testing.T) {
	// A tool that fails silently is the worst case: nothing in the record says
	// anything went wrong.
	r, _ := runWithResult(t, agent.ToolResult{ExitCode: 2})
	obs := r.Obs[0]
	if obs.OK {
		t.Fatal("a non-zero exit must not be recorded as ok")
	}
	if obs.Error == "" {
		t.Fatal("a failure with no reason must still carry one, or it reads as an empty success")
	}
	if !strings.Contains(obs.Error, "2") {
		t.Fatalf("the reason should name the exit code, got %q", obs.Error)
	}
}

func TestASuccessfulEmptyResultIsStillASuccess(t *testing.T) {
	// The other half. A tool can legitimately return nothing, and treating empty
	// output as failure would make the model retry work that already succeeded.
	r, _ := runWithResult(t, agent.ToolResult{ExitCode: 0, Output: ""})
	obs := r.Obs[0]
	if !obs.OK {
		t.Fatalf("an empty but successful result must be ok, got error %q", obs.Error)
	}
	if obs.Error != "" {
		t.Fatalf("a success must carry no error, got %q", obs.Error)
	}
}

func TestASuccessKeepsItsOutput(t *testing.T) {
	r, _ := runWithResult(t, agent.ToolResult{ExitCode: 0, Output: "file contents"})
	obs := r.Obs[0]
	if !obs.OK || obs.Content != "file contents" {
		t.Fatalf("a success must keep its output: %+v", obs)
	}
}

func TestAnErrorWithAZeroExitIsStillAFailure(t *testing.T) {
	// The two answers disagree; the error is the tool telling us directly, so it
	// wins over the exit code.
	if (agent.ToolResult{ExitCode: 0, Error: "denied"}).OK() {
		t.Fatal("an error must count as a failure whatever the exit code says")
	}
}

func TestAFailureIsVisibleOnTheTimeline(t *testing.T) {
	// A run quietly making no progress looks exactly like one that is working.
	_, tools := runWithResult(t, agent.ToolResult{ExitCode: 1, Error: "boom"})
	found := false
	for _, ev := range tools.events {
		if ev.Kind == "tool.failed" {
			found = true
			if ev.Payload["tool"] != "fs.read" {
				t.Fatalf("the event must name the tool: %v", ev.Payload)
			}
			if ev.Payload["error"] != "boom" {
				t.Fatalf("the event must carry the reason: %v", ev.Payload)
			}
		}
	}
	if !found {
		t.Fatal("a failed tool must be visible on the timeline, not only in the conversation")
	}
}

func TestTruncationIsRecorded(t *testing.T) {
	// A truncated output is not the whole result, and the model should know that
	// rather than reason from a partial answer.
	r, _ := runWithResult(t, agent.ToolResult{ExitCode: 0, Output: "first bytes", Truncated: true})
	if !r.Obs[0].Truncated {
		t.Fatal("a truncated result must be marked as truncated")
	}
}
