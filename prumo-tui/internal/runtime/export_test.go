package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo-tui/internal/llm/models"
)

// The export exists to be read where the terminal is not, so it has to be plain
// text, complete enough to reconstruct what happened, and identical when the
// same run is exported twice.
func TestTheTimelineExportsAsPlainText(t *testing.T) {
	client := &timelineClient{status: "complete"}
	client.append(
		Event{Kind: "text_delta", Payload: map[string]any{"text": "hello"}},
		Event{Kind: "usage", Payload: map[string]any{
			"prompt_tokens": float64(120), "completion_tokens": float64(30),
			"cache_read_tokens": float64(900), "cache_write_tokens": float64(50), "cost_usd": 0.002,
		}},
		Event{Kind: "file.changed", Payload: map[string]any{"path": "internal/x.go", "operation": "modified"}},
	)
	workspace := t.TempDir()
	r := NewRunner(Options{
		Client: client, Sessions: newTestRunner(client, models.Model{}).sessions,
		Messages: newTestRunner(client, models.Model{}).messages, Workspace: workspace,
	})

	path, err := r.ExportTimeline(context.Background(), "S1")
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if !strings.HasPrefix(path, workspace) {
		t.Fatalf("the export landed outside the workspace: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the export: %v", err)
	}
	text := string(data)

	// What a session spent, by kind, is what makes the export worth keeping: a
	// file with a total alone could not answer "why did this cost that".
	for _, want := range []string{"run S1", "events 3", "text_delta", "hello", "usage", "prompt_tokens=120", "cache_read_tokens=900", "cache_write_tokens=50", "cost_usd=0.002", "file.changed"} {
		if !strings.Contains(text, want) {
			t.Errorf("the export lost %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "\x1b[") {
		t.Errorf("the export carries terminal escapes, which is the one thing it must not:\n%q", text)
	}

	// The same run exports the same bytes: a payload's keys are sorted, so two
	// exports can be compared.
	first := text
	if _, err := r.ExportTimeline(context.Background(), "S1"); err != nil {
		t.Fatalf("second export: %v", err)
	}
	again, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the second export: %v", err)
	}
	if string(again) != first {
		t.Fatal("two exports of the same run differ")
	}

	if _, err := filepath.Rel(filepath.Join(workspace, ExportDir), path); err != nil {
		t.Fatalf("the export is not where it says it is: %v", err)
	}
}

// A run the daemon does not know cannot be exported, and saying so is the point
// of asking rather than writing an empty file.
func TestExportingWithoutAHarnessIsRefused(t *testing.T) {
	r := NewRunner(Options{Workspace: t.TempDir()})
	if _, err := r.ExportTimeline(context.Background(), "S1"); err == nil {
		t.Fatal("an export with no harness attached was accepted")
	}
}
