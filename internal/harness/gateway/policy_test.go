package gateway

import "testing"

func TestPolicyCheapFirst(t *testing.T) {
	g := New()
	r := g.SelectWithPolicy([]RouteTarget{
		{Provider: "expensive", CostPer1k: 1.0},
		{Provider: "cheap", CostPer1k: 0.01},
		{Provider: "unknown"},
	}, Policy{Prefer: "cheap"})
	if r.Primary.Provider != "cheap" || len(r.Fallbacks) != 2 || r.Fallbacks[1].Provider != "unknown" {
		t.Fatalf("cheap-first failed: %+v", r)
	}
}

func TestPolicyFilters(t *testing.T) {
	g := New()
	r := g.SelectWithPolicy([]RouteTarget{
		{Provider: "no-tools"},
		{Provider: "ext", Tools: true, Privacy: "external"},
		{Provider: "local", Tools: true, Privacy: "local"},
	}, Policy{RequireTools: true, LocalOnly: true})
	if r.Primary.Provider != "local" || len(r.Fallbacks) != 0 {
		t.Fatalf("capability+privacy filter failed: %+v", r)
	}
	empty := g.SelectWithPolicy([]RouteTarget{{Provider: "x"}}, Policy{AllowedProviders: []string{"y"}})
	if empty.Primary.Provider != "" {
		t.Fatalf("empty pool must yield empty route: %+v", empty)
	}
}
