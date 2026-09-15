package environment

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalEnvironment(t *testing.T) {
	tempDir := t.TempDir()
	env := NewLocalEnvironment(tempDir, true)

	if err := env.Prepare(); err != nil {
		t.Fatalf("prepare failed: %v", err)
	}

	// Safe command inside root
	res, err := env.Execute("echo", []string{"hello-world"}, nil, tempDir)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if !strings.Contains(string(res.Output), "hello-world") {
		t.Fatalf("unexpected output: %s", string(res.Output))
	}

	// Destructive command blocked by safe mode
	_, errDestr := env.Execute("rm", []string{"-rf", "/"}, nil, tempDir)
	if errDestr == nil {
		t.Fatalf("expected destructive command to be blocked in safe mode")
	}

	// Workdir outside project root blocked by safe mode
	otherDir := t.TempDir()
	_, errOut := env.Execute("echo", []string{"outside"}, nil, otherDir)
	if errOut == nil {
		t.Fatalf("expected out-of-root workdir to be blocked in safe mode")
	}

	if err := env.Cleanup(); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
}

func TestWorktreeEnvironment(t *testing.T) {
	// Use the current repo to test worktree isolation
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tempWorktree := filepath.Join(t.TempDir(), "wt-test")
	branchName := "test-wt-branch-d2"

	// Idempotency: a previous interrupted run may have left the branch
	// behind (fixed name, repo-level). Prune before and after so the test
	// never fails on its own residue.
	prune := func() {
		_ = exec.Command("git", "-C", cwd, "worktree", "prune").Run()
		_ = exec.Command("git", "-C", cwd, "branch", "-D", branchName).Run()
	}
	prune()
	t.Cleanup(prune)

	wtEnv := NewWorktreeEnvironment(cwd, tempWorktree, branchName)
	if err := wtEnv.Prepare(); err != nil {
		t.Fatalf("worktree prepare failed: %v", err)
	}

	res, err := wtEnv.Execute("git", []string{"status", "--short"}, nil, tempWorktree)
	if err != nil {
		t.Fatalf("worktree execute git status failed: %v (%s)", err, string(res.Output))
	}

	if err := wtEnv.Cleanup(); err != nil {
		t.Fatalf("worktree cleanup failed: %v", err)
	}
}
