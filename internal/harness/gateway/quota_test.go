package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
)

func TestQuotaExhaustionSkips(t *testing.T) {
	g := New()
	future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano)
	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339Nano)
	g.SetQuota("drained", 0, future)
	g.SetQuota("reset", 0, past)
	g.SetQuota("rich", 100, "")
	r := g.Select([]RouteTarget{{Provider: "drained"}, {Provider: "reset"}, {Provider: "rich"}, {Provider: "unknown"}})
	if r.Primary.Provider != "reset" && r.Primary.Provider != "rich" && r.Primary.Provider != "unknown" {
		t.Fatalf("drained must be skipped: %+v", r)
	}
	for _, fb := range r.Fallbacks {
		if fb.Provider == "drained" {
			t.Fatalf("drained must not even fallback: %+v", r)
		}
	}
}

func TestRateLimitCoolsDown(t *testing.T) {
	g := New()
	g.Retry = RetryPolicy{Attempts: 1}
	g.Register(&flakyProvider{left: 99})
	route := g.Select([]RouteTarget{{Provider: "flaky"}})
	if _, _, err := g.StreamWithFallback(context.Background(), route, agent.ModelRequest{RequestID: "r"}, false); err == nil {
		t.Fatal("expected failure")
	}
	// "overload" is a rate signal: quota cooldown engages.
	r2 := g.Select([]RouteTarget{{Provider: "flaky"}})
	if r2.Primary.Provider != "" {
		t.Fatalf("rate-limited provider must cool down: %+v", r2)
	}
}
