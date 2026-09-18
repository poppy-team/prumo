package model

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

// fakeOpenCode puts a script named `opencode` on PATH that prints `stdout` and
// exits with `code` — so the provider can be exercised without a model, a
// network or a quota.
func fakeOpenCode(t *testing.T, stdout string, code int, stderr string) {
	t.Helper()
	dir := t.TempDir()
	terminated := func(text string) string {
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		return text
	}
	script := "#!/bin/sh\n"
	if stdout != "" {
		script += "cat <<'EOF'\n" + terminated(stdout) + "EOF\n"
	}
	if stderr != "" {
		script += "cat >&2 <<'EOF'\n" + terminated(stderr) + "EOF\n"
	}
	script += "exit " + string(rune('0'+code)) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "opencode"), []byte(script), 0o755); err != nil {
		t.Fatalf("cannot write the fake opencode: %v", err)
	}
	// The fake goes first, but the shell it runs in still needs to find `cat`.
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PRUMO_OPENCODE_BIN", "")
}

func collect(t *testing.T, ch <-chan agent.ModelEvent) []agent.ModelEvent {
	t.Helper()
	var out []agent.ModelEvent
	for ev := range ch {
		out = append(out, ev)
	}
	return out
}

func eventKinds(events []agent.ModelEvent) []string {
	var out []string
	for _, ev := range events {
		out = append(out, string(ev.Kind))
	}
	return out
}

