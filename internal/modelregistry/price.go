package modelregistry

import "github.com/raillen/prumo/internal/harness/agent"

// Price applies a pricing table to a usage report, filling in what the provider
// did not send.
//
// Neither the OpenAI-compatible nor the Anthropic endpoint returns a cost, so
// every usage event carried CostUSD zero. A budget's US dollar dimension then
// compared spent against a limit where spent never moved, and --budget-usd was
// inert for exactly the two providers a real run uses (GAP-129).
//
// The result is derived, and it says so: PricingVersion travels with it, so a
// cost on a run record can be reproduced by naming the table that produced it.
// A model the table does not price is left at zero rather than guessed, and
// Pricable reports that so a caller can tell "free" from "unknown".
func Price(table PricingTable, provider, model string, usage *agent.Usage) {
	if usage == nil {
		return
	}
	pricing, err := table.Lookup(provider, model)
	if err != nil {
		return
	}
	usage.CostUSD = pricing.Cost(TokenCounts{
		Input:      usage.InputTokens,
		Output:     usage.OutputTokens,
		CacheRead:  usage.CacheReadTokens,
		CacheWrite: usage.CacheWriteTokens,
	})
}

// Pricable reports whether the table prices this model, so a caller can tell a
// genuinely free model from one nobody has priced yet.
func Pricable(table PricingTable, provider, model string) bool {
	return table.Known(provider, model)
}
