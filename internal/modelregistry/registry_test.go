package modelregistry

import (
	"strings"
	"testing"
)

func TestRestrictedRequiresLocal(t *testing.T) {
	m := Descriptor{Privacy: External, ContextTokens: 1000}
	if Compatible(m, RouteRequest{DataClass: "restricted"}) {
		t.Fatal("external model accepted restricted data")
	}
}

func TestToolRoute(t *testing.T) {
	m := Descriptor{Privacy: Trusted, ContextTokens: 2000, ToolUse: true, StructuredOutput: true}
	if !Compatible(m, RouteRequest{NeedTools: true, NeedStructuredOutput: true, MinimumContext: 1000}) {
		t.Fatal("compatible model rejected")
	}
}

func TestModelRouterAndFallbacks(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Descriptor{
		ID:               "local-llama",
		Provider:         "ollama",
		Privacy:          Local,
		ContextTokens:    8000,
		ToolUse:          true,
		StructuredOutput: true,
		LatencyClass:     "standard",
	})
	reg.Register(Descriptor{
		ID:               "cloud-fast",
		Provider:         "openai",
		Privacy:          External,
		ContextTokens:    16000,
		ToolUse:          true,
		StructuredOutput: true,
		LatencyClass:     "fast",
	})
	reg.Register(Descriptor{
		ID:               "cloud-deep",
		Provider:         "anthropic",
		Privacy:          External,
		ContextTokens:    32000,
		ToolUse:          true,
		StructuredOutput: true,
		LatencyClass:     "deep",
	})

	// Restricted data route -> only local-llama should qualify
	resp, err := reg.Route(RouteRequest{DataClass: "restricted", MinimumContext: 4000, NeedTools: true})
	if err != nil {
		t.Fatalf("unexpected route error: %v", err)
	}
	if resp.Primary.ID != "local-llama" {
		t.Fatalf("expected local-llama, got: %s", resp.Primary.ID)
	}
	if len(resp.Fallbacks) != 0 {
		t.Fatalf("expected 0 fallbacks for restricted route, got %d", len(resp.Fallbacks))
	}

	// Provider unavailable -> filtered out
	reg.SetHealth("openai", ProviderHealth{Provider: "openai", Status: HealthUnavailable})
	resp2, err := reg.Route(RouteRequest{LatencyPreference: "fast", MinimumContext: 1000})
	if err != nil {
		t.Fatalf("unexpected route error: %v", err)
	}
	if resp2.Primary.ID == "cloud-fast" {
		t.Fatalf("unavailable provider was chosen as primary route")
	}
}

func TestCanaryDriftEvaluation(t *testing.T) {
	eval := EvaluateDrift("model-v1", 0.95, 0.80, 0.10)
	if !eval.DriftDetected {
		t.Fatalf("expected drift detected when drop exceeds tolerance")
	}

	evalNoDrift := EvaluateDrift("model-v1", 0.95, 0.92, 0.10)
	if evalNoDrift.DriftDetected {
		t.Fatalf("did not expect drift detected within tolerance")
	}
}

// The catalog named providers of its own invention — "local", "trusted-cloud",
// "external-cloud" — that appear in no factory and no deployment. A model
// descriptor could therefore describe something that cannot be constructed, and
// nothing noticed until a run tried to route to it (GAP-133).

func TestAModelNamingAProviderNobodyCanBuildIsRejected(t *testing.T) {
	r := NewRegistry()
	err := r.Register(Descriptor{ID: "made-up", Provider: "trusted-cloud"})
	if err == nil {
		t.Fatal("a provider that exists in no factory must be refused where the descriptor is written")
	}
	if !strings.Contains(err.Error(), "trusted-cloud") {
		t.Fatalf("the error must name the provider so it can be fixed, got %v", err)
	}
	if _, present := r.Get("made-up"); present {
		t.Fatal("a refused descriptor must not be stored as though it had been accepted")
	}
}

func TestAProviderTheFactoryBuildsIsAccepted(t *testing.T) {
	r := NewRegistry()
	for _, provider := range []string{"openai-compat", "anthropic", "opencode", "fake"} {
		if err := r.Register(Descriptor{ID: "m-" + provider, Provider: provider}); err != nil {
			t.Errorf("%s is a provider the factory builds and must be accepted: %v", provider, err)
		}
	}
}

func TestAProviderTheDeploymentSuppliesIsAccepted(t *testing.T) {
	// A model served by something else in the deployment — a local ollama, an
	// openai-compatible proxy — is a legitimate catalog entry even though this
	// process cannot construct it. Refusing those would make the catalog unable
	// to describe the deployment it is installed into.
	r := NewRegistry()
	if err := r.Register(Descriptor{ID: "local-llama", Provider: "ollama", Privacy: Local}); err != nil {
		t.Fatalf("an externally provided provider must be accepted: %v", err)
	}
	if err := r.Register(Descriptor{ID: "typo", Provider: "ollamma"}); err == nil {
		t.Fatal("a misspelling must not pass as an external provider")
	}
}

func TestEveryBuiltInProviderIsEitherBuildableOrDeclaredExternal(t *testing.T) {
	// The catalog's own claim, checked: nothing in it is a name with nothing
	// behind it.
	for _, spec := range KnownProviders() {
		if spec.External {
			continue
		}
		if spec.Name == "" {
			t.Fatal("a provider spec with no name describes nothing")
		}
		if spec.Privacy == "" {
			t.Errorf("provider %q has no privacy class; the data-class firewall reads it", spec.Name)
		}
	}
}
