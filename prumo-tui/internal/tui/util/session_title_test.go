package util

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultSessionTitleUsesGitBranchWhenPresent(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("cannot create .git: %v", err)
	}
	headFile := filepath.Join(gitDir, "HEAD")
	if err := os.WriteFile(headFile, []byte("ref: refs/heads/feature/my-agent-tui\n"), 0o644); err != nil {
		t.Fatalf("cannot write HEAD: %v", err)
	}

	title := DefaultSessionTitle(dir)
	if title != "feature/my-agent-tui" {
		t.Fatalf("expected title 'feature/my-agent-tui', got: %q", title)
	}
}

func TestDefaultSessionTitleFallbackDateAndProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cool-project")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("cannot create dir: %v", err)
	}

	title := DefaultSessionTitle(dir)
	today := time.Now().Format("2006-01-02")
	expectedPrefix := today + " - cool-project"
	if title != expectedPrefix {
		t.Fatalf("expected title %q, got: %q", expectedPrefix, title)
	}
}

func TestGitBranchDetachedHead(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("cannot create .git: %v", err)
	}
	headFile := filepath.Join(gitDir, "HEAD")
	if err := os.WriteFile(headFile, []byte("0123456789abcdef\n"), 0o644); err != nil {
		t.Fatalf("cannot write HEAD: %v", err)
	}

	branch := GitBranch(dir)
	if branch != "0123456" {
		t.Fatalf("expected short commit hash '0123456', got: %q", branch)
	}
}

func TestGitBranchEmptyWhenNoGit(t *testing.T) {
	dir := t.TempDir()
	branch := GitBranch(dir)
	if branch != "" {
		t.Fatalf("expected empty branch for non-git dir, got: %q", branch)
	}
}
