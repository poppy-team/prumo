package modelregistry

import (
	"strings"
	"testing"
)

// The pricing table is the thing a cost is explained by later: "why did this run
// cost what it cost" has to resolve to a version, a date and a source, not to a
// number somebody typed into a struct.

func almost(got, want float64) bool {
	diff := got - want
	return diff < 1e-12 && diff > -1e-12
}

func TestTheBuiltInTableCanExplainItself(t *testing.T) {
	table := DefaultPricing()
	if err := table.Validate(); err != nil {
		t.Fatalf("the table we ship must validate: %v", err)
	}
	if table.Version != PricingVersion {
		t.Fatalf("version = %q, want %q", table.Version, PricingVersion)
	}
	for key, entry := range table.Entries {
		if strings.TrimSpace(entry.Source) == "" {
			t.Errorf("entry %q has no source; a number nobody can check is not a price", key)
		}
		if entry.EffectiveFrom == "" {
			t.Errorf("entry %q has no effective date", key)
		}
	}
}

func TestCostIsDollarsPerMillionTokens(t *testing.T) {
	// The rates are published per million, and the arithmetic has to match the
	// published unit or every cost is wrong by a factor of a thousand.
	p := Pricing{USDPerMillionInput: 2.50, USDPerMillionOutput: 10.00}
	if got, want := p.Cost(TokenCounts{Input: 1_000_000}), 2.50; !almost(got, want) {
		t.Errorf("one million input at $2.50/M = %g, want %g", got, want)
	}
	if got, want := p.Cost(TokenCounts{Output: 1_000_000}), 10.00; !almost(got, want) {
		t.Errorf("one million output at $10/M = %g, want %g", got, want)
	}
}

func TestAllFourTokenDimensionsAreCharged(t *testing.T) {
	p := Pricing{
		USDPerMillionInput:      2.00,
		USDPerMillionOutput:     8.00,
		USDPerMillionCacheWrite: 2.50,
		USDPerMillionCacheRead:  0.20,
	}
	got := p.Cost(TokenCounts{
		Input: 1_000_000, Output: 1_000_000,
		CacheWrite: 1_000_000, CacheRead: 1_000_000,
	})
	if want := 12.70; !almost(got, want) {
		t.Fatalf("all four dimensions = %g, want %g", got, want)
	}
}

func TestACacheReadIsFreeWhenTheProviderSaysItIs(t *testing.T) {
	// Not every provider discounts a cache read. One that does not bills it at
	// the input rate; charging 0.1x because that looked plausible understates it.
	p := Pricing{USDPerMillionInput: 2.00, USDPerMillionCacheRead: 0.20, CacheReadIsFree: true}
	if got, want := p.Cost(TokenCounts{CacheRead: 1_000_000}), 2.00; !almost(got, want) {
		t.Fatalf("an undiscounted cache read = %g, want the input rate %g", got, want)
	}
}

func TestACacheReadNotSeparatelyBilledIsNotChargedAsADiscount(t *testing.T) {
	p := Pricing{
		USDPerMillionInput:     2.00,
		USDPerMillionCacheRead: CacheReadNotSeparatelyBilled,
	}
	if got := p.Cost(TokenCounts{CacheRead: 1_000_000}); !almost(got, 0) {
		t.Fatalf("a sentinel rate must not be multiplied into a cost, got %g", got)
	}
}

func TestAnUnpricedModelIsAnErrorNotAZero(t *testing.T) {
	// Zero means "this model is free". "Nobody has priced this model" is a
	// different answer, and returning zero for it is how a US dollar budget
	// becomes decorative without anyone noticing.
	table := DefaultPricing()
	if _, err := table.Lookup("openai", "a-model-nobody-priced"); err == nil {
		t.Fatal("an unpriced model must be an error")
	}
	if table.Known("openai", "a-model-nobody-priced") {
		t.Fatal("the table must not claim to price a model it does not")
	}
}

func TestALookupIsCaseAndSpaceInsensitive(t *testing.T) {
	// Model names arrive from config files, CLI flags and provider responses,
	// spelled inconsistently. A miss on " GPT-4O " would report no cost for a
	// model the table plainly prices.
	table := DefaultPricing()
	if _, err := table.Lookup("OpenAI", " GPT-4o "); err != nil {
		t.Fatalf("lookup must normalise: %v", err)
	}
}

func TestTheSameModelNameAtTwoProvidersIsTwoPrices(t *testing.T) {
	// The provider is part of the key: one vendor can serve a model name another
	// sells at a different rate, and keying on the model alone would silently
	// charge one vendor's price for the other's usage.
	table := DefaultPricing()
	other := table.Entries["openai/gpt-4o"]
	other.Provider = "other-vendor"
	other.USDPerMillionInput = 99
	table.Entries[pricingKey("other-vendor", "gpt-4o")] = other

	openai, err := table.Lookup("openai", "gpt-4o")
	if err != nil {
		t.Fatal(err)
	}
	vendor, err := table.Lookup("other-vendor", "gpt-4o")
	if err != nil {
		t.Fatal(err)
	}
	if almost(openai.USDPerMillionInput, vendor.USDPerMillionInput) {
		t.Fatal("two providers must not collapse onto one price")
	}
}

func TestValidateRejectsATableThatCouldNotBeExplainedLater(t *testing.T) {
	cases := []struct {
		name  string
		table PricingTable
		want  string
	}{
		{"no version", PricingTable{Entries: map[string]PricingEntry{"a/b": {Provider: "a", Model: "b"}}}, "no version"},
		{"no entries", PricingTable{Version: "v"}, "no entries"},
		{"no source", PricingTable{Version: "v", Entries: map[string]PricingEntry{
			"a/b": {Provider: "a", Model: "b", EffectiveFrom: "2026-09-01"},
		}}, "no source"},
		{"bad date", PricingTable{Version: "v", Entries: map[string]PricingEntry{
			"a/b": {Provider: "a", Model: "b", EffectiveFrom: "last tuesday", Source: "s"},
		}}, "effective date"},
		{"negative input", PricingTable{Version: "v", Entries: map[string]PricingEntry{
			"a/b": {Provider: "a", Model: "b", EffectiveFrom: "2026-09-01", Source: "s",
				Pricing: Pricing{USDPerMillionInput: -2}},
		}}, "negative input"},
		{"no model", PricingTable{Version: "v", Entries: map[string]PricingEntry{
			"a/": {Provider: "a", EffectiveFrom: "2026-09-01", Source: "s"},
		}}, "no provider or model"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.table.Validate()
			if err == nil {
				t.Fatalf("expected %s to be rejected", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}

func TestModelsListsWhatIsActuallyPriced(t *testing.T) {
	table := DefaultPricing()
	got := table.Models("openai")
	if len(got) != 2 {
		t.Fatalf("openai has two priced models, got %v", got)
	}
	for _, m := range got {
		if !strings.HasPrefix(m, "openai/") {
			t.Errorf("Models(provider) returned %q, which is not that provider", m)
		}
	}
	if all := table.Models(""); len(all) != 4 {
		t.Errorf("the table prices four models, got %v", all)
	}
}
