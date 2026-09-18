package main

import (
	"encoding/json"
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
	code, out := captureOutput(func() int {
		return run([]string{"trace", "code-trace"})
	})
	if code != exitOK {
		t.Fatalf("expected exitOK (%d), got %d; out: %s", exitOK, code, out)
	}
	if !strings.Contains(out, "PRUMO TRACEABILITY: Traceability Engine") {
		t.Errorf("expected header in trace output, got:\n%s", out)
	}
	if !strings.Contains(out, "Upstream Lineage") {
		t.Errorf("expected Upstream Lineage in trace output")
	}
	if !strings.Contains(out, "Downstream Lineage") {
		t.Errorf("expected Downstream Lineage in trace output")
	}
}

func TestRunTraceJSON(t *testing.T) {
	code, out := captureOutput(func() int {
		return run([]string{"--json", "trace", "code-trace"})
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
