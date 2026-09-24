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

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/modelregistry"
)

// priceUsage fills in a cost the endpoint did not report.
//
// The provider name is what the pricing table is keyed by, and the model is the
// one actually requested — not the provider's configured default — because a
// request that names a different model is billed at that model's rate.
func priceUsage(table modelregistry.PricingTable, provider, model string, usage *agent.Usage) {
	if table.Version == "" {
		return
	}
	modelregistry.Price(table, provider, model, usage)
}

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
	// The catalog is the single declaration of what exists; the switch below
	// builds it. A name the catalog lists that this factory cannot build is a
	// promise the catalog cannot keep, and a name this factory builds that the
	// catalog omits is a provider the catalog cannot describe. VerifiedAgainstCatalog
	// checks the two agree, so neither drift is silent (GAP-133).
	case "", "fake":
		return NewFake(map[string][]ScriptStep{"*": {{Kind: "text", Text: "hello"}, {Kind: "complete"}}}), nil
	case "fake-tools":
		// Deterministic and offline, but it asks for one tool. The plain fake
		// never does, so nothing in a live run ever reached the permission
		// gate — which is how the approval surface stayed unimplemented
		// without anyone noticing. fs.read is deliberately harmless.
		return NewFake(map[string][]ScriptStep{"*": {
			{Kind: "text", Text: "reading the workspace"},
			{Kind: "tool_call", Tool: &agent.ToolCall{
				ID: "c1", TurnID: "turn-1", Name: "fs.read",
				Arguments:      map[string]any{"path": "README.md"},
				IdempotencyKey: "fake-tools:c1",
			}},
			{Kind: "complete"},
		}}), nil
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
		if err := ValidateDestinationURL(baseURL, DefaultDestinationPolicy()); err != nil {
			return nil, err
		}
		return NewOpenAICompat(baseURL, apiKey, mdl), nil
	case "gemini":
		if apiKey == "" {
			apiKey = envOr("GEMINI_API_KEY", "")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("gemini requires api-key or GEMINI_API_KEY")
		}
		geminiBase := envOr("PRUMO_GEMINI_BASE_URL", "")
		if geminiBase != "" {
			if err := ValidateDestinationURL(geminiBase, DefaultDestinationPolicy()); err != nil {
				return nil, err
			}
		}
		return NewGeminiWithPolicy(geminiBase, apiKey, mdl, DefaultDestinationPolicy()), nil
	case "opencode":
		// A delegated turn: opencode runs it with its own tools, its own
		// permission policy and its own authentication (including the models it
		// serves for free). Prumo gets the answer and the spend, and never sees
		// the tool calls — see OpenCode's doc comment for why that is stated
		// rather than papered over.
		return NewOpenCode(mdl), nil
	case "anthropic":
		if baseURL == "" {
			baseURL = envOr("PRUMO_MODEL_BASE_URL", "")
		}
		if apiKey == "" {
			apiKey = envOr("PRUMO_MODEL_API_KEY", "")
		}
		if baseURL != "" {
			if err := ValidateDestinationURL(baseURL, DefaultDestinationPolicy()); err != nil {
				return nil, err
			}
		}
		return NewAnthropic(baseURL, apiKey, mdl), nil
	default:
		return nil, fmt.Errorf("unknown provider %s (fake|fake-tools|openai-compat|anthropic|gemini|opencode)", name)
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
	// Provider is the vendor whose published prices apply, for the many
	// compatible endpoints that are not OpenAI.
	Provider string
	// Pricing prices this provider's usage. The zero value has no rates, so a
	// provider built without one reports no cost — which is honest, and loud
	// through modelregistry.Pricable, rather than a silent zero.
	Pricing modelregistry.PricingTable
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

// NewOpenAICompat builds a provider against any OpenAI-compatible endpoint.
//
// The client is destination-checked: the base URL is a request field, so
// without this the harness would send the conversation and the API key to any
// address the caller named, including the cloud metadata endpoint (GAP-111).
func NewOpenAICompat(baseURL, apiKey, model string) *OpenAICompat {
	return NewOpenAICompatWithPolicy(baseURL, apiKey, model, DefaultDestinationPolicy())
}

