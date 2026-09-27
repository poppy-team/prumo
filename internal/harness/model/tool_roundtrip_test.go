package model

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

// A tool result used to reach the provider as a bare message with no call to
// attach it to, and the assistant turn that asked for the tool was never
// recorded at all. A conversation that reports a result for a call nobody made
// is not one any provider accepts (GAP-115).

// toolRoundTrip is the shape a second request has: what the model asked for,
// then what came back.
func toolRoundTrip() []agent.Message {
	return []agent.Message{
		{ID: "m1", Role: agent.RoleUser, Content: "read the readme"},
		{ID: "m2", Role: agent.RoleAgent, ToolCalls: []agent.ToolCall{{
			ID: "call-1", Name: "fs.read", Arguments: map[string]any{"path": "README.md"},
		}}},
		{ID: "m3", Role: agent.RoleTool, ToolCallID: "call-1", Content: "# readme"},
	}
}

func captureOpenAI(t *testing.T, messages []agent.Message) []map[string]any {
	t.Helper()
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{}}]}\n\ndata: [DONE]\n\n"))
	}))
	defer srv.Close()

	p := NewOpenAICompatWithPolicy(srv.URL, "k", "m", LocalDevelopmentDestinationPolicy())
	ch, err := p.Stream(context.Background(), agent.ModelRequest{RequestID: "r1", Messages: messages})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	for range ch {
	}
	raw, _ := captured["messages"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, m := range raw {
		if asMap, ok := m.(map[string]any); ok {
			out = append(out, asMap)
		}
	}
	return out
}

func TestOpenAIToolResultCarriesItsCallID(t *testing.T) {
	msgs := captureOpenAI(t, toolRoundTrip())
	var toolMsg map[string]any
	for _, m := range msgs {
		if m["role"] == "tool" {
			toolMsg = m
		}
	}
	if toolMsg == nil {
		t.Fatalf("no tool message was sent: %v", msgs)
	}
	if toolMsg["tool_call_id"] != "call-1" {
		t.Fatalf("a tool result must name the call it answers: %v", toolMsg)
	}
	if toolMsg["content"] != "# readme" {
		t.Fatalf("the result content must be sent: %v", toolMsg)
	}
}

func TestOpenAIAssistantTurnSaysWhatItAskedFor(t *testing.T) {
	msgs := captureOpenAI(t, toolRoundTrip())
	var assistant map[string]any
	for _, m := range msgs {
		if m["role"] == "assistant" {
			assistant = m
		}
	}
	if assistant == nil {
		t.Fatalf("the assistant turn that requested the tool was not sent: %v", msgs)
	}
	calls, _ := assistant["tool_calls"].([]any)
	if len(calls) != 1 {
		t.Fatalf("the tool request must be sent, not only its result: %v", assistant)
	}
	call, _ := calls[0].(map[string]any)
	if call["id"] != "call-1" || call["type"] != "function" {
		t.Fatalf("the call must be identified: %v", call)
	}
	fn, _ := call["function"].(map[string]any)
	if fn["name"] != "fs.read" {
		t.Fatalf("the function name must be sent: %v", fn)
	}
	// Arguments are a JSON string in this API, not an object.
	args, isString := fn["arguments"].(string)
	if !isString {
		t.Fatalf("arguments must be a JSON string, got %T", fn["arguments"])
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(args), &decoded); err != nil {
		t.Fatalf("arguments must be valid JSON: %v", err)
	}
	if decoded["path"] != "README.md" {
		t.Fatalf("the arguments must survive: %v", decoded)
	}
}

func TestAnAssistantTurnWithoutToolsHasNoToolCallsKey(t *testing.T) {
	msgs := captureOpenAI(t, []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "hello"}})
	if _, present := msgs[0]["tool_calls"]; present {
		t.Fatalf("an ordinary turn must not carry an empty tool_calls: %v", msgs[0])
	}
}

