package model

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

func anthropicTestServer(body string, status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

func collectAnthropic(t *testing.T, p *Anthropic, id string) []agent.ModelEvent {
	t.Helper()
	ch, err := p.Stream(context.Background(), agent.ModelRequest{RequestID: id, TurnID: "T", Messages: []agent.Message{{ID: "m", Role: agent.RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("stream failed: %v", err)
	}
	var out []agent.ModelEvent
	for ev := range ch {
		out = append(out, ev)
	}
	return out
}

func kinds(evs []agent.ModelEvent) map[agent.ModelEventKind]int {
	m := map[agent.ModelEventKind]int{}
	for _, e := range evs {
		m[e.Kind]++
	}
	return m
}

func TestAnthropicTextAndUsage(t *testing.T) {
	srv := anthropicTestServer("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":3,\"output_tokens\":0}}}\n\nevent: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hello\"}}\n\nevent: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"input_tokens\":0,\"output_tokens\":5}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n", http.StatusOK)
	defer srv.Close()
	p := NewAnthropicWithPolicy(srv.URL, "test-key", "claude-test", LocalDevelopmentDestinationPolicy())
	if h, err := p.Health(context.Background()); err != nil || h != "healthy" {
		t.Fatalf("health: %s %v", h, err)
	}
	if caps := p.Capabilities(); !caps.Streaming || !caps.ToolCalls || !caps.Usage || !caps.Cancel || !caps.Health {
		t.Fatalf("missing baseline capabilities: %+v", caps)
	}
	evs := collectAnthropic(t, p, "r1")
	k := kinds(evs)
	if k[agent.EventTextDelta] != 1 || k[agent.EventUsageUpdated] != 2 || k[agent.EventCompleted] != 1 {
		t.Fatalf("unexpected event mix: %+v", evs)
	}
	if evs[0].Usage == nil || evs[0].Usage.InputTokens != 3 {
		t.Fatalf("first usage wrong: %+v", evs[0])
	}
}

func TestAnthropicToolUse(t *testing.T) {
	body := "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu-1\",\"name\":\"fs.read\"}}\n\nevent: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"path\\\": \\\"a.go\\\"\"}}\n\nevent: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"}\" }}\n\nevent: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	srv := anthropicTestServer(body, http.StatusOK)
	defer srv.Close()
	p := NewAnthropicWithPolicy(srv.URL, "k", "m", LocalDevelopmentDestinationPolicy())
	evs := collectAnthropic(t, p, "r-tool")
	var tool *agent.ToolCall
	for _, e := range evs {
		if e.Kind == agent.EventToolCallReady {
			tool = e.ToolCall
		}
	}
	if tool == nil {
		t.Fatalf("expected tool_call_ready: %+v", evs)
	}
	if tool.Name != "fs.read" || tool.IdempotencyKey == "" {
		t.Fatalf("bad tool normalization: %+v", tool)
	}
	if v, _ := tool.Arguments["path"].(string); v != "a.go" {
		t.Fatalf("bad args merge: %+v", tool.Arguments)
	}
}

func TestAnthropicRetryableError(t *testing.T) {
	srv := anthropicTestServer("", 429)
	defer srv.Close()
	p := NewAnthropicWithPolicy(srv.URL, "k", "m", LocalDevelopmentDestinationPolicy())
	evs := collectAnthropic(t, p, "r-err")
	if len(evs) != 1 || evs[0].Kind != agent.EventError || !evs[0].Retryable {
		t.Fatalf("expected single retryable error: %+v", evs)
	}
	srv2 := anthropicTestServer("event: error\ndata: {\"type\":\"error\",\"error\":{\"type\":\"overloaded_error\",\"message\":\"busy\"}}\n\n", http.StatusOK)
	defer srv2.Close()
	p2 := NewAnthropicWithPolicy(srv2.URL, "k", "m", LocalDevelopmentDestinationPolicy())
	evs2 := collectAnthropic(t, p2, "r-err2")
	found := false
	for _, e := range evs2 {
		if e.Kind == agent.EventError && e.Retryable && e.Error == "busy" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected overloaded retryable error: %+v", evs2)
	}
}

func TestAnthropicCancel(t *testing.T) {
	srv := anthropicTestServer("event: message_start\ndata: {\"type\":\"message_start\"}\n\n", http.StatusOK)
	defer srv.Close()
	p := NewAnthropicWithPolicy(srv.URL, "k", "m", LocalDevelopmentDestinationPolicy())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ch, err := p.Stream(ctx, agent.ModelRequest{RequestID: "r-cancel", TurnID: "T"})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for ev := range ch {
		n++
		_ = fmt.Sprint(ev.Kind)
	}
	if n == 0 {
		t.Fatal("expected terminal event")
	}
}

func TestAnthropicMultipartImagePayload(t *testing.T) {
	var receivedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer srv.Close()

	p := NewAnthropicWithPolicy(srv.URL, "test-key", "claude-3-5", LocalDevelopmentDestinationPolicy())
	req := agent.ModelRequest{
		RequestID: "req-img",
		TurnID:    "T1",
		Messages: []agent.Message{
			{
				ID:   "m1",
				Role: agent.RoleUser,
				Parts: []agent.ContentPart{
					{Type: "text", Text: "explain this:"},
					{Type: "image", MimeType: "image/png", Data: "iVBORw0KGgoAAAANSUhEUg=="},
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
	blocks, ok := msgMap["content"].([]any)
	if !ok || len(blocks) != 2 {
		t.Fatalf("expected 2 content blocks, got: %v", msgMap["content"])
	}
	b0 := blocks[0].(map[string]any)
	if b0["type"] != "text" || b0["text"] != "explain this:" {
		t.Errorf("block 0 mismatch: %v", b0)
	}
	b1 := blocks[1].(map[string]any)
	if b1["type"] != "image" {
		t.Errorf("block 1 not image: %v", b1)
	}
	src, ok := b1["source"].(map[string]any)
	if !ok || src["type"] != "base64" || src["media_type"] != "image/png" || src["data"] != "iVBORw0KGgoAAAANSUhEUg==" {
		t.Errorf("block 1 source mismatch: %v", src)
	}
}
