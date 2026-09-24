package model

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/modelregistry"
)

// Gemini speaks Google's generateContent dialect. The shape here was pinned
// against the live API before any of it was written, because four of its details
// are not guessable and three of them are hard failures rather than quirks:
//
//   - Tool arguments come back as `args`, not `arguments`.
//   - The assistant role is `model`, not `assistant`.
//   - A functionCall part carries a `thoughtSignature`, and a follow-up turn that
//     replays the call without it is a 400. The signature does not appear on
//     text parts for replay purposes, so it is carried on the call and nowhere
//     else.
//   - Reasoning is `thoughtsTokenCount`, separate from the candidate count, and
//     cache detail is an array rather than an object.
//
// The pinned expectations live in gemini_contract_test.go behind the
// integration_live tag, so a change in the API fails there rather than quietly
// mis-parsing here.

// geminiThoughtSignatureKey namespaces the opaque token so the core never has to
// know whose it is and a second provider cannot collide with it.
// GeminiThoughtSignatureKey is where the adapter stores its opaque round-trip
// token on a tool call. It is exported so a conformance test can assert the key
// is present without reaching into the adapter's internals.
const GeminiThoughtSignatureKey = "gemini.thought_signature"

// Gemini calls {BaseURL}/models/{model}:streamGenerateContent.
type Gemini struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
	// Provider names the vendor whose published prices apply.
	Provider string
	// Pricing prices this provider's usage. The endpoint reports tokens and no
	// cost, so without it a US dollar budget compares a fixed zero against a limit
	// and never moves (GAP-129).
	Pricing modelregistry.PricingTable
}

const defaultGeminiBase = "https://generativelanguage.googleapis.com/v1beta"

// NewGemini builds a Gemini provider. The client is destination-checked for the
// same reason as the other adapters: the base URL can come from configuration, and
// a provider that will dial anything it is handed is an SSRF primitive.
func NewGemini(apiKey, mdl string) *Gemini {
	return NewGeminiWithPolicy("", apiKey, mdl, DefaultDestinationPolicy())
}

// NewGeminiWithPolicy builds a Gemini provider with an explicit destination policy.
func NewGeminiWithPolicy(baseURL, apiKey, mdl string, policy DestinationPolicy) *Gemini {
	if baseURL == "" {
		baseURL = defaultGeminiBase
	}
	provider := "gemini"
	return &Gemini{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		APIKey:   apiKey,
		Model:    mdl,
		Provider: provider,
		Client:   NewDestinationHTTPClient(policy),
		Pricing:  modelregistry.DefaultPricing(),
	}
}

func (g *Gemini) Name() string { return "gemini" }

// ProviderKey is the vendor whose published prices apply.
func (g *Gemini) ProviderKey() string {
	if g.Provider != "" {
		return g.Provider
	}
	return "gemini"
}

// Capabilities reports what this adapter does. ToolCalls is true, which is what
// separates it from a delegating provider like the opencode CLI: this one hands
// the calls back and lets Prumo run them under its own permission policy.
func (g *Gemini) Capabilities() Capabilities {
	return Capabilities{Streaming: true, ToolCalls: true, Usage: true, Cancel: true, Health: true}
}

// Models lists the models this endpoint advertises.
func (g *Gemini) Models(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.BaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	g.setAuth(req)
	resp, err := g.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gemini models: http %d", resp.StatusCode)
	}
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(payload.Models))
	for _, m := range payload.Models {
		if name := strings.TrimPrefix(m.Name, "models/"); name != "" {
			out = append(out, name)
		}
	}
	return out, nil
}

// Health probes the model list, which is the cheapest authenticated call.
func (g *Gemini) Health(ctx context.Context) (string, error) {
	if _, err := g.Models(ctx); err != nil {
		return "unreachable", err
	}
	return "healthy", nil
}

// setAuth attaches the key as a header rather than a query parameter.
//
// The API accepts both, and the query form puts a live credential in the URL where
// it lands in proxy logs, error strings and any request dump. A key that has to be
// scrubbed out of diagnostics is a key that will eventually be read out of one.
func (g *Gemini) setAuth(req *http.Request) {
	// The environment is the fallback rather than the override: an explicit
	// key passed by a caller is a decision, and silently replacing it with
	// whatever the environment happens to hold is how a run talks to the wrong
	// account.
	if g.APIKey != "" {
		req.Header.Set("x-goog-api-key", g.APIKey)
		return
	}
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		req.Header.Set("x-goog-api-key", key)
	}
}

// geminiContents is the request body.
type geminiContents struct {
	Contents         []geminiContent         `json:"contents"`
	Tools            []geminiTool            `json:"tools,omitempty"`
	ToolConfig       *geminiToolConfig       `json:"toolConfig,omitempty"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text             string               `json:"text,omitempty"`
	FunctionCall     *geminiFunctionCall  `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionReply `json:"functionResponse,omitempty"`
	ThoughtSignature string               `json:"thoughtSignature,omitempty"`
}

type geminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
	ID   string         `json:"id,omitempty"`
}

type geminiFunctionReply struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type geminiTool struct {
	FunctionDeclarations []geminiDeclaration `json:"functionDeclarations"`
}

type geminiDeclaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type geminiToolConfig struct {
	FunctionCallingConfig geminiCallingConfig `json:"functionCallingConfig"`
}

type geminiCallingConfig struct {
	Mode                 string   `json:"mode"`
	AllowedFunctionNames []string `json:"allowedFunctionNames,omitempty"`
}

type geminiGenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
}

// Stream issues the request and reads the SSE response.
func (g *Gemini) Stream(ctx context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	modelName := req.Model
	if modelName == "" {
		modelName = g.Model
	}
	body, err := json.Marshal(g.requestBody(req))
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", g.BaseURL, modelName)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	g.setAuth(httpReq)

	resp, err := g.Client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read the status and Retry-After here, where they still exist, so the
		// router can tell a quota from a fault and wait the interval asked for
		// (GAP-108).
		message := parseProviderError(resp.Body)
		resp.Body.Close()
		pe := NewProviderError(g.ProviderKey(), resp, message)
		ch := make(chan agent.ModelEvent, 1)
		retryable := resp.StatusCode == 429 || resp.StatusCode >= 500 || pe.Quota
		ch <- agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID, Error: pe.Error(),
			Retryable: retryable, Quota: pe.Quota, RetryAfter: pe.RetryAfter}
		close(ch)
		return ch, nil
	}

	ch := make(chan agent.ModelEvent, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		// The endpoint reports usage on every chunk, cumulatively, so the same
		// tracker the Anthropic path uses is what keeps the last event a total
		// rather than the last fragment (GAP-131).
		seen := newCumulativeUsage(g.Pricing, g.ProviderKey(), modelName)
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 1024*1024), 8*1024*1024)
		emitted := false
		// This endpoint sends no [DONE] and no stop event. What it does send is a
		// finishReason on the candidate that ends the response, and that — not the
		// stream simply ending — is what says the answer is complete. Treating the
		// end of the body as completion is the GAP-131 shape in a new dialect: a
		// connection that dropped mid-answer would be reported as a finished run.
		finished := false
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload == "" || payload == "[DONE]" {
				continue
			}
			var chunk geminiStreamChunk
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				continue
			}
			for _, candidate := range chunk.Candidates {
				if candidate.FinishReason != "" {
					finished = true
				}
			}
			for _, ev := range g.eventsFromChunk(&chunk, req, seen) {
				emitted = true
				select {
				case ch <- ev:
				case <-ctx.Done():
					return
				}
			}
		}
		// A stream that ends without the endpoint saying it finished is a cut
		// stream. Reporting completion there records a truncated answer as a
		// finished run (GAP-131).
		if err := sc.Err(); err != nil {
			select {
			case ch <- agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID,
				Error: "gemini stream ended before completion: " + err.Error(), Retryable: true}:
			case <-ctx.Done():
			}
			return
		}
		switch {
		case !emitted:
			select {
			case ch <- agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID,
				Error: "gemini stream produced no events", Retryable: true}:
			case <-ctx.Done():
			}
			return
		case !finished:
			select {
			case ch <- agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID,
				Error: "gemini stream ended without a finish reason", Retryable: true}:
			case <-ctx.Done():
			}
			return
		}
		select {
		case ch <- agent.ModelEvent{Kind: agent.EventCompleted, RequestID: req.RequestID, Finished: true}:
		case <-ctx.Done():
		}
	}()
	return ch, nil
}

