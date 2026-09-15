package schemaruntime

import (
	"encoding/json"
	"reflect"
	"testing"

	docengine "github.com/raillen/prumo/internal/documentation"
	"github.com/raillen/prumo/internal/gauntlet"
	"github.com/raillen/prumo/internal/knowledge"
	"github.com/raillen/prumo/internal/protocol"
)

func roundTrip(t *testing.T, label string, in any, out any) {
	t.Helper()
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("%s: marshal: %v", label, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("%s: unmarshal: %v", label, err)
	}
	got := reflect.ValueOf(out).Elem().Interface()
	if !reflect.DeepEqual(in, got) {
		t.Fatalf("%s: round-trip changed the value:\n in:  %#v\n out: %#v", label, in, got)
	}
}

// TestGoTypesRoundTrip is W1.3: JSON → Go struct → JSON must be lossless for
// the contract-bearing runtime types.
func TestGoTypesRoundTrip(t *testing.T) {
	coverage := docengine.Coverage{
		ContractID:        "architecture.system",
		State:             docengine.Verified,
		Sources:           []string{"docs/architecture/overview.md"},
		MissingKnowledge:  []string{"dependency direction"},
		BlockingQuestions: []string{"Who owns canonical state?"},
		Warnings:          []string{"unknown contract: x"},
	}
	roundTrip(t, "docengine.Coverage", coverage, &docengine.Coverage{})

	roundTrip(t, "docengine.AuditReport", docengine.AuditReport{
		Profiles:            []string{"core-software", "cli"},
		ApplicableContracts: []string{"product.vision"},
		Coverage:            []docengine.Coverage{coverage},
		Warnings:            []string{},
	}, &docengine.AuditReport{})

	roundTrip(t, "docengine.ReadinessReport", docengine.ReadinessReport{
		Ready:               false,
		Goal:                "M5",
		BlockingContracts:   []string{"architecture.system"},
		BlockingQuestions:   []string{"Who owns canonical state?"},
		Warnings:            []string{},
		ApplicableContracts: []string{"product.vision"},
		Coverage:            []docengine.Coverage{coverage},
	}, &docengine.ReadinessReport{})

	roundTrip(t, "docengine.AuthorityReport", docengine.AuthorityReport{
		CurrentVersion: "0.6.0",
		Files:          127, Canonical: 97, Projection: 2, Historical: 28,
		Findings: []docengine.AuthorityFinding{{Kind: "version_drift", Path: "AGENTS.md", Detail: "v0.4"}},
		OK:       false,
	}, &docengine.AuthorityReport{})

	policy := gauntlet.DefaultPolicy()
	policy.Mode = gauntlet.ModeAuto
	policy.Dimensions = []string{"semantic_coverage", "freshness"}
	policy.Stopping.NoImprovementRounds = 2
	roundTrip(t, "gauntlet.Policy", policy, &gauntlet.Policy{})

	roundTrip(t, "protocol.Envelope", protocol.OkEnvelope(nil), &protocol.Envelope{})

	// Knowledge identity must survive JSON transport as its stable string.
	id, err := knowledge.NewID("cli.agent-run")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(id)
	if err != nil || string(encoded) != `"ku:cli.agent-run"` {
		t.Fatalf("knowledge ID must marshal as its stable string, got %s (%v)", encoded, err)
	}
	var decoded knowledge.ID
	if err := json.Unmarshal(encoded, &decoded); err != nil || decoded != id {
		t.Fatalf("knowledge ID must unmarshal back to the same identity, got %q (%v)", decoded, err)
	}
}
