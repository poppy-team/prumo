package model

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

// Anthropic declared ToolCalls: true and then never put a single schema in the
// request, so a model had no way to know the tools existed. The capability was a
// claim in a struct (GAP-114).

// captureBody runs a request against a stub and returns what the provider was
// actually sent.
func captureBody(t *testing.T, stream func(ctx context.Context, base, apiKey string, p *Anthropic) (<-chan agent.ModelEvent, error)) map[string]any {
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
	ch, err := stream(context.Background(), srv.URL, "k", p)
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	for range ch {
	}
	if captured == nil {
		t.Fatal("the provider was never called")
	}
	return captured
}

func TestAnthropicSendsTheToolCatalogue(t *testing.T) {
	body := captureBody(t, func(ctx context.Context, base, key string, p *Anthropic) (<-chan agent.ModelEvent, error) {
		return p.Stream(ctx, agent.ModelRequest{
			RequestID: "r1",
			Messages:  []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "read the readme"}},
			Tools: []agent.ToolSpec{{
				Name:        "fs.read",
				Description: "read a file as text",
				Schema:      map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}, "required": []string{"path"}},
			}},
		})
	})

	tools, ok := body["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("the tool catalogue must reach the provider, got %v", body["tools"])
	}
	tool, _ := tools[0].(map[string]any)
	if tool["name"] != "fs.read" {
		t.Fatalf("the tool name must be sent: %v", tool)
	}
	if tool["description"] != "read a file as text" {
		t.Fatalf("the description must be sent: %v", tool)
	}
	// Anthropic names the argument schema input_schema, not parameters, and takes
	// no wrapper type. Sending the OpenAI shape here would be silently ignored.
	schema, ok := tool["input_schema"].(map[string]any)
	if !ok {
		t.Fatalf("the argument schema must be sent as input_schema: %v", tool)
	}
	props, _ := schema["properties"].(map[string]any)
	if _, ok := props["path"]; !ok {
		t.Fatalf("the schema must describe the arguments: %v", schema)
	}
	if _, wrong := tool["parameters"]; wrong {
		t.Fatal("the OpenAI-shaped parameters key does not belong in an Anthropic request")
	}
}

func TestAnthropicOmitsToolsWhenThereAreNone(t *testing.T) {
	body := captureBody(t, func(ctx context.Context, base, key string, p *Anthropic) (<-chan agent.ModelEvent, error) {
		return p.Stream(ctx, agent.ModelRequest{
			RequestID: "r1",
			Messages:  []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "hello"}},
		})
	})
	if _, present := body["tools"]; present {
		t.Fatal("a request with no tools must not carry an empty catalogue")
	}
}

func TestAToolWithoutASchemaStillGetsOne(t *testing.T) {
	// A model asked to call a tool with no declared arguments tends to send none,
	// and the call then fails for a reason nothing told it. An explicit empty
	// object is the honest "this tool takes nothing".
	body := captureBody(t, func(ctx context.Context, base, key string, p *Anthropic) (<-chan agent.ModelEvent, error) {
		return p.Stream(ctx, agent.ModelRequest{
			RequestID: "r1",
			Messages:  []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "status"}},
			Tools:     []agent.ToolSpec{{Name: "git.status"}},
		})
	})
	tools, _ := body["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("the tool must be sent: %v", body["tools"])
	}
	tool, _ := tools[0].(map[string]any)
	schema, ok := tool["input_schema"].(map[string]any)
	if !ok {
		t.Fatalf("a tool with no schema must still declare one: %v", tool)
	}
	if schema["type"] != "object" {
		t.Fatalf("the fallback schema must be an object: %v", schema)
	}
}

func TestOpenAICompatSendsTheToolCatalogue(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{}}]}\n\ndata: [DONE]\n\n"))
	}))
	defer srv.Close()

	p := NewOpenAICompatWithPolicy(srv.URL, "k", "m", LocalDevelopmentDestinationPolicy())
	ch, err := p.Stream(context.Background(), agent.ModelRequest{
		RequestID: "r1",
		Messages:  []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "read"}},
		Tools: []agent.ToolSpec{{
			Name: "fs.read", Description: "read a file",
			Schema: map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}},
		}},
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	for range ch {
	}
	tools, ok := captured["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("the OpenAI-compatible provider must send the catalogue: %v", captured["tools"])
	}
	fn, _ := tools[0].(map[string]any)["function"].(map[string]any)
	if fn["name"] != "fs.read" {
		t.Fatalf("the function name must be sent: %v", tools[0])
	}
	if _, ok := fn["parameters"]; !ok {
		t.Fatalf("the OpenAI shape names the schema parameters: %v", fn)
	}
}

func TestValidateArgsChecksAgainstTheSentSchemas(t *testing.T) {
	// The validation the provider does on a tool call has to be the same schemas
	// that were offered, or a model is held to a contract it never saw.
	tools := []agent.ToolSpec{{
		Name: "fs.read",
		Schema: map[string]any{
			"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}},
			"required": []string{"path"},
		},
	}}
	if err := ValidateArgs(tools, "fs.read", map[string]any{"path": "README.md"}); err != nil {
		t.Fatalf("a valid call must pass: %v", err)
	}
	if err := ValidateArgs(tools, "fs.read", map[string]any{}); err == nil {
		t.Fatal("a call missing a required argument must be refused")
	}
}