// The shapes below are the ones the installed opencode 1.18.31 emits
// (`{type, timestamp, sessionID, ...extra}`), so the parser is tested against
// the wire, not against a convenient invention.
func TestTheDelegatedTurnBecomesTextUsageAndCompletion(t *testing.T) {
	fakeOpenCode(t, `{"type":"step_start","timestamp":1,"sessionID":"ses_1","part":{"type":"step-start"}}
{"type":"text","timestamp":2,"sessionID":"ses_1","part":{"type":"text","text":"a resposta"}}
{"type":"reasoning","timestamp":3,"sessionID":"ses_1","part":{"type":"reasoning","text":"pensando"}}
{"type":"tool_use","timestamp":4,"sessionID":"ses_1","part":{"type":"tool","tool":"bash"}}
{"type":"step_finish","timestamp":5,"sessionID":"ses_1","part":{"reason":"stop","tokens":{"input":120,"output":30,"cache":{"read":900,"write":15}},"cost":0.0042}}
`, 0, "")

	provider := &OpenCode{Binary: "opencode"}
	ch, err := provider.Stream(context.Background(), agent.ModelRequest{
		RequestID: "r1",
		Messages:  []agent.Message{{Role: agent.RoleUser, Content: "faça isso"}},
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	events := collect(t, ch)

	if got := strings.Join(eventKinds(events), ","); got != "text_delta,reasoning_delta,usage_updated,completed" {
		t.Fatalf("the turn became %q", got)
	}
	if events[0].Text != "a resposta" {
		t.Errorf("the answer was not carried: %q", events[0].Text)
	}
	usage := events[2].Usage
	if usage == nil {
		t.Fatal("the turn reported no usage")
	}
	// Every number the provider sent arrives, and in its own field: a cached
	// token is not an input token and a client that summed them could not say
	// what the turn cost.
	if usage.InputTokens != 120 || usage.OutputTokens != 30 || usage.CacheReadTokens != 900 || usage.CacheWriteTokens != 15 {
		t.Fatalf("the usage lost a number: %+v", usage)
	}
	if usage.CostUSD != 0.0042 {
		t.Fatalf("the cost was lost: %v", usage.CostUSD)
	}
	if !events[3].Finished {
		t.Fatal("the turn did not say it finished")
	}
}

// Forwarding opencode's tool calls would make Prumo execute tools that were
// already executed.
func TestTheDelegatedToolCallsAreNotClaimed(t *testing.T) {
	fakeOpenCode(t, `{"type":"tool_use","timestamp":4,"sessionID":"ses_1","part":{"type":"tool","tool":"bash","state":{"input":{"command":"rm -rf /"}}}}
{"type":"step_finish","timestamp":5,"sessionID":"ses_1","part":{"tokens":{"input":1,"output":1}}}
`, 0, "")

	provider := &OpenCode{Binary: "opencode"}
	ch, _ := provider.Stream(context.Background(), agent.ModelRequest{Messages: []agent.Message{{Role: agent.RoleUser, Content: "x"}}})
	for _, ev := range collect(t, ch) {
		if ev.Kind == agent.EventToolCallReady || ev.Kind == agent.EventToolCallDelta {
			t.Fatalf("a delegated tool call was handed to the runtime: %+v", ev)
		}
	}
}

// A provider error is reported with what the provider said, so the person sees
// the cause (an exhausted quota, an expired login) instead of "failed".
func TestAProviderErrorIsReportedVerbatim(t *testing.T) {
	fakeOpenCode(t, `{"type":"error","timestamp":1,"sessionID":"ses_1","error":{"name":"UnknownError","data":{"message":"Unexpected server error. Check server logs for details.","ref":"err_513b6d92"}}}
`, 1, "")

	provider := &OpenCode{Binary: "opencode"}
	ch, _ := provider.Stream(context.Background(), agent.ModelRequest{Messages: []agent.Message{{Role: agent.RoleUser, Content: "x"}}})
	events := collect(t, ch)

	var text string
	for _, ev := range events {
		if ev.Kind == agent.EventError {
			text = ev.Error
		}
	}
	for _, want := range []string{"Unexpected server error", "err_513b6d92"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the error lost %q: %q", want, text)
		}
	}
}

// An exit that failed without an error event still fails: "the process ended" is
// not "the turn worked".
func TestAFailedExitIsAnError(t *testing.T) {
	fakeOpenCode(t, `{"type":"text","timestamp":1,"sessionID":"ses_1","part":{"type":"text","text":"parcial"}}
`, 3, "cannot reach the provider")

	provider := &OpenCode{Binary: "opencode"}
	ch, _ := provider.Stream(context.Background(), agent.ModelRequest{Messages: []agent.Message{{Role: agent.RoleUser, Content: "x"}}})
	events := collect(t, ch)

	last := events[len(events)-1]
	if last.Kind != agent.EventError || !strings.Contains(last.Error, "cannot reach the provider") {
		t.Fatalf("a failed exit ended as %+v", last)
	}
}

// Nothing is invented from a line this provider does not understand.
func TestAStrayLineIsReportedNotGuessed(t *testing.T) {
	fakeOpenCode(t, `isto não é json
{"type":"step_finish","timestamp":5,"sessionID":"ses_1","part":{"tokens":{"input":5,"output":2}}}
`, 0, "")

	provider := &OpenCode{Binary: "opencode"}
	ch, _ := provider.Stream(context.Background(), agent.ModelRequest{Messages: []agent.Message{{Role: agent.RoleUser, Content: "x"}}})
	events := collect(t, ch)

	if events[0].Kind != agent.EventWarning {
		t.Fatalf("a stray line became %q", events[0].Kind)
	}
	if events[0].Text != "" {
		t.Fatalf("text was invented from a stray line: %q", events[0].Text)
	}
}

func TestNoGoalIsAnExplicitError(t *testing.T) {
	provider := &OpenCode{Binary: "opencode"}
	if _, err := provider.Stream(context.Background(), agent.ModelRequest{}); err == nil {
		t.Fatal("a turn with no goal started anyway")
	}
}

func TestDiscoveryAndHealthAreHonest(t *testing.T) {
	fakeOpenCode(t, "opencode/mimo-v2.5-free\nopencode/big-pickle\n", 0, "")
	provider := &OpenCode{Binary: "opencode"}

	models, err := provider.Models(context.Background())
	if err != nil {
		t.Fatalf("models: %v", err)
	}
	if len(models) != 2 || models[0] != "opencode/big-pickle" {
		t.Fatalf("the catalogue came back as %v", models)
	}

	// A missing binary is unavailable with the reason, never "available".
	t.Setenv("PATH", t.TempDir())
	if _, err := provider.Health(context.Background()); err == nil {
		t.Fatal("a missing opencode reported itself healthy")
	}
}

// The provider says out loud that it does not hand tool calls back.
func TestToolCallsAreDeclaredDelegated(t *testing.T) {
	caps := (&OpenCode{}).Capabilities()
	if caps.ToolCalls {
		t.Fatal("the provider claims tool calls it never forwards")
	}
	if !caps.Streaming || !caps.Usage {
		t.Fatal("the provider hides what it does do")
	}
}
