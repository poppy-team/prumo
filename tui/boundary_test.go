package tui

import (
	"os/exec"
	"strings"
	"testing"
)

// TestBoundaryNoInternalImports is the TUI split-gate, the sibling of
// sdk/prumo's test of the same name and the machine-checkable form of H10
// acceptance criterion 1.
//
// The TUI is a client of the harness, not a part of it: it reaches the daemon
// through `sdk/prumo` (the public typed SDK) and through the `prumo` binary it
// supervises. The day `tui/` imports `prumo/internal` is the day the split
// becomes a rename, and this test fails before that lands rather than after.
//
// Test files are included deliberately. A boundary that holds only for
// non-test code is not a boundary: a `_test` helper that reaches into
// `internal/harness/agent` would let the next production file follow it.
func TestBoundaryNoInternalImports(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go tool unavailable")
	}
	// `-deps` covers the package and everything it transitively links,
	// `-test` includes test files of the package itself.
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
