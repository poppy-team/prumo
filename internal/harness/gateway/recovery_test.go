package gateway

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/model"
)

// A circuit that opens and can never close is not a circuit breaker; it is a
// permanent exclusion. The cooldown was written on the third failure and read by
// nothing, so a provider that had a bad minute was dropped from every route for
// the life of the process, and the recovery the breaker existed to enable was
// unreachable (GAP-103).

func TestAnOpenCircuitIsProbedAgainAfterItsCooldown(t *testing.T) {
	g := New()
	g.Retry = RetryPolicy{Attempts: 1}
	g.Register(&alwaysFailProvider{name: "flaky"})
	targets := []RouteTarget{{Provider: "flaky"}}

	// Three failures trip the breaker.
	for i := 0; i < 3; i++ {
		route := g.Select(targets)
		if _, _, err := g.StreamWithFallback(context.Background(), route, agent.ModelRequest{RequestID: "r"}, false); err == nil {
			t.Fatal("expected the provider to keep failing")
		}
	}
	if r := g.Select(targets); r.Primary.Provider == "flaky" {
		t.Fatal("a provider that just failed three times must not stay primary")
	}

	// The cooldown elapses.
	g.mu.Lock()
	g.health["flaky"].CooldownUntil = time.Now().UTC().Add(-time.Second).Format(time.RFC3339Nano)
	g.mu.Unlock()

	// It must be tried again. Before the cooldown was read, it stayed excluded
	// forever and a provider that recovered was never used again.
	if r := g.Select(targets); r.Primary.Provider != "flaky" {
		t.Fatalf("after its cooldown the provider must be probed again, got %+v", r)
	}
}

func TestAnUnreadableCooldownDoesNotExcludeAProviderForever(t *testing.T) {
	g := New()
	g.Register(&alwaysFailProvider{name: "flaky"})
	g.mu.Lock()
	g.health["flaky"] = &Health{Status: "open", CooldownUntil: "not a time"}
	g.mu.Unlock()
	if r := g.Select([]RouteTarget{{Provider: "flaky"}}); r.Primary.Provider != "flaky" {
		t.Fatalf("a cooldown nobody can read must not hold a provider out of routing, got %+v", r)
	}
}

// quotaExhausted read the quotas map unguarded while SetQuota and the rate-limit
// path wrote it under the lock, so a selection running beside a quota update raced
// on the same map (GAP-103).

func TestQuotaReadsAndWritesDoNotRace(t *testing.T) {
	g := New()
	targets := []RouteTarget{{Provider: "a"}, {Provider: "b"}, {Provider: "c"}}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				g.SetQuota("a", float64(j%2), "")
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				g.Select(targets)
			}
		}()
	}
	wg.Wait()
}

// LocalOnly is a firewall, and a firewall that opens for "we have not classified
// this" is not a firewall. The filter allowed an empty privacy value through, so
// the one case where nobody had classified the target was the one case allowed to
// receive restricted data (GAP-156).

func TestLocalOnlyRefusesATargetWhosePrivacyIsUnknown(t *testing.T) {
	g := New()
	targets := []RouteTarget{
		{Provider: "known", Privacy: "local"},
		{Provider: "unknown", Privacy: ""},
	}
	route := g.SelectWithPolicy(targets, Policy{LocalOnly: true})
	if route.Primary.Provider != "known" {
		t.Fatalf("only a target known to be local may be used, got %+v", route)
	}
	for _, fb := range route.Fallbacks {
		if fb.Provider == "unknown" {
			t.Fatal("an unclassified target must not even be a fallback")
		}
	}
}

func TestLocalOnlyStillRefusesExternalAndTrusted(t *testing.T) {
	g := New()
	targets := []RouteTarget{
		{Provider: "trusted", Privacy: "trusted"},
		{Provider: "external", Privacy: "external"},
		{Provider: "nonsense", Privacy: "not-a-privacy-class"},
	}
	if route := g.SelectWithPolicy(targets, Policy{LocalOnly: true}); route.Primary.Provider != "" {
		t.Fatalf("nothing but local may be selected, got %+v", route)
	}
}

