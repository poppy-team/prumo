package humandocs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	docengine "github.com/raillen/prumo/internal/documentation"
)

func testRegistry() docengine.Registry {
	return docengine.Registry{
		Contracts: map[string]docengine.Contract{
			"c-run": {
				ID: "c-run", Role: "operator", Description: "how to run the harness safely",
				RequiredKnowledge:    []string{"run lifecycle", "budgets"},
				BlockingQuestions:    []string{"who approves production runs?"},
				EvidenceRequirements: []string{"green suite"},
			},
		},
		Profiles: map[string]docengine.Profile{
			"p-ops": {ID: "p-ops", Capabilities: []string{"operate"}, Contracts: []string{"c-run"}},
		},
	}
}

func TestComposeProfile(t *testing.T) {
	base := docengine.Profile{ID: "b", Capabilities: []string{"a", "b"}, Contracts: []string{"c1", "c2"}}
	over := docengine.Profile{ID: "o", Capabilities: []string{"c"}, Contracts: []string{"c3"}, ExcludedContracts: []string{"c2"}}
	got := ComposeProfile(base, over)
	if len(got.Capabilities) != 3 || len(got.Contracts) != 2 {
		t.Fatalf("bad composition: %+v", got)
	}
	for _, c := range got.Contracts {
		if c == "c2" {
			t.Fatal("excluded contract leaked")
		}
	}
	// Deterministic.
	if strings.Join(got.Contracts, ",") != "c1,c3" {
		t.Fatalf("order not deterministic: %v", got.Contracts)
	}
}

func TestPlannerGapsNotHallucinated(t *testing.T) {
	spec := Spec{ID: "hd", Title: "HD", Audience: "ops", Profiles: []string{"p-ops", "p-missing"}}
	plan, err := Planner(spec, testRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Units) != 2+4 {
		t.Fatalf("expected readme+reference+4 intents, got %d", len(plan.Units))
	}
	_, missing := Coverage(plan)
	if len(missing) != 4 {
		t.Fatalf("unsourced units must be missing: %v", missing)
	}
	ready, blockers := Readiness(plan)
	if ready {
		t.Fatal("unknown profile must block readiness")
	}
	found := false
	for _, b := range blockers {
		if b == "gap:profile:p-missing" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected profile gap blocker: %v", blockers)
	}
}

func TestPlannerSourcedReady(t *testing.T) {
	spec := Spec{ID: "hd", Title: "HD", Audience: "ops", Profiles: []string{"p-ops"},
		Sources: map[string][]string{"c-run": {"docs/run.md"}}}
	plan, err := Planner(spec, testRegistry())
	if err != nil {
		t.Fatal(err)
	}
	ready, missing := Coverage(plan)
	if len(missing) != 0 || len(ready) != 6 {
		t.Fatalf("sourced plan must be covered: %v %v", ready, missing)
	}
	if ok, blockers := Readiness(plan); !ok {
		t.Fatalf("sourced plan must be ready: %v", blockers)
	}
}

func TestGenerateTreeNoOp(t *testing.T) {
	dir := t.TempDir()
	spec := Spec{ID: "hd", Title: "HD", Audience: "ops", Profiles: []string{"p-ops"},
		Sources: map[string][]string{"c-run": {"docs/run.md"}}}
	plan, err := Planner(spec, testRegistry())
	if err != nil {
		t.Fatal(err)
	}
	arts, err := GenerateTree(spec, plan, testRegistry(), dir)
	if err != nil || len(arts) != 3 {
		t.Fatalf("tree failed: %+v %v", arts, err)
	}
	for _, p := range []string{"README.md", "docs/reference.md", "docs/INDEX.md"} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Fatalf("missing %s: %v", p, err)
		}
	}
	// Missing units get index rows, never files.
	if _, err := os.Stat(filepath.Join(dir, "docs", "tutorial")); !os.IsNotExist(err) {
		t.Fatal("no invented unit files allowed")
	}
	arts2, err := GenerateTree(spec, plan, testRegistry(), dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range arts2 {
		if !a.NoOp {
			t.Fatalf("second generation must be no-op: %+v", a)
		}
	}
	ref, _ := os.ReadFile(filepath.Join(dir, "docs/reference.md"))
	if !strings.Contains(string(ref), "c-run") || !strings.Contains(string(ref), "green suite") {
		t.Fatalf("reference must render contracts:\n%s", ref)
	}
}

func TestBriefNoClaimsBeyondInputs(t *testing.T) {
	u := Unit{ID: "hd:how-to:c-run", Intent: IntentHowTo, Audience: "ops", Title: "how-to", Status: "missing"}
	c := testRegistry().Contracts["c-run"]
	brief := Brief(u, c)
	for _, want := range []string{"TODO(CURATED): who approves production runs?", "run lifecycle", "do not write prose without sources", "green suite"} {
		if !strings.Contains(brief, want) {
			t.Fatalf("brief missing %q:\n%s", want, brief)
		}
	}
}

func TestGraphCycleDetected(t *testing.T) {
	g := Graph{Pages: []Page{
		{Unit: Unit{ID: "a"}, Links: []string{"b"}},
		{Unit: Unit{ID: "b"}, Links: []string{"a"}},
	}}
	if _, err := g.Order(); err == nil {
		t.Fatal("cycle must error")
	}
}
