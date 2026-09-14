package contextv2

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSized(t *testing.T, path string, n int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, n), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCompileWorkspacePacksGoalFirst(t *testing.T) {
	dir := t.TempDir()
	writeSized(t, filepath.Join(dir, "README.md"), 400)
	writeSized(t, filepath.Join(dir, "a.go"), 4000)
	m := CompileWorkspace("R-ws", "fix the bug", dir, 8000, "L1")
	if m.Version != 2 || m.Level != "L1" {
		t.Fatalf("bad manifest: %+v", m)
	}
	refs := map[string]bool{}
	for _, it := range m.Included {
		refs[it.Ref] = true
	}
	if !refs["goal"] {
		t.Fatalf("goal must always pack (canonical): %+v", m.Included)
	}
	if !refs["repo-map"] {
		t.Fatalf("repo-map must pack: %+v", m.Included)
	}
	if m.EstimatedTokens > 8000 {
		t.Fatalf("budget exceeded: %d", m.EstimatedTokens)
	}
}

func TestCompileWorkspaceBudgetPressure(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 10; i++ {
		writeSized(t, filepath.Join(dir, "f", string(rune('a'+i))+".go"), 4000)
	}
	m := CompileWorkspace("R-ws", "goal", dir, 300, "")
	if m.EstimatedTokens > 300 {
		t.Fatalf("budget exceeded: %d", m.EstimatedTokens)
	}
	if len(m.Excluded) == 0 {
		t.Fatal("expected exclusions under pressure")
	}
	// Deterministic: same input, same order.
	m2 := CompileWorkspace("R-ws", "goal", dir, 300, "")
	if len(m.Included) != len(m2.Included) || m.Included[0].Ref != m2.Included[0].Ref {
		t.Fatal("compilation must be deterministic")
	}
}

func TestCompileWorkspaceDegradesGracefully(t *testing.T) {
	m := CompileWorkspace("R-ws", "goal", filepath.Join(t.TempDir(), "missing"), 8000, "")
	if len(m.Included) != 1 || m.Included[0].Ref != "goal" {
		t.Fatalf("missing root must degrade to goal-only: %+v", m)
	}
}
