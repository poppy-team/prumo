package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/protocol"
	"github.com/raillen/prumo/internal/traceability"
)

func TestRunTraceMissingRef(t *testing.T) {
	code, out := captureOutput(func() int {
		return run([]string{"trace"})
	})
	if code != exitUsage {
		t.Fatalf("expected exitUsage (%d), got %d; out: %s", exitUsage, code, out)
	}
}

func TestRunTraceHuman(t *testing.T) {
	root := t.TempDir()
	goalDir := filepath.Join(root, ".ai", "goals")
	if err := os.MkdirAll(goalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	goalJSON := `{"id":"G-CORE","title":"Core Engine","phase":"P00","state":"LOCKED","objective":"Build engine."}`
	if err := os.WriteFile(filepath.Join(goalDir, "G-CORE.goal.json"), []byte(goalJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out := captureOutput(func() int {
		return run([]string{"trace", "G-CORE", "--path", root})
	})
	if code != exitOK {
		t.Fatalf("expected exitOK (%d), got %d; out: %s", exitOK, code, out)
	}
	if !strings.Contains(out, "PRUMO TRACEABILITY: Core Engine") {
		t.Errorf("expected header in trace output, got:\n%s", out)
	}
	if !strings.Contains(out, "Node ID:   G-CORE") {
		t.Errorf("expected G-CORE node ID, got:\n%s", out)
	}
}

func TestRunTraceJSON(t *testing.T) {
	root := t.TempDir()
	goalDir := filepath.Join(root, ".ai", "goals")
	if err := os.MkdirAll(goalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	goalJSON := `{"id":"G-CORE","title":"Core Engine","phase":"P00","state":"LOCKED","objective":"Build engine."}`
	if err := os.WriteFile(filepath.Join(goalDir, "G-CORE.goal.json"), []byte(goalJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out := captureOutput(func() int {
		return run([]string{"--json", "trace", "G-CORE", "--path", root})
	})
	if code != exitOK {
		t.Fatalf("expected exitOK (%d), got %d; out: %s", exitOK, code, out)
	}
	var env protocol.Envelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("failed unmarshaling trace json: %v; out:\n%s", err, out)
	}
	if !env.Ok {
		t.Fatalf("expected env.Ok == true")
	}
	dataMap, ok := env.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected data map in envelope")
	}
	if dataMap["root"] == nil {
		t.Errorf("expected root node in trace envelope")
	}
}

func TestRunTraceMissingNodeHonestError(t *testing.T) {
	root := t.TempDir()
	goalDir := filepath.Join(root, ".ai", "goals")
	if err := os.MkdirAll(goalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goalDir, "G-EXISTS.goal.json"), []byte(`{"id":"G-EXISTS","title":"Exists"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out := captureOutput(func() int {
		return run([]string{"--json", "trace", "G-NONEXISTENT", "--path", root})
	})
	if code != exitValidation {
		t.Fatalf("expected exitValidation (%d), got %d", exitValidation, code)
	}
	if !strings.Contains(out, "G-NONEXISTENT") || !strings.Contains(out, "G-EXISTS") {
		t.Errorf("expected error naming missing node and known nodes, got: %s", out)
	}
}

func TestRunJournal(t *testing.T) {
	tmpDir := t.TempDir()
	storeDir := filepath.Join(tmpDir, ".prumo", "traceability")

	j := traceability.NewJournal()
	_ = j.Add(traceability.JournalEntry{
		ID:          "jour-test",
		Goal:        "M8",
		Title:       "Traceability Verification",
		Summary:     "Verified end-to-end trace query and journal storage.",
		Decisions:   []string{"dec-clean-arch"},
		CodeChanges: []string{"cmd/prumo/trace_commands.go"},
	})
	if err := traceability.SaveJournal(storeDir, j); err != nil {
		t.Fatalf("SaveJournal failed: %v", err)
	}

	// Human mode
	code, out := captureOutput(func() int {
		return run([]string{"journal", "--path", tmpDir, "--goal", "M8"})
	})
	if code != exitOK {
		t.Fatalf("expected exitOK, got %d; out: %s", code, out)
	}
	if !strings.Contains(out, "IMPLEMENTATION JOURNAL") || !strings.Contains(out, "jour-test") {
		t.Errorf("expected journal entry in output, got:\n%s", out)
	}

	// JSON mode
	codeJSON, outJSON := captureOutput(func() int {
		return run([]string{"--json", "journal", "--path", tmpDir, "--goal", "M8"})
	})
	if codeJSON != exitOK {
		t.Fatalf("expected exitOK, got %d; out: %s", codeJSON, outJSON)
	}
	var env protocol.Envelope
	if err := json.Unmarshal([]byte(outJSON), &env); err != nil {
		t.Fatalf("failed parsing journal JSON: %v", err)
	}
	if !env.Ok {
		t.Errorf("expected env.Ok == true")
	}
}
