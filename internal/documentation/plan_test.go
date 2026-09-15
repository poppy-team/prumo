package docengine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildPlanRecordsObligationsAndReasons(t *testing.T) {
	root, _ := filepath.Abs("../..")
	plan, err := BuildPlan(root, "W17", []string{"schemas/goal.schema.json", "docs/agents/instruction-ir.json"}, "2026-09-15T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Goal != "W17" || len(plan.Obligations) == 0 {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	kinds := map[string]bool{}
	for _, o := range plan.Obligations {
		kinds[o.Kind] = true
		if o.Applicable && o.Reason == "" {
			t.Fatalf("applicable obligation without reason: %#v", o)
		}
	}
	for _, want := range []string{ObligationSchema, ObligationAgentSurface} {
		if !kinds[want] {
			t.Fatalf("expected a %s obligation in %#v", want, plan.Obligations)
		}
	}
	if len(plan.Findings) != 0 {
		t.Fatalf("plan must validate cleanly: %#v", plan.Findings)
	}
}

func TestPlanRequiresExplicitNotApplicableReason(t *testing.T) {
	plan := DocumentationPlan{Goal: "G1", Obligations: []PlanObligation{
		{ID: "a", Kind: ObligationTranslation, Applicable: false},
	}}
	findings := plan.Validate()
	if !hasSemanticFinding(findings, "plan-silent-not-applicable") {
		t.Fatalf("silent N/A must be a finding: %#v", findings)
	}
	plan.Obligations[0].NotApplicableReason = "the project has no localized surfaces"
	findings = plan.Validate()
	if hasSemanticFinding(findings, "plan-silent-not-applicable") {
		t.Fatalf("explicit N/A must pass: %#v", findings)
	}
}

func TestPlanValidateRejectsDuplicatesAndMissingGoal(t *testing.T) {
	plan := DocumentationPlan{Obligations: []PlanObligation{
		{ID: "dup", Kind: ObligationSchema, Applicable: true, Reason: "schema changed"},
		{ID: "dup", Kind: ObligationSchema, Applicable: true, Reason: "schema changed"},
	}}
	findings := plan.Validate()
	if !hasSemanticFinding(findings, "plan-duplicate-obligation") {
		t.Fatalf("duplicate obligation must be a finding: %#v", findings)
	}
	if !hasSemanticFinding(findings, "plan-missing-goal") {
		t.Fatalf("missing goal must be a finding: %#v", findings)
	}
}

func TestReconcileDiffsPredictionAgainstReality(t *testing.T) {
	plan := DocumentationPlan{Goal: "G1", PredictedSet: []string{"product.vision", "cli.reference"}}
	actual := []Impact{{ContractID: "cli.reference"}, {ContractID: "architecture.system"}}
	got := Reconcile(plan, actual)
	if len(got.Missing) != 1 || got.Missing[0] != "product.vision" {
		t.Fatalf("expected product.vision under-predicted: %#v", got)
	}
	if len(got.Unpredicted) != 1 || got.Unpredicted[0] != "architecture.system" {
		t.Fatalf("expected architecture.system over-predicted: %#v", got)
	}
	if got.DriftRatio != 1 {
		t.Fatalf("expected drift ratio 1.0, got %v", got.DriftRatio)
	}
	if !hasSemanticFinding(got.Findings, "plan-underpredicted") || !hasSemanticFinding(got.Findings, "plan-overpredicted") {
		t.Fatalf("expected both drift findings: %#v", got.Findings)
	}
}

func TestPlanRoundTripsThroughRuntimeState(t *testing.T) {
	repo, _ := filepath.Abs("../..")
	root := t.TempDir()
	plan, err := BuildPlan(repo, "Goal: One", []string{"docs/readme.md"}, "2026-09-15T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if err := SavePlan(root, plan); err != nil {
		t.Fatal(err)
	}
	stored, err := LoadPlan(root, "Goal: One")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Goal != plan.Goal || len(stored.Obligations) != len(plan.Obligations) {
		t.Fatalf("round trip mismatch: %#v vs %#v", stored, plan)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(PlanRoot))); err != nil {
		t.Fatalf("plans must live under the derived runtime root: %v", err)
	}
	if _, err := LoadPlan(root, "unknown"); err == nil {
		t.Fatal("expected a missing plan to error")
	}
}

func TestPlanExplainExplainsAnObligation(t *testing.T) {
	plan := DocumentationPlan{Goal: "G", Obligations: []PlanObligation{
		{ID: "doc:cli.reference", ContractID: "cli.reference", Kind: ObligationDocumentUpdate,
			Applicable: true, Reason: "trigger:path:docs/reference", Target: []string{"docs/reference/cli.md"}},
	}}
	out, err := plan.Explain("cli.reference")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"cli.reference", "trigger:path:docs/reference", "docs/reference/cli.md"} {
		if !containsSubstring(out, want) {
			t.Fatalf("explanation is missing %q:\n%s", want, out)
		}
	}
	if _, err := plan.Explain("nope"); err == nil {
		t.Fatal("expected an error for an unknown obligation")
	}
}

func hasSemanticFinding(findings []SemanticFinding, kind string) bool {
	for _, f := range findings {
		if f.Kind == kind {
			return true
		}
	}
	return false
}

func containsSubstring(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
