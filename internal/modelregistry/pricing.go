package modelregistry

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Pricing lives in one place, with units, currency, a date and a source.
//
// Three representations existed: float costs on a model descriptor with no unit,
// an untyped map on the same descriptor, and a rate table in the budget package
// denominated in cents of a currency that defaulted to BRL while the descriptor
// fields were plainly USD. Nothing carried an effective date or a source, so two
// of them could disagree and nothing could say which was right (GAP-133, GAP-155).
//
// Rates here are **US dollars per million tokens**, which is how providers
// publish them. Per-1k and per-1m conversions happen at the edge, in one place,
// instead of at each call site with a different divisor.

// PricingVersion identifies the table. It is part of every cost a run reports, so
// a cost can be explained later by naming the table that produced it.
const PricingVersion = "2026-09"

// Pricing is one model's rates. A zero rate means the dimension is free or not
// billed; it is not "unknown", which is a different answer and is represented by
// the table not having an entry.
type Pricing struct {
	// USDPerMillionInput is the rate for uncached input tokens.
	USDPerMillionInput float64 `json:"usd_per_million_input"`
	// USDPerMillionOutput is the rate for output tokens, including reasoning
	// tokens, which providers bill as output.
	USDPerMillionOutput float64 `json:"usd_per_million_output"`
	// USDPerMillionCacheWrite is the rate for writing a prompt prefix to a
	// provider's cache.
	USDPerMillionCacheWrite float64 `json:"usd_per_million_cache_write,omitempty"`
	// USDPerMillionCacheRead is the rate for reading a cached prefix, which is
	// normally a fraction of the input rate. A negative value means "not
	// separately billed", which is not the same as zero-billed: some providers
	// do not discount cache reads, and saying so beats pricing them at 0.1x
	// because that looked plausible.
	USDPerMillionCacheRead float64 `json:"usd_per_million_cache_read,omitempty"`
	// CacheReadIsFree records that this provider bills cache reads at the input
	// rate rather than a discounted one.
	CacheReadIsFree bool `json:"cache_read_is_free,omitempty"`
}

// PricingEntry is one model's pricing with its provenance.
type PricingEntry struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Pricing
	// EffectiveFrom is when these rates start applying. A rate with no date
	// cannot be reasoned about when a bill disagrees with it.
	EffectiveFrom string `json:"effective_from"`
	// Source says where the numbers came from, so an operator can check them
	// rather than trust them.
	Source string `json:"source"`
}

// PricingTable is a versioned set of rates.
type PricingTable struct {
	Version string                  `json:"version"`
	Entries map[string]PricingEntry `json:"entries"`
}

// pricingKey identifies an entry. A provider may serve the same model name at
// different rates, so the provider is part of the key.
func pricingKey(provider, model string) string {
	return strings.ToLower(strings.TrimSpace(provider)) + "/" + strings.ToLower(strings.TrimSpace(model))
}

// Lookup finds the rates for a provider and model.
//
// A model with no entry returns an error rather than zero cost: a run that
// cannot be priced must say so, because a silent zero is how a budget in US
// dollars ends up decorative without anyone noticing.
func (t PricingTable) Lookup(provider, model string) (Pricing, error) {
	if t.Entries == nil {
		return Pricing{}, fmt.Errorf("pricing table %q has no entries", t.Version)
	}
	entry, ok := t.Entries[pricingKey(provider, model)]
	if !ok {
		return Pricing{}, fmt.Errorf("no pricing for %s/%s in table %s", provider, model, t.Version)
	}
	return entry.Pricing, nil
}

// Known reports whether the table prices this model.
func (t PricingTable) Known(provider, model string) bool {
	_, err := t.Lookup(provider, model)
	return err == nil
}

// Models lists the priced models, sorted, for a provider or for all of them.
func (t PricingTable) Models(provider string) []string {
	out := make([]string, 0, len(t.Entries))
	for _, entry := range t.Entries {
		if provider == "" || strings.EqualFold(entry.Provider, provider) {
			out = append(out, entry.Provider+"/"+entry.Model)
		}
	}
	sort.Strings(out)
	return out
}

// Validate rejects a table that could not be explained later.
func (t PricingTable) Validate() error {
	if strings.TrimSpace(t.Version) == "" {
		return fmt.Errorf("pricing table has no version")
	}
	if len(t.Entries) == 0 {
		return fmt.Errorf("pricing table %q has no entries", t.Version)
	}
	for key, entry := range t.Entries {
		if entry.Provider == "" || entry.Model == "" {
			return fmt.Errorf("pricing entry %q names no provider or model", key)
		}
		if _, err := time.Parse("2006-01-02", entry.EffectiveFrom); err != nil {
			return fmt.Errorf("pricing entry %q has an unparseable effective date %q", key, entry.EffectiveFrom)
		}
		if strings.TrimSpace(entry.Source) == "" {
			return fmt.Errorf("pricing entry %q has no source; a number nobody can check is not a price", key)
		}
		for name, rate := range map[string]float64{
			"input":       entry.USDPerMillionInput,
			"output":      entry.USDPerMillionOutput,
			"cache write": entry.USDPerMillionCacheWrite,
			"cache read":  entry.USDPerMillionCacheRead,
		} {
			if rate < 0 && rate != CacheReadNotSeparatelyBilled {
				return fmt.Errorf("pricing entry %q has a negative %s rate %g", key, name, rate)
			}
		}
	}
	return nil
}

