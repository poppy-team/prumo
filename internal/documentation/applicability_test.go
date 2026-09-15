package docengine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// waivableContract exercises the applicability path without depending on the
// UI registry, so the mechanism is tested on its own terms.
func waivableContract() Contract {
	return Contract{ID: "ui.example", Version: 1, Role: "ui",
		RequiredKnowledge: []string{"state matrix"}}
}

// TestWaiverRequiresAReason is the invariant the mechanism exists for: an
// obligation may be waived, but never silently. A waiver without a reason is
// indistinguishable from a forgotten gate, so it fails closed.
func TestWaiverRequiresAReason(t *testing.T) {
	contract := waivableContract()

	waived := evaluate(t.TempDir(), contract, Binding{
		ContractID: contract.ID, Applicability: ApplicabilityNotApplicable,
		NotApplicableReason: "no locale surface ships in this increment",
	})
	if waived.State != NotApplicable {
		t.Fatalf("a reasoned waiver must be not-applicable, got %s", waived.State)
	}
	if len(waived.Findings) != 0 {
		t.Fatalf("a reasoned waiver must not produce findings: %#v", waived.Findings)
	}

	silent := evaluate(t.TempDir(), contract, Binding{
		ContractID: contract.ID, Applicability: ApplicabilityNotApplicable,
	})
	if silent.State != Missing {
		t.Fatalf("a waiver without a reason must fail closed to missing, got %s", silent.State)
	}
	if !hasFinding(silent.Findings, "not-applicable-without-reason") {
		t.Fatalf("expected not-applicable-without-reason, got %#v", silent.Findings)
	}
	if len(silent.BlockingQuestions) == 0 && len(contract.BlockingQuestions) > 0 {
		t.Fatal("an unexplained waiver must still surface the contract's questions")
	}
}

// TestWaiverNeverCountsAsBoundWork keeps the audit honest: a waived contract is
// neither a semantic contract nor a lexical one, so the lexical count reflects
// work a reviewer still has to do.
func TestWaiverNeverCountsAsBoundWork(t *testing.T) {
	root := t.TempDir()
	writeJSONFile(t, root, "docs/contracts/bindings.json", []map[string]any{
		{"contract_id": "ui.example", "sources": []string{}, "ownership": "prumo-maintained",
			"authority": "canonical-documentation", "applicability": "not-applicable",
			"not_applicable_reason": "surface does not exist yet"},
	})
	bindings, err := LoadBindings(root)
	if err != nil {
		t.Fatal(err)
	}
	coverage := evaluate(root, waivableContract(), bindings[0])
	if coverage.Authoritative {
		t.Fatal("a waived contract must never be authoritative")
	}
	if coverage.State != NotApplicable {
		t.Fatalf("expected not-applicable, got %s", coverage.State)
	}

	report := ReadinessReport{Ready: true, UnverifiedContracts: []string{}, NotApplicable: []string{}}
	applyCoverage(&report, coverage)
	if !report.Ready {
		t.Fatal("a reasoned waiver must not block readiness")
	}
	if len(report.BlockingContracts) != 0 || len(report.UnverifiedContracts) != 0 {
		t.Fatalf("a reasoned waiver must not appear as blocking or unverified: %#v", report)
	}
}

// TestRepositoryWaiversAreAllReasoned is the dogfood assertion: every waiver in
// the canonical bindings carries a specific reason. Without this, the waiver
// list would rot into a mute suppression file.
func TestRepositoryWaiversAreAllReasoned(t *testing.T) {
	root, _ := filepath.Abs("../..")
	bindings, err := LoadBindings(root)
	if err != nil {
		t.Fatal(err)
	}
	waived := 0
	for _, b := range bindings {
		if b.Applicability != ApplicabilityNotApplicable {
			continue
		}
		waived++
		if len(b.NotApplicableReason) < 40 {
			t.Errorf("%s: waiver reason is too thin to review (%q)", b.ContractID, b.NotApplicableReason)
		}
	}
	if waived == 0 {
		t.Log("no waivers declared; the assertion is vacuous until one is added")
	}
}

// TestBindingSchemaRequiresReasonForWaiver guards the machine contract, not just
// the reader: a waiver without a reason must be invalid JSON Schema, so an
// external tool cannot write one.
func TestBindingSchemaRequiresReasonForWaiver(t *testing.T) {
	root, _ := filepath.Abs("../..")
	data, err := os.ReadFile(filepath.Join(root, "schemas", "documentation-binding.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Properties map[string]any `json:"properties"`
		AllOf      []struct {
			If struct {
				Properties map[string]any `json:"properties"`
				Required   []string       `json:"required"`
			} `json:"if"`
			Then struct {
				Required []string `json:"required"`
			} `json:"then"`
		} `json:"allOf"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	if _, ok := schema.Properties["applicability"]; !ok {
		t.Fatal("the binding schema must declare applicability")
	}
	if _, ok := schema.Properties["not_applicable_reason"]; !ok {
		t.Fatal("the binding schema must declare not_applicable_reason")
	}
	for _, clause := range schema.AllOf {
		if len(clause.Then.Required) != 1 || clause.Then.Required[0] != "not_applicable_reason" {
			continue
		}
		if contains(clause.If.Required, "applicability") {
			return
		}
	}
	t.Fatal("the binding schema must require not_applicable_reason when applicability is not-applicable")
}

func writeJSONFile(t *testing.T, root, rel string, value any) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
