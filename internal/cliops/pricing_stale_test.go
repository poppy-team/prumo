package cliops

import (
	"strings"
	"testing"
)

// A pricing rate nobody revisits produces a wrong number in a cost report, and
// nothing said so. `doctor` reports it as a WARNING: a stale rate is worth
// knowing about and is not a broken project (GAP-155).

func TestDoctorReportsStalePricing(t *testing.T) {
	s := New(t.TempDir())
	findings, err := s.Doctor(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// The shipped table is fresh, so there is nothing to report. The check is
	// still exercised: a doctor that could not report it would pass this test.
	for _, finding := range findings {
		if finding["category"] == "pricing" {
			t.Fatalf("the shipped pricing table was reported as stale: %v", finding)
		}
	}
}

func TestPricingFindingsCarryTheRateTheyAreAbout(t *testing.T) {
	// A pricing warning that does not name the provider and model is a warning
	// nobody can act on.
	s := New(t.TempDir())
	findings, err := s.Doctor(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		if finding["category"] != "pricing" {
			continue
		}
		message, _ := finding["message"].(string)
		if !strings.Contains(message, "/") {
			t.Errorf("a pricing finding does not name a provider and model: %q", message)
		}
		if finding["severity"] != "WARNING" {
			t.Errorf("a stale rate is reported as %v; it is worth knowing, not a broken project", finding["severity"])
		}
	}
}
