package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
)

// The Gemini wire shape was pinned against the live API first; this is the part
// that must hold without a credential, because the shapes here are the ones a
// future edit can quietly break and a live call is too expensive to notice with.

func geminiTestServer(t *testing.T, sse string, status int) (*httptest.Server, *http.Request) {
	t.Helper()
	captured := &http.Request{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*captured = *r
		if status != http.StatusOK {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":{"code":403,"message":"forbidden"}}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sse))
	}))
	t.Cleanup(srv.Close)
	return srv, captured
}

func TestTheGeminiRequestCarriesTheSignatureBackOnTheCall(t *testing.T) {
	// This is the whole reason ProviderOpaque exists. A conversation that replays
	// a functionCall without its thoughtSignature is refused with a 400, so an
	// adapter that parsed the call and dropped the signature would look correct
	// on the first turn and fail on every turn after it (GAP-174).
	p := &Gemini{}
	body := p.requestBody(agent.ModelRequest{Messages: []agent.Message{
		{ID: "m1", Role: agent.RoleUser, Content: "weather?"},
		{ID: "m2", Role: agent.RoleAgent, ToolCalls: []agent.ToolCall{{
			ID: "c1", Name: "get_weather", Arguments: map[string]any{"city": "Paris"},
			ProviderOpaque: map[string]string{GeminiThoughtSignatureKey: "SIG-123"},
		}}},
		{ID: "m3", Role: agent.RoleTool, ToolCallID: "c1", Content: "17C"},
	}})

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	wire := string(encoded)
	if !strings.Contains(wire, `"thoughtSignature":"SIG-123"`) {
		t.Fatalf("the signature did not reach the wire: %s", wire)
	}
	// The result must be a functionResponse under the tool's own name, or the
	// API has a response with no matching call.
	if !strings.Contains(wire, `"functionResponse"`) || !strings.Contains(wire, `"name":"get_weather"`) {
		t.Fatalf("the tool result was not sent as a named functionResponse: %s", wire)
	}
}

func TestTheToolResultNameIsRecoveredFromTheCallItAnswers(t *testing.T) {
	// A tool result carries the call id, not the name, and this API matches by
	// name. If the lookup fails the request is a result for a call nobody made.
	p := &Gemini{}
	body := p.requestBody(agent.ModelRequest{Messages: []agent.Message{
		{ID: "m1", Role: agent.RoleAgent, ToolCalls: []agent.ToolCall{{ID: "c9", Name: "lookup_order"}}},
		{ID: "m2", Role: agent.RoleTool, ToolCallID: "c9", Content: "order 42"},
	}})
	encoded, _ := json.Marshal(body.Contents[1])
	if !strings.Contains(string(encoded), `"name":"lookup_order"`) {
		t.Fatalf("the result lost the tool name: %s", encoded)
	}
}

func TestTheAssistantRoleIsModelAndNotAssistant(t *testing.T) {
	p := &Gemini{}
	body := p.requestBody(agent.ModelRequest{Messages: []agent.Message{
		{ID: "m1", Role: agent.RoleAgent, Content: "I said this"},
		{ID: "m2", Role: agent.RoleUser, Content: "and you said that"},
	}})
	if body.Contents[0].Role != "model" {
		t.Fatalf("assistant role sent as %q; this API expects model", body.Contents[0].Role)
	}
	if body.Contents[1].Role != "user" {
		t.Fatalf("user role sent as %q", body.Contents[1].Role)
	}
}

func TestToolsAreOfferedInAutoModeNotAny(t *testing.T) {
	// ANY means "you must call a tool". On the first turn that looks helpful; on
	// the follow-up, where the result is already in the conversation, it forces
	// another call and the model calls the same tool forever. A live round trip
	// produced three get_weather calls and no answer before this changed.
	p := &Gemini{}
	body := p.requestBody(agent.ModelRequest{
		Messages: []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}},
		Tools:    []agent.ToolSpec{{Name: "f"}},
	})
	if body.ToolConfig == nil {
		t.Fatal("tools were offered with no calling config")
	}
	if mode := body.ToolConfig.FunctionCallingConfig.Mode; mode != "AUTO" {
		t.Fatalf("calling mode = %q; ANY loops on the follow-up turn", mode)
	}
}

func TestNoToolConfigWhenThereAreNoTools(t *testing.T) {
	p := &Gemini{}
	body := p.requestBody(agent.ModelRequest{Messages: []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "x"}}})
	if body.ToolConfig != nil {
		t.Fatal("a tool config was sent on a turn with no tools")
	}
}