// NewOpenAICompatWithPolicy is the form a caller uses when it has a destination
// policy of its own, such as a local gateway reached over loopback.
func NewOpenAICompatWithPolicy(baseURL, apiKey, model string, policy DestinationPolicy) *OpenAICompat {
	return &OpenAICompat{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		Model:   model,
		Client:  NewDestinationHTTPClient(policy),
		// Priced by default. This endpoint returns tokens and no cost, so a
		// provider built without a table reports a cost of zero for every call and
		// a US dollar budget never moves (GAP-129).
		Pricing: modelregistry.DefaultPricing(),
	}
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

// ProviderKey is the vendor whose published prices apply. The adapter is a
// protocol, not a vendor: the same client talks to OpenAI, OpenRouter and
// anything else compatible, and their rates differ. A caller that knows the
// vendor sets it.
//
// It defaults to OpenAI because that is the vendor the built-in table is keyed
// by. Defaulting to the adapter's own name instead looked right and priced
// nothing at all: "openai-compat/gpt-4o" is not in the table, so every lookup
// missed and every call reported a cost of zero.
func (o *OpenAICompat) ProviderKey() string {
	if o.Provider != "" {
		return o.Provider
	}
	return "openai"
}

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
	Content any    `json:"content"`
	// ToolCalls is what an assistant message asked for, and ToolCallID names the
	// call a tool message answers. Both are required by the API for a tool round
	// trip: a tool result with no tool_call_id cannot be matched to its request,
	// and an assistant turn that asked for tools but does not say so leaves the
	// result answering a call that was never made (GAP-115).
	ToolCalls  []chatToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type chatToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function chatToolFunc `json:"function"`
}

type chatToolFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func (o *OpenAICompat) Stream(ctx context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	model := req.Model
	if model == "" {
		model = o.Model
	}
	msgs := make([]chatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		role := openAIRole(m.Role)
		if len(m.Parts) == 0 {
			content := any(m.Content)
			if m.Role == agent.RoleTool {
				content = toolResultContent(m)
			}
			msg := chatMessage{Role: role, Content: content, ToolCallID: m.ToolCallID}
			if len(m.ToolCalls) > 0 {
				// An assistant that asked for tools says so; the arguments go as
				// a JSON string, which is what this API expects.
				for _, tc := range m.ToolCalls {
					args, err := json.Marshal(tc.Arguments)
					if err != nil {
						args = []byte("{}")
					}
					msg.ToolCalls = append(msg.ToolCalls, chatToolCall{
						ID:   tc.ID,
						Type: "function",
						Function: chatToolFunc{
							Name:      tc.Name,
							Arguments: string(args),
						},
					})
				}
			}
			msgs = append(msgs, msg)
			continue
		}
		parts := make([]any, 0, len(m.Parts))
		for _, p := range m.Parts {
			switch p.Type {
			case "image":
				mime := p.MimeType
				if mime == "" {
					mime = "image/png"
				}
				dataURL := fmt.Sprintf("data:%s;base64,%s", mime, p.Data)
				parts = append(parts, map[string]any{
					"type": "image_url",
					"image_url": map[string]any{
						"url": dataURL,
					},
				})
			default:
				parts = append(parts, map[string]any{
					"type": "text",
					"text": p.Text,
				})
			}
		}
		msgs = append(msgs, chatMessage{Role: role, Content: parts})
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
		// The status code and the Retry-After header are read here, where they
		// still exist. Reducing this to a message string meant the gateway had to
		// guess whether it was a rate limit, and the delay the provider asked for
		// was thrown away, so a retry came back inside the limit window and was
		// rejected again (GAP-108).
		pe := NewProviderError(o.ProviderKey(), resp, msg)
		retryable := resp.StatusCode == 429 || resp.StatusCode >= 500 || pe.Quota
		ch := make(chan agent.ModelEvent, 1)
		ch <- agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID, Error: pe.Error(),
			Retryable: retryable, Quota: pe.Quota, RetryAfter: pe.RetryAfter}
		close(ch)
		return ch, nil
	}
	ch := make(chan agent.ModelEvent, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		// Usage is priced against the model this request named, not the
		// adapter's configured default, and accumulates across chunks so the
		// last event is the call's total.
		seen := newCumulativeUsage(o.Pricing, o.ProviderKey(), model)
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
					// A cached prefix is reported inside the prompt count, not
					// beside it: the detail is what says how much of the prompt
					// the provider did not have to read again.
					PromptTokensDetails *struct {
						CachedTokens int `json:"cached_tokens"`
					} `json:"prompt_tokens_details"`
					// Reasoning tokens arrive as a breakdown of the completion
					// count. They are read so a run can show that its cost was
					// mostly thinking rather than answering, and are deliberately
					// not added to any total: the provider already includes them
					// in CompletionTokens (GAP-130).
					CompletionTokensDetails *struct {
						ReasoningTokens int `json:"reasoning_tokens"`
					} `json:"completion_tokens_details"`
				} `json:"usage"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if chunk.Usage != nil {
				// A cached token is counted inside the prompt and reported here
				// too, so it is moved rather than added: a total that counted it
				// twice would overstate the run.
				cacheRead := 0
				if chunk.Usage.PromptTokensDetails != nil {
					cacheRead = chunk.Usage.PromptTokensDetails.CachedTokens
				}
				input := chunk.Usage.PromptTokens - cacheRead
				usage := seen.advance(input, chunk.Usage.CompletionTokens, cacheRead, 0)
				if chunk.Usage.CompletionTokensDetails != nil {
					usage.ReasoningTokens = chunk.Usage.CompletionTokensDetails.ReasoningTokens
				}
				ch <- agent.ModelEvent{Kind: agent.EventUsageUpdated, RequestID: req.RequestID, Usage: usage}
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
		// Reaching here without having seen [DONE] means the stream was cut: a
		// connection that dropped, a read that failed, or a line past the
		// scanner's buffer. The loop ending is not the provider saying it
		// finished — only [DONE] is — so reporting completion here would record a
		// truncated answer as a finished run, with no indication that the tail is
		// missing (GAP-131).
		if err := sc.Err(); err != nil {
			ch <- agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID,
				Error: "provider stream ended before completion: " + err.Error(), Retryable: true}
			return
		}
		ch <- agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID,
			Error: "provider stream ended without a completion marker", Retryable: true}
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

// openAIRole maps an internal role onto one this API accepts.
//
// The internal vocabulary has "agent" for a turn the assistant produced; the
// wire vocabulary does not, and a role it does not know is a rejected request
// rather than a tolerated one. Passing it through verbatim meant every tool
// round trip was sent with a role the provider would refuse (GAP-115).
func openAIRole(role agent.Role) string {
	switch role {
	case agent.RoleSystem, agent.RoleUser, agent.RoleTool:
		return string(role)
	case agent.RoleAgent:
		// "agent" is this codebase's word for the assistant's turn; the wire word
		// is "assistant", and anything else is a rejected request.
		return "assistant"
	default:
		return "user"
	}
}

// toolResultContent renders a tool message's body.
//
// A successful result is its output. A failed one says so, because a tool that
// failed leaves its output empty, and an empty tool result is indistinguishable
// from a tool that succeeded and returned nothing — which is how a model ends up
// reasoning from a failure as though it were a result.
func toolResultContent(m agent.Message) string {
	if m.Metadata == nil {
		return m.Content
	}
	if okFlag, present := m.Metadata["ok"]; present {
		if isOK, isBool := okFlag.(bool); isBool && isOK {
			return m.Content
		}
	} else {
		// No verdict recorded: the result is taken at face value.
		return m.Content
	}
	reason, _ := m.Metadata["error"].(string)
	if reason == "" {
		if code, present := m.Metadata["exit_code"]; present {
			reason = fmt.Sprintf("exited %v", code)
		} else {
			reason = "reported a failure with no reason"
		}
	}
	if m.Content != "" {
		return fmt.Sprintf("tool failed: %s\npartial output: %s", reason, m.Content)
	}
	return fmt.Sprintf("tool failed: %s", reason)
}

// cumulativeUsage turns a provider's split usage reports into totals.
//
// Anthropic announces the input cost at message_start and the output cost at
// message_delta, as two separate events. Forwarding each as it arrives means the
// last event is the output alone: a run that spent 1000 input and 500 output
// tokens reads as having spent 500, and every cost, budget and report derived
// from that last event understates the call by the input that was already paid
// for.
//
// Each dimension therefore keeps its high-water mark and every event reports the
// total so far. The figures are cumulative per message, so a later event carries
// a larger or equal value, never a smaller one; a provider that corrected a
// figure downward would be ignored, which is preferable to double-counting a
// bill.
type cumulativeUsage struct {
	table    modelregistry.PricingTable
	provider string
	model    string
	last     agent.Usage
}

func newCumulativeUsage(table modelregistry.PricingTable, provider, model string) *cumulativeUsage {
	return &cumulativeUsage{table: table, provider: provider, model: model}
}

func (c *cumulativeUsage) advance(in, out, cacheRead, cacheWrite int) *agent.Usage {
	if in > c.last.InputTokens {
		c.last.InputTokens = in
	}
	if out > c.last.OutputTokens {
		c.last.OutputTokens = out
	}
	if cacheRead > c.last.CacheReadTokens {
		c.last.CacheReadTokens = cacheRead
	}
	if cacheWrite > c.last.CacheWriteTokens {
		c.last.CacheWriteTokens = cacheWrite
	}
	usage := c.last
	// The cost is derived from the pricing table (GAP-129): these endpoints report
	// tokens and no cost, so without it a US dollar budget compares a fixed zero
	// against a limit and never moves.
	priceUsage(c.table, c.provider, c.model, &usage)
	return &usage
}

func (c *cumulativeUsage) advanceAnthropic(block *anthropicUsageBlock) *agent.Usage {
	return c.advance(block.InputTokens, block.OutputTokens, block.CacheReadInputTokens, block.CacheCreationInputTokens)
}

// buildableNames is what this factory can actually construct, read off the switch
// above by asking it. It is a probe, not a second list: adding a case to ForName
// changes this without anyone maintaining a parallel declaration.
func buildableNames() map[string]bool {
	names := map[string]bool{}
	for _, spec := range modelregistry.KnownProviders() {
		// A provider needing a base URL cannot be probed without one, so it is
		// taken on the catalog's word; the offline ones are checked by building.
		if spec.NeedsBaseURL {
			continue
		}
		if _, err := ForName(spec.Name, "", "", "m"); err == nil {
			names[spec.Name] = true
		}
	}
	return names
}

// VerifiedAgainstCatalog reports what the factory and the catalog disagree about.
//
// Two ways to drift, and both are worth surfacing rather than assuming away: a
// provider the catalog describes that nothing can build — the original GAP-133 —
// and a provider the factory can build that the catalog has never heard of,
// which is the same problem in the other direction.
func VerifiedAgainstCatalog() error {
	buildable := buildableNames()
	var missing []string
	for _, spec := range modelregistry.KnownProviders() {
		if buildable[spec.Name] || spec.External {
			// A provider the deployment supplies rather than this factory is
			// described by the catalog and built by something else; that is a
			// legitimate pairing, not a missing one.
			continue
		}
		if spec.NeedsBaseURL || spec.NeedsAPIKey {
			// Cannot be probed without a credential or an endpoint. ForName's own
			// error messages cover those, and pretending otherwise would make the
			// check report a provider missing when it is merely unconfigured.
			continue
		}
		missing = append(missing, spec.Name)
	}
	if len(missing) > 0 {
		return fmt.Errorf("catalog describes providers the factory cannot build: %s", strings.Join(missing, ", "))
	}
	for name := range buildable {
		if _, known := modelregistry.ProviderSpecFor(name); !known {
			return fmt.Errorf("factory can build %q but the catalog has no descriptor for it", name)
		}
	}
	return nil
}
