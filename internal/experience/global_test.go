package experience

import (
	"testing"
)

func TestGlobalLearningRegistry_AggregationAndScope(t *testing.T) {
	reg := NewGlobalLearningRegistry()

	// 1. Observation from Project A
	pat1 := reg.RecordObservation("proj-alpha", PatternPhilosophy, "Clean Code Pragmático", "Explicit boundaries and composition", 0.70)
	if pat1.Scope != ScopeProject {
		t.Fatalf("expected ScopeProject for single project, got %s", pat1.Scope)
	}
	if pat1.ProjectCount != 1 {
		t.Fatalf("expected 1 project, got %d", pat1.ProjectCount)
	}

	// 2. Same observation from Project B -> should aggregate to ScopeGlobal
	pat2 := reg.RecordObservation("proj-beta", PatternPhilosophy, "Clean Code Pragmático", "Explicit boundaries and composition", 0.70)
	if pat2.Scope != ScopeGlobal {
		t.Fatalf("expected ScopeGlobal after 2 projects, got %s", pat2.Scope)
	}
	if pat2.ProjectCount != 2 {
		t.Fatalf("expected 2 projects, got %d", pat2.ProjectCount)
	}
	if pat2.Confidence <= 0.70 {
		t.Fatalf("expected reinforced confidence > 0.70, got %f", pat2.Confidence)
	}

	// 3. Review and promotion
	err := reg.ReviewCandidate(pat2.ID, ReviewPromoted, "policy:clean-code-pragmatico")
	if err != nil {
		t.Fatalf("review candidate failed: %v", err)
	}
	if pat2.ReviewStatus != ReviewPromoted {
		t.Fatalf("expected ReviewPromoted, got %s", pat2.ReviewStatus)
	}
	if pat2.PromotionTarget != "policy:clean-code-pragmatico" {
		t.Fatalf("expected promotion target, got %s", pat2.PromotionTarget)
	}

	// 4. Filtering
	promoted := reg.ListPatterns(ScopeGlobal, ReviewPromoted)
	if len(promoted) != 1 {
		t.Fatalf("expected 1 promoted global pattern, got %d", len(promoted))
	}
}