func TestWithoutLocalOnlyPrivacyIsNotFiltered(t *testing.T) {
	g := New()
	targets := []RouteTarget{{Provider: "unknown", Privacy: ""}}
	if route := g.SelectWithPolicy(targets, Policy{}); route.Primary.Provider != "unknown" {
		t.Fatalf("privacy is only a gate when the caller asked for one, got %+v", route)
	}
}

// The adapters reduced a response to a message string, so the gateway guessed the
// meaning from prose and threw away the Retry-After the provider sent. A retry
// then came back inside the limit window and was rejected for the same reason
// (GAP-108).

func TestTheProvidersRequestedWaitIsHonoured(t *testing.T) {
	g := New()
	g.Retry = RetryPolicy{Attempts: 2, Backoff: time.Millisecond}
	g.Register(&quotaProvider{name: "limited", retryAfter: 40 * time.Millisecond})

	start := time.Now()
	route := g.Select([]RouteTarget{{Provider: "limited"}})
	if _, _, err := g.StreamWithFallback(context.Background(), route, agent.ModelRequest{RequestID: "r"}, false); err == nil {
		t.Fatal("expected the provider to keep failing")
	}
	// Two attempts at a 1ms backoff would take about 1ms. The provider asked for
	// 40ms, and the gateway has to wait at least that long, or it retries inside
	// the limit window and is rejected again.
	if waited := time.Since(start); waited < 35*time.Millisecond {
		t.Fatalf("waited %s; the provider asked for 40ms and that is not a suggestion", waited)
	}
}

func TestResourceExhaustedIsRecognisedWithoutAKnownStatus(t *testing.T) {
	// Some compatible gateways report exhaustion inside a successful response,
	// with no 429 to read. Matching prose is still the only signal there.
	g := New()
	g.Retry = RetryPolicy{Attempts: 1}
	g.Register(&alwaysFailProvider{name: "gem", errorText: "RESOURCE_EXHAUSTED: quota limit"})
	g.Select([]RouteTarget{{Provider: "gem"}})
	route := g.Select([]RouteTarget{{Provider: "gem"}})
	if _, _, err := g.StreamWithFallback(context.Background(), route, agent.ModelRequest{RequestID: "r"}, false); err == nil {
		t.Fatal("expected failure")
	}
	// Exhaustion must cool the provider down rather than retrying straight at it.
	if r := g.Select([]RouteTarget{{Provider: "gem"}}); r.Primary.Provider == "gem" {
		t.Fatal("a resource-exhausted provider must cool down")
	}
}

func TestBackoffGrowsRatherThanStayingLinear(t *testing.T) {
	p := RetryPolicy{Attempts: 6, Backoff: 10 * time.Millisecond}
	first := backoffFor(p, 1, nil)
	second := backoffFor(p, 2, nil)
	third := backoffFor(p, 3, nil)
	if second <= first {
		t.Fatalf("backoff must grow: %s then %s", first, second)
	}
	if third <= second {
		t.Fatalf("backoff must keep growing: %s then %s", second, third)
	}
	// Linear backoff would have been 1x, 2x, 3x; the point is that the later
	// attempts back off further than a linear ramp would.
	if third != 4*first {
		t.Fatalf("attempt 3 should wait 4x the base, got %s vs base %s", third, first)
	}
}

func TestBackoffIsCappedSoAPolicyCannotProduceAnUnwaitableDelay(t *testing.T) {
	p := RetryPolicy{Attempts: 40, Backoff: time.Second}
	if d := backoffFor(p, 40, nil); d > 2*time.Minute {
		t.Fatalf("a generous policy produced a %s delay; the cap is what keeps a retry loop finite in practice", d)
	}
}

func TestAMalformedRetryAfterFallsBackToTheConfiguredBackoff(t *testing.T) {
	// A header nobody can read is not a reason to invent a delay: the gateway
	// waits what its own policy says, and nothing more.
	p := RetryPolicy{Attempts: 2, Backoff: 10 * time.Millisecond}
	malformed := &failure{err: errString("429"), retryAfter: model.ParseRetryAfter("soon")}
	unreadable := &failure{err: errString("429")}
	if model.ParseRetryAfter("soon") != 0 {
		t.Fatal("an unparseable Retry-After must parse to zero, not to a guess")
	}
	if got, want := backoffFor(p, 2, malformed), backoffFor(p, 2, unreadable); got != want {
		t.Fatalf("a malformed Retry-After changed the delay to %s; the policy's own %s is the fallback", got, want)
	}
}

