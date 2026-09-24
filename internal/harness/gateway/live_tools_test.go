//go:build integration_live

package gateway_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/gateway"
	"github.com/raillen/prumo/internal/harness/model"
)

// This is the test the opencode limitation made impossible to write.
//
// The opencode adapter delegates the whole turn: it runs its own loop with its own
// tools and hands back text and cost, never a tool call. That is the right design
// for it — forwarding its tool_use events would make Prumo execute tools the
// delegate had already executed — and it also means no amount of routing work
// could exercise tool calling across two real providers through it.
//
// Gemini is the other kind: it serves tool calls and lets Prumo run them under its
// own permission policy. So the two together are the pool the gateway was built
// for, and this is where that is checked against real endpoints rather than
// doubles.
//
// Credentials come from the environment. Both are needed: the second route has to
// be a different provider, and opencode is what supplies one.
func liveToolPool(t *testing.T) (*gateway.Gateway, string, string) {
	t.Helper()
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("set GEMINI_API_KEY to run the live tool-calling test")
	}
	if os.Getenv("OPENCODE_API_KEY") == "" {
		t.Skip("set OPENCODE_API_KEY: the second route must be a different provider")
	}
	geminiModel := os.Getenv("GEMINI_LIVE_MODEL")
	if geminiModel == "" {
		geminiModel = "gemini-3-flash-preview"
	}
	// A model the Gemini endpoint does not serve, so the first route fails for a
	// real reason and the second is exercised by actually being reached.
	deadModel := "gemini-no-such-model-exists"
	workingModel := os.Getenv("OPENCODE_LIVE_MODEL")
	if workingModel == "" {
		workingModel = "opencode/muse-spark-1.3-contributor-free"
	}

	g := gateway.New()
	g.RegisterAs("gemini-dead", model.NewGemini(os.Getenv("GEMINI_API_KEY"), deadModel))
	g.RegisterAs("opencode", model.NewOpenCode(workingModel))
	g.DeclareTarget(gateway.RouteTarget{Provider: "gemini-dead", Model: deadModel, Privacy: "external", Tools: true})
	g.DeclareTarget(gateway.RouteTarget{Provider: "opencode", Model: workingModel, Privacy: "external", Tools: true})
	g.Retry = gateway.RetryPolicy{Attempts: 1, Backoff: 10 * time.Millisecond}
	return g, deadModel, workingModel
}

func liveToolRequest() agent.ModelRequest {
	return agent.ModelRequest{
		RequestID: "live-tools-1",
		TurnID:    "turn-1",
		Messages: []agent.Message{
			{ID: "m1", Role: agent.RoleUser, Content: "What is the weather in Paris? Use the get_weather tool."},
		},
		Tools: []agent.ToolSpec{{
			Name:        "get_weather",
			Description: "Get the current weather for a city",
			Schema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"city": map[string]any{"type": "string"}},
				"required":   []any{"city"},
			},
		}},
	}
}

