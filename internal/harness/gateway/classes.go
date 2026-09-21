package gateway

import (
	"fmt"
	"time"
)

// Routing class constants per Constitution 87.
const (
	RouteClassCheap    = "ROUTE-CHEAP"
	RouteClassPrimary  = "ROUTE-PRIMARY"
	RouteClassFallback = "ROUTE-FALLBACK"
)

// Quota state status per Constitution 87 / Rule 26.
const (
	QuotaStatusKnown     = "known"
	QuotaStatusEstimated = "estimated"
	QuotaStatusUnknown   = "unknown"
	QuotaStatusCooldown  = "cooldown"
	QuotaStatusExhausted = "exhausted"
)

// Explainability Reason Codes for model routing.
const (
	ReasonCodeCheapPreferred         = "CHEAP_PREFERRED"
	ReasonCodePrimaryStandard        = "PRIMARY_STANDARD"
	ReasonCodeFallbackDegraded       = "FALLBACK_DEGRADED"
	ReasonCodeQuotaCooldown          = "QUOTA_COOLDOWN"
	ReasonCodeQuotaExhausted         = "QUOTA_EXHAUSTED"
	ReasonCodePostMutationRefusal    = "POST_MUTATION_REFUSAL"
)

// RouteExplanation provides structured insight into routing decisions.
type RouteExplanation struct {
	RouteClass   string   `json:"route_class"`
	ReasonCode   string   `json:"reason_code"`
	Details      string   `json:"details"`
	CandidateIDs []string `json:"candidate_ids"`
}

// SelectByClass applies high-level routing class policy.
func (g *Gateway) SelectByClass(targets []RouteTarget, routeClass string) (ModelRoute, RouteExplanation) {
	var pol Policy
	var reasonCode string
	var details string

	switch routeClass {
	case RouteClassCheap:
		pol = Policy{Prefer: "cheap"}
		reasonCode = ReasonCodeCheapPreferred
		details = "Prioritizing lowest cost per 1k tokens with healthy provider"
	case RouteClassFallback:
		pol = Policy{Prefer: "fast"}
		reasonCode = ReasonCodeFallbackDegraded
		details = "Emergency fast-latency fallback path selected"
	case RouteClassPrimary:
		fallthrough
	default:
		pol = Policy{}
		reasonCode = ReasonCodePrimaryStandard
		details = "Standard primary route selected with healthy-first deterministic order"
	}

	route := g.SelectWithPolicy(targets, pol)
	route.Reason = fmt.Sprintf("%s: %s", reasonCode, details)

	candidates := make([]string, 0, len(targets))
	for _, t := range targets {
		candidates = append(candidates, fmt.Sprintf("%s/%s", t.Provider, t.Model))
	}

	explanation := RouteExplanation{
		RouteClass:   routeClass,
		ReasonCode:   reasonCode,
		Details:      details,
		CandidateIDs: candidates,
	}

	return route, explanation
}

// GetQuotaStatus reports the current quota status classification for a provider.
func (g *Gateway) GetQuotaStatus(provider string) string {
	g.mu.Lock()
	defer g.mu.Unlock()

	q, ok := g.quotas[provider]
	if !ok {
		return QuotaStatusUnknown
	}

	if q.Remaining == 0 {
		if q.ResetsAt == "" {
			return QuotaStatusExhausted
		}
		t, err := time.Parse(time.RFC3339Nano, q.ResetsAt)
		if err != nil {
			t, err = time.Parse(time.RFC3339, q.ResetsAt)
		}
		if err == nil && time.Now().UTC().Before(t) {
			return QuotaStatusCooldown
		}
		return QuotaStatusExhausted
	}

	if q.Remaining > 0 {
		return QuotaStatusKnown
	}

	return QuotaStatusEstimated
}
