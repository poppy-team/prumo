package model

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

	p := NewOpenAICompatWithPolicy(srv.URL, "test-key", "gpt-4o", LocalDevelopmentDestinationPolicy())
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

func TestForNameProviders(t *testing.T) {
	// Fake
	fake, err := ForName("fake", "", "", "")
	if err != nil || fake.Name() != "fake" {
		t.Fatalf("fake failed: %v", err)
	}

	// OpenAI compat
	openai, err := ForName("openai-compat", "http://localhost:11434/v1", "test-key", "deepseek-coder")
	if err != nil || openai.Name() != "openai-compat" {
		t.Fatalf("openai-compat failed: %v", err)
	}

	// Anthropic
	claude, err := ForName("anthropic", "https://api.anthropic.com", "test-key", "claude-3-5-sonnet-20241022")
	if err != nil || claude.Name() != "anthropic" {
		t.Fatalf("anthropic failed: %v", err)
	}

	// Gemini / Google
	gemini, err := ForName("gemini", "", "test-gemini-key", "gemini-2.5-pro")
	if err != nil || gemini.Name() != "openai-compat" {
		t.Fatalf("gemini failed: %v", err)
	}

	// Antigravity ACF
	acf, err := ForName("antigravity", "", "test-acf-token", "gemini-2.5-pro")
	if err != nil || acf.Name() != "openai-compat" {
		t.Fatalf("antigravity failed: %v", err)
	}

	// DeepSeek
	deepseek, err := ForName("deepseek", "", "test-deepseek-key", "deepseek-reasoner")
	if err != nil || deepseek.Name() != "openai-compat" {
		t.Fatalf("deepseek failed: %v", err)
	}
	compat, ok := deepseek.(*OpenAICompat)
	if !ok || compat.ProviderKey() != "deepseek" {
		t.Fatalf("expected deepseek provider key, got: %v", compat.ProviderKey())
	}
}

func TestOpenAICompatParsesRetryAfter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"type":"rate_limit_error","message":"rate limit exceeded"}}`))
	}))
	defer srv.Close()

	policy := LocalDevelopmentDestinationPolicy()
	p := NewOpenAICompatWithPolicy(srv.URL, "key", "gpt-4o", policy)
	ch, err := p.Stream(context.Background(), agent.ModelRequest{RequestID: "req-retry"})
	if err != nil {
		t.Fatalf("Stream() err = %v", err)
	}
	ev := <-ch
	if ev.Kind != agent.EventError {
		t.Fatalf("expected EventError, got %v", ev.Kind)
	}
	if !ev.Retryable {
		t.Errorf("expected Retryable=true")
	}
	if !strings.Contains(ev.Error, "Retry-After: 120") {
		t.Errorf("expected error message to contain Retry-After: 120, got %q", ev.Error)
	}
}

func TestOpenAICompatStreamsReasoningDelta(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"thinking step 1\"}}]}\n\n")
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"final answer\"}}]}\n\n")
		fmt.Fprintf(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	policy := LocalDevelopmentDestinationPolicy()
	p := NewOpenAICompatWithPolicy(srv.URL, "key", "deepseek-reasoner", policy)
	ch, err := p.Stream(context.Background(), agent.ModelRequest{RequestID: "req-reasoning"})
	if err != nil {
		t.Fatalf("Stream() err = %v", err)
	}

	var events []agent.ModelEvent
	for ev := range ch {
		events = append(events, ev)
	}

	if len(events) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(events))
	}
	if events[0].Kind != agent.EventReasoningDelta || events[0].Text != "thinking step 1" {
		t.Errorf("event 0 mismatch: %+v", events[0])
	}
	if events[1].Kind != agent.EventTextDelta || events[1].Text != "final answer" {
		t.Errorf("event 1 mismatch: %+v", events[1])
	}
	if events[2].Kind != agent.EventCompleted {
		t.Errorf("event 2 mismatch: %+v", events[2])
	}
}