// TestLiveToolCallingReachesARealToolServingProvider is the multi-provider
// tool-calling check: a first route that cannot serve the model, a second that
// can, and a real tool specification on the request.
func TestLiveToolCallingReachesARealToolServingProvider(t *testing.T) {
	g, deadModel, workingModel := liveToolPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	route := g.Select([]gateway.RouteTarget{{Provider: "gemini-dead", Model: deadModel}, {Provider: "opencode", Model: workingModel}})
	if route.Primary.Provider != "gemini-dead" {
		t.Fatalf("the first route should be the one that fails: %+v", route)
	}
	ch, _, err := g.StreamWithFallback(ctx, route, liveToolRequest(), false)
	if err != nil {
		t.Fatalf("the pool produced no route at all: %v", err)
	}
	var text string
	var sawQuota bool
	for ev := range ch {
		switch ev.Kind {
		case agent.EventTextDelta:
			text += ev.Text
		case agent.EventError:
			if ev.Quota {
				sawQuota = true
				t.Skipf("a route's API quota is spent, so the pool was not exercised: %s", ev.Error)
			}
			t.Logf("route error: %s", ev.Error)
		}
	}
	if strings.TrimSpace(text) == "" {
		t.Fatalf("no provider in the pool answered (quota seen: %v)", sawQuota)
	}
	// "Some text came back" is not tool calling, and accepting it as such is how
	// this test would have passed while proving nothing.
	//
	// The live run that first wrote this made exactly that mistake. The fallback
	// route is the opencode delegate, which ignores the tool specifications it is
	// handed because it runs its own tools — and the model said so in as many
	// words: "I don't have a get_weather tool available to me". The route was
	// correct and the test was satisfied, and neither fact says anything about
	// tool calling across two providers.
	//
	// So a delegate answering is reported as what it is. Multi-provider tool
	// calling needs two tool-serving routes, and until the pool has one it is
	// unproven, which is worth saying rather than papering over.
	if providerServesTools(g, workingModel) {
		t.Logf("a tool-serving route answered with: %q", text)
		return
	}
	t.Skipf("the fallback route is a delegating provider, which does not serve "+
		"tool calls; routing is proven but multi-provider tool calling is not. Answer was: %q", text)
}

// TestLiveGeminiRoundTripsARealToolCall is the half that needs only Gemini, and it
// is the one that proves the opaque round-trip token survives: the call is issued,
// answered and returned, and the conversation is accepted.
func TestLiveGeminiRoundTripsARealToolCall(t *testing.T) {
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("set GEMINI_API_KEY to run the live Gemini tool round trip")
	}
	mdl := os.Getenv("GEMINI_LIVE_MODEL")
	if mdl == "" {
		mdl = "gemini-3-flash-preview"
	}
	p := model.NewGemini(os.Getenv("GEMINI_API_KEY"), mdl)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	req := liveToolRequest()
	first, err := p.Stream(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	var call *agent.ToolCall
	for ev := range first {
		if ev.Kind == agent.EventError && ev.Quota {
			t.Skipf("the API quota is spent, so the round trip was not exercised: %s", ev.Error)
		}
		if ev.Kind == agent.EventToolCallReady {
			call = ev.ToolCall
		}
	}
	if call == nil {
		t.Fatal("the live model did not call the tool that was offered")
	}
	if call.ProviderOpaque[model.GeminiThoughtSignatureKey] == "" {
		t.Fatal("no round-trip token on the call; the follow-up turn will be refused")
	}

	second := req
	second.RequestID = "live-tools-2"
	second.Messages = []agent.Message{
		req.Messages[0],
		{ID: "m2", Role: agent.RoleAgent, ToolCalls: []agent.ToolCall{*call}},
		{ID: "m3", Role: agent.RoleTool, ToolCallID: call.ID, Content: "17C and cloudy"},
	}
	stream, err := p.Stream(ctx, second)
	if err != nil {
		t.Fatal(err)
	}
	var answer string
	for ev := range stream {
		switch ev.Kind {
		case agent.EventTextDelta:
			answer += ev.Text
		case agent.EventError:
			t.Fatalf("the follow-up turn was refused: %s", ev.Error)
		}
	}
	if strings.TrimSpace(answer) == "" {
		t.Fatal("the follow-up turn produced no answer")
	}
	t.Logf("round trip complete: %q", answer)
}

// providerServesTools reports whether the route for a model can hand tool calls
// back to Prumo.
//
// The answer is read from the adapter's own declared capability rather than
// guessed, because that declaration is exactly what the router's safety depends
// on and exactly what a live answer can contradict.
func providerServesTools(g *gateway.Gateway, modelID string) bool {
	for _, target := range g.DeclaredTargets() {
		if target.Model != modelID {
			continue
		}
		return g.CapabilitiesFor(target.Provider).ToolCalls
	}
	return false
}
