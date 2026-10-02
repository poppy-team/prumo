package goals

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadWorkspace(t *testing.T) {
	ws := t.TempDir()
	goalsDir := filepath.Join(ws, ".ai", "goals", "p01")
	if err := os.MkdirAll(goalsDir, 0755); err != nil {
		t.Fatal(err)
	}

	valid := `{
		"id": "P01-G01",
		"title": "Protocol Conformance",
		"phase": "P01",
		"state": "EXECUTING",
		"acceptance": ["Contract holds"],
		"constraints": ["Zero allocation"],
		"gates": {"test": "required"}
	}`
	if err := os.WriteFile(filepath.Join(goalsDir, "P01-G01.goal.json"), []byte(valid), 0644); err != nil {
		t.Fatal(err)
	}

	malformed := `{ "id": "P01-G02", invalid json`
	if err := os.WriteFile(filepath.Join(goalsDir, "P01-G02.goal.json"), []byte(malformed), 0644); err != nil {
		t.Fatal(err)
	}

	outsideDir := t.TempDir()
	targetOutside := filepath.Join(outsideDir, "outside.goal.json")
	if err := os.WriteFile(targetOutside, []byte(valid), 0644); err != nil {
		t.Fatal(err)
	}
	_ = os.Symlink(targetOutside, filepath.Join(goalsDir, "symlink.goal.json"))

	list := ReadWorkspace(ws)
	if len(list) != 1 {
		t.Fatalf("expected 1 valid goal, got %d", len(list))
	}
	g := list[0]
	if g.ID != "P01-G01" || g.State != "EXECUTING" {
		t.Fatalf("unexpected goal read: %+v", g)
	}
	if len(g.Acceptance) != 1 || g.Acceptance[0] != "Contract holds" {
		t.Fatalf("unexpected acceptance: %+v", g.Acceptance)
	}
	if len(g.Constraints) != 1 || g.Constraints[0] != "Zero allocation" {
		t.Fatalf("unexpected constraints: %+v", g.Constraints)
	}
	if g.Gates["test"] != "required" {
		t.Fatalf("unexpected gates: %+v", g.Gates)
	}
}

func TestBoundaryNoInternalImports(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go tool unavailable")
	}
	out, err := exec.Command("go", "list", "-deps", "-test", "./...").Output()
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}
	for _, dep := range strings.Fields(string(out)) {
		if strings.Contains(dep, "github.com/raillen/prumo/internal/") {
			t.Fatalf("tui depends on internal package %q", dep)
		}
	}
}
