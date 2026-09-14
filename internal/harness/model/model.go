// Package model defines the ModelProvider boundary (HA1) and ships a
// deterministic FakeProvider plus an OpenAI-compatible HTTP adapter.
// Vendor message types never escape: adapters normalize to agent.ModelEvent.
package model

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
)

// Capabilities advertises what an adapter can do.
type Capabilities struct {
	Streaming        bool `json:"streaming"`
	ToolCalls        bool `json:"tool_calls"`
	StructuredOutput bool `json:"structured_output"`
	Usage            bool `json:"usage"`
	Cancel           bool `json:"cancel"`
	Health           bool `json:"health"`
	ModelDiscovery   bool `json:"model_discovery"`
}

// Provider is the canonical ModelProvider contract.
type Provider interface {
	Name() string
	Capabilities() Capabilities
	Models(ctx context.Context) ([]string, error)
	Stream(ctx context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error)
	Health(ctx context.Context) (string, error)
}

// ---- FakeProvider (deterministic, scripted) ----

// ScriptStep is one scripted behavior unit.
type ScriptStep struct {
	Kind      string          // "text" | "tool_call" | "usage" | "error" | "complete" | "cancel"
	Text      string          // for text
	Tool      *agent.ToolCall // for tool_call
	Error     string          // for error
	Retryable bool            // for error
	Usage     *agent.Usage    // for usage
}

// FakeProvider replays scripts deterministically for conformance tests.
type FakeProvider struct {
	Scripts map[string][]ScriptStep // keyed by RequestID; "*" is default
	Calls   int
}

func NewFake(scripts map[string][]ScriptStep) *FakeProvider {
	return &FakeProvider{Scripts: scripts}
}

func (f *FakeProvider) Name() string { return "fake" }

func (f *FakeProvider) Capabilities() Capabilities {
	return Capabilities{Streaming: true, ToolCalls: true, StructuredOutput: true, Usage: true, Cancel: true, Health: true, ModelDiscovery: true}
}

func (f *FakeProvider) Models(_ context.Context) ([]string, error) {
	return []string{"fake-default"}, nil
}

func (f *FakeProvider) Health(_ context.Context) (string, error) { return "healthy", nil }

func (f *FakeProvider) Stream(ctx context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	f.Calls++
	steps := f.Scripts[req.RequestID]
	if steps == nil {
		steps = f.Scripts["*"]
	}
	ch := make(chan agent.ModelEvent, len(steps)+1)
	go func() {
		defer close(ch)
		for _, s := range steps {
			select {
			case <-ctx.Done():
				ch <- agent.ModelEvent{Kind: agent.EventCancelled, RequestID: req.RequestID}
				return
			default:
			}
			switch s.Kind {
			case "text":
				ch <- agent.ModelEvent{Kind: agent.EventTextDelta, RequestID: req.RequestID, Text: s.Text}
			case "tool_call":
				tool := s.Tool
				if tool == nil {
					tool = &agent.ToolCall{ID: "call-1", TurnID: req.TurnID, Name: "fs.read"}
				}
				ch <- agent.ModelEvent{Kind: agent.EventToolCallReady, RequestID: req.RequestID, ToolCall: tool}
			case "usage":
				ch <- agent.ModelEvent{Kind: agent.EventUsageUpdated, RequestID: req.RequestID, Usage: s.Usage}
			case "error":
				ch <- agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID, Error: s.Error, Retryable: s.Retryable}
			case "cancel":
				ch <- agent.ModelEvent{Kind: agent.EventCancelled, RequestID: req.RequestID}
				return
			case "complete":
				ch <- agent.ModelEvent{Kind: agent.EventCompleted, RequestID: req.RequestID, Finished: true}
				return
			}
		}
		ch <- agent.ModelEvent{Kind: agent.EventCompleted, RequestID: req.RequestID, Finished: true}
	}()
	return ch, nil
}

