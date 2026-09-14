package model

import (
	"context"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

// runConformance executes the shared contract suite against any Provider.
func runConformance(t *testing.T, p Provider) {
	t.Helper()
	ctx := context.Background()
	caps := p.Capabilities()
	if !caps.Streaming || !caps.ToolCalls || !caps.Usage || !caps.Cancel || !caps.Health {
		t.Fatalf("provider %s missing baseline capabilities: %+v", p.Name(), caps)
	}
	if _, err := p.Health(ctx); err != nil {
		t.Fatalf("health failed: %v", err)
	}
	if _, err := p.Models(ctx); err != nil {
		t.Fatalf("models failed: %v", err)
	}

	collect := func(req agent.ModelRequest) []agent.ModelEvent {
		t.Helper()
		ch, err := p.Stream(ctx, req)
		if err != nil {
			t.Fatalf("stream failed: %v", err)
		}
		var out []agent.ModelEvent
		for ev := range ch {
			out = append(out, ev)
		}
		return out
	}
	mkReq := func(id string) agent.ModelRequest {
		return agent.ModelRequest{RequestID: id, RunID: "R-1", TurnID: "T-1", Model: "fake-default", Messages: []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "hi"}}}
	}

	// simple completion ends with completed
	evs := collect(mkReq("completion"))
	if len(evs) == 0 || evs[len(evs)-1].Kind != agent.EventCompleted {
		t.Fatalf("completion must end with completed, got %+v", evs)
	}
	// tool trajectory delivers a tool call
	evs = collect(mkReq("tools"))
	found := false
	for _, e := range evs {
		if e.Kind == agent.EventToolCallReady {
			found = true
		}
	}
	if !found {
		t.Fatalf("tools script must deliver tool_call_ready")
	}
	// failure surfaces retryable error
	evs = collect(mkReq("failure"))
	foundErr := false
	for _, e := range evs {
		if e.Kind == agent.EventError && e.Retryable {
			foundErr = true
		}
	}
	if !foundErr {
		t.Fatalf("failure script must surface retryable error")
	}
	// cancel: cancelled context yields cancelled event
	cctx, cancel := context.WithCancel(ctx)
	cancel()
	ch, err := p.Stream(cctx, mkReq("cancel"))
	if err != nil {
		t.Fatalf("cancel stream failed: %v", err)
	}
	sawCancel := false
	for e := range ch {
		if e.Kind == agent.EventCancelled || e.Kind == agent.EventCompleted {
			sawCancel = true
		}
	}
	if !sawCancel {
		t.Fatalf("cancel must terminate stream")
	}
}

func fakeForConformance() *FakeProvider {
	return NewFake(map[string][]ScriptStep{
		"*":          {{Kind: "text", Text: "hello"}, {Kind: "complete"}},
		"completion": {{Kind: "text", Text: "done"}, {Kind: "usage", Usage: &agent.Usage{InputTokens: 10, OutputTokens: 5}}, {Kind: "complete"}},
		"tools":      {{Kind: "text", Text: "reading"}, {Kind: "tool_call", Tool: &agent.ToolCall{ID: "c1", TurnID: "T-1", Name: "fs.read", IdempotencyKey: "k1"}}, {Kind: "complete"}},
		"failure":    {{Kind: "error", Error: "rate limit", Retryable: true}},
		"cancel":     {{Kind: "cancel"}},
	})
}

func TestFakeConformance(t *testing.T) { runConformance(t, fakeForConformance()) }

func TestFakeStreamingToolArgs(t *testing.T) {
	f := NewFake(map[string][]ScriptStep{
		"*": {{Kind: "tool_call", Tool: &agent.ToolCall{ID: "c1", Name: "edit.patch", IdempotencyKey: "k"}}, {Kind: "complete"}},
	})
	ch, _ := f.Stream(context.Background(), agent.ModelRequest{RequestID: "x", TurnID: "T"})
	n := 0
	for ev := range ch {
		n++
		if ev.Kind == agent.EventToolCallReady && ev.ToolCall.IdempotencyKey == "" {
			t.Fatal("tool call must carry idempotency key")
		}
	}
	if n == 0 {
		t.Fatal("expected events")
	}
}