func TestAReadableRetryAfterOverridesTheComputedBackoff(t *testing.T) {
	p := RetryPolicy{Attempts: 2, Backoff: 10 * time.Millisecond}
	short := backoffFor(p, 2, &failure{err: errString("500"), retryAfter: time.Second})
	if short != time.Second {
		t.Fatalf("delay = %s; the provider asked for a second, so that is the wait", short)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

// alwaysFailProvider fails every call with a retryable error.
type alwaysFailProvider struct {
	name      string
	errorText string
}

func (f *alwaysFailProvider) Name() string { return f.name }
func (f *alwaysFailProvider) Capabilities() model.Capabilities {
	return model.Capabilities{Streaming: true}
}
func (f *alwaysFailProvider) Models(context.Context) ([]string, error) {
	return []string{"m"}, nil
}
func (f *alwaysFailProvider) Health(context.Context) (string, error) { return "healthy", nil }
func (f *alwaysFailProvider) Stream(_ context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	text := f.errorText
	if text == "" {
		text = "server error"
	}
	ch := make(chan agent.ModelEvent, 1)
	ch <- agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID, Error: text, Retryable: true}
	close(ch)
	return ch, nil
}

// quotaProvider reports exhaustion the way an adapter does: classified on the
// event, with the interval the provider asked for.
type quotaProvider struct {
	name       string
	retryAfter time.Duration
}

func (f *quotaProvider) Name() string                     { return f.name }
func (f *quotaProvider) Capabilities() model.Capabilities { return model.Capabilities{Streaming: true} }
func (f *quotaProvider) Models(context.Context) ([]string, error) {
	return []string{"m"}, nil
}
func (f *quotaProvider) Health(context.Context) (string, error) { return "healthy", nil }
func (f *quotaProvider) Stream(_ context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	ch := make(chan agent.ModelEvent, 1)
	ch <- agent.ModelEvent{
		Kind: agent.EventError, RequestID: req.RequestID,
		Error: "rate limited", Retryable: true, Quota: true, RetryAfter: f.retryAfter,
	}
	close(ch)
	return ch, nil
}

// Register keys by the adapter's own name, so a pool with two instances of the
// same adapter silently collapsed to one: the second overwrote the first and
// every route to that name reached the survivor. With one provider it is
// invisible; with a pool of two it defeats the reason the gateway exists.
//
// This is not hypothetical. The opencode adapter serves many models through one
// implementation, so a pool of two models through it is two instances named
// "opencode".

func TestTwoInstancesOfTheSameAdapterCoexistUnderDifferentKeys(t *testing.T) {
	g := New()
	first := &alwaysFailProvider{name: "shared-adapter", errorText: "first instance"}
	second := &alwaysFailProvider{name: "shared-adapter", errorText: "second instance"}
	g.RegisterAs("route-first", first)
	g.RegisterAs("route-second", second)

	g.mu.Lock()
	if len(g.providers) != 2 {
		t.Fatalf("the pool has %d providers; the second registration overwrote the first", len(g.providers))
	}
	if g.providers["route-first"] != model.Provider(first) {
		t.Error("route-first must still resolve to the instance registered under it")
	}
	if g.providers["route-second"] != model.Provider(second) {
		t.Error("route-second must still resolve to the instance registered under it")
	}
}

func TestTwoRoutesReachTwoDifferentInstancesOfTheSameAdapter(t *testing.T) {
	g := New()
	primary := &recordingNameProvider{name: "shared-adapter", answer: "from the first"}
	backup := &recordingNameProvider{name: "shared-adapter", answer: "from the second"}
	g.RegisterAs("route-first", primary)
	g.RegisterAs("route-second", backup)
	g.DeclareTarget(RouteTarget{Provider: "route-first", Model: "m1"})
	g.DeclareTarget(RouteTarget{Provider: "route-second", Model: "m1"})
	g.Retry = RetryPolicy{Attempts: 1}

	route := g.Select([]RouteTarget{{Provider: "route-first"}, {Provider: "route-second"}})
	if route.Primary.Provider != "route-first" {
		t.Fatalf("selection is deterministic by name, so route-first comes first: %+v", route)
	}
	ch, err := g.Stream(context.Background(), agent.ModelRequest{Model: "m1"})
	if err != nil {
		t.Fatal(err)
	}
	var text string
	for ev := range ch {
		if ev.Kind == agent.EventTextDelta {
			text += ev.Text
		}
	}
	if primary.calls != 1 {
		t.Errorf("the primary instance was called %d times, want 1", primary.calls)
	}
	if backup.calls != 0 {
		t.Errorf("the backup was called %d times; only the selected route should be called", backup.calls)
	}
	if text == "" {
		t.Error("the selected instance produced nothing")
	}
}

// recordingNameProvider answers with a fixed string and counts its calls. It
// reports the same adapter name for two instances, which is the collision.
type recordingNameProvider struct {
	name   string
	answer string
	calls  int
}

func (p *recordingNameProvider) Name() string { return p.name }
func (p *recordingNameProvider) Capabilities() model.Capabilities {
	return model.Capabilities{Streaming: true}
}
func (p *recordingNameProvider) Models(context.Context) ([]string, error) {
	return []string{"m1"}, nil
}
func (p *recordingNameProvider) Health(context.Context) (string, error) { return "healthy", nil }
func (p *recordingNameProvider) Stream(_ context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	p.calls++
	ch := make(chan agent.ModelEvent, 2)
	ch <- agent.ModelEvent{Kind: agent.EventTextDelta, Text: p.answer, RequestID: req.RequestID}
	ch <- agent.ModelEvent{Kind: agent.EventCompleted, Finished: true, RequestID: req.RequestID}
	close(ch)
	return ch, nil
}

// A provider that reports ToolCalls: false is not a provider Prumo drives — it
// owns its own loop, runs its own tools, and can change the world at any moment
// after it starts. Swapping it for another provider mid-run hands the turn to
// something that has not seen what the first one did, and the second re-plans
// from a conversation with no record of it.
//
// The gateway read afterSideEffects from the caller, which is Prumo's own effect
// flag. For a delegate that flag is always false, because Prumo executed no tools
// — so the one case that most needs the guard was the case the guard could not
// see. The live opencode test is what made it concrete (GAP-173).

func TestAFailureInsideADelegatingProviderIsNotSilentlyRoutedElsewhere(t *testing.T) {
	g := New()
	delegate := &failingDelegate{name: "delegate", started: true}
	other := &recordingNameProvider{name: "other", answer: "started over"}
	g.RegisterAs("delegate", delegate)
	g.RegisterAs("other", other)
	g.DeclareTarget(RouteTarget{Provider: "delegate", Model: "m1"})
	g.DeclareTarget(RouteTarget{Provider: "other", Model: "m1"})
	g.Retry = RetryPolicy{Attempts: 1}

	// The delegate emits a preamble and then fails, which is what a real
	// delegated run does when its own provider goes away mid-work.
	_, _, err := g.StreamWithFallback(context.Background(),
		ModelRoute{Primary: RouteTarget{Provider: "delegate", Model: "m1"},
			Fallbacks: []RouteTarget{{Provider: "other", Model: "m1"}}},
		agent.ModelRequest{RequestID: "r"}, false)
	if err == nil {
		t.Fatal("a delegate that failed after doing work must not have the turn handed to another provider")
	}
	if other.calls != 0 {
		t.Error("the second provider was called; it would re-plan from a conversation that never saw the delegate's work")
	}
}

func TestADelegateThatFailedBeforeDoingAnythingMayStillFallBack(t *testing.T) {
	// A delegate that never started cannot have had effects, so falling back is
	// safe. Refusing here would make a provider that is simply unreachable take
	// the whole run down with it.
	g := New()
	delegate := &failingDelegate{name: "delegate", started: false}
	other := &recordingNameProvider{name: "other", answer: "answered here"}
	g.RegisterAs("delegate", delegate)
	g.RegisterAs("other", other)
	g.DeclareTarget(RouteTarget{Provider: "delegate", Model: "m1"})
	g.DeclareTarget(RouteTarget{Provider: "other", Model: "m1"})
	g.Retry = RetryPolicy{Attempts: 1}

	ch, _, err := g.StreamWithFallback(context.Background(),
		ModelRoute{Primary: RouteTarget{Provider: "delegate", Model: "m1"},
			Fallbacks: []RouteTarget{{Provider: "other", Model: "m1"}}},
		agent.ModelRequest{RequestID: "r"}, false)
	if err != nil {
		t.Fatalf("a delegate that never started must not block the fallback: %v", err)
	}
	var text string
	for ev := range ch {
		if ev.Kind == agent.EventTextDelta {
			text += ev.Text
		}
	}
	if text == "" {
		t.Fatal("the fallback produced nothing")
	}
}

func TestAToolServingProviderStillFallsBackAfterAPartialAnswer(t *testing.T) {
	// The guard must not disable the fallback the gateway exists to provide. A
	// tool-serving provider has run no tools of its own, so handing the turn to
	// another one loses nothing.
	g := New()
	primary := &failingToolProvider{name: "primary"}
	backup := &recordingNameProvider{name: "backup", answer: "the backup answered"}
	g.RegisterAs("primary", primary)
	g.RegisterAs("backup", backup)
	g.DeclareTarget(RouteTarget{Provider: "primary", Model: "m1"})
	g.DeclareTarget(RouteTarget{Provider: "backup", Model: "m1"})
	g.Retry = RetryPolicy{Attempts: 1}

	ch, _, err := g.StreamWithFallback(context.Background(),
		ModelRoute{Primary: RouteTarget{Provider: "primary", Model: "m1"},
			Fallbacks: []RouteTarget{{Provider: "backup", Model: "m1"}}},
		agent.ModelRequest{RequestID: "r"}, false)
	if err != nil {
		t.Fatalf("a tool-serving provider must still fall back: %v", err)
	}
	var text string
	for ev := range ch {
		if ev.Kind == agent.EventTextDelta {
			text += ev.Text
		}
	}
	if text == "" {
		t.Fatal("the fallback produced nothing")
	}
}

// failingDelegate reports ToolCalls:false — it owns its own loop.
type failingDelegate struct {
	name    string
	started bool
}

func (f *failingDelegate) Name() string { return f.name }
func (f *failingDelegate) Capabilities() model.Capabilities {
	return model.Capabilities{Streaming: true, ToolCalls: false, Usage: true}
}
func (f *failingDelegate) Models(context.Context) ([]string, error) { return []string{"m1"}, nil }
func (f *failingDelegate) Health(context.Context) (string, error)   { return "healthy", nil }
func (f *failingDelegate) Stream(_ context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	ch := make(chan agent.ModelEvent, 2)
	if f.started {
		// Preamble, then a failure: the shape a delegated run takes when its own
		// provider disappears after it has already done work.
		ch <- agent.ModelEvent{Kind: agent.ModelEventKind("step_start"), RequestID: req.RequestID}
	}
	ch <- agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID,
		Error: "upstream went away", Retryable: true}
	close(ch)
	return ch, nil
}

// failingToolProvider reports ToolCalls:true — Prumo drives the loop.
type failingToolProvider struct{ name string }

func (f *failingToolProvider) Name() string { return f.name }
func (f *failingToolProvider) Capabilities() model.Capabilities {
	return model.Capabilities{Streaming: true, ToolCalls: true, Usage: true}
}
func (f *failingToolProvider) Models(context.Context) ([]string, error) {
	return []string{"m1"}, nil
}
func (f *failingToolProvider) Health(context.Context) (string, error) { return "healthy", nil }
func (f *failingToolProvider) Stream(_ context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	ch := make(chan agent.ModelEvent, 2)
	ch <- agent.ModelEvent{Kind: agent.EventTextDelta, Text: "partial", RequestID: req.RequestID}
	ch <- agent.ModelEvent{Kind: agent.EventError, RequestID: req.RequestID,
		Error: "boom", Retryable: true}
	close(ch)
	return ch, nil
}
