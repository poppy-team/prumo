package gateway

import (
	"strings"
	"testing"
	"time"
)

func TestSelectByClass(t *testing.T) {
	gw := New()

	targets := []RouteTarget{
		{Provider: "prov-fast", Model: "m-fast", Latency: "fast", CostPer1k: 0.05},
		{Provider: "prov-cheap", Model: "m-cheap", Latency: "standard", CostPer1k: 0.002},
		{Provider: "prov-primary", Model: "m-primary", Latency: "standard", CostPer1k: 0.02},
	}

	// 1. ROUTE-CHEAP
	routeCheap, expCheap := gw.SelectByClass(targets, RouteClassCheap)
	if routeCheap.Primary.Provider != "prov-cheap" {
		t.Errorf("expected prov-cheap for ROUTE-CHEAP, got %s", routeCheap.Primary.Provider)
	}
	if expCheap.ReasonCode != ReasonCodeCheapPreferred {
		t.Errorf("expected reason code %s, got %s", ReasonCodeCheapPreferred, expCheap.ReasonCode)
	}

	// 2. ROUTE-FALLBACK (fast preferred)
	routeFast, expFast := gw.SelectByClass(targets, RouteClassFallback)
	if routeFast.Primary.Provider != "prov-fast" {
		t.Errorf("expected prov-fast for ROUTE-FALLBACK, got %s", routeFast.Primary.Provider)
	}
	if expFast.ReasonCode != ReasonCodeFallbackDegraded {
		t.Errorf("expected reason code %s, got %s", ReasonCodeFallbackDegraded, expFast.ReasonCode)
	}

	// 3. Quota Status verification
	if st := gw.GetQuotaStatus("unknown-provider"); st != QuotaStatusUnknown {
		t.Errorf("expected QuotaStatusUnknown, got %s", st)
	}

	gw.SetQuota("prov-cheap", 1000.0, "")
	if st := gw.GetQuotaStatus("prov-cheap"); st != QuotaStatusKnown {
		t.Errorf("expected QuotaStatusKnown, got %s", st)
	}

	gw.SetQuota("prov-exhausted", 0.0, "")
	if st := gw.GetQuotaStatus("prov-exhausted"); st != QuotaStatusExhausted {
		t.Errorf("expected QuotaStatusExhausted, got %s", st)
	}

	futureReset := time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339)
	gw.SetQuota("prov-cooldown", 0.0, futureReset)
	if st := gw.GetQuotaStatus("prov-cooldown"); st != QuotaStatusCooldown {
		t.Errorf("expected QuotaStatusCooldown, got %s", st)
	}
}

func TestAccountPool(t *testing.T) {
	ap := NewAccountPool()

	ap.RegisterAccount("acc-1", "openai", "sk-test-1")
	ap.RegisterAccount("acc-2", "openai", "sk-test-2")

	// Test round robin
	accA, err := ap.PickAccount("openai")
	if err != nil || accA.ID != "acc-1" {
		t.Fatalf("expected acc-1, got %v (err: %v)", accA, err)
	}
	accB, err := ap.PickAccount("openai")
	if err != nil || accB.ID != "acc-2" {
		t.Fatalf("expected acc-2, got %v (err: %v)", accB, err)
	}

	// Fail acc-1 with rate limit -> goes to cooldown
	ap.RecordFailure("acc-1", true)

	// Now picking should only return acc-2
	accC, err := ap.PickAccount("openai")
	if err != nil || accC.ID != "acc-2" {
		t.Fatalf("expected acc-2 while acc-1 in cooldown, got %v (err: %v)", accC, err)
	}

	// Fail acc-2 with rate limit -> both in cooldown
	ap.RecordFailure("acc-2", true)

	_, err = ap.PickAccount("openai")
	if err == nil || !strings.Contains(err.Error(), "all accounts in cooldown") {
		t.Fatalf("expected error when all accounts in cooldown, got: %v", err)
	}

	// Clear acc-1 -> recovers
	ap.RecordSuccess("acc-1")
	accD, err := ap.PickAccount("openai")
	if err != nil || accD.ID != "acc-1" {
		t.Fatalf("expected acc-1 after recovery, got %v (err: %v)", accD, err)
	}
}
