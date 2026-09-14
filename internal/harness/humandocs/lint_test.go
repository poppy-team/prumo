package humandocs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLintCleanTree(t *testing.T) {
	dir := t.TempDir()
	spec := Spec{ID: "hd", Title: "HD", Audience: "ops", Profiles: []string{"p-ops"},
		Sources: map[string][]string{"c-run": {"docs/run.md"}}}
	reg := testRegistry()
	plan, err := Planner(spec, reg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateTree(spec, plan, reg, dir); err != nil {
		t.Fatal(err)
	}
	if err := LintError(dir, plan); err != nil {
		t.Fatalf("clean tree must lint: %v", err)
	}
}

func TestLintFindsHoles(t *testing.T) {
	dir := t.TempDir()
	plan := Plan{Units: []Unit{{ID: "u1", Status: "ready"}}}
	issues := Lint(dir, plan)
	if len(issues) < 3 {
		t.Fatalf("empty dir must report readme+reference+index: %+v", issues)
	}
	bad := filepath.Join(dir, "README.md")
	if err := os.WriteFile(bad, []byte("x TODO(CURATED): write me"), 0o644); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, is := range Lint(dir, plan) {
		if is.Rule == "generated-pure" {
			found = true
		}
	}
	if !found {
		t.Fatal("curated marker in GENERATED file must be flagged")
	}
}