func TestAFailedToolIsNotSentToOpenAIAsAnEmptyResult(t *testing.T) {
	messages := []agent.Message{
		{ID: "m1", Role: agent.RoleUser, Content: "delete it"},
		{ID: "m2", Role: agent.RoleAgent, ToolCalls: []agent.ToolCall{{ID: "call-1", Name: "edit.delete", Arguments: map[string]any{"path": "a.txt"}}}},
		{ID: "m3", Role: agent.RoleTool, ToolCallID: "call-1", Content: "",
			Metadata: map[string]any{"ok": false, "exit_code": 1, "error": "no such file"}},
	}
	msgs := captureOpenAI(t, messages)
	var toolMsg map[string]any
	for _, m := range msgs {
		if m["role"] == "tool" {
			toolMsg = m
		}
	}
	if toolMsg == nil {
		t.Fatalf("the tool message must still be sent: %v", msgs)
	}
	if content, _ := toolMsg["content"].(string); content == "" {
		t.Fatal("a failed tool must not reach the model as an empty result, which reads as an empty success")
	}
}

func captureAnthropic(t *testing.T, messages []agent.Message) []map[string]any {
	t.Helper()
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer srv.Close()

	p := NewAnthropicWithPolicy(srv.URL, "k", "claude-test", LocalDevelopmentDestinationPolicy())
	ch, err := p.Stream(context.Background(), agent.ModelRequest{RequestID: "r1", Messages: messages})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	for range ch {
	}
	raw, _ := captured["messages"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, m := range raw {
		if asMap, ok := m.(map[string]any); ok {
			out = append(out, asMap)
		}
	}
	return out
}

func TestAnthropicUsesToolUseAndToolResultBlocks(t *testing.T) {
	msgs := captureAnthropic(t, toolRoundTrip())
	if len(msgs) != 3 {
		t.Fatalf("the whole round trip must be sent, got %d messages: %v", len(msgs), msgs)
	}
	// The assistant turn carries tool_use.
	assistant := msgs[1]
	if assistant["role"] != "assistant" {
		t.Fatalf("the requesting turn must stay an assistant turn: %v", assistant)
	}
	useBlocks, _ := assistant["content"].([]any)
	if len(useBlocks) != 1 {
		t.Fatalf("the tool request must be a tool_use block: %v", assistant)
	}
	use, _ := useBlocks[0].(map[string]any)
	if use["type"] != "tool_use" || use["id"] != "call-1" || use["name"] != "fs.read" {
		t.Fatalf("the tool_use block is wrong: %v", use)
	}
	input, _ := use["input"].(map[string]any)
	if input["path"] != "README.md" {
		t.Fatalf("the tool input must be sent: %v", use)
	}

	// The result comes back as a user turn with a tool_result block.
	result := msgs[2]
	if result["role"] != "user" {
		t.Fatalf("a tool result is a user turn in this API: %v", result)
	}
	resultBlocks, _ := result["content"].([]any)
	if len(resultBlocks) != 1 {
		t.Fatalf("the result must be a tool_result block: %v", result)
	}
	tr, _ := resultBlocks[0].(map[string]any)
	if tr["type"] != "tool_result" || tr["tool_use_id"] != "call-1" {
		t.Fatalf("the tool_result block is wrong: %v", tr)
	}
	if tr["content"] != "# readme" {
		t.Fatalf("the result content must be sent: %v", tr)
	}
}

func TestAnthropicTellsTheModelWhenAToolFailed(t *testing.T) {
	messages := []agent.Message{
		{ID: "m1", Role: agent.RoleUser, Content: "delete it"},
		{ID: "m2", Role: agent.RoleAgent, ToolCalls: []agent.ToolCall{{ID: "call-1", Name: "edit.delete", Arguments: map[string]any{"path": "a.txt"}}}},
		{ID: "m3", Role: agent.RoleTool, ToolCallID: "call-1", Content: "",
			Metadata: map[string]any{"ok": false, "exit_code": 1, "error": "no such file"}},
	}
	msgs := captureAnthropic(t, messages)
	last := msgs[len(msgs)-1]
	blocks, _ := last["content"].([]any)
	if len(blocks) != 1 {
		t.Fatalf("expected a tool_result block: %v", last)
	}
	tr, _ := blocks[0].(map[string]any)
	content, _ := tr["content"].(string)
	if content == "" {
		t.Fatal("a failed tool must not reach the model as an empty result")
	}
	if !strings.Contains(content, "no such file") {
		t.Fatalf("the reason must reach the model: %q", content)
	}
}
