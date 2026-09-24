package modelregistry

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Privacy string

const (
	Local    Privacy = "local"
	Trusted  Privacy = "trusted"
	External Privacy = "external"
	Unknown  Privacy = "unknown"
)

type HealthStatus string

const (
	HealthHealthy     HealthStatus = "healthy"
	HealthDegraded    HealthStatus = "degraded"
	HealthUnavailable HealthStatus = "unavailable"
)

type ProviderHealth struct {
	Provider     string       `json:"provider"`
	Status       HealthStatus `json:"status"`
	LatencyMS    int          `json:"latency_ms"`
	LastChecked  string       `json:"last_checked"`
	FailureCount int          `json:"failure_count"`
}

type Descriptor struct {
	ID                    string         `json:"id"`
	Version               int            `json:"version"`
	Provider              string         `json:"provider"`
	Privacy               Privacy        `json:"privacy"`
	ContextTokens         int            `json:"context_tokens"`
	StructuredOutput      bool           `json:"structured_output"`
	ToolUse               bool           `json:"tool_use"`
	LatencyClass          string         `json:"latency_class,omitempty"` // "fast", "standard", "deep"
	CostPer1kInputTokens  float64        `json:"cost_per_1k_input,omitempty"`
	CostPer1kOutputTokens float64        `json:"cost_per_1k_output,omitempty"`
	CanaryScore           float64        `json:"canary_score,omitempty"`
	Pricing               map[string]any `json:"pricing,omitempty"`
	RateLimit             map[string]any `json:"rate_limit,omitempty"`
}

type RouteRequest struct {
	Risk                 string `json:"risk,omitempty"`
	DataClass            string `json:"data_class,omitempty"`
	NeedTools            bool   `json:"need_tools,omitempty"`
	NeedStructuredOutput bool   `json:"need_structured_output,omitempty"`
	MinimumContext       int    `json:"minimum_context,omitempty"`
	LatencyPreference    string `json:"latency_preference,omitempty"` // "fast", "standard", "any"
}

type RouteResponse struct {
	Primary     Descriptor   `json:"primary"`
	Fallbacks   []Descriptor `json:"fallbacks"`
	Explanation string       `json:"explanation"`
	Score       float64      `json:"score"`
}

func Compatible(model Descriptor, request RouteRequest) bool {
	if request.DataClass == "restricted" && model.Privacy != Local {
		return false
	}
	if request.NeedTools && !model.ToolUse {
		return false
	}
	if request.NeedStructuredOutput && !model.StructuredOutput {
		return false
	}
	return model.ContextTokens >= request.MinimumContext
}

type Registry struct {
	models map[string]Descriptor
	health map[string]ProviderHealth
}

func NewRegistry() *Registry {
	return &Registry{
		models: make(map[string]Descriptor),
		health: make(map[string]ProviderHealth),
	}
}

// Register adds a descriptor, refusing one that describes something the factory
// cannot build.
//
// It used to accept anything, so the registry could name providers that exist in
// no factory — "local", "trusted-cloud", "external-cloud" were all registered and
// none was constructible. The failure surfaced as a routing error at run time,
// with the descriptor that caused it sitting in a file nobody had connected to
// anything (GAP-133). A catalog is a promise; an entry that cannot be delivered
// is rejected where it is written, not discovered where it is used.
func (r *Registry) Register(d Descriptor) error {
	if d.ID == "" {
		return fmt.Errorf("model descriptor has no id")
	}
	if _, buildable := ProviderSpecFor(d.Provider); !buildable {
		known := make([]string, 0, 5)
		for _, spec := range KnownProviders() {
			known = append(known, spec.Name)
		}
		return fmt.Errorf("model %q names provider %q, which the factory cannot build (known: %s)",
			d.ID, d.Provider, strings.Join(known, ", "))
	}
	r.models[d.ID] = d
	return nil
}

func (r *Registry) Get(id string) (Descriptor, bool) {
	d, ok := r.models[id]
	return d, ok
}

