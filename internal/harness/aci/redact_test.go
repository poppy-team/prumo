package aci

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

func TestOutputsRedacted(t *testing.T) {
	dir := t.TempDir()
	secret := "sk-abcdefghijklmnopqrstuvwx"
	if err := os.WriteFile(filepath.Join(dir, "k.txt"), []byte("key="+secret+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := New(dir)
	res, _ := e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "fs.read", Arguments: map[string]any{"path": "k.txt"}})
	if res.ExitCode != 0 {
		t.Fatalf("read failed: %+v", res)
	}
	if strings.Contains(res.Output, secret) || !strings.Contains(res.Output, "[REDACTED]") {
		t.Fatalf("secret leaked to model surface: %q", res.Output)
	}
}

func TestRedactNilDisables(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "k.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := New(dir)
	e.Redact = nil
	res, _ := e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "fs.read", Arguments: map[string]any{"path": "k.txt"}})
	if res.Output != "x" {
		t.Fatalf("nil redactor must pass through: %q", res.Output)
	}
}
