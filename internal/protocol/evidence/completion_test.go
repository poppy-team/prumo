package evidence

import (
	"testing"
)

func TestIsAgentSelfDeclaration(t *testing.T) {
	cases := []struct {
		text     string
		expected bool
	}{
		{"I am done with the implementation.", true},
		{"Task is complete and ready for use.", true},
		{"All requirements are met now.", true},
		{"Work is complete.", true},
		{"Successfully implemented feature X.", true},
		{"Let us review the compiler tests.", false},
		{"The parser returns an error on line 42.", false},
	}

	for _, c := range cases {
		if got := IsAgentSelfDeclaration(c.text); got != c.expected {
			t.Errorf("IsAgentSelfDeclaration(%q) = %v, want %v", c.text, got, c.expected)
		}
	}
}

func TestEvaluateCompletion_TotalAssurance(t *testing.T) {
	// Rule: implemented != reachable != exercised != evidenced != verified != accepted != released.
	// An implemented-only criterion must NEVER declare done.
	criteria := []AcceptanceCriterion{
		{
			ID:          "AC-1",
			Description: "Must implement parser",
			Required:    true,
			Status:      AcceptanceImplemented, // Implemented only!
		},
	}

	eval := EvaluateCompletion("goal-1", criteria, nil)
	if eval.CanComplete {
		t.Fatalf("expected CanComplete to be false for implemented-only criterion")
	}
	if eval.DerivedState != "incomplete" {
		t.Fatalf("expected DerivedState 'incomplete', got %q", eval.DerivedState)
	}

	// Now add evidence
	rec := NewRecord("EV-001", "test", "ci", "goal-1", "pass")
	criteria[0].Status = AcceptanceEvidenced
	criteria[0].EvidenceIDs = []string{"EV-001"}

	evalEvidenced := EvaluateCompletion("goal-1", criteria, []Record{rec})
	if !evalEvidenced.CanComplete {
		t.Fatalf("expected CanComplete to be true with valid evidence: %s", evalEvidenced.Explanation)
	}
	if evalEvidenced.DerivedState != "verified" {
		t.Fatalf("expected DerivedState 'verified', got %q", evalEvidenced.DerivedState)
	}

	// If evidence is stale, completion is blocked
	recStale := rec
	recStale.Stale = true
	evalStale := EvaluateCompletion("goal-1", criteria, []Record{recStale})
	if evalStale.CanComplete {
		t.Fatalf("expected CanComplete to be false with stale evidence")
	}
	if evalStale.StaleEvidenceCount != 1 {
		t.Fatalf("expected 1 stale evidence count, got %d", evalStale.StaleEvidenceCount)
	}

	// Waived criterion without reason is blocked
	criteriaWaived := []AcceptanceCriterion{
		{
			ID:          "AC-WAIVE",
			Description: "Optional feature",
			Required:    true,
			Status:      AcceptanceWaived,
			WaivedReason: "", // Empty!
		},
	}
	evalWaived := EvaluateCompletion("goal-1", criteriaWaived, nil)
	if evalWaived.CanComplete {
		t.Fatalf("expected CanComplete to be false for unjustified waiver")
	}
}
