// Package gateway implements the internal Model Gateway: routing, fallback,
// retry and circuit breaker over ModelProviders. It never routes external
// agent runtimes through the model path.
package gateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/model"
)

// RouteTarget is one selectable provider+model.
type RouteTarget struct {
	Provider   string  `json:"provider"`
	Model      string  `json:"model"`
	Weight     int     `json:"weight,omitempty"`
	Tools      bool    `json:"tools,omitempty"`
	Structured bool    `json:"structured_output,omitempty"`
	Latency    string  `json:"latency,omitempty"`     // fast|standard|deep
	Privacy    string  `json:"privacy,omitempty"`     // local|trusted|external
	CostPer1k  float64 `json:"cost_per_1k,omitempty"` // blended USD, 0 = unknown
}

// Select picks primary+fallbacks: healthy first, deterministic order.
func (g *Gateway) Select(targets []RouteTarget) ModelRoute {
	return g.SelectWithPolicy(targets, Policy{})
}

// Policy tunes selection beyond healthy-first (GAP-005 first slice).
type Policy struct {
	// Prefer orders healthy targets: "cheap" (cost asc, unknown last),
	// "fast" (latency rank), "" keeps deterministic provider order.
	Prefer string
	// RequireTools/RequireStructured filter incapable targets.
	RequireTools      bool
	RequireStructured bool
	// LocalOnly keeps privacy=local targets (restricted data classes).
	LocalOnly bool
	// AllowedProviders restricts the pool; empty allows all.
	AllowedProviders []string
}

// SelectWithPolicy filters by capability/privacy/pool, then orders healthy
// targets by policy preference with deterministic tiebreaks.
func (g *Gateway) SelectWithPolicy(targets []RouteTarget, p Policy) ModelRoute {
	allowed := map[string]bool{}
	for _, a := range p.AllowedProviders {
		allowed[a] = true
	}
	kept := []RouteTarget{}
	for _, t := range targets {
		if len(allowed) > 0 && !allowed[t.Provider] {
			continue
		}
		if p.RequireTools && !t.Tools {
			continue
		}
		if p.RequireStructured && !t.Structured {
			continue
		}
		// LocalOnly is a firewall, so it fails closed. A target whose privacy is
		// unknown is not evidence that it is local: `t.Privacy != ""` let an empty
		// value through, which meant the one case where nobody had classified the
		// target was the one case allowed to receive restricted data (GAP-156).
		if p.LocalOnly && t.Privacy != "local" {
			continue
		}
		if g.quotaExhausted(t.Provider) {
			continue
		}
		kept = append(kept, t)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	healthy, degraded := []RouteTarget{}, []RouteTarget{}
	for _, t := range kept {
		h := g.health[t.Provider]
		if h == nil {
			healthy = append(healthy, t)
			continue
		}
		// An open circuit whose cooldown has passed is tried again, as a
		// half-open probe. The cooldown used to be written and never read, so a
		// provider that failed three times was excluded from every route for the
		// life of the process — the breaker opened and could never close, and the
		// recovery it existed to enable was unreachable (GAP-103).
		if h.Status == "open" && cooldownElapsed(h.CooldownUntil) {
			h.Status = "degraded"
			h.CooldownUntil = ""
		}
		switch h.Status {
		case "healthy":
			healthy = append(healthy, t)
		case "degraded":
			degraded = append(degraded, t)
		}
	}
	order := func(ts []RouteTarget) {
		sort.SliceStable(ts, func(i, j int) bool { return ts[i].Provider < ts[j].Provider })
		switch p.Prefer {
		case "cheap":
			sort.SliceStable(ts, func(i, j int) bool {
				a, b := ts[i].CostPer1k, ts[j].CostPer1k
				if (a == 0) != (b == 0) {
					return b == 0 // known costs first
				}
				return a < b
			})
		case "fast":
			rank := map[string]int{"fast": 0, "standard": 1, "deep": 2, "": 3}
			sort.SliceStable(ts, func(i, j int) bool { return rank[ts[i].Latency] < rank[ts[j].Latency] })
		}
	}
	order(healthy)
	order(degraded)
	ordered := append(healthy, degraded...)
	if len(ordered) == 0 {
		return ModelRoute{}
	}
	return ModelRoute{Primary: ordered[0], Fallbacks: ordered[1:], Reason: "policy-ordered deterministic"}
}

// ModelRoute is the chosen route with fallbacks.
type ModelRoute struct {
	Primary   RouteTarget   `json:"primary"`
	Fallbacks []RouteTarget `json:"fallbacks,omitempty"`
	Reason    string        `json:"reason,omitempty"`
}

// Health tracks per-provider circuit state.
type Health struct {
	Status        string `json:"status"` // healthy|degraded|open
	Failures      int    `json:"failures"`
	LastFailure   string `json:"last_failure,omitempty"`
	CooldownUntil string `json:"cooldown_until,omitempty"`
}

// RetryPolicy bounds same-provider retries for retryable failures.
// Attempts counts total tries (1 = no retry); Backoff is the unit delay.
// Class multipliers: rate-limit 5x, server-5xx 2x, other 1x, times attempt
// (linear). Jitter adds a deterministic attempt-hashed offset (no RNG in
// the hot path, stable under test).
type RetryPolicy struct {
	Attempts int
	Backoff  time.Duration
	Jitter   bool
}

// DefaultRetryPolicy retries twice with a 200ms base backoff.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{Attempts: 3, Backoff: 200 * time.Millisecond}
}

