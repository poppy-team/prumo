package gateway

import (
	"context"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/model"
)

func TestFallbackBeforeSideEffects(t *testing.T) {
	g := New()
	bad := model.NewFake(map[string][]model.ScriptStep{"*": []model.ScriptStep{{Kind: "error", Error: "boom", Retryable: true}}})
	good := model.NewFake(map[string][]model.ScriptStep{"*": []model.ScriptStep{{Kind: "text", Text: "ok"}, {Kind: "complete"}}})
	// Wrap names via custom providers
	g.Register(named{"bad", bad})
	g.Register(named{"good", good})
	route := g.Select([]RouteTarget{{Provider: "bad"}, {Provider: "good"}})
	ch, used, err := g.StreamWithFallback(context.Background(), route, agent.ModelRequest{RequestID: "r", TurnID: "t"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if used != "good" {
		t.Fatalf("expected fallback to good, got %s", used)
	}
	n := 0
	for range ch {
		n++
	}
	if n == 0 {
		t.Fatal("expected events")
	}
}

func TestNoTransparentFallbackAfterSideEffects(t *testing.T) {
	g := New()
	bad := model.NewFake(map[string][]model.ScriptStep{"*": []model.ScriptStep{{Kind: "error", Error: "boom", Retryable: true}}})
	good := model.NewFake(map[string][]model.ScriptStep{"*": []model.ScriptStep{{Kind: "complete"}}})
	g.Register(named{"bad", bad})
	g.Register(named{"good", good})
	route := g.Select([]RouteTarget{{Provider: "bad"}, {Provider: "good"}})
	_, _, err := g.StreamWithFallback(context.Background(), route, agent.ModelRequest{RequestID: "r"}, true)
	if err == nil {
		t.Fatal("expected Handoff-required error after side effects")
	}
}

type named struct {
	name string
	p    *model.FakeProvider
}

func (n named) Name() string                                 { return n.name }
func (n named) Capabilities() model.Capabilities             { return n.p.Capabilities() }
func (n named) Models(ctx context.Context) ([]string, error) { return n.p.Models(ctx) }
func (n named) Health(ctx context.Context) (string, error)   { return n.p.Health(ctx) }
func (n named) Stream(ctx context.Context, r agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	return n.p.Stream(ctx, r)
}

// flakyProvider fails n times with a retryable error, then completes.
type flakyProvider struct {
	left int
}

func (f *flakyProvider) Name() string                     { return "flaky" }
func (f *flakyProvider) Capabilities() model.Capabilities { return model.Capabilities{Streaming: true} }
func (f *flakyProvider) Models(context.Context) ([]string, error) {
	return []string{"m"}, nil
}
func (f *flakyProvider) Health(context.Context) (string, error) { return "healthy", nil }
func (f *flakyProvider) Stream(_ context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	ch := make(chan agent.ModelEvent, 2)
	if f.left > 0 {
		f.left--
		ch <- agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID, Error: "overload", Retryable: true}
	} else {
		ch <- agent.ModelEvent{Kind: agent.EventCompleted, RequestID: req.RequestID, Finished: true}
	}
	close(ch)
	return ch, nil
}

func TestRetryRecoversWithoutFallback(t *testing.T) {
	g := New()
	g.Retry = RetryPolicy{Attempts: 3}
	g.Register(&flakyProvider{left: 2})
	route := g.Select([]RouteTarget{{Provider: "flaky"}})
	ch, used, err := g.StreamWithFallback(context.Background(), route, agent.ModelRequest{RequestID: "r"}, false)
	if err != nil || used != "flaky" {
		t.Fatalf("expected recovery on same provider: %v %s", err, used)
	}
	n := 0
	for range ch {
		n++
	}
	if n == 0 {
		t.Fatal("expected events")
	}
}

func TestRetryExhaustionFallsBack(t *testing.T) {
	g := New()
	g.Retry = RetryPolicy{Attempts: 2}
	g.Register(&flakyProvider{left: 99})
	good := model.NewFake(map[string][]model.ScriptStep{"*": {{Kind: "complete"}}})
	g.Register(named{"good", good})
	route := g.Select([]RouteTarget{{Provider: "flaky"}, {Provider: "good"}})
	ch, used, err := g.StreamWithFallback(context.Background(), route, agent.ModelRequest{RequestID: "r"}, false)
	if err != nil || used != "good" {
		t.Fatalf("expected fallback after retry exhaustion: %v %s", err, used)
	}
	for range ch {
	}
}
