package tui_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestBoundaryNoCoreInternals fails the build if this client ever imports the
// core's internal packages.
//
// ADR 013 made the boundary structural by moving the client into its own
// module, but structure alone does not stop someone adding an import that only
// works because of the local `replace` directive. This is the check that does:
// the client reaches the harness through the SDK and the protocol, and a
// dependency on `internal/` would mean it had started to know how the runtime
// works.
//
// It asks the toolchain rather than reading files, so transitive dependencies
// count. Test files count too: a test that reaches inside the core would make
// the boundary a claim rather than a fact.
func TestBoundaryNoCoreInternals(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go tool unavailable")
	}
	cmd := exec.Command(goBin, "list", "-deps", "-test", "./...")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list failed: %v\n%s", err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		pkg := strings.TrimSpace(line)
		if strings.HasPrefix(pkg, "github.com/raillen/prumo/internal/") {
			t.Errorf("the client imports %s: it must reach the harness only through github.com/raillen/prumo/sdk/prumo", pkg)
		}
	}
}
