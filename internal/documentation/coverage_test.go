package docengine

import (
	"path/filepath"
	"testing"
)

func TestPrumoAuditAndReadiness(t *testing.T) {
	root, _ := filepath.Abs("../..")
	audit, err := Audit(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(audit.ApplicableContracts) == 0 {
		t.Fatal("expected applicable contracts")
	}
	readiness, err := Readiness(root, "M5")
	if err != nil {
		t.Fatal(err)
	}
	if len(readiness.Coverage) == 0 {
		t.Fatal("expected coverage")
	}
}

func TestVerifiedCoverageRequiresEvidence(t *testing.T) {
	contract := Contract{ID: "verified", RequiredKnowledge: []string{"architecture"}, EvidenceRequirements: []string{"review"}}
	withoutEvidence := evaluate("../..", contract, Binding{Sources: []string{"docs/architecture/overview.md"}})
	if withoutEvidence.State != ImplementationReady {
		t.Fatalf("unexpected state: %s", withoutEvidence.State)
	}
	withEvidence := evaluate("../..", contract, Binding{Sources: []string{"docs/architecture/overview.md"}, Evidence: []string{"review-result"}})
	if withEvidence.State != Verified {
		t.Fatalf("expected verified, got %s", withEvidence.State)
	}
}

func TestFalseGreenKnowledgeMatchingPrevented(t *testing.T) {
	// 1. A document containing only the stopword "and" must NOT satisfy "problem and users"
	contentWithStopwordOnly := "this is a document and nothing else"
	if matchesKnowledgeRequirement(contentWithStopwordOnly, "problem and users") {
		t.Fatal("expected 'problem and users' to fail when only stopword 'and' is present")
	}

	// 2. A document containing only one content word ("permission") must NOT satisfy "permission model and recovery guarantees"
	contentWithSingleWord := "this document mentions permission only"
	if matchesKnowledgeRequirement(contentWithSingleWord, "permission model and recovery guarantees") {
		t.Fatal("expected multi-word requirement to fail when key terms ('model', 'recovery', 'guarantees') are missing")
	}

	// 3. A document containing all key words should pass
	contentWithAllKeyWords := "the permission model provides crash recovery guarantees"
	if !matchesKnowledgeRequirement(contentWithAllKeyWords, "permission model and recovery guarantees") {
		t.Fatal("expected requirement to pass when all key terms are present")
	}
}

func TestEvidenceDistinctAndNonEmpty(t *testing.T) {
	contract := Contract{
		ID:                   "multi-evidence",
		RequiredKnowledge:    []string{"architecture"},
		EvidenceRequirements: []string{"test-suite", "security-audit"},
	}

	// Duplicate evidence entries should not satisfy 2 distinct requirements
	duplicateEvidence := evaluate("../..", contract, Binding{
		Sources:  []string{"docs/architecture/overview.md"},
		Evidence: []string{"test-suite", "test-suite"},
	})
	if duplicateEvidence.State == Verified {
		t.Fatal("duplicate evidence entries must not satisfy multi-evidence requirement")
	}

	// Empty strings must not satisfy requirements
	emptyEvidence := evaluate("../..", contract, Binding{
		Sources:  []string{"docs/architecture/overview.md"},
		Evidence: []string{"test-suite", "   "},
	})
	if emptyEvidence.State == Verified {
		t.Fatal("empty whitespace evidence must not satisfy evidence requirement")
	}

	// Distinct valid evidence entries should satisfy
	validEvidence := evaluate("../..", contract, Binding{
		Sources:  []string{"docs/architecture/overview.md"},
		Evidence: []string{"test-suite", "audit-report"},
	})
	if validEvidence.State != Verified {
		t.Fatalf("expected verified, got %s", validEvidence.State)
	}
}
