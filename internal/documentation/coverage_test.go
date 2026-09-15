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
	// Every applicable contract lands in exactly one bucket. A waived contract is
	// its own bucket: it is neither bound work nor a blocker, and folding it into
	// the lexical count would overstate what a reviewer still has to do.
	classified := audit.SemanticContracts + audit.LexicalContracts + len(audit.NotApplicableContracts)
	if classified != len(audit.Coverage) {
		t.Fatalf("mode classification mismatch: semantic=%d lexical=%d not-applicable=%d coverage=%d",
			audit.SemanticContracts, audit.LexicalContracts, len(audit.NotApplicableContracts), len(audit.Coverage))
	}
	for _, c := range audit.Coverage {
		if c.Authoritative != (c.Mode == ModeSemantic) {
			t.Fatalf("contract %s: authoritative=%v with mode=%s", c.ContractID, c.Authoritative, c.Mode)
		}
	}
	readiness, err := Readiness(root, "M5")
	if err != nil {
		t.Fatal(err)
	}
	if len(readiness.Coverage) == 0 {
		t.Fatal("expected coverage")
	}
}

// TestLexicalMatchNeverAuthoritative pins W15.5: a word match may be reported,
// but it can never make a contract ready.
func TestLexicalMatchNeverAuthoritative(t *testing.T) {
	contract := Contract{ID: "lexical", RequiredKnowledge: []string{"architecture"}}
	coverage := evaluate("../..", contract, Binding{
		Sources:  []string{"docs/architecture/overview.md"},
		Evidence: []string{"review-result"},
	})
	if coverage.State != Unverified {
		t.Fatalf("expected unverified, got %s", coverage.State)
	}
	if coverage.Authoritative {
		t.Fatal("lexical coverage must not be authoritative")
	}
	if coverage.Mode != ModeLexical {
		t.Fatalf("expected lexical mode, got %s", coverage.Mode)
	}
	report := ReadinessReport{Ready: true}
	applyCoverage(&report, coverage)
	if report.Ready {
		t.Fatal("readiness must not pass on a lexical match")
	}
	if len(report.UnverifiedContracts) != 1 {
		t.Fatalf("expected the contract to be reported unverified, got %v", report.UnverifiedContracts)
	}
}

func semanticContract() Contract {
	return Contract{
		ID:                "semantic",
		RequiredKnowledge: []string{"recovery guarantees"},
	}
}

func semanticBinding(evidence []EvidenceRef, status string) Binding {
	return Binding{
		Sources:        []string{"docs/architecture/overview.md"},
		RequirementMap: map[string]string{"recovery guarantees": "REQ-RECOVERY"},
		Claims: []ClaimBinding{{
			ID:        "CLAIM-RECOVERY",
			Statement: "Crash recovery guarantees are documented per operation.",
			Status:    status,
			Authority: "canonical-documentation",
			Revision:  3,
			Requires:  []string{"REQ-RECOVERY"},
			Evidence:  evidence,
		}},
	}
}

func verifiedEvidence(id string) EvidenceRef {
	return EvidenceRef{
		ID:       id,
		Type:     "test-suite",
		Target:   "CLAIM-RECOVERY",
		Revision: "3",
		Verified: true,
	}
}

func TestSemanticBindingReachesVerified(t *testing.T) {
	coverage := evaluate("../..", semanticContract(), semanticBinding([]EvidenceRef{verifiedEvidence("EV-1")}, "accepted"))
	if coverage.State != Verified {
		t.Fatalf("expected verified, got %s (%v)", coverage.State, coverage.Findings)
	}
	if !coverage.Authoritative || coverage.Mode != ModeSemantic {
		t.Fatalf("semantic coverage must be authoritative: %#v", coverage)
	}
	report := ReadinessReport{Ready: true}
	applyCoverage(&report, coverage)
	if !report.Ready {
		t.Fatal("a semantically verified contract must not block readiness")
	}
}