// ForName builds a provider by name. baseURL falls back to
// PRUMO_MODEL_BASE_URL for openai-compat; apiKey falls back to
// PRUMO_MODEL_API_KEY for network adapters.
func ForName(name, baseURL, apiKey, mdl string) (Provider, error) {
	switch name {
	case "", "fake":
		return NewFake(map[string][]ScriptStep{"*": {{Kind: "text", Text: "hello"}, {Kind: "complete"}}}), nil
	case "openai-compat":
		if baseURL == "" {
			baseURL = envOr("PRUMO_MODEL_BASE_URL", "")
		}
		if baseURL == "" {
			return nil, fmt.Errorf("openai-compat requires base-url or PRUMO_MODEL_BASE_URL")
		}
		if apiKey == "" {
			apiKey = envOr("PRUMO_MODEL_API_KEY", "")
		}
		return NewOpenAICompat(baseURL, apiKey, mdl), nil
	case "anthropic":
		if baseURL == "" {
			baseURL = envOr("PRUMO_MODEL_BASE_URL", "")
		}
		if apiKey == "" {
			apiKey = envOr("PRUMO_MODEL_API_KEY", "")
		}
		return NewAnthropic(baseURL, apiKey, mdl), nil
	default:
		return nil, fmt.Errorf("unknown provider %s (fake|openai-compat|anthropic)", name)
	}
}

// OpenAICompat calls any OpenAI-compatible /chat/completions endpoint with
// stream=true (SSE) and normalizes deltas to agent.ModelEvent.
type OpenAICompat struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
	// ExtraHeaders are sent on every request (e.g. x-session-id on gateways
	// whose free tier only serves requests tied to an account session).
	ExtraHeaders map[string]string
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ModelHeaders parses PRUMO_MODEL_HEADERS (a JSON object of header name to
// value) into a map. Empty or malformed values yield nil.
func ModelHeaders() map[string]string {
	raw := os.Getenv("PRUMO_MODEL_HEADERS")
	if raw == "" {
		return nil
	}
	m := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil
	}
	return m
}

func NewOpenAICompat(baseURL, apiKey, model string) *OpenAICompat {
	return &OpenAICompat{BaseURL: strings.TrimRight(baseURL, "/"), APIKey: apiKey, Model: model, Client: &http.Client{Timeout: 120 * time.Second}}
}

// WithHeaders sets extra headers sent on every request and returns the
// provider so callers can chain configuration.
func (o *OpenAICompat) WithHeaders(headers map[string]string) *OpenAICompat {
	o.ExtraHeaders = headers
	return o
}

func (o *OpenAICompat) applyHeaders(req *http.Request) {
	for name, value := range o.ExtraHeaders {
		req.Header.Set(name, value)
	}
}

func (o *OpenAICompat) Name() string { return "openai-compat" }

func (o *OpenAICompat) Capabilities() Capabilities {
	return Capabilities{Streaming: true, ToolCalls: true, StructuredOutput: true, Usage: true, Cancel: true, Health: true, ModelDiscovery: false}
}

func (o *OpenAICompat) Models(ctx context.Context) ([]string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, o.BaseURL+"/models", nil)
	if o.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.APIKey)
	}
	o.applyHeaders(req)
	if resp, err := o.Client.Do(req); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var doc struct {
				Data []struct {
					ID string `json:"id"`
				} `json:"data"`
			}
			if err := json.Unmarshal(readAll(resp), &doc); err == nil && len(doc.Data) > 0 {
				ids := make([]string, 0, len(doc.Data))
				for _, d := range doc.Data {
					if d.ID != "" {
						ids = append(ids, d.ID)
					}
				}
				if len(ids) > 0 {
					return ids, nil
				}
			}
		}
	}
	if o.Model != "" {
		return []string{o.Model}, nil
	}
	return []string{"default"}, nil
}

