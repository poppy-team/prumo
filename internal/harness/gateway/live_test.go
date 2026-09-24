//go:build integration_live

package gateway_test

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/gateway"
	"github.com/raillen/prumo/internal/harness/model"
)

// This test talks to real models.
//
// Every other test in this package uses a double, which is the right default —
// doubles are deterministic and free — but it left the one property that matters
// most about a routing layer unexercised: whether a route survives a real
// connection, a real stream, a real tokenizer and a real provider's idea of its
// own name. A gateway that routes correctly against doubles and drops the call
// against a real endpoint is a gateway that looks finished.
//
// It is behind the integration_live build tag and needs credentials in the
// environment, never in this file.
//
// Models, chosen because they are two different providers behind one adapter —
// the case that made Register key by adapter name a real defect rather than a
// theoretical one:
//
//	Space Bunny Alpha      openrouter/stealth/space-bunny-alpha
//	Muse Spark 1.3 Contrib opencode/muse-spark-1.3-contributor-free
//
// The paid muse-spark-1.3-contributor route is accepted via PRUMO_LIVE_MODEL_B
// for an account with credit; the default is the free tier of the same model.

const (
	liveModelA = "openrouter/stealth/space-bunny-alpha"
	liveModelB = "opencode/muse-spark-1.3-contributor-free"
)

func liveModels(t *testing.T) (string, string) {
	t.Helper()
	if os.Getenv("PRUMO_LIVE_MODEL_A") != "" {
		return os.Getenv("PRUMO_LIVE_MODEL_A"), os.Getenv("PRUMO_LIVE_MODEL_B")
	}
	return liveModelA, liveModelB
}

// liveProvider builds the opencode-backed provider for a model, or skips when
// there is nothing to run against.
func liveProvider(t *testing.T, modelID string) model.Provider {
	t.Helper()
	if os.Getenv("OPENCODE_API_KEY") == "" {
		t.Skip("set OPENCODE_API_KEY to run the live routing test")
	}
	// LookPath rather than a hard-coded location: the binary is wherever the
	// user's PATH puts it, and a check against a fixed path reports "not
	// installed" for a working setup and skips the test for the wrong reason.
	if binary := model.NewOpenCode(modelID).Binary; binary == "" {
		t.Skip("no opencode binary")
	} else if _, err := exec.LookPath(binary); err != nil {
		if _, err := exec.LookPath(model.NewOpenCode(modelID).Binary); err != nil {
			t.Skipf("opencode CLI not found on PATH: %v", err)
		}
	}
	return model.NewOpenCode(modelID)
}

func TestLiveGatewayRoutesBetweenTwoRealModels(t *testing.T) {
	modelA, modelB := liveModels(t)
	providerA := liveProvider(t, modelA)
	providerB := liveProvider(t, modelB)

	g := gateway.New()
	// Two instances of the same adapter, so they must be registered under
	// distinct keys. Registering both by adapter name would leave one provider
	// in the pool and every route to the other name reaching the survivor.
	g.RegisterAs("model-a", providerA)
	g.RegisterAs("model-b", providerB)
	g.DeclareTarget(gateway.RouteTarget{Provider: "model-a", Model: modelA, Privacy: "external"})
	g.DeclareTarget(gateway.RouteTarget{Provider: "model-b", Model: modelB, Privacy: "external"})
	// One attempt: this is a routing test, not a patience test, and a retry here
	// would hide a failure behind a second chance.
	g.Retry = gateway.RetryPolicy{Attempts: 1}

	for _, modelID := range []string{modelA, modelB} {
		t.Run(modelID, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
			defer cancel()

			ch, err := g.Stream(ctx, agent.ModelRequest{
				RequestID: "live-" + modelID,
				Model:     modelID,
				Messages: []agent.Message{
					{ID: "m1", Role: agent.RoleUser, Content: "Reply with exactly: ok"},
				},
			})
			if err != nil {
				t.Fatalf("routing to %s failed: %v", modelID, err)
			}

			var text string
			var usage *agent.Usage
			var failed bool
			for ev := range ch {
				switch ev.Kind {
				case agent.EventTextDelta:
					text += ev.Text
				case agent.EventUsageUpdated:
					usage = ev.Usage
				case agent.EventError:
					failed = true
					t.Logf("provider error: %s (quota=%v retryAfter=%s)", ev.Error, ev.Quota, ev.RetryAfter)
				}
			}
			if failed {
				t.Fatalf("%s reported a failure", modelID)
			}
			if text == "" {
				t.Fatalf("%s returned no text", modelID)
			}
			t.Logf("%s -> %q", modelID, text)
			if usage != nil {
				t.Logf("%s usage: in=%d out=%d cacheRead=%d reasoning=%d cost=%.6f",
					modelID, usage.InputTokens, usage.OutputTokens,
					usage.CacheReadTokens, usage.ReasoningTokens, usage.CostUSD)
			} else {
				t.Logf("%s reported no usage event", modelID)
			}
		})
	}
}

// TestLiveGatewayFallsBackToTheOtherRealModel checks the property doubles cannot:
// that a failure on one real route lands on another real route. The first route is
// made to fail by naming a model the provider does not serve, which is a real
// 404 rather than a simulated error.
func TestLiveGatewayFallsBackToTheOtherRealModel(t *testing.T) {
	_, modelB := liveModels(t)
	providerA := liveProvider(t, "openrouter/stealth/no-such-model-exists")
	providerB := liveProvider(t, modelB)

	g := gateway.New()
	g.RegisterAs("broken", providerA)
	g.RegisterAs("working", providerB)
	g.DeclareTarget(gateway.RouteTarget{Provider: "broken", Model: "openrouter/stealth/no-such-model-exists"})
	g.DeclareTarget(gateway.RouteTarget{Provider: "working", Model: modelB})
	g.Retry = gateway.RetryPolicy{Attempts: 1, Backoff: 10 * time.Millisecond}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	ch, err := g.Stream(ctx, agent.ModelRequest{
		RequestID: "live-fallback",
		Messages:  []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: "Reply with exactly: ok"}},
	})
	if err != nil {
		t.Fatalf("a broken first route must fall through to the second: %v", err)
	}
	var text string
	for ev := range ch {
		if ev.Kind == agent.EventTextDelta {
			text += ev.Text
		}
	}
	if text == "" {
		t.Fatal("the fallback route produced no text")
	}
	t.Logf("fell back to the live model -> %q", text)
}