func (r *Registry) List() []Descriptor {
	out := make([]Descriptor, 0, len(r.models))
	for _, d := range r.models {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (r *Registry) SetHealth(provider string, h ProviderHealth) {
	r.health[provider] = h
}

func (r *Registry) GetHealth(provider string) ProviderHealth {
	h, ok := r.health[provider]
	if !ok {
		return ProviderHealth{Provider: provider, Status: HealthHealthy}
	}
	return h
}

func (r *Registry) Route(req RouteRequest) (RouteResponse, error) {
	candidates := make([]Descriptor, 0)
	for _, m := range r.models {
		if !Compatible(m, req) {
			continue
		}
		h := r.GetHealth(m.Provider)
		if h.Status == HealthUnavailable {
			continue
		}
		candidates = append(candidates, m)
	}

	if len(candidates) == 0 {
		return RouteResponse{}, errors.New("no compatible model available for request")
	}

	// Score candidates: healthy +10, low latency +5, local privacy if confidential/internal +5
	type scoredModel struct {
		model Descriptor
		score float64
	}
	scored := make([]scoredModel, 0, len(candidates))
	for _, c := range candidates {
		s := 10.0
		h := r.GetHealth(c.Provider)
		if h.Status == HealthDegraded {
			s -= 5.0
		}
		if req.LatencyPreference == "fast" && c.LatencyClass == "fast" {
			s += 5.0
		}
		if (req.DataClass == "confidential" || req.DataClass == "internal") && c.Privacy == Local {
			s += 5.0
		}
		if c.CanaryScore > 0 {
			s += c.CanaryScore * 2.0
		}
		scored = append(scored, scoredModel{model: c, score: s})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	primary := scored[0].model
	fallbacks := make([]Descriptor, 0, len(scored)-1)
	for i := 1; i < len(scored); i++ {
		fallbacks = append(fallbacks, scored[i].model)
	}

	explanation := fmt.Sprintf("Selected %s (provider=%s, privacy=%s, score=%.1f) from %d candidate(s)",
		primary.ID, primary.Provider, primary.Privacy, scored[0].score, len(candidates))

	return RouteResponse{
		Primary:     primary,
		Fallbacks:   fallbacks,
		Explanation: explanation,
		Score:       scored[0].score,
	}, nil
}

// Canary & drift tracking
type CanaryEval struct {
	ModelID       string  `json:"model_id"`
	BaselineScore float64 `json:"baseline_score"`
	CurrentScore  float64 `json:"current_score"`
	DriftDetected bool    `json:"drift_detected"`
	EvaluatedAt   string  `json:"evaluated_at"`
}

func EvaluateDrift(modelID string, baseline, current, tolerance float64) CanaryEval {
	diff := baseline - current
	drift := diff > tolerance
	return CanaryEval{
		ModelID:       modelID,
		BaselineScore: baseline,
		CurrentScore:  current,
		DriftDetected: drift,
		EvaluatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
}

// ProviderSpec is what the factory knows how to build, and it lives here rather
// than in the adapters because this package is the leaf everything else imports.
// The adapters read it to decide what they can build, which makes the catalog the
// single place a provider is declared — the arrangement GAP-133 asked for and did
// not have.
//
// The registry used to name providers of its own invention — "local",
// "trusted-cloud", "external-cloud" — that appear in no factory anywhere, so a
// descriptor could describe a provider that could not be constructed and nothing
// noticed until a run tried to route to it.
type ProviderSpec struct {
	// Name is what the provider factory accepts.
	Name string `json:"name"`
	// Privacy is where a request's data goes when this provider is used. Both the
	// data-class firewall and the gateway's LocalOnly filter read it, so it is
	// declared once here instead of restated per descriptor.
	Privacy Privacy `json:"privacy"`
	// NeedsBaseURL reports that the provider is useless without an endpoint.
	NeedsBaseURL bool `json:"needs_base_url,omitempty"`
	// BuildableOffline is true for the deterministic providers, which need no
	// endpoint and no key.
	BuildableOffline bool `json:"buildable_offline,omitempty"`
	// External marks a provider the catalog may describe but Prumo does not
	// construct: a model served by something else in the deployment, reached
	// through a proxy or a local runtime.
	//
	// It exists because "Prumo can build this" and "this is a real provider" are
	// different questions. The catalog legitimately describes a model this
	// process cannot dial — that is what a local ollama or an openai-compatible
	// proxy is. What it may not describe is a provider that exists in no factory
	// and no deployment, which is what the invented "local", "trusted-cloud" and
	// "external-cloud" entries were.
	External bool `json:"external,omitempty"`
}

// knownProviders is the single declaration of what can be built. The adapter
// factory mirrors it; a name added there and not here is a provider the catalog
// cannot describe, and a name here the factory cannot build is a provider the
// catalog promised and cannot deliver.
var knownProviders = []ProviderSpec{
	{Name: "openai-compat", Privacy: ExternalPrivacy, NeedsBaseURL: true},
	{Name: "anthropic", Privacy: ExternalPrivacy, NeedsBaseURL: true},
	{Name: "opencode", Privacy: Local, BuildableOffline: true},
	{Name: "fake", Privacy: Local, BuildableOffline: true},
	{Name: "fake-tools", Privacy: Local, BuildableOffline: true},
	// Provided by the deployment rather than constructed here. Naming them is
	// what makes a typo in a model descriptor visible; leaving the set open would
	// make every misspelling look like one of these.
	{Name: "openai", Privacy: ExternalPrivacy, External: true},
	{Name: "ollama", Privacy: Local, External: true},
	{Name: "openrouter", Privacy: ExternalPrivacy, External: true},
}

// ExternalPrivacy is the privacy class for data that leaves the machine. It is
// named apart from the ProviderSpec.External field so the two, which mean
// unrelated things, are not confused when read together.
const ExternalPrivacy Privacy = "external"

// KnownProviders lists the providers the catalog can describe.
func KnownProviders() []ProviderSpec {
	out := make([]ProviderSpec, len(knownProviders))
	copy(out, knownProviders)
	return out
}

// ProviderSpecFor returns what is known about a provider name.
func ProviderSpecFor(name string) (ProviderSpec, bool) {
	for _, spec := range knownProviders {
		if spec.Name == name {
			return spec, true
		}
	}
	return ProviderSpec{}, false
}