type geminiStreamChunk struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
			Role  string       `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata *geminiUsage `json:"usageMetadata"`
}

type geminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
	ThoughtsTokenCount   int `json:"thoughtsTokenCount"`
	CachedContentTokens  int `json:"cachedContentTokenCount"`
	// PromptTokensDetails is an array of per-modality entries, not an object.
	// Reading it as an object silently yields nothing, which is a usage report
	// that is wrong by exactly the amount it undercounts.
	PromptTokensDetails []struct {
		Modality   string `json:"modality"`
		TokenCount int    `json:"tokenCount"`
	} `json:"promptTokensDetails"`
}

// eventsFromChunk turns one streamed chunk into runtime events.
func (g *Gemini) eventsFromChunk(chunk *geminiStreamChunk, req agent.ModelRequest, seen *cumulativeUsage) []agent.ModelEvent {
	var out []agent.ModelEvent
	for _, candidate := range chunk.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				out = append(out, agent.ModelEvent{Kind: agent.EventTextDelta, RequestID: req.RequestID, Text: part.Text})
			}
			if call := part.FunctionCall; call != nil {
				toolCall := &agent.ToolCall{
					ID: call.ID, TurnID: req.TurnID, Name: call.Name, Arguments: call.Args,
					IdempotencyKey: req.RequestID + ":" + call.Name,
				}
				// The signature has to come back on the next turn or the API refuses
				// the whole conversation with a 400, so it travels with the call
				// rather than being dropped as vendor noise (GAP-174).
				if part.ThoughtSignature != "" {
					toolCall.ProviderOpaque = map[string]string{
						GeminiThoughtSignatureKey: part.ThoughtSignature,
					}
				}
				out = append(out, agent.ModelEvent{Kind: agent.EventToolCallReady, RequestID: req.RequestID, ToolCall: toolCall})
			}
		}
	}
	if usage := chunk.UsageMetadata; usage != nil {
		// Cache reads are reported inside the prompt count, so they are moved
		// rather than added: a total that counted them twice would overstate the
		// run.
		cacheRead := usage.CachedContentTokens
		for _, entry := range usage.PromptTokensDetails {
			if strings.EqualFold(entry.Modality, "TEXT") && entry.TokenCount > cacheRead {
				cacheRead = 0
			}
		}
		reported := seen.advance(usage.PromptTokenCount-cacheRead, usage.CandidatesTokenCount, cacheRead, 0)
		reported.ReasoningTokens = usage.ThoughtsTokenCount
		out = append(out, agent.ModelEvent{Kind: agent.EventUsageUpdated, RequestID: req.RequestID, Usage: reported})
	}
	return out
}

// RequestBody maps the neutral request onto Google's shape. It is exported so a
// conformance test can assert what goes on the wire without a live call, which is
// the only way to check the parts a 400 would otherwise reveal only in
// production.
func (g *Gemini) RequestBody(req agent.ModelRequest) any { return g.requestBody(req) }

func (g *Gemini) requestBody(req agent.ModelRequest) geminiContents {
	body := geminiContents{GenerationConfig: &geminiGenerationConfig{
		MaxOutputTokens: req.MaxTokens,
		Temperature:     req.Temperature,
	}}
	for _, message := range req.Messages {
		role := geminiRole(message.Role)
		content := geminiContent{Role: role}
		switch {
		case message.Role == agent.RoleTool:
			// A tool result is a functionResponse, and the name has to survive:
			// a response with no matching call is a conversation about a call
			// nobody made (GAP-115).
			// A tool result carries the call id, not the name, and this API
			// matches a functionResponse by name.
			name := toolNameForResult(message, req)
			content.Parts = append(content.Parts, geminiPart{FunctionResponse: &geminiFunctionReply{
				Name:     name,
				Response: map[string]any{"result": message.Content},
			}})
		case len(message.ToolCalls) > 0:
			if message.Content != "" {
				content.Parts = append(content.Parts, geminiPart{Text: message.Content})
			}
			for _, call := range message.ToolCalls {
				part := geminiPart{FunctionCall: &geminiFunctionCall{
					Name: call.Name, Args: call.Arguments, ID: call.ID,
				}}
				if call.ProviderOpaque != nil {
					part.ThoughtSignature = call.ProviderOpaque[GeminiThoughtSignatureKey]
				}
				content.Parts = append(content.Parts, part)
			}
		default:
			if message.Content != "" {
				content.Parts = append(content.Parts, geminiPart{Text: message.Content})
			}
		}
		if len(content.Parts) > 0 {
			body.Contents = append(body.Contents, content)
		}
	}
	if len(req.Tools) > 0 {
		declarations := make([]geminiDeclaration, 0, len(req.Tools))
		for _, spec := range req.Tools {
			declarations = append(declarations, geminiDeclaration{
				Name:        spec.Name,
				Description: spec.Description,
				Parameters:  spec.Schema,
			})
		}
		body.Tools = []geminiTool{{FunctionDeclarations: declarations}}
		// AUTO, not ANY. ANY means "you must call a tool", which is right on the
		// first turn and a guaranteed loop afterwards: the follow-up turn already
		// carries the result, ANY still demands another call, and the model calls
		// the same tool again forever. A live round-trip showed exactly that —
		// three calls to get_weather and no answer. Prumo's loop is what decides
		// between prose and a call, so the mode's job is only to keep the tools
		// available, not to force them.
		body.ToolConfig = &geminiToolConfig{FunctionCallingConfig: geminiCallingConfig{Mode: "AUTO"}}
	}
	return body
}

// geminiRole maps a neutral role onto Google's two.
func geminiRole(role agent.Role) string {
	// "user" and "model" are the only two. There is no system role in the
	// contents list, so a system message is folded into the first user turn
	// rather than dropped: silently discarding the run's instructions would make
	// every system-prompted run behave differently from what was configured.
	if role == agent.RoleAgent {
		return "model"
	}
	return "user"
}

// toolNameForResult recovers the tool name a result answers.
//
// A tool result carries the call id but not the name, and this API matches a
// functionResponse by name. Looking the name up from the request is the only
// place left to get it, so an unmatched result is reported as unknown rather than
// guessed at a name that would be rejected.
func toolNameForResult(message agent.Message, req agent.ModelRequest) string {
	for _, prior := range req.Messages {
		for _, call := range prior.ToolCalls {
			if call.ID == message.ToolCallID {
				return call.Name
			}
		}
	}
	return ""
}