// CacheReadNotSeparatelyBilled marks a provider that does not discount cache
// reads. It is a named sentinel rather than a negative literal at each call site,
// because a negative rate is otherwise a bug waiting to happen.
const CacheReadNotSeparatelyBilled = -1

// TokenCounts is what a cost is computed from. It mirrors the provider's usage
// report without importing the harness, so this package stays a leaf.
type TokenCounts struct {
	Input      int
	Output     int
	CacheRead  int
	CacheWrite int
}

// Cost returns the US dollar cost of these tokens at these rates.
//
// The arithmetic is done in float64 from integer token counts. A cent is a
// hundred-millionth of a dollar at current rates, so float64 is exact well past
// the precision any bill is stated at; integer micro-dollars would add a
// rounding decision at every step for no accuracy anyone can observe.
func (p Pricing) Cost(tokens TokenCounts) float64 {
	const perMillion = 1_000_000.0
	cost := float64(tokens.Input) * p.USDPerMillionInput / perMillion
	cost += float64(tokens.Output) * p.USDPerMillionOutput / perMillion
	cost += float64(tokens.CacheWrite) * p.USDPerMillionCacheWrite / perMillion
	switch {
	case tokens.CacheRead == 0:
	case p.CacheReadIsFree:
		cost += float64(tokens.CacheRead) * p.USDPerMillionInput / perMillion
	case p.USDPerMillionCacheRead >= 0:
		cost += float64(tokens.CacheRead) * p.USDPerMillionCacheRead / perMillion
	}
	return cost
}

// DefaultPricing is the built-in table.
//
// The rates are the published list prices at the version date and are recorded
// so a cost can be reproduced later. They are a baseline, not an oracle: a
// deployment on negotiated or discounted rates replaces this table rather than
// editing the numbers here, so the version still says what was applied.
func DefaultPricing() PricingTable {
	entry := func(provider, model string, in, out, cacheWrite, cacheRead float64, free bool) PricingEntry {
		return PricingEntry{
			Provider: provider, Model: model,
			Pricing: Pricing{
				USDPerMillionInput: in, USDPerMillionOutput: out,
				USDPerMillionCacheWrite: cacheWrite, USDPerMillionCacheRead: cacheRead,
				CacheReadIsFree: free,
			},
			EffectiveFrom: "2026-09-01",
			Source:        "provider published list price",
		}
	}
	return PricingTable{
		Version: PricingVersion,
		Entries: map[string]PricingEntry{
			"openai/gpt-4o":               entry("openai", "gpt-4o", 2.50, 10.00, 0, 1.25, false),
			"openai/gpt-4o-mini":          entry("openai", "gpt-4o-mini", 0.15, 0.60, 0, 0.075, false),
			"anthropic/claude-3-5-sonnet": entry("anthropic", "claude-3-5-sonnet", 3.00, 15.00, 3.75, 0.30, false),
			"anthropic/claude-3-5-haiku":  entry("anthropic", "claude-3-5-haiku", 0.80, 4.00, 1.00, 0.08, false),
		},
	}
}

// StaleAfter is how long a pricing entry is trusted before it is called out.
//
// It is a date, not a network call, for two reasons: a CI run must be
// deterministic, and a check that reaches a provider's API every time is a check
// that fails for reasons unrelated to this repository. The cost of a stale rate
// is a wrong number in a report, which is worth detecting and not worth breaking
// a build over (GAP-155).
const StaleAfter = 90 * 24 * time.Hour

// StaleEntry is one pricing entry that has aged past what is trusted.
type StaleEntry struct {
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	EffectiveFrom string `json:"effective_from"`
	AgeDays       int    `json:"age_days"`
}

// Stale reports the entries whose effective date has aged past StaleAfter, as of
// now.
//
// An entry with an unparseable date is reported rather than skipped: Validate
// catches it, and a check that quietly ignores what it cannot read is how a bad
// date becomes a permanent one.
func (t PricingTable) Stale(now time.Time) []StaleEntry {
	out := []StaleEntry{}
	for _, key := range t.Keys() {
		entry, ok := t.Entries[key]
		if !ok {
			continue
		}
		effective, err := time.Parse("2006-01-02", entry.EffectiveFrom)
		if err != nil {
			out = append(out, StaleEntry{
				Provider: entry.Provider, Model: entry.Model,
				EffectiveFrom: entry.EffectiveFrom, AgeDays: -1,
			})
			continue
		}
		age := now.Sub(effective)
		if age > StaleAfter {
			out = append(out, StaleEntry{
				Provider: entry.Provider, Model: entry.Model,
				EffectiveFrom: entry.EffectiveFrom, AgeDays: int(age.Hours() / 24),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Provider != out[j].Provider {
			return out[i].Provider < out[j].Provider
		}
		return out[i].Model < out[j].Model
	})
	return out
}

// Keys lists every priced key, sorted. A caller that iterates the table directly
// gets a different order every run, which makes a report that quotes it
// unreproducible.
func (t PricingTable) Keys() []string {
	keys := make([]string, 0, len(t.Entries))
	for key := range t.Entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