func TestTheStreamMapsTextCallsAndUsage(t *testing.T) {
	sse := "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Checking\"}],\"role\":\"model\"}}],\"usageMetadata\":{\"promptTokenCount\":10,\"candidatesTokenCount\":2,\"totalTokenCount\":30,\"thoughtsTokenCount\":18}}\n\n" +
		"data: {\"candidates\":[{\"content\":{\"parts\":[{\"functionCall\":{\"name\":\"get_weather\",\"args\":{\"city\":\"Paris\"},\"id\":\"call_1\"},\"thoughtSignature\":\"SIG\"}],\"role\":\"model\"}}]}\n\n" +
		"data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"done\"}],\"role\":\"model\"},\"finishReason\":\"STOP\"}]}\n\n"
	srv, _ := geminiTestServer(t, sse, http.StatusOK)
	p := NewGeminiWithPolicy(srv.URL, "k", "gemini-test", LocalDevelopmentDestinationPolicy())

	ch, err := p.Stream(context.Background(), agent.ModelRequest{
		RequestID: "r", Messages: []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "weather?"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var text string
	var call *agent.ToolCall
	var usage *agent.Usage
	for ev := range ch {
		switch ev.Kind {
		case agent.EventTextDelta:
			text += ev.Text
		case agent.EventToolCallReady:
			call = ev.ToolCall
		case agent.EventUsageUpdated:
			usage = ev.Usage
		case agent.EventError:
			t.Fatalf("unexpected error: %s", ev.Error)
		}
	}
	if text != "Checkingdone" {
		t.Errorf("text = %q, want the two deltas joined", text)
	}
	if call == nil || call.Name != "get_weather" || call.Arguments["city"] != "Paris" {
		t.Fatalf("tool call = %+v", call)
	}
	// Arguments arrive under args; reading them as arguments yields a call with no
	// arguments, which fails at execution with an unhelpful missing-parameter.
	if call.ID != "call_1" {
		t.Errorf("call id = %q", call.ID)
	}
	if call.ProviderOpaque == nil || call.ProviderOpaque[GeminiThoughtSignatureKey] != "SIG" {
		t.Error("the thought signature was dropped from the call")
	}
	if usage == nil {
		t.Fatal("no usage event")
	}
	// Reasoning is a separate count, not part of the candidate count.
	if usage.ReasoningTokens != 18 {
		t.Errorf("reasoning = %d, want 18", usage.ReasoningTokens)
	}
	if usage.OutputTokens != 2 {
		t.Errorf("output = %d, want 2", usage.OutputTokens)
	}
}

func TestAQuotaResponseIsClassifiedAndCarriesTheRequestedWait(t *testing.T) {
	// A 429 with a Retry-After is the case the router needs in order to wait
	// instead of hammering a provider that asked it not to (GAP-108).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":429,"message":"quota"}}`))
	}))
	defer srv.Close()
	p := NewGeminiWithPolicy(srv.URL, "k", "gemini-test", LocalDevelopmentDestinationPolicy())

	ch, err := p.Stream(context.Background(), agent.ModelRequest{RequestID: "r"})
	if err != nil {
		t.Fatal(err)
	}
	var got agent.ModelEvent
	for ev := range ch {
		got = ev
	}
	if got.Kind != agent.EventError {
		t.Fatalf("kind = %s, want an error", got.Kind)
	}
	if !got.Quota {
		t.Error("a 429 was not classified as a quota")
	}
	if !got.Retryable {
		t.Error("a 429 must be retryable")
	}
	if got.RetryAfter.String() != "7s" {
		t.Errorf("retry after = %s, want the 7s the provider asked for", got.RetryAfter)
	}
}

func TestAClientErrorIsNotRetryable(t *testing.T) {
	// A 403 will be a 403 on every provider. Retrying it wastes the run and
	// hides the reason.
	srv, _ := geminiTestServer(t, "", http.StatusForbidden)
	p := NewGeminiWithPolicy(srv.URL, "k", "gemini-test", LocalDevelopmentDestinationPolicy())
	ch, err := p.Stream(context.Background(), agent.ModelRequest{RequestID: "r"})
	if err != nil {
		t.Fatal(err)
	}
	var got agent.ModelEvent
	for ev := range ch {
		got = ev
	}
	if got.Kind != agent.EventError || got.Retryable {
		t.Fatalf("a 403 must be a terminal error, got kind=%s retryable=%v", got.Kind, got.Retryable)
	}
}

