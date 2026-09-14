package aci

import (
	"context"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

func TestPathContainment(t *testing.T) {
	e := New(t.TempDir())
	res, _ := e.Execute(context.Background(), agent.ToolCall{ID: "c1", Name: "fs.read", Arguments: map[string]any{"path": "../../etc/passwd"}})
	if res.ExitCode == 0 {
		t.Fatal("path escape must be blocked")
	}
}

func TestCatalogKinds(t *testing.T) {
	e := New(t.TempDir())
	if e.KindOf("fs.read") != "read-only" {
		t.Fatal("fs.read must be read-only")
	}
	if e.KindOf("edit.delete") != "destructive" {
		t.Fatal("edit.delete must be destructive")
	}
}
