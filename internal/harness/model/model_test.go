package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestOpenAICompatMultipartImagePayload(t *testing.T) {
	var receivedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	p := NewOpenAICompat(srv.URL, "test-key", "gpt-4o")
	req := agent.ModelRequest{
		RequestID: "req-img",
		TurnID:    "T1",
		Messages: []agent.Message{
			{
				ID:   "m1",
				Role: agent.RoleUser,
				Parts: []agent.ContentPart{
					{Type: "text", Text: "describe:"},
					{Type: "image", MimeType: "image/png", Data: "iVBORw0KGgoAAA=="},
				},
			},
		},
	}
	ch, err := p.Stream(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	for range ch {
	}

	msgs, ok := receivedBody["messages"].([]any)
	if !ok || len(msgs) != 1 {
		t.Fatalf("expected 1 message in payload, got: %v", receivedBody)
	}
	msgMap := msgs[0].(map[string]any)
	parts, ok := msgMap["content"].([]any)
	if !ok || len(parts) != 2 {
		t.Fatalf("expected 2 content parts, got: %v", msgMap["content"])
	}
	p0 := parts[0].(map[string]any)
	if p0["type"] != "text" || p0["text"] != "describe:" {
		t.Errorf("part 0 mismatch: %v", p0)
	}
	p1 := parts[1].(map[string]any)
	if p1["type"] != "image_url" {
		t.Errorf("part 1 not image_url: %v", p1)
	}
	imgURL, ok := p1["image_url"].(map[string]any)
	if !ok || imgURL["url"] != "data:image/png;base64,iVBORw0KGgoAAA==" {
		t.Errorf("part 1 url mismatch: %v", imgURL)
	}
}

