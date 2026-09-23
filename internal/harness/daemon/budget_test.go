package daemon

import "testing"

// The daemon built every tracker with NewTracker(0, 0, 0), and a zero limit is
// unlimited in the envelope. Every run through the daemon therefore had no
// ceiling on tokens, cost or tool calls until the provider declined (GAP-098).

func TestDefaultRunBudgetIsARealCeiling(t *testing.T) {
	b := DefaultRunBudget()
	if b.Tokens <= 0 || b.CostUSD <= 0 || b.ToolCalls <= 0 {
		t.Fatalf("the default budget must bound every dimension, got %+v", b)
	}
}

func TestAnUnsetBudgetFallsBackToTheDefaultRatherThanUnlimited(t *testing.T) {
	// This is the exact shape of the old bug: a zero Budget must not mean "no
	// limit", or an unconfigured daemon reproduces it.
	tokens, usd, tools := Budget{}.limits()
	def := DefaultRunBudget()
	if tokens != def.Tokens || usd != def.CostUSD || tools != def.ToolCalls {
		t.Fatalf("an unset budget must fall back to the default, got %v/%v/%v (default %v/%v/%v)",
			tokens, usd, tools, def.Tokens, def.CostUSD, def.ToolCalls)
	}
	if tokens <= 0 || usd <= 0 || tools <= 0 {
		t.Fatalf("the fallback must bound something, got %v/%v/%v", tokens, usd, tools)
	}
}

func TestAPartialBudgetKeepsItsOwnCeiling(t *testing.T) {
	// A deployment that wants a tighter token ceiling must get it, and the
	// dimensions it did not name must still be bounded rather than opened.
	tokens, usd, tools := Budget{Tokens: 500}.limits()
	if tokens != 500 {
		t.Fatalf("an explicit token ceiling must be honoured, got %v", tokens)
	}
	def := DefaultRunBudget()
	if usd != def.CostUSD || tools != def.ToolCalls {
		t.Fatalf("the unspecified dimensions must still be bounded, got %v/%v", usd, tools)
	}
}

func TestANegativeBudgetIsNotAnEscapeHatchToUnlimited(t *testing.T) {
	// A negative number is a mistake, not a request for no limit.
	tokens, usd, tools := Budget{Tokens: -1, CostUSD: -1, ToolCalls: -1}.limits()
	if tokens <= 0 || usd <= 0 || tools <= 0 {
		t.Fatalf("a negative budget must fall back, got %v/%v/%v", tokens, usd, tools)
	}
}
