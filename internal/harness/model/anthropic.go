// Anthropic Messages API adapter (second real ModelProvider).
//
// Normalizes Anthropic SSE streams to agent.ModelEvent. Vendor types never
// escape this file. Live credentials remain environmental; httptest covers
// the wire mapping deterministically.
package model

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/modelregistry"
)

// Anthropic calls {BaseURL}/v1/messages with stream=true.
type Anthropic struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
	// Provider names the vendor whose published prices apply.
	Provider string
	// Pricing prices this provider's usage. The endpoint returns tokens and no
	// cost, so without it a US dollar budget compares a fixed zero against a
	// limit and never moves (GAP-129).
	Pricing modelregistry.PricingTable
}

// ProviderKey is the vendor whose published prices apply. The endpoint can be
// pointed at a compatible gateway whose rates differ, so it is set explicitly
// rather than assumed from the base URL.
func (a *Anthropic) ProviderKey() string {
	if a.Provider != "" {
		return a.Provider
	}
	return "anthropic"
}

// NewAnthropic builds an Anthropic provider. The client is destination-checked
// for the same reason as openai-compat: the base URL can come from a request
// (GAP-111).
func NewAnthropic(baseURL, apiKey, model string) *Anthropic {
	return NewAnthropicWithPolicy(baseURL, apiKey, model, DefaultDestinationPolicy())
}

// NewAnthropicWithPolicy takes an explicit destination policy.
func NewAnthropicWithPolicy(baseURL, apiKey, model string, policy DestinationPolicy) *Anthropic {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	return &Anthropic{
		BaseURL: baseURL, APIKey: apiKey, Model: model,
		Client: NewDestinationHTTPClient(policy),
		// Priced by default: this endpoint returns tokens and no cost (GAP-129).
		Pricing: modelregistry.DefaultPricing(),
	}
}

func (a *Anthropic) Name() string { return "anthropic" }

func (a *Anthropic) Capabilities() Capabilities {
	return Capabilities{Streaming: true, ToolCalls: true, StructuredOutput: true, Usage: true, Cancel: true, Health: true, ModelDiscovery: false}
}

func (a *Anthropic) Models(_ context.Context) ([]string, error) {
	if a.Model != "" {
		return []string{a.Model}, nil
	}
	return []string{"default"}, nil
}

func (a *Anthropic) Health(ctx context.Context) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, a.BaseURL+"/v1/models", nil)
	a.setAuth(req)
	resp, err := a.Client.Do(req)
	if err != nil {
		return "unavailable", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return "degraded", fmt.Errorf("upstream %d", resp.StatusCode)
	}
	return "healthy", nil
}

func (a *Anthropic) setAuth(req *http.Request) {
	if a.APIKey != "" {
		req.Header.Set("x-api-key", a.APIKey)
	}
	req.Header.Set("anthropic-version", "2023-06-01")
}

type anthropicOutboundMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

