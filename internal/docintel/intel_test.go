package docintel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectRepositoryMetrics(t *testing.T) {
	root, _ := filepath.Abs("../..")
	metrics, err := Collect(root)
	if err != nil {
		t.Fatal(err)
	}
	if metrics.Pages == 0 {
		t.Fatalf("expected pages in the graph: %#v", metrics)
	}
	if metrics.HistoricalPages == 0 {
		t.Fatal("expected historical pages to be counted separately")
	}
	if metrics.AgentSurfaces == 0 || metrics.InstructionTokens == 0 {
		t.Fatalf("expected agent surfaces in the metrics: %#v", metrics)
	}
	if metrics.LexicalContracts+metrics.SemanticContracts == 0 {
		t.Fatalf("lexical/semantic split must reflect the repository: %#v", metrics)
	}
	if metrics.SemanticContracts == 0 {
		t.Fatal("semantic bindings must be counted once the repository carries them")
	}
	if metrics.FreshnessScore <= 0 || metrics.FreshnessScore > 1 {
		t.Fatalf("expected a bounded freshness score in (0,1], got %v", metrics.FreshnessScore)
	}
	// The repository may or may not be debt-free; what must always hold is that
	// unverified contracts are named as debt and nothing else is invented.
	if len(metrics.UnverifiedContracts) > 0 && len(metrics.DocumentationDebt) == 0 {
		t.Fatalf("unverified contracts must be visible as documentation debt: %#v", metrics)
	}
}

// TestDebtAndScoreFollowVerificationState exercises the debt path hermetically,
// so it stays covered even while the repository itself is debt-free (W21.1,
// W21.12).
func TestDebtAndScoreFollowVerificationState(t *testing.T) {
	verified := Metrics{Pages: 10, SemanticContracts: 3}
	if debt := documentationDebt(verified); len(debt) != 0 {
		t.Fatalf("a verified metric set must carry no debt: %#v", debt)
	}
	if score := freshnessScore(verified); score != 1 {
		t.Fatalf("a verified metric set must score 1, got %v", score)
	}
	unverified := Metrics{Pages: 10, SemanticContracts: 3, UnverifiedContracts: []string{"testing.strategy"}}
	debt := documentationDebt(unverified)
	if len(debt) != 1 {
		t.Fatalf("expected one named debt item, got %#v", debt)
	}
	if score := freshnessScore(unverified); score <= 0 || score >= 1 {
		t.Fatalf("unverified contracts must lower a bounded score, got %v", score)
	}
}

func TestMetricsComputeWithoutDerivedIndex(t *testing.T) {
	// The search index is derived: metrics must not depend on it existing.
	root := t.TempDir()
	if entries, err := retrievalEntries(root); err != nil || entries != 0 {
		t.Fatalf("a missing derived index must report 0, got %d (%v)", entries, err)
	}
}

func TestFreshnessScoreIsOneWithoutDebt(t *testing.T) {
	clean := Metrics{Pages: 10}
	if score := freshnessScore(clean); score != 1 {
		t.Fatalf("expected a clean score of 1, got %v", score)
	}
	debt := Metrics{Pages: 10, BrokenLinks: 2}
	if score := freshnessScore(debt); score >= 1 {
		t.Fatalf("findings must lower the score, got %v", score)
	}
}

func TestInspectProposesBindingsWithoutApplyingThem(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"README.md":             "# Project\n",
		"docs/vision.md":        "# Vision\n",
		"docs/security.md":      "# Security\n",
		"docs/architecture.md":  "# Architecture\n",
		"docs/user/usage.md":    "# Usage\n",
		"docs/archive/usage.md": "# Old usage\n",
	}
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	report, err := Inspect(root)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Signals["readme"] || !report.Signals["docs-directory"] {
		t.Fatalf("expected discovery signals: %#v", report.Signals)
	}
	if len(report.Proposals) != 3 {
		t.Fatalf("expected three inferred bindings, got %#v", report.Proposals)
	}
	for _, proposal := range report.Proposals {
		if proposal.Confidence != "inferred" || proposal.Promotion == "" {
			t.Fatalf("proposals must state their confidence and promotion path: %#v", proposal)
		}
	}
	if len(report.Duplicates) != 1 {
		t.Fatalf("expected the duplicated usage document, got %#v", report.Duplicates)
	}
	// Adoption never writes into the target: nothing was created or modified.
	if _, err := os.Stat(filepath.Join(root, "docs", "contracts")); err == nil {
		t.Fatal("adoption inspection must not create canonical artifacts")
	}
}

func TestProposeRequiresReview(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# P\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := Propose(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan["note"] == "" {
		t.Fatal("the adoption plan must state that promotion is a human decision")
	}
	if _, ok := plan["promotion_required"]; !ok {
		t.Fatal("the plan must state whether promotion is required")
	}
	if _, err := Inspect(filepath.Join(root, "does-not-exist")); err == nil {
		t.Fatal("inspection of a missing target must fail")
	}
}
