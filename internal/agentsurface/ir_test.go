package agentsurface

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleIR() IR {
	return IR{
		Version:     1,
		TokenBudget: Budget{Root: 1000, Scope: 1000},
		Rules: []Rule{
			{ID: "root-b", Scope: "", Precedence: 20, Text: "Second root rule."},
			{ID: "root-a", Scope: "", Precedence: 10, Text: "First root rule."},
			{ID: "inner", Scope: "internal/harness", Precedence: 10, Text: "Harness rule."},
		},
	}
}

func TestEffectiveRulesOrderAndInheritance(t *testing.T) {
	ir := sampleIR()
	root := ir.EffectiveRules("")
	if len(root) != 2 || root[0].ID != "root-a" || root[1].ID != "root-b" {
		t.Fatalf("root scope must be ordered by precedence: %#v", root)
	}
	inner := ir.EffectiveRules("internal/harness/agent")
	if len(inner) != 3 {
		t.Fatalf("nested scope must inherit root rules: %#v", inner)
	}
	if inner[2].ID != "inner" {
		t.Fatalf("more specific scope must compile last: %#v", inner)
	}
	if got := ir.EffectiveRules("internal/documentation"); len(got) != 2 {
		t.Fatalf("sibling scope must not inherit the harness rule: %#v", got)
	}
	if scopes := ir.Scopes(); len(scopes) != 2 || scopes[0] != "" {
		t.Fatalf("scopes must list root first: %#v", scopes)
	}
}

func TestDuplicateAndConflictingRulesDetected(t *testing.T) {
	ir := IR{Version: 1, Rules: []Rule{
		{ID: "a", Text: "Keep the entrypoint short."},
		{ID: "b", Text: "  keep   the ENTRYPOINT short. "},
	}}
	if conflicts := ir.Conflicts(""); len(conflicts) != 1 || conflicts[0].Kind != "duplicate-instruction" {
		t.Fatalf("expected duplicate instruction, got %#v", conflicts)
	}

	sameID := IR{Version: 1, Rules: []Rule{
		{ID: "shared", Scope: "", Text: "Root statement."},
		{ID: "shared", Scope: "internal", Text: "Different statement."},
	}}
	if conflicts := sameID.Conflicts("internal"); len(conflicts) != 1 || conflicts[0].Kind != "conflicting-rule" {
		t.Fatalf("expected conflicting rule, got %#v", conflicts)
	}
}

func TestValidateRejectsStructuralProblems(t *testing.T) {
	ir := IR{Version: 1, Rules: []Rule{
		{ID: "", Text: "No id."},
		{ID: "empty", Text: "   "},
		{ID: "unknown", Text: "Bad adapter.", Adapters: []string{"not-a-tool"}},
		{ID: "escaping", Scope: "../outside", Text: "Bad scope."},
	}}
	kinds := map[string]bool{}
	for _, p := range ir.Validate() {
		kinds[p.Kind] = true
	}
	for _, want := range []string{"missing-rule-id", "empty-rule-text", "unknown-adapter", "invalid-scope"} {
		if !kinds[want] {
			t.Fatalf("expected %s in %#v", want, kinds)
		}
	}
}

// TestCoreHasNoVendorKnowledge pins the layer invariant: only adapter
// implementations may know a vendor format (W16.13).
func TestCoreHasNoVendorKnowledge(t *testing.T) {
	for _, name := range []string{"ir.go", "compile.go", "references.go"} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		source := strings.ToLower(string(data))
		for _, vendor := range []string{"copilot", "cursor", "claude", "skill.md", ".mdc"} {
			if strings.Contains(source, vendor) {
				t.Fatalf("%s must not reference the vendor format %q", name, vendor)
			}
		}
	}
}

func TestLoadIRRejectsEmptyIR(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs", "agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(IRPath))
	cases := []string{
		`{"version":1,"rules":[]}`,
		`{"version":0,"rules":[{"id":"a","text":"x"}]}`,
		`{}`,
	}
	for _, body := range cases {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadIR(root); err == nil {
			t.Fatalf("expected %s to be rejected", body)
		}
	}
}
