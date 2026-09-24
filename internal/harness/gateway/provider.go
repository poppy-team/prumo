package gateway

import (
	"context"
	"errors"
	"fmt"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/model"
)

// The gateway is the routing layer the runner is supposed to call, and nothing
// called it: the runtime held a model.Provider and was handed the raw adapter, so
// selection, fallback, retry, the circuit breaker and the quota filter were all
// built, tested, and unreachable. Every fix to them was a fix to code no run
// touched (GAP-102).
//
// Implementing model.Provider is the wiring. The runner's call site does not
// change; a caller that passes a gateway instead of an adapter gets routing
// without the loop knowing routing exists, and a caller with one provider can keep
// passing the adapter directly.

var (
	// ErrNoProviders means the gateway has nothing to route to.
	ErrNoProviders = errors.New("gateway: no providers registered")
	// ErrNoTarget means the model a caller pinned is not in the declared pool.
	ErrNoTarget = errors.New("gateway: no target serves the requested model")
)

// Name identifies the gateway as a provider.
func (g *Gateway) Name() string { return "gateway" }

// Capabilities reports what the registered providers can do. A gateway that has
// nothing registered can do nothing, which it says rather than implying a
// capability it does not have.
func (g *Gateway) Capabilities() model.Capabilities {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, p := range g.providers {
		return p.Capabilities()
	}
	return model.Capabilities{}
}

// Models lists the models the registered providers offer, skipping a provider
// that cannot be asked: one unreachable provider must not empty the list.
func (g *Gateway) Models(ctx context.Context) ([]string, error) {
	g.mu.Lock()
	providers := make([]model.Provider, 0, len(g.providers))
	for _, p := range g.providers {
		providers = append(providers, p)
	}
	g.mu.Unlock()
	if len(providers) == 0 {
		return nil, ErrNoProviders
	}
	var out []string
	for _, p := range providers {
		models, err := p.Models(ctx)
		if err != nil {
			continue
		}
		out = append(out, models...)
	}
	if len(out) == 0 {
		return nil, ErrNoProviders
	}
	return out, nil
}

// Health reports the first registered provider's health.
func (g *Gateway) Health(ctx context.Context) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, p := range g.providers {
		return p.Health(ctx)
	}
	return "", ErrNoProviders
}

// Stream routes the request and streams the chosen target's response.
func (g *Gateway) Stream(ctx context.Context, req agent.ModelRequest) (<-chan agent.ModelEvent, error) {
	route, err := g.routeFor(req)
	if err != nil {
		return nil, err
	}
	ch, _, err := g.StreamWithFallback(ctx, route, req, req.NoTransparentFallback)
	return ch, err
}

// DeclareTarget describes a provider and model the gateway may route to.
//
// The descriptor is separate from Register because the two answer different
// questions: Register says "this process can talk to that provider", and a target
// says "and here is what it costs, where its data goes, and whether it can do
// what the request needs". Selection filters on the target, so a provider
// registered without one is routable but unclassified — and an unclassified
// provider is refused wherever a classification is required, rather than being
// treated as though it had passed.
func (g *Gateway) DeclareTarget(target RouteTarget) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, existing := range g.targets {
		if existing.Provider == target.Provider && existing.Model == target.Model {
			return
		}
	}
	g.targets = append(g.targets, target)
}

// routeFor picks the route for a request.
//
// A request that names a model is routed only to a target that serves exactly
// that model. Substituting a different model than the one asked for is how a run
// ends up measured against a capability it never had, so a pinned model that
// nothing serves is an error rather than a quiet preference.
func (g *Gateway) routeFor(req agent.ModelRequest) (ModelRoute, error) {
	g.mu.Lock()
	targets := append([]RouteTarget(nil), g.targets...)
	g.mu.Unlock()
	if len(targets) == 0 {
		return ModelRoute{}, ErrNoProviders
	}
	if req.Model != "" {
		serving := make([]RouteTarget, 0, len(targets))
		for _, t := range targets {
			if t.Model == req.Model {
				serving = append(serving, t)
			}
		}
		if len(serving) == 0 {
			return ModelRoute{}, fmt.Errorf("%w: %s", ErrNoTarget, req.Model)
		}
		targets = serving
	}
	route := g.Select(targets)
	if route.Primary.Provider == "" && route.Primary.Model == "" {
		return ModelRoute{}, ErrNoProviders
	}
	return route, nil
}

// DeclaredTargets reports what the catalog knows about this gateway's pool, for
// a caller deciding what it can ask for.
func (g *Gateway) DeclaredTargets() []RouteTarget {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]RouteTarget(nil), g.targets...)
}

// CapabilitiesFor reports what a route's provider can do.
//
// A route key with no provider behind it has no capabilities, which is the honest
// answer: a route that names nothing cannot be asked to do anything.
func (g *Gateway) CapabilitiesFor(key string) model.Capabilities {
	g.mu.Lock()
	defer g.mu.Unlock()
	provider, ok := g.providers[key]
	if !ok {
		return model.Capabilities{}
	}
	return provider.Capabilities()
}
