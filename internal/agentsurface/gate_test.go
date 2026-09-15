package agentsurface

import (
	"path/filepath"
	"testing"
)

// TestAgentSurfaceContextRotGate is the repository's context-rot CI gate
// (W16.14). It fails when a compiled agent surface is stale or when any
// instruction document points at a path, command, symbol or test that no
// longer exists.
func TestAgentSurfaceContextRotGate(t *testing.T) {
	root, _ := filepath.Abs("../..")
	report, err := Verify(root)
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK {
		for _, f := range report.Findings {
			t.Errorf("%s: %s (%s) — %s", f.Kind, f.Target, f.Value, f.Detail)
		}
		t.Fatalf("agent surface gate failed with %d findings", len(report.Findings))
	}
	if report.Surfaces == 0 || report.IRVersion < 1 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

// TestContextRotGuardHasTeeth proves the gate would catch rot: the same check
// reports a finding once an instruction document references a missing path.
func TestContextRotGuardHasTeeth(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "AGENTS.md", "- Start with `ENTRYPOINT.md` and `docs/removed-surface.md`.\n")
	writeFile(t, root, "ENTRYPOINT.md", "# ENTRYPOINT\n")
	writeFile(t, root, IRPath, mustJSON(t, smallIR()))

	findings, err := CheckContextRot(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Value != "docs/removed-surface.md" {
		t.Fatalf("expected exactly the removed pointer, got %#v", findings)
	}
}