func TestACutStreamIsAnErrorNotACompletion(t *testing.T) {
	// The endpoint does not send a completion marker, so the stream ending is the
	// only signal — and ending is not finishing. A truncated answer reported as
	// a finished run is the GAP-131 shape, in a new dialect.
	srv, _ := geminiTestServer(t, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"half\"}]}}]}\n\n", http.StatusOK)
	p := NewGeminiWithPolicy(srv.URL, "k", "gemini-test", LocalDevelopmentDestinationPolicy())
	ch, err := p.Stream(context.Background(), agent.ModelRequest{RequestID: "r"})
	if err != nil {
		t.Fatal(err)
	}
	var sawCompletion, sawError bool
	for ev := range ch {
		switch ev.Kind {
		case agent.EventCompleted:
			sawCompletion = true
		case agent.EventError:
			sawError = true
		}
	}
	if sawCompletion {
		t.Error("a stream that simply ended was reported as a completed run")
	}
	if !sawError {
		t.Error("a stream that ended without a completion marker must say so")
	}
}

func TestTheApiKeyTravelsInAHeaderNotTheURL(t *testing.T) {
	// The API accepts the key as a query parameter, and that form puts a live
	// credential in the URL where proxy logs, error strings and request dumps all
	// keep it. A key that has to be scrubbed out of diagnostics is a key that
	// eventually gets read out of one.
	srv, captured := geminiTestServer(t, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"x\"}]},\"finishReason\":\"STOP\"}]}\n\n", http.StatusOK)
	p := NewGeminiWithPolicy(srv.URL, "secret-key-value", "gemini-test", LocalDevelopmentDestinationPolicy())
	ch, err := p.Stream(context.Background(), agent.ModelRequest{RequestID: "r"})
	if err != nil {
		t.Fatal(err)
	}
	for range ch {
	}
	if got := captured.Header.Get("x-goog-api-key"); got != "secret-key-value" {
		t.Errorf("x-goog-api-key = %q", got)
	}
	if strings.Contains(captured.URL.String(), "secret-key-value") {
		t.Error("the key reached the URL")
	}
}

func TestTheKeyFallsBackToTheEnvironment(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "from-env")
	srv, captured := geminiTestServer(t, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"x\"}]},\"finishReason\":\"STOP\"}]}\n\n", http.StatusOK)
	p := NewGeminiWithPolicy(srv.URL, "", "gemini-test", LocalDevelopmentDestinationPolicy())
	ch, err := p.Stream(context.Background(), agent.ModelRequest{RequestID: "r"})
	if err != nil {
		t.Fatal(err)
	}
	for range ch {
	}
	if got := captured.Header.Get("x-goog-api-key"); got != "from-env" {
		t.Errorf("the environment key was not used: %q", got)
	}
}

func TestGeminiIsAToolServingProvider(t *testing.T) {
	// The distinction the gateway relies on: this one hands calls back for Prumo
	// to run under its own policy, which is what makes transparent fallback safe
	// after a partial answer. The opencode CLI is the other kind.
	p := &Gemini{}
	if !p.Capabilities().ToolCalls {
		t.Fatal("gemini serves tool calls and must say so")
	}
}

func TestARetryDelayInTheBodyIsHonouredWhenThereIsNoHeader(t *testing.T) {
	// Google puts the retry hint in the error body and sends no Retry-After. A
	// router that only reads the header falls back to its computed backoff for a
	// provider that said exactly when it would be ready (GAP-108).
	body := `{"error":{"code":429,"status":"RESOURCE_EXHAUSTED","details":[
		{"@type":"type.googleapis.com/google.rpc.Help"},
		{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"19s"}],
		"message":"quota exceeded"}}`
	pe := NewProviderError("gemini", &http.Response{StatusCode: 429}, body)
	if !pe.Quota {
		t.Error("a RESOURCE_EXHAUSTED body was not classified as a quota")
	}
	if pe.RetryAfter != 19*time.Second {
		t.Errorf("retry after = %s, want the 19s the body asked for", pe.RetryAfter)
	}
}

func TestAQuotaBodyWithNoDelayFallsBackToTheComputedBackoff(t *testing.T) {
	// A missing or unreadable hint is not a reason to invent a delay.
	pe := NewProviderError("gemini", &http.Response{StatusCode: 429},
		`{"error":{"code":429,"status":"RESOURCE_EXHAUSTED","message":"quota"}}`)
	if pe.RetryAfter != 0 {
		t.Errorf("retry after = %s, want zero when the body says nothing", pe.RetryAfter)
	}
}