func (a *Anthropic) Stream(ctx context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	modelName := req.Model
	if modelName == "" {
		modelName = a.Model
	}
	msgs := make([]anthropicOutboundMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		role := string(m.Role)
		if role != "user" && role != "assistant" {
			role = "user"
		}
		// A tool result is a user turn carrying a tool_result block, not prose.
		// Sent as plain text it reads as something the person said, and the model
		// has no way to tie it to the call that produced it (GAP-115).
		if m.Role == agent.RoleTool {
			text := toolResultContent(m)
			msgs = append(msgs, anthropicOutboundMessage{
				Role: "user",
				Content: []map[string]any{{
					"type":        "tool_result",
					"tool_use_id": m.ToolCallID,
					"content":     text,
				}},
			})
			continue
		}
		if len(m.Parts) == 0 {
			if len(m.ToolCalls) > 0 {
				// An assistant that asked for tools says so with tool_use blocks,
				// so the result that follows has something to answer.
				blocks := make([]map[string]any, 0, len(m.ToolCalls))
				for _, tc := range m.ToolCalls {
					input := tc.Arguments
					if input == nil {
						input = map[string]any{}
					}
					blocks = append(blocks, map[string]any{
						"type":  "tool_use",
						"id":    tc.ID,
						"name":  tc.Name,
						"input": input,
					})
				}
				msgs = append(msgs, anthropicOutboundMessage{Role: "assistant", Content: blocks})
				continue
			}
			msgs = append(msgs, anthropicOutboundMessage{Role: role, Content: m.Content})
			continue
		}
		blocks := make([]map[string]any, 0, len(m.Parts))
		for _, p := range m.Parts {
			switch p.Type {
			case "image":
				mediaType := p.MimeType
				if mediaType == "" {
					mediaType = "image/png"
				}
				blocks = append(blocks, map[string]any{
					"type": "image",
					"source": map[string]any{
						"type":       "base64",
						"media_type": mediaType,
						"data":       p.Data,
					},
				})
			default:
				blocks = append(blocks, map[string]any{
					"type": "text",
					"text": p.Text,
				})
			}
		}
		msgs = append(msgs, anthropicOutboundMessage{Role: role, Content: blocks})
	}
	payload := map[string]any{
		"model": modelName, "max_tokens": 1024, "stream": true, "messages": msgs,
	}
	// The tool catalogue goes in the request. This provider declared
	// ToolCalls: true and then never sent a single schema, so a model had no way
	// to know the tools existed — the capability was a claim in a struct
	// (GAP-114). Anthropic names the schema "input_schema" and takes no wrapper
	// type, unlike the OpenAI-compatible shape.
	if len(req.Tools) > 0 {
		tools := make([]any, 0, len(req.Tools))
		for _, ts := range req.Tools {
			tool := map[string]any{"name": ts.Name}
			if ts.Description != "" {
				tool["description"] = ts.Description
			}
			// An absent schema is worse than an empty one here: a model asked to
			// call a tool with no declared arguments tends to send none, and the
			// call then fails for a reason nothing told it.
			schema := ts.Schema
			if schema == nil {
				schema = map[string]any{"type": "object", "properties": map[string]any{}}
			}
			tool["input_schema"] = schema
			tools = append(tools, tool)
		}
		payload["tools"] = tools
	}
	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.BaseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	a.setAuth(httpReq)
	resp, err := a.Client.Do(httpReq)
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
		resp.Body.Close()
		retryable := resp.StatusCode == 429 || resp.StatusCode >= 500
		ch := make(chan agent.ModelEvent, 1)
		ch <- agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID, Error: fmt.Sprintf("provider http %d", resp.StatusCode), Retryable: retryable}
		close(ch)
		return ch, nil
	}
	ch := make(chan agent.ModelEvent, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		a.consumeSSE(ctx, req, modelName, resp, ch)
	}()
	return ch, nil
}

type anthropicToolBuf struct {
	id   string
	name string
	json strings.Builder
}