func TestSemanticReadinessRejectsFalseGreens(t *testing.T) {
	misbound := verifiedEvidence("EV-1")
	misbound.Target = "CLAIM-SOMETHING-ELSE"
	stale := verifiedEvidence("EV-2")
	stale.Revision = "2"
	artifact := verifiedEvidence("EV-3")
	artifact.Artifact = "docs/does-not-exist-sentinel.md"
	unverified := verifiedEvidence("EV-4")
	unverified.Verified = false

	cases := []struct {
		name   string
		bind   Binding
		state  CoverageState
		expect string
	}{
		{
			name:   "evidence bound to another claim",
			bind:   semanticBinding([]EvidenceRef{misbound}, "accepted"),
			state:  Unverified,
			expect: "unverified-claim",
		},
		{
			name:   "evidence proves an older revision",
			bind:   semanticBinding([]EvidenceRef{stale}, "accepted"),
			state:  Unverified,
			expect: "stale-evidence",
		},
		{
			name:   "evidence artifact is missing",
			bind:   semanticBinding([]EvidenceRef{artifact}, "accepted"),
			state:  Unverified,
			expect: "missing-artifact",
		},
		{
			name:   "evidence is not verified",
			bind:   semanticBinding([]EvidenceRef{unverified}, "accepted"),
			state:  Unverified,
			expect: "unverified-claim",
		},
		{
			name:   "claim is not accepted",
			bind:   semanticBinding([]EvidenceRef{verifiedEvidence("EV-5")}, "proposed"),
			state:  Unverified,
			expect: "claim-not-accepted",
		},
		{
			name: "no claim satisfies the requirement",
			bind: Binding{
				Sources:        []string{"docs/architecture/overview.md"},
				RequirementMap: map[string]string{"recovery guarantees": "REQ-RECOVERY"},
			},
			state:  Partial,
			expect: "missing-claim",
		},
		{
			name: "required knowledge is not mapped",
			bind: Binding{
				Sources: []string{"docs/architecture/overview.md"},
				Claims:  []ClaimBinding{{ID: "CLAIM-1", Status: "accepted", Requires: []string{"REQ-OTHER"}}},
			},
			state:  Partial,
			expect: "unmapped-requirement",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			coverage := evaluate("../..", semanticContract(), tc.bind)
			if coverage.State != tc.state {
				t.Fatalf("state = %s, want %s (%v)", coverage.State, tc.state, coverage.Findings)
			}
			if !hasFinding(coverage.Findings, tc.expect) {
				t.Fatalf("expected finding %q in %v", tc.expect, coverage.Findings)
			}
			report := ReadinessReport{Ready: true}
			applyCoverage(&report, coverage)
			if report.Ready {
				t.Fatal("readiness must not pass with unresolved semantic findings")
			}
		})
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

// TestEvidenceDistinctAndNonEmpty pins that duplicated evidence never
// double-counts and that evidence without a stable id can never prove a claim.
func TestEvidenceDistinctAndNonEmpty(t *testing.T) {
	duplicate := evaluate("../..", semanticContract(), semanticBinding([]EvidenceRef{
		verifiedEvidence("EV-DUP"), verifiedEvidence("EV-DUP"),
	}, "accepted"))
	if duplicate.State != Verified {
		t.Fatalf("expected verified with one distinct evidence, got %s (%v)", duplicate.State, duplicate.Findings)
	}
	if len(duplicate.Requirements) != 1 || duplicate.Requirements[0].EvidenceCount != 1 {
		t.Fatalf("duplicated evidence must count once, got %#v", duplicate.Requirements)
	}

	blank := verifiedEvidence("   ")
	blankCoverage := evaluate("../..", semanticContract(), semanticBinding([]EvidenceRef{blank}, "accepted"))
	if blankCoverage.State == Verified {
		t.Fatalf("evidence without an id must not verify a claim (%v)", blankCoverage.Findings)
	}
	if !hasFinding(blankCoverage.Findings, "invalid-evidence") {
		t.Fatalf("expected invalid-evidence finding, got %v", blankCoverage.Findings)
	}

	distinct := evaluate("../..", semanticContract(), semanticBinding([]EvidenceRef{
		verifiedEvidence("EV-1"), verifiedEvidence("EV-2"),
	}, "accepted"))
	if distinct.State != Verified {
		t.Fatalf("expected verified with distinct evidence, got %s (%v)", distinct.State, distinct.Findings)
	}
	if distinct.Requirements[0].EvidenceCount != 2 {
		t.Fatalf("expected two counted evidence items, got %#v", distinct.Requirements)
	}
}

// TestSemanticReadinessDogfood runs readiness against Prumo itself (W15.11). It
// asserts the mechanism, never a fixed verdict, so a green result cannot be
// obtained by inventing claims — only by binding real ones.
func TestSemanticReadinessDogfood(t *testing.T) {
	root, _ := filepath.Abs("../..")
	audit, err := Audit(root)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Readiness(root, "W15")
	if err != nil {
		t.Fatal(err)
	}
	// A lexical-only contract must be surfaced, not absorbed. Checking
	// UnverifiedContracts alone is not enough: a contract with no binding at all
	// is reported as `missing` and reaches the operator through
	// BlockingContracts, so a project could declare a capability, fail every
	// gate it brings, and still see an empty unverified list.
	surfaced := map[string]bool{}
	for _, id := range append(append([]string{}, report.UnverifiedContracts...), report.BlockingContracts...) {
		surfaced[id] = true
	}
	if len(surfaced) < audit.LexicalContracts {
		t.Fatalf("expected every lexical contract to be reported, got %d lexical / %d surfaced",
			audit.LexicalContracts, len(surfaced))
	}
	for _, c := range audit.Coverage {
		if !c.Authoritative && c.State == Verified {
			t.Fatalf("contract %s reached verified without a semantic binding", c.ContractID)
		}
		if c.State == Unverified && len(c.Findings) == 0 && len(c.Warnings) == 0 {
			t.Fatalf("contract %s is unverified with no explanation", c.ContractID)
		}
	}
	t.Logf("semantic=%d lexical=%d unverified=%v blocking=%v",
		audit.SemanticContracts, audit.LexicalContracts, report.UnverifiedContracts, report.BlockingContracts)
}

func hasFinding(findings []SemanticFinding, kind string) bool {
	for _, f := range findings {
		if f.Kind == kind {
			return true
		}
	}
	return false
}