// QuotaState tracks remaining quota per provider (GAP-005 remainder).
// Unknown providers are treated as unlimited; callers feed this from
// 429 headers and usage accounting.
type QuotaState struct {
	Remaining float64 `json:"remaining"`
	ResetsAt  string  `json:"resets_at,omitempty"`
}

// Gateway routes model requests across registered providers.
type Gateway struct {
	mu        sync.Mutex
	providers map[string]model.Provider
	health    map[string]*Health
	quotas    map[string]QuotaState
	// targets are the declared provider+model pairs selection filters over.
	targets []RouteTarget
	Retry   RetryPolicy
	// AfterSideEffects=false allows transparent fallback; once a Run has
	// observable effects, callers must use explicit Handoff instead.
}

func New() *Gateway {
	return &Gateway{
		providers: map[string]model.Provider{},
		health:    map[string]*Health{},
		quotas:    map[string]QuotaState{},
		Retry:     DefaultRetryPolicy(),
	}
}

// SetQuota records remaining quota (negative = unlimited).
func (g *Gateway) SetQuota(provider string, remaining float64, resetsAt string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.quotas[provider] = QuotaState{Remaining: remaining, ResetsAt: resetsAt}
}

// quotaExhausted reports whether a provider is known to be out of quota.
//
// It takes the lock. It used to read the map unguarded while SetQuota and the
// rate-limit path in streamWithRetry wrote it under the lock, so a concurrent
// selection and a quota update raced on the same map (GAP-103).
//
// An unparseable reset time is treated as exhausted rather than as unlimited:
// "we could not tell when this clears" is not a reason to send more traffic at a
// provider that has already said it is full.
func (g *Gateway) quotaExhausted(provider string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	q, ok := g.quotas[provider]
	if !ok || q.Remaining != 0 {
		return false
	}
	if q.ResetsAt == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339Nano, q.ResetsAt)
	if err != nil {
		if t2, err2 := time.Parse(time.RFC3339, q.ResetsAt); err2 == nil {
			t = t2
		} else {
			return true
		}
	}
	return time.Now().UTC().Before(t)
}

// cooldownElapsed reports whether a cooldown has run out. An absent or
// unparseable cooldown is treated as elapsed: a circuit that says "wait until
// some time nobody can read" must not hold a provider out of routing forever.
func cooldownElapsed(until string) bool {
	if until == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339Nano, until)
	if err != nil {
		return true
	}
	return !time.Now().UTC().Before(t)
}

func (g *Gateway) Register(p model.Provider) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.providers[p.Name()] = p
	g.health[p.Name()] = &Health{Status: "healthy"}
}

