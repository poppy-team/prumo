package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/modelregistry"
)

// Neither endpoint returns a cost. Every usage event therefore carried CostUSD
// zero, and a budget's US dollar dimension compared a fixed number against a
// limit the spend never reached: --budget-usd was inert for exactly the two
// providers a real run uses (GAP-129).

func usageOf(t *testing.T, events <-chan agent.ModelEvent) *agent.Usage {
	t.Helper()
	for ev := range events {
		if ev.Kind == agent.EventUsageUpdated && ev.Usage != nil {
			return ev.Usage
		}
	}
	return nil
}

func TestAnthropicReportsACostForTheTokensItReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(
			"event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":1000,\"output_tokens\":0}}}\n\n" +
				"event: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":500}}\n\n" +
				"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer srv.Close()

	p := NewAnthropicWithPolicy(srv.URL, "k", "claude-3-5-sonnet", LocalDevelopmentDestinationPolicy())
	ch, err := p.Stream(context.Background(), agent.ModelRequest{
		RequestID: "r1",
		Messages:  []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var last *agent.Usage
	for ev := range ch {
		if ev.Kind == agent.EventUsageUpdated && ev.Usage != nil {
			last = ev.Usage
		}
	}
	if last == nil {
		t.Fatal("no usage was reported")
	}
	if last.CostUSD <= 0 {
		t.Fatalf("a call that spent tokens must report a cost, got %g", last.CostUSD)
	}
	// 1000 input at $3/M plus 500 output at $15/M is 0.003 + 0.0075.
	want := 0.0105
	if diff := last.CostUSD - want; diff > 0.000001 || diff < -0.000001 {
		t.Fatalf("cost = %g, want %g", last.CostUSD, want)
	}
}

func TestOpenAICompatReportsACostForTheTokensItReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(
			`data: {"choices":[{"delta":{}}],"usage":{"prompt_tokens":1000,"completion_tokens":500}}` + "\n\n" +
				"data: [DONE]\n\n"))
	}))
	defer srv.Close()

	p := NewOpenAICompatWithPolicy(srv.URL, "k", "gpt-4o", LocalDevelopmentDestinationPolicy())
	ch, err := p.Stream(context.Background(), agent.ModelRequest{
		RequestID: "r1",
		Messages:  []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	usage := usageOf(t, ch)
	if usage == nil {
		t.Fatal("no usage was reported")
	}
	if usage.CostUSD <= 0 {
		t.Fatalf("a call that spent tokens must report a cost, got %g", usage.CostUSD)
	}
	// 1000 input at $2.50/M plus 500 output at $10/M is 0.0025 + 0.005.
	want := 0.0075
	if diff := usage.CostUSD - want; diff > 0.000001 || diff < -0.000001 {
		t.Fatalf("cost = %g, want %g", usage.CostUSD, want)
	}
}

func TestAnUnpricedModelReportsNoCostRatherThanAGuess(t *testing.T) {
	// Zero and unknown are different answers. Reporting zero for a model nobody
	// has priced would make an unpriced model look free.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(
			`data: {"choices":[{"delta":{}}],"usage":{"prompt_tokens":1000,"completion_tokens":500}}` + "\n\n" +
				"data: [DONE]\n\n"))
	}))
	defer srv.Close()

	p := NewOpenAICompatWithPolicy(srv.URL, "k", "a-model-nobody-priced", LocalDevelopmentDestinationPolicy())
	ch, err := p.Stream(context.Background(), agent.ModelRequest{
		RequestID: "r1",
		Messages:  []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	usage := usageOf(t, ch)
	if usage == nil {
		t.Fatal("no usage was reported")
	}
	if usage.CostUSD != 0 {
		t.Fatalf("an unpriced model must not be given a made-up cost, got %g", usage.CostUSD)
	}
	if modelregistry.Pricable(p.Pricing, p.ProviderKey(), "a-model-nobody-priced") {
		t.Fatal("the table must not claim to price a model it does not")
	}
}

func TestTheCostUsesTheRequestedModelNotTheConfiguredDefault(t *testing.T) {
	// A request that names a different model is billed at that model's rate.
	// Pricing under the adapter's default would charge one model at another's.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(
			`data: {"choices":[{"delta":{}}],"usage":{"prompt_tokens":1000,"completion_tokens":0}}` + "\n\n" +
				"data: [DONE]\n\n"))
	}))
	defer srv.Close()

	p := NewOpenAICompatWithPolicy(srv.URL, "k", "gpt-4o", LocalDevelopmentDestinationPolicy())
	ch, err := p.Stream(context.Background(), agent.ModelRequest{
		RequestID: "r1", Model: "gpt-4o-mini",
		Messages: []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	usage := usageOf(t, ch)
	if usage == nil {
		t.Fatal("no usage")
	}
	// gpt-4o-mini input is $0.15/M, so 1000 tokens is 0.00015 — an order of
	// magnitude below gpt-4o's 0.0025.
	if usage.CostUSD > 0.001 {
		t.Fatalf("cost %g was priced at the configured default rather than the requested model", usage.CostUSD)
	}
}

func TestAProviderWithNoTableReportsNoCostRatherThanPanicking(t *testing.T) {
	p := &OpenAICompat{Pricing: modelregistry.PricingTable{}}
	usage := agent.Usage{InputTokens: 100}
	priceUsage(p.Pricing, p.ProviderKey(), "gpt-4o", &usage)
	if usage.CostUSD != 0 {
		t.Fatalf("a provider with no table must report no cost, got %g", usage.CostUSD)
	}
}

func TestTheCostTravelsInTheEventTheBudgetReads(t *testing.T) {
	// The budget consumes Usage.CostUSD. A cost that does not reach the event is
	// a cost nothing can charge.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(
			`data: {"choices":[{"delta":{}}],"usage":{"prompt_tokens":2000,"completion_tokens":1000}}` + "\n\n" +
				"data: [DONE]\n\n"))
	}))
	defer srv.Close()

	p := NewOpenAICompatWithPolicy(srv.URL, "k", "gpt-4o", LocalDevelopmentDestinationPolicy())
	ch, err := p.Stream(context.Background(), agent.ModelRequest{
		RequestID: "r1", Messages: []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for ev := range ch {
		if ev.Kind == agent.EventUsageUpdated && ev.Usage != nil {
			data, _ := json.Marshal(ev.Usage)
			if !strings.Contains(string(data), "cost_usd") {
				t.Fatal("the cost must be serialised on the event, not only held in memory")
			}
			if ev.Usage.CostUSD <= 0 {
				t.Fatal("the event must carry a positive cost")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("no usage event was emitted")
	}
}

func TestAnthropicUsageTotalsGrowAcrossSplitReports(t *testing.T) {
	// Anthropic announces input at message_start and output at message_delta.
	// Forwarding each raw report means the last event is the output alone, so a
	// run that spent 1000 input and 500 output reads as having spent 500 — and
	// every cost, budget and report built from that last event understates the
	// call by the input already paid for.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(
			"event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":1000,\"output_tokens\":0}}}\n\n" +
				"event: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":200}}\n\n" +
				"event: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":500}}\n\n" +
				"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer srv.Close()

	p := NewAnthropicWithPolicy(srv.URL, "k", "claude-3-5-sonnet", LocalDevelopmentDestinationPolicy())
	ch, err := p.Stream(context.Background(), agent.ModelRequest{
		RequestID: "r1",
		Messages:  []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var last *agent.Usage
	for ev := range ch {
		if ev.Kind == agent.EventUsageUpdated && ev.Usage != nil {
			last = ev.Usage
		}
	}
	if last == nil {
		t.Fatal("no usage was reported")
	}
	if last.InputTokens != 1000 || last.OutputTokens != 500 {
		t.Fatalf("the last usage event must carry the total, got input=%d output=%d",
			last.InputTokens, last.OutputTokens)
	}
	// The full 0.0105, not the 0.0075 that only the output accounts for.
	if diff := last.CostUSD - 0.0105; diff > 0.000001 || diff < -0.000001 {
		t.Fatalf("cost = %g, want the whole call at 0.0105", last.CostUSD)
	}
}

func TestOpenAICompatUsageTotalsGrowAcrossChunks(t *testing.T) {
	// The same split-report hazard as Anthropic: if usage arrives on more than
	// one chunk, forwarding each raw means the last event carries only what that
	// chunk said, and the total a caller reads depends on which chunk it kept.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(
			`data: {"choices":[{"delta":{}}],"usage":{"prompt_tokens":1000,"completion_tokens":0}}` + "\n\n" +
				`data: {"choices":[{"delta":{}}],"usage":{"prompt_tokens":1000,"completion_tokens":200}}` + "\n\n" +
				`data: {"choices":[{"delta":{}}],"usage":{"prompt_tokens":1000,"completion_tokens":500}}` + "\n\n" +
				"data: [DONE]\n\n"))
	}))
	defer srv.Close()

	p := NewOpenAICompatWithPolicy(srv.URL, "k", "gpt-4o", LocalDevelopmentDestinationPolicy())
	ch, err := p.Stream(context.Background(), agent.ModelRequest{
		RequestID: "r1",
		Messages:  []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var last *agent.Usage
	for ev := range ch {
		if ev.Kind == agent.EventUsageUpdated && ev.Usage != nil {
			last = ev.Usage
		}
	}
	if last == nil {
		t.Fatal("no usage was reported")
	}
	if last.InputTokens != 1000 || last.OutputTokens != 500 {
		t.Fatalf("the last usage event must carry the total, got input=%d output=%d",
			last.InputTokens, last.OutputTokens)
	}
	// 1000 input at $2.50/M plus 500 output at $10/M is 0.0025 + 0.005.
	if diff := last.CostUSD - 0.0075; diff > 0.000001 || diff < -0.000001 {
		t.Fatalf("cost = %g, want the whole call at 0.0075", last.CostUSD)
	}
}

func TestACachedTokenIsCountedOnce(t *testing.T) {
	// The provider counts a cached token inside the prompt and also reports it in
	// the details. Adding the two would overstate the run; so would leaving the
	// cached tokens out of the input that gets billed at the cache-read rate.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(
			`data: {"choices":[{"delta":{}}],"usage":{"prompt_tokens":1000,"completion_tokens":0,"prompt_tokens_details":{"cached_tokens":400}}}` + "\n\n" +
				"data: [DONE]\n\n"))
	}))
	defer srv.Close()

	p := NewOpenAICompatWithPolicy(srv.URL, "k", "gpt-4o", LocalDevelopmentDestinationPolicy())
	ch, err := p.Stream(context.Background(), agent.ModelRequest{
		RequestID: "r1",
		Messages:  []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	usage := usageOf(t, ch)
	if usage == nil {
		t.Fatal("no usage was reported")
	}
	if usage.InputTokens != 600 {
		t.Errorf("input = %d, want 600: the 400 cached tokens belong to the cache-read dimension", usage.InputTokens)
	}
	if usage.CacheReadTokens != 400 {
		t.Errorf("cache read = %d, want 400", usage.CacheReadTokens)
	}
	if usage.InputTokens+usage.CacheReadTokens != 1000 {
		t.Errorf("the dimensions must still add up to the provider's prompt total, got %d",
			usage.InputTokens+usage.CacheReadTokens)
	}
}
