package aci

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

func initGitRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s %v", args, out, err)
		}
	}
	run("init", "-q")
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		run("add", name)
	}
	run("commit", "-qm", "seed")
	return dir
}

func TestEditPatchApplies(t *testing.T) {
	dir := initGitRepo(t, map[string]string{"a.txt": "one\ntwo\nthree\n"})
	e := New(dir)
	diff := "--- a/a.txt\n+++ b/a.txt\n@@ -1,3 +1,3 @@\n one\n-two\n+TWO\n three\n"
	res, err := e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "edit.patch", Arguments: map[string]any{"patch": diff}})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("patch failed: %+v %v", res, err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "a.txt"))
	if string(data) != "one\nTWO\nthree\n" {
		t.Fatalf("bad result: %q", data)
	}
	// Re-apply must fail, not duplicate: git apply is not idempotent.
	res, _ = e.Execute(context.Background(), agent.ToolCall{ID: "c2", Name: "edit.patch", Arguments: map[string]any{"patch": diff}})
	if res.ExitCode == 0 {
		t.Fatal("re-apply must fail")
	}
}

func TestEditPatchBadContextAtomic(t *testing.T) {
	dir := initGitRepo(t, map[string]string{"a.txt": "one\nCHANGED\nthree\n"})
	e := New(dir)
	diff := "--- a/a.txt\n+++ b/a.txt\n@@ -1,3 +1,3 @@\n one\n-two\n+TWO\n three\n"
	res, _ := e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "edit.patch", Arguments: map[string]any{"patch": diff}})
	if res.ExitCode == 0 {
		t.Fatal("mismatched context must fail")
	}
	data, _ := os.ReadFile(filepath.Join(dir, "a.txt"))
	if string(data) != "one\nCHANGED\nthree\n" {
		t.Fatalf("failed patch must leave file untouched: %q", data)
	}
}

func TestEditPatchBaseRevGuard(t *testing.T) {
	dir := initGitRepo(t, map[string]string{"a.txt": "one\n"})
	e := New(dir)
	diff := "--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-one\n+two\n"
	res, _ := e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "edit.patch",
		Arguments: map[string]any{"patch": diff, "base_rev": "deadbeef"}})
	if res.ExitCode == 0 || !strings.Contains(res.Error, "base revision moved") {
		t.Fatalf("stale base must be refused: %+v", res)
	}
}

func TestEditDeleteMove(t *testing.T) {
	dir := initGitRepo(t, map[string]string{"a.txt": "x", "sub/b.txt": "y"})
	e := New(dir)
	if res, _ := e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "edit.move", Arguments: map[string]any{"from": "a.txt", "to": "sub/moved.txt"}}); res.ExitCode != 0 {
		t.Fatalf("move failed: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(dir, "sub", "moved.txt")); err != nil {
		t.Fatal("moved file missing")
	}
	if res, _ := e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "edit.delete", Arguments: map[string]any{"path": "sub/moved.txt"}}); res.ExitCode != 0 {
		t.Fatalf("delete failed: %+v", res)
	}
	if res, _ := e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "edit.delete", Arguments: map[string]any{"path": "../escape"}}); res.ExitCode == 0 {
		t.Fatal("delete escape must be blocked")
	}
	if res, _ := e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "edit.delete", Arguments: map[string]any{"path": "sub"}}); res.ExitCode == 0 {
		t.Fatal("directory delete must be refused")
	}
}

func TestSearchBackendAgnostic(t *testing.T) {
	dir := initGitRepo(t, map[string]string{"a.txt": "needle in haystack\n"})
	e := New(dir)
	res, _ := e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "fs.search", Arguments: map[string]any{"pattern": "needle"}})
	if res.ExitCode != 0 || !strings.Contains(res.Output, "needle") {
		t.Fatalf("search failed: %+v", res)
	}
}