func (g *Gateway) recordFailure(provider string) {
	h := g.health[provider]
	if h == nil {
		return
	}
	h.Failures++
	h.LastFailure = time.Now().UTC().Format(time.RFC3339Nano)
	if h.Failures >= 3 {
		h.Status = "open"
		h.CooldownUntil = time.Now().UTC().Add(30 * time.Second).Format(time.RFC3339Nano)
	} else if h.Failures >= 1 {
		h.Status = "degraded"
	}
}

func (g *Gateway) recordSuccess(provider string) {
	h := g.health[provider]
	if h == nil {
		return
	}
	h.Failures = 0
	h.Status = "healthy"
	// The cooldown is cleared with the status. Leaving it behind would mean a
	// later failure set a fresh one, but a stale past timestamp left in place
	// reads as "still cooling" to anything that consults it directly.
	h.CooldownUntil = ""
}

// StreamWithFallback tries primary then fallbacks on retryable errors.
// afterSideEffects=true disables transparent fallback (caller must Handoff).
func (g *Gateway) StreamWithFallback(ctx context.Context, route ModelRoute, req agent.ModelRequest, afterSideEffects bool) (<-chan agent.ModelEvent, string, error) {
	chain := append([]RouteTarget{route.Primary}, route.Fallbacks...)
	var lastErr error
	for i, t := range chain {
		if i > 0 && afterSideEffects {
			return nil, "", fmt.Errorf("transparent fallback denied after side effects: use explicit Handoff (would try %s)", t.Provider)
		}
		g.mu.Lock()
		p := g.providers[t.Provider]
		g.mu.Unlock()
		if p == nil {
			lastErr = fmt.Errorf("unknown provider %s", t.Provider)
			continue
		}
		r := req
		if t.Model != "" {
			r.Model = t.Model
		}
		ch, err := g.streamWithRetry(ctx, p, t.Provider, r)
		if err != nil {
			lastErr = err
			continue
		}
		return ch, t.Provider, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no route targets")
	}
	return nil, "", lastErr
}

// streamWithRetry opens the provider stream and peeks the first event,
// retrying retryable failures per policy with linear backoff. Successes
// reset the circuit; terminal provider exhaustion records one failure.
func (g *Gateway) streamWithRetry(ctx context.Context, p model.Provider, name string, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	attempts := g.Retry.Attempts
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for a := 0; a < attempts; a++ {
		if a > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoffFor(g.Retry, a, lastErr)):
			}
		}
		ch, err := p.Stream(ctx, req)
		if err != nil {
			lastErr = err
			continue
		}
		select {
		case ev, ok := <-ch:
			if !ok {
				lastErr = &failure{err: fmt.Errorf("provider %s closed stream", name)}
				continue
			}
			if ev.Kind == agent.EventError && ev.Retryable {
				// The adapter's classification travels on the event, so the
				// retry interval the provider asked for and whether this is a
				// quota are both still here. Wrapping only the message is what
				// made the gateway read prose to decide what it was looking at
				// (GAP-108).
				lastErr = &failure{
					err:        fmt.Errorf("provider %s: %s", name, ev.Error),
					quota:      ev.Quota,
					retryAfter: ev.RetryAfter,
				}
				continue
			}
			g.mu.Lock()
			g.recordSuccess(name)
			g.mu.Unlock()
			out := make(chan agent.ModelEvent, 64)
			out <- ev
			go func() {
				defer close(out)
				for e := range ch {
					out <- e
				}
			}()
			return out, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	g.mu.Lock()
	g.recordFailure(name)
	if isRateLimit(lastErr) {
		// Rate-limited providers cool down; selection skips them until reset.
		// The provider's own Retry-After is the reset point when it gave one, so
		// the gateway stops sending at the time the limit actually lifts rather
		// than at a fixed guess (GAP-108).
		cooldown := 30 * time.Second
		if requested := retryAfterFrom(lastErr); requested > 0 {
			cooldown = requested
		}
		g.quotas[name] = QuotaState{
			Remaining: 0,
			ResetsAt:  time.Now().UTC().Add(cooldown).Format(time.RFC3339Nano),
		}
	}
	g.mu.Unlock()
	if lastErr == nil {
		lastErr = fmt.Errorf("provider %s exhausted retries", name)
	}
	return nil, lastErr
}

