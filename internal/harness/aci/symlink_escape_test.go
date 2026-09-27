package aci

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

// The executor used to check containment with filepath.Rel alone. Every test
// here failed against that check: a symlink reads as inside the workspace and
// resolves outside it, and Rel never asks the filesystem anything.

func TestExecutorRejectsSymlinkEscapeOnRead(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("classified"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "innocent.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	e := New(root)
	res, _ := e.Execute(t.Context(), agent.ToolCall{Name: "fs.read", Arguments: map[string]any{"path": "innocent.txt"}})
	if res.ExitCode == 0 {
		t.Fatalf("reading through a symlink out of the workspace must fail; got %q", res.Output)
	}
}

func TestExecutorRejectsSymlinkEscapeOnCreate(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	e := New(root)
	res, _ := e.Execute(t.Context(), agent.ToolCall{
		Name:      "edit.create",
		Arguments: map[string]any{"path": "escape/new.txt", "content": "planted"},
	})
	if res.ExitCode == 0 {
		t.Fatal("creating a file through a symlinked parent out of the workspace must fail")
	}
	if _, err := os.Stat(filepath.Join(outside, "new.txt")); err == nil {
		t.Fatal("the file was written outside the workspace anyway")
	}
}

func TestExecutorRejectsSymlinkEscapeOnDelete(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "keep.txt")
	if err := os.WriteFile(secret, []byte("keep"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "innocent.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	e := New(root)
	res, _ := e.Execute(t.Context(), agent.ToolCall{Name: "edit.delete", Arguments: map[string]any{"path": "innocent.txt"}})
	if res.ExitCode == 0 {
		t.Fatal("deleting through a symlink out of the workspace must fail")
	}
	if _, err := os.Stat(secret); err != nil {
		t.Fatal("the file outside the workspace was deleted anyway")
	}
}

func TestExecutorRejectsParentTraversalOnMoveDestination(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	src := filepath.Join(root, "src.txt")
	if err := os.WriteFile(src, []byte("data"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	e := New(root)
	res, _ := e.Execute(t.Context(), agent.ToolCall{
		Name: "edit.move",
		Arguments: map[string]any{
			"from": "src.txt",
			"to":   filepath.Join("..", filepath.Base(outside), "stolen.txt"),
		},
	})
	if res.ExitCode == 0 {
		t.Fatal("moving to a parent-relative destination must fail")
	}
}

func TestExecutorStillReadsAndWritesOrdinaryPaths(t *testing.T) {
	// The containment test must not become a blanket denial.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	e := New(root)

	created, _ := e.Execute(t.Context(), agent.ToolCall{
		Name:      "edit.create",
		Arguments: map[string]any{"path": "pkg/file.txt", "content": "hello"},
	})
	if created.ExitCode != 0 {
		t.Fatalf("creating an ordinary file must succeed: %s", created.Error)
	}
	read, _ := e.Execute(t.Context(), agent.ToolCall{Name: "fs.read", Arguments: map[string]any{"path": "pkg/file.txt"}})
	if read.ExitCode != 0 || read.Output != "hello" {
		t.Fatalf("reading it back must succeed, got %q (%s)", read.Output, read.Error)
	}
}