// consumeSSE reads the stream. It takes the resolved model name because the
// stream is a separate function and the cost is priced per model — pricing under
// the adapter's configured default would charge a request for one model at
// another's rate.
func (a *Anthropic) consumeSSE(ctx context.Context, req agent.ModelRequest, modelName string, resp *http.Response, ch chan<- agent.ModelEvent) {
	seen := newCumulativeUsage(a.Pricing, a.ProviderKey(), modelName)
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	eventName := ""
	tools := map[int]*anthropicToolBuf{}
	emit := func(ev agent.ModelEvent) bool {
		select {
		case <-ctx.Done():
			ch <- agent.ModelEvent{Kind: agent.EventCancelled, RequestID: req.RequestID}
			return false
		case ch <- ev:
			return true
		}
	}
	flush := func(index int) {
		buf, ok := tools[index]
		if !ok || buf.name == "" {
			return
		}
		args := map[string]any{}
		if s := buf.json.String(); s != "" {
			_ = json.Unmarshal([]byte(s), &args)
		}
		if err := ValidateArgs(req.Tools, buf.name, args); err != nil {
			emit(agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID, Error: "invalid tool call: " + err.Error(), Retryable: false})
			delete(tools, index)
			return
		}
		id := buf.id
		if id == "" {
			id = fmt.Sprintf("tool-%d", index)
		}
		ch <- agent.ModelEvent{Kind: agent.EventToolCallReady, RequestID: req.RequestID,
			ToolCall: &agent.ToolCall{ID: id, TurnID: req.TurnID, Name: buf.name, Arguments: args, IdempotencyKey: req.RequestID + ":" + id}}
		delete(tools, index)
	}
	for sc.Scan() {
		select {
		case <-ctx.Done():
			ch <- agent.ModelEvent{Kind: agent.EventCancelled, RequestID: req.RequestID}
			return
		default:
		}
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var payload struct {
			Type  string `json:"type"`
			Index *int   `json:"index"`
			Delta *struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				PartialJSON string `json:"partial_json"`
			} `json:"delta"`
			ContentBlock *struct {
				Type string `json:"type"`
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"content_block"`
			Usage   *anthropicUsageBlock `json:"usage"`
			Message *struct {
				Usage *anthropicUsageBlock `json:"usage"`
			} `json:"message"`
			Error *struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			continue
		}
		if payload.Error != nil || payload.Type == "error" || eventName == "error" {
			msg := "provider error"
			retryable := false
			if payload.Error != nil {
				msg = payload.Error.Message
				if msg == "" {
					msg = payload.Error.Type
				}
				switch payload.Error.Type {
				case "overloaded_error", "rate_limit_error", "api_error", "timeout_error":
					retryable = true
				}
			}
			if !emit(agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID, Error: msg, Retryable: retryable}) {
				return
			}
			continue
		}
		if payload.Usage != nil {
			if !emit(agent.ModelEvent{Kind: agent.EventUsageUpdated, RequestID: req.RequestID, Usage: seen.advanceAnthropic(payload.Usage)}) {
				return
			}
		}
		if payload.Message != nil && payload.Message.Usage != nil {
			if !emit(agent.ModelEvent{Kind: agent.EventUsageUpdated, RequestID: req.RequestID, Usage: seen.advanceAnthropic(payload.Message.Usage)}) {
				return
			}
		}
		if payload.Delta != nil {
			switch payload.Delta.Type {
			case "text_delta":
				if payload.Delta.Text != "" {
					if !emit(agent.ModelEvent{Kind: agent.EventTextDelta, RequestID: req.RequestID, Text: payload.Delta.Text}) {
						return
					}
				}
			case "input_json_delta":
				idx := 0
				if payload.Index != nil {
					idx = *payload.Index
				}
				buf, ok := tools[idx]
				if !ok {
					buf = &anthropicToolBuf{}
					tools[idx] = buf
				}
				buf.json.WriteString(payload.Delta.PartialJSON)
			}
		}
		if payload.ContentBlock != nil && payload.ContentBlock.Type == "tool_use" {
			idx := 0
			if payload.Index != nil {
				idx = *payload.Index
			}
			buf, ok := tools[idx]
			if !ok {
				buf = &anthropicToolBuf{}
				tools[idx] = buf
			}
			buf.id = payload.ContentBlock.ID
			buf.name = payload.ContentBlock.Name
		}
		if payload.Type == "content_block_stop" && payload.Index != nil {
			flush(*payload.Index)
		}
		if payload.Type == "message_stop" {
			for idx := range tools {
				flush(idx)
			}
			emit(agent.ModelEvent{Kind: agent.EventCompleted, RequestID: req.RequestID, Finished: true})
			return
		}
	}
	for idx := range tools {
		flush(idx)
	}
	ch <- agent.ModelEvent{Kind: agent.EventCompleted, RequestID: req.RequestID, Finished: true}
}

// anthropicUsageBlock is the usage report this endpoint returns, named so the
// mapping onto ours is written once. The same four numbers arrive on two
// different events, and duplicating the mapping is how one of them drifts.
type anthropicUsageBlock struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// anthropicUsage maps a provider usage report onto ours.
func anthropicUsage(u *anthropicUsageBlock) *agent.Usage {
	return &agent.Usage{
		InputTokens:      u.InputTokens,
		OutputTokens:     u.OutputTokens,
		CacheReadTokens:  u.CacheReadInputTokens,
		CacheWriteTokens: u.CacheCreationInputTokens,
	}
}
