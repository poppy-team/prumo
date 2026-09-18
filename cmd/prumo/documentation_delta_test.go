package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type docsDeltaRecord struct {
	ID       string   `json:"id"`
	State    string   `json:"state"`
	Source   string   `json:"source"`
	Evidence []string `json:"evidence"`
}

type docsDeltaEnvelope struct {
	Data docsDeltaRecord `json:"data"`
}

func writeDeltaBindings(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "docs", "contracts", "bindings.json")
	content := `[
		{"contract_id": "architecture.system", "sources": ["docs/architecture/overview.md"], "ownership": "human", "authority": "canonical"}
	]` + "\n"
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func runDocsDeltaJSON(t *testing.T, args []string) docsDeltaRecord {
	t.Helper()
	code, output := captureOutput(func() int { return run(args) })
	if code != 0 {
		t.Fatalf("docs delta command failed: %d %q", code, output)
	}
	var envelope docsDeltaEnvelope
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("invalid docs delta JSON: %v %q", err, output)
	}
	return envelope.Data
}

func TestDocsDeltaLifecycle(t *testing.T) {
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	root := t.TempDir()
	writeDeltaBindings(t, root)
	changed := "docs/architecture/overview.md"

	proposed := runDocsDeltaJSON(t, []string{"--json", "docs", "delta", "propose", "--goal", "G042", "--path", root, changed})
	if !strings.HasPrefix(proposed.ID, "DD-") || proposed.State != "proposed" || proposed.Source != "G042" {
		t.Fatalf("unexpected proposed delta: %#v", proposed)
	}

	reviewed := runDocsDeltaJSON(t, []string{"--json", "docs", "delta", "transition", "--id", proposed.ID, "--state", "reviewed", "--evidence", "EV-001", "--path", root})
	if reviewed.State != "reviewed" || len(reviewed.Evidence) != 1 {
		t.Fatalf("unexpected reviewed delta: %#v", reviewed)
	}

	accepted := runDocsDeltaJSON(t, []string{"--json", "docs", "delta", "transition", "--id", proposed.ID, "--state", "accepted", "--path", root})
	if accepted.State != "accepted" {
		t.Fatalf("unexpected accepted delta: %#v", accepted)
	}

	applied := runDocsDeltaJSON(t, []string{"--json", "docs", "delta", "transition", "--id", proposed.ID, "--state", "applied", "--evidence", "EV-002", "--path", root})
	if applied.State != "applied" || len(applied.Evidence) != 2 {
		t.Fatalf("unexpected applied delta: %#v", applied)
	}

	if code := run([]string{"docs", "delta", "transition", "--id", proposed.ID, "--state", "reviewed", "--path", root}); code != 1 {
		t.Fatalf("expected backward transition to fail, got %d", code)
	}
}

func TestDocsDeltaProposeRequiresGoal(t *testing.T) {
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	if code := run([]string{"docs", "delta", "propose", "--path", t.TempDir()}); code != 2 {
		t.Fatalf("expected usage error, got %d", code)
	}
}
