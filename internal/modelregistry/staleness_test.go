package modelregistry

import (
	"testing"
	"time"
)

// Pricing tables diverged between providers and carried no version, so a report
// could quote a rate that no provider charged. The table is now versioned and
// dated, but a dated table still goes stale silently: the cost of a stale rate
// is a wrong number in a report, and nothing said so (GAP-155).
//
// The check is a date rather than a network call on purpose. A CI run must be
// deterministic, and a check that reaches a provider's API on every build fails
// for reasons that have nothing to do with this repository.

func TestAFreshTableIsNotStale(t *testing.T) {
	now := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	table := PricingTable{Entries: map[string]PricingEntry{
		"p/m": {Provider: "p", Model: "m", EffectiveFrom: now.Add(-7 * 24 * time.Hour).Format("2006-01-02")},
	}}
	if stale := table.Stale(now); len(stale) != 0 {
		t.Fatalf("a table dated a week ago was reported as stale: %+v", stale)
	}
}

func TestAnAgedTableIsReported(t *testing.T) {
	now := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	table := PricingTable{Entries: map[string]PricingEntry{
		"p/m": {Provider: "p", Model: "m", EffectiveFrom: now.Add(-120 * 24 * time.Hour).Format("2006-01-02")},
	}}
	stale := table.Stale(now)
	if len(stale) != 1 {
		t.Fatalf("a rate 120 days old was not reported: %+v", stale)
	}
	if stale[0].AgeDays < 119 {
		t.Fatalf("the age reported is %d days, which is wrong", stale[0].AgeDays)
	}
}

func TestAnUnreadableDateIsReportedRatherThanIgnored(t *testing.T) {
	// A check that quietly skips what it cannot read is how a bad date becomes a
	// permanent one.
	now := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	table := PricingTable{Entries: map[string]PricingEntry{
		"p/m": {Provider: "p", Model: "m", EffectiveFrom: "not a date"},
	}}
	stale := table.Stale(now)
	if len(stale) != 1 {
		t.Fatalf("an unparseable date was ignored: %+v", stale)
	}
	if stale[0].AgeDays != -1 {
		t.Fatalf("an unparseable date is reported as age %d", stale[0].AgeDays)
	}
}

func TestTheDefaultTableIsNotAlreadyStale(t *testing.T) {
	// The table this repository ships must pass its own check, or the check is
	// noise from the first day.
	now := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	if stale := DefaultPricing().Stale(now); len(stale) != 0 {
		names := make([]string, 0, len(stale))
		for _, entry := range stale {
			names = append(names, entry.Provider+"/"+entry.Model+" ("+entry.EffectiveFrom+")")
		}
		t.Fatalf("the shipped pricing table is already stale: %v", names)
	}
}

func TestStaleOutputIsOrdered(t *testing.T) {
	// A report that quotes an unordered list differs between two runs of the same
	// code, which makes it impossible to diff.
	now := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	old := now.Add(-200 * 24 * time.Hour).Format("2006-01-02")
	table := PricingTable{Entries: map[string]PricingEntry{
		"z/last":  {Provider: "z", Model: "last", EffectiveFrom: old},
		"a/first": {Provider: "a", Model: "first", EffectiveFrom: old},
		"m/mid":   {Provider: "m", Model: "mid", EffectiveFrom: old},
	}}
	first := table.Stale(now)
	for range 20 {
		again := table.Stale(now)
		for i := range first {
			if first[i] != again[i] {
				t.Fatal("the stale report came out in a different order on a second run")
			}
		}
	}
	if first[0].Provider != "a" || first[2].Provider != "z" {
		t.Fatalf("the report is not sorted by provider: %+v", first)
	}
}