// failure is what a provider call left behind: the error, plus the two facts
// that decide how to retry it. It is carried as its own type because a bare error
// cannot hold them, and putting them in the message is what this whole change
// removes.
type failure struct {
	err        error
	quota      bool
	retryAfter time.Duration
}

func (f *failure) Error() string {
	if f == nil || f.err == nil {
		return ""
	}
	return f.err.Error()
}

func (f *failure) Unwrap() error {
	if f == nil {
		return nil
	}
	return f.err
}

// isRateLimit recognizes quota/rate signals across provider dialects.
//
// A typed provider failure is authoritative: it carries the classification the
// adapter made from the status code and the payload. Substring matching is the
// fallback for an error that never went through an adapter, and is deliberately
// never consulted first, because matching prose is how a vendor that says
// "RESOURCE_EXHAUSTED" gets treated as an ordinary fault and retried straight
// back into the limit (GAP-108).
func isRateLimit(err error) bool {
	if err == nil {
		return false
	}
	var f *failure
	if errors.As(err, &f) {
		if f.quota {
			return true
		}
		if f.retryAfter > 0 {
			// A provider that asked us to wait is telling us it is busy, whether
			// or not it labelled the failure a quota.
			return true
		}
		// The adapter did not classify this one, so the message is all there is.
		// A provider built outside this package — a test double, or a plugin —
		// never sets Quota, and dropping the text signals for it would make a
		// rate-limited provider look like an ordinary fault.
		return containsRateSignal(f.Error())
	}
	if pe, ok := model.ProviderErrorFrom(err); ok {
		if pe.Quota {
			return true
		}
		if pe.RetryAfter > 0 {
			return true
		}
		return pe.StatusCode == http.StatusTooManyRequests
	}
	return containsRateSignal(err.Error())
}

// containsRateSignal is the text fallback for failures that were never
// classified by an adapter.
func containsRateSignal(message string) bool {
	lowered := strings.ToLower(message)
	for _, signal := range []string{
		"429", "rate_limit", "rate limit", "ratelimit", "overload", "quota",
		"resource_exhausted", "resource exhausted", "quota_exceeded", "too many requests",
	} {
		if strings.Contains(lowered, signal) {
			return true
		}
	}
	return false
}

// backoffFor computes the pre-attempt delay (attempt starts at 1).
//
// The delay is exponential, because linear backoff retries a struggling
// provider at nearly the same rate it just failed at. A provider under load
// needs less traffic, not the same traffic sooner. The exponent is capped so a
// generous policy cannot produce a delay longer than a run is willing to wait.
//
// Retry-After overrides the computed delay when the provider sent one: it is the
// provider saying when it will be ready, and guessing a shorter wait than the one
// asked for means the retry is rejected for the same reason (GAP-108).
func backoffFor(p RetryPolicy, attempt int, err error) time.Duration {
	mult := 1.0
	if isRateLimit(err) {
		mult = 5
	} else if isServerError(err) {
		mult = 2
	}
	// 2^(attempt-1), capped, so attempt 1 waits 1x, attempt 2 waits 2x, and the
	// growth stops somewhere a caller can still reason about.
	factor := 1 << min(attempt-1, 6)
	d := time.Duration(float64(p.Backoff) * float64(factor) * mult)
	if requested := retryAfterFrom(err); requested > d {
		d = requested
	}
	if p.Jitter && d > 0 {
		d += time.Duration((attempt*37)%100) * p.Backoff / 100
	}
	return d
}

// retryAfterFrom reads the delay the provider asked for out of a typed failure.
func retryAfterFrom(err error) time.Duration {
	var f *failure
	if errors.As(err, &f) {
		return f.retryAfter
	}
	if pe, ok := model.ProviderErrorFrom(err); ok {
		return pe.RetryAfter
	}
	return 0
}

func isServerError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	for _, sig := range []string{"500", "502", "503", "504", "5xx", "http 5", "timeout", "unavailable"} {
		if strings.Contains(s, sig) {
			return true
		}
	}
	return false
}
