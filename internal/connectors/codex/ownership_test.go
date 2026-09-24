package codex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/raillen/prumo/internal/connectors"
	"github.com/raillen/prumo/internal/install"
)

// The whole path, end to end: a user who already has AGENTS.md installs the
// connector and then uninstalls it. The file has to be there afterwards, with its
// content, because it belongs to them and nothing in this framework can restore
// it (GAP-141).
//
// The test goes through the connector rather than the install helper because the
// bug lived in the wiring: the write was unconditional and the path was appended
// to the created list, so the uninstall had every reason and no way to know the
// file was not its own.
func TestInstallingAndUninstallingLeavesAUsersAgentsFileAlone(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	agentsPath := filepath.Join(root, "AGENTS.md")
	const mine = "# My project\n\nRun the linter before committing.\n"
	if err := os.WriteFile(agentsPath, []byte(mine), 0o644); err != nil {
		t.Fatal(err)
	}

	connector := NewConnector()
	result, err := connector.Install(home, root, connectors.InstallOptions{})
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	for _, p := range result.CreatedPaths {
		if p == agentsPath {
			t.Fatal("the user's AGENTS.md was claimed as created; the uninstall would delete it")
		}
	}
	if len(result.PreservedPaths) != 1 || result.PreservedPaths[0] != agentsPath {
		t.Fatalf("the skipped file must be reported, got %v", result.PreservedPaths)
	}
	after, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("the install removed the user's file: %v", err)
	}
	if string(after) != mine {
		t.Fatalf("the install rewrote the user's file: %q", after)
	}

	// The rest of the install must still have happened: refusing to clobber one
	// file is not a reason to abandon the connector.
	if len(result.CreatedPaths) == 0 {
		t.Error("the connector created nothing, so the refusal broke the install rather than protecting a file")
	}

	// And the uninstall must not take the file with it, whether it goes through
	// the cleanup manifest or the direct path.
	removed, _ := install.RemoveManagedPaths(home, result.CreatedPaths)
	t.Logf("removed %d of %d created paths", len(removed), len(result.CreatedPaths))
	final, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("the uninstall removed the user's file: %v", err)
	}
	if string(final) != mine {
		t.Fatalf("the uninstall modified the user's file: %q", final)
	}
}
