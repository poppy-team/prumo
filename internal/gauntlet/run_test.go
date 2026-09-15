package gauntlet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildRunPassesAndStopsAtGates(t *testing.T) {
	record := BuildRun(DefaultPolicy(), "M6", "run-1", []string{"authority", "documentation-links"}, nil)
	if !record.Passed() {
		t.Fatalf("expected a passing run: %#v", record)
	}
	if record.StopReason != string(StopGatesPassed) {
		t.Fatalf("unexpected stop reason: %s", record.StopReason)
	}
	if len(record.Rounds) != 1 || record.Rounds[0].ModelCritic == "" {
		t.Fatalf("the model critic skip must be explicit: %#v", record.Rounds)
	}
}

func TestBuildRunRecordsFailuresWithoutSilentIteration(t *testing.T) {
	problems := []Finding{
		{ID: "claim-drift", Dimension: DimensionFor("claim-drift"), Severity: SeverityFor("claim-drift"),
			Status: "open", Critic: "deterministic", Detail: "docs/x.md:12"},
		{ID: "broken-link", Dimension: DimensionFor("broken-link"), Severity: SeverityFor("broken-link"),
			Status: "open", Critic: "deterministic"},
	}
	record := BuildRun(DefaultPolicy(), "M6", "run-2", []string{"documentation-links"}, problems)
	if record.Passed() {
		t.Fatal("a run with failing checks must not pass")
	}
	// Default policy is off: the gauntlet returns to a human instead of
	// iterating on its own.
	if record.StopReason != string(StopHuman) {
		t.Fatalf("expected human-decision stop with the gauntlet off, got %s", record.StopReason)
	}
	if record.Findings[0].Dimension != "factual_grounding" || record.Findings[0].Severity != "critical" {
		t.Fatalf("claim drift must be a critical factual-grounding finding: %#v", record.Findings[0])
	}
	if len(record.Rounds[0].Deterministic.Failed) != 2 {
		t.Fatalf("every failing check must be listed: %#v", record.Rounds[0].Deterministic)
	}
}

func TestBuildRunWithAutoModeStaysBounded(t *testing.T) {
	policy := DefaultPolicy()
	policy.Mode = ModeAuto
	policy.Stopping.BudgetTokens = 1000
	problems := []Finding{{ID: "broken-link", Dimension: "reference_integrity", Severity: "high", Status: "open", Critic: "deterministic"}}
	record := BuildRun(policy, "M6", "run-3", []string{"documentation-links"}, problems)
	if record.StopReason != string(StopBudget) {
		t.Fatalf("expected a budget stop, got %s", record.StopReason)
	}
	if record.Rounds[0].ModelCritic == "" {
		t.Fatal("an auto run must still record the critic status explicitly")
	}
}

func TestSaveRunRecordMatchesSchemaShape(t *testing.T) {
	root := t.TempDir()
	record := BuildRun(DefaultPolicy(), "M6", "run-4", []string{"authority"}, nil)
	rel, err := Save(root, record)
	if err != nil {
		t.Fatal(err)
	}
	if rel != RunRoot+"/run-4.json" {
		t.Fatalf("run records must live under runtime state, got %s", rel)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"run_id", "mode", "rounds", "stop_reason", "findings"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("run record is missing the schema-required key %q", key)
		}
	}
	if _, err := Save(root, RunRecord{RunID: ""}); err == nil {
		t.Fatal("a run record without an id must be rejected")
	}
}

func TestDimensionCoverageForDocumentationChecks(t *testing.T) {
	cases := map[string]string{
		"broken-link":             "reference_integrity",
		"authority:version_drift": "factual_grounding",
		"claim-drift":             "factual_grounding",
		"surface-stale":           "token_efficiency",
		"dead-internal-link":      "information_architecture",
		"authority:unclassified":  "reference_integrity",
		"something-unknown":       "cross_document_consistency",
	}
	for kind, want := range cases {
		if got := DimensionFor(kind); got != want {
			t.Fatalf("DimensionFor(%q) = %s, want %s", kind, got, want)
		}
	}
	for _, dimension := range []string{"reference_integrity", "factual_grounding", "token_efficiency", "information_architecture", "cross_document_consistency"} {
		found := false
		for _, d := range Dimensions() {
			if d == dimension {
				found = true
			}
		}
		if !found {
			t.Fatalf("dimension %s is not part of the gauntlet policy dimensions", dimension)
		}
	}
}