func (o *OpenAICompat) Health(ctx context.Context) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, o.BaseURL+"/models", nil)
	if o.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.APIKey)
	}
	o.applyHeaders(req)
	resp, err := o.Client.Do(req)
	if err != nil {
		return "unavailable", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return "degraded", fmt.Errorf("upstream %d", resp.StatusCode)
	}
	return "healthy", nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (o *OpenAICompat) Stream(ctx context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	model := req.Model
	if model == "" {
		model = o.Model
	}
	msgs := make([]chatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		msgs = append(msgs, chatMessage{Role: string(m.Role), Content: m.Content})
	}
	payload := map[string]any{"model": model, "messages": msgs, "stream": true}
	if len(req.Tools) > 0 {
		tools := make([]any, 0, len(req.Tools))
		for _, ts := range req.Tools {
			tools = append(tools, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name": ts.Name, "description": ts.Description, "parameters": ts.Schema,
				},
			})
		}
		payload["tools"] = tools
	}
	if len(req.ResponseFormat) > 0 {
		payload["response_format"] = map[string]any{"type": "json_schema", "json_schema": req.ResponseFormat}
	}
	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if o.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.APIKey)
	}
	o.applyHeaders(httpReq)
	resp, err := o.Client.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			ch := make(chan agent.ModelEvent, 1)
			ch <- agent.ModelEvent{Kind: agent.EventCancelled, RequestID: req.RequestID}
			close(ch)
			return ch, nil
		}
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := parseProviderError(resp.Body)
		resp.Body.Close()
		if msg == "" {
			msg = fmt.Sprintf("provider http %d", resp.StatusCode)
		} else {
			msg = fmt.Sprintf("provider http %d: %s", resp.StatusCode, msg)
		}
		retryable := resp.StatusCode == 429 || resp.StatusCode >= 500
		ch := make(chan agent.ModelEvent, 1)
		ch <- agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID, Error: msg, Retryable: retryable}
		close(ch)
		return ch, nil
	}
	ch := make(chan agent.ModelEvent, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 1024*1024), 1024*1024)
		for sc.Scan() {
			select {
			case <-ctx.Done():
				ch <- agent.ModelEvent{Kind: agent.EventCancelled, RequestID: req.RequestID}
				return
			default:
			}
			line := strings.TrimSpace(sc.Text())
			if !strings.HasPrefix(line, "data:") {
				if em := parseProviderError(strings.NewReader(line)); em != "" {
					ch <- agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID, Error: "provider error: " + em, Retryable: false}
					return
				}
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				ch <- agent.ModelEvent{Kind: agent.EventCompleted, RequestID: req.RequestID, Finished: true}
				return
			}
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content   string `json:"content"`
						ToolCalls []struct {
							ID       string `json:"id"`
							Function struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							} `json:"function"`
						} `json:"tool_calls"`
					} `json:"delta"`
				} `json:"choices"`
				Usage *struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if chunk.Usage != nil {
				ch <- agent.ModelEvent{Kind: agent.EventUsageUpdated, RequestID: req.RequestID, Usage: &agent.Usage{InputTokens: chunk.Usage.PromptTokens, OutputTokens: chunk.Usage.CompletionTokens}}
				continue
			}
			for _, c := range chunk.Choices {
				if c.Delta.Content != "" {
					ch <- agent.ModelEvent{Kind: agent.EventTextDelta, RequestID: req.RequestID, Text: c.Delta.Content}
				}
				for _, tc := range c.Delta.ToolCalls {
					args := map[string]any{}
					if tc.Function.Arguments != "" {
						_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
					}
					if err := ValidateArgs(req.Tools, tc.Function.Name, args); err != nil {
						ch <- agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID, Error: "invalid tool call: " + err.Error(), Retryable: false}
						continue
					}
					ch <- agent.ModelEvent{Kind: agent.EventToolCallReady, RequestID: req.RequestID, ToolCall: &agent.ToolCall{ID: tc.ID, TurnID: req.TurnID, Name: tc.Function.Name, Arguments: args, IdempotencyKey: req.RequestID + ":" + tc.ID}}
				}
			}
		}
		ch <- agent.ModelEvent{Kind: agent.EventCompleted, RequestID: req.RequestID, Finished: true}
	}()
	return ch, nil
}

func parseProviderError(r io.Reader) string {
	body, _ := io.ReadAll(io.LimitReader(r, 64*1024))
	if len(body) == 0 {
		return ""
	}
	var envelope struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Error.Message == "" {
		return ""
	}
	if envelope.Error.Type != "" {
		return envelope.Error.Type + ": " + envelope.Error.Message
	}
	return envelope.Error.Message
}
