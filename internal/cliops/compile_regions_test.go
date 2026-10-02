package cliops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCompilePreservesHumanInstructionRegions is the regression test for the
// finding that `prumo compile` used to overwrite CLAUDE.md and AGENTS.md
// wholesale, deleting project rules on every recompile. The generated block is
// now delimited by stable region markers, so a human edit outside the block —
// and the human block itself — survives recompilation, while the generated
// content still refreshes.
func TestCompilePreservesHumanInstructionRegions(t *testing.T) {
	root := t.TempDir()
	repoRoot := testRepoRootForCompile(t)
	svc := New(repoRoot)

	// First compile creates the file.
	if _, err := svc.Compile(root, "claude-code"); err != nil {
		t.Fatalf("first compile: %v", err)
	}
	claude := filepath.Join(root, "CLAUDE.md")
	data, err := os.ReadFile(claude)
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	if !strings.Contains(string(data), beginRegion(agentsCoreRegionID)) {
		t.Fatalf("expected a managed region marker in a freshly compiled instruction file, got:\n%s", data)
	}

	// A human adds project rules outside the managed region.
	withHuman := string(data) + "\n## Project Rules\n- Never force-push main.\n- Migrations require review.\n"
	if err := os.WriteFile(claude, []byte(withHuman), 0o644); err != nil {
		t.Fatal(err)
	}

	// Recompiling must not delete the human section.
	if _, err := svc.Compile(root, "claude-code"); err != nil {
		t.Fatalf("second compile: %v", err)
	}
	after, err := os.ReadFile(claude)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "## Project Rules") {
		t.Fatalf("recompile destroyed human-maintained content, got:\n%s", after)
	}
	if !strings.Contains(string(after), "Never force-push main.") {
		t.Fatalf("recompile destroyed human rules, got:\n%s", after)
	}
	// And it must not accumulate duplicate managed regions.
	if got := strings.Count(string(after), beginRegion(agentsCoreRegionID)); got != 1 {
		t.Fatalf("expected exactly one managed region after recompile, got %d", got)
	}
}

// TestCompileAdoptsUnmarkedInstructionFile covers the upgrade path: a project
// that already has an instruction file written before managed regions existed
// keeps its content, and the generated block is appended rather than replacing
// what the human wrote.
func TestCompileAdoptsUnmarkedInstructionFile(t *testing.T) {
	root := t.TempDir()
	svc := New(testRepoRootForCompile(t))

	claude := filepath.Join(root, "CLAUDE.md")
	existing := "# CLAUDE.md\n\nMy own hand-written guidance that must survive.\n"
	if err := os.WriteFile(claude, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Compile(root, "claude-code"); err != nil {
		t.Fatalf("compile: %v", err)
	}
	after, err := os.ReadFile(claude)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "hand-written guidance that must survive") {
		t.Fatalf("compile destroyed a pre-existing unmarked instruction file, got:\n%s", after)
	}
	if !strings.Contains(string(after), beginRegion(agentsCoreRegionID)) {
		t.Fatalf("expected the generated block to be appended, got:\n%s", after)
	}
}

func testRepoRootForCompile(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}
