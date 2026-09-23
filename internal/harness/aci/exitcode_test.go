package aci

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

// These tests are the guard for GAP-097.
//
// Before the fix, every command-returning tool discarded the command's error and
// reported ExitCode: 0. `test.run` additionally piped through `head -c 8000`,
// and a pipeline reports the status of its last command, so even a correctly
// propagated error would have been head's. The practical effect was that a
// failing `go test ./...` produced tool output that looked like success, and the
// strict quality gate — which trusts that exit code — passed it.
//
// The tests below assert the opposite property: a command that fails reports its
// real status, and a command that succeeds reports zero. A gate is only as
// useful as the honesty of the number it reads.

func newTestExecutor(t *testing.T, root string) *Executor {
	t.Helper()
	return &Executor{Root: root, OutputMax: 32 * 1024}
}

func runTool(t *testing.T, exec *Executor, name string, args map[string]string) agent.ToolResult {
	t.Helper()
	arguments := map[string]any{}
	for key, value := range args {
		arguments[key] = value
	}
	res, err := exec.Execute(context.Background(), agent.ToolCall{
		ID:        "call-1",
		Name:      name,
		Arguments: arguments,
	})
	if err != nil {
		t.Fatalf("%s returned a transport error instead of a result: %v", name, err)
	}
	return res
}

func TestProcessExecReportsFailingCommandStatus(t *testing.T) {
	root := t.TempDir()
	exec := newTestExecutor(t, root)

	res := runTool(t, exec, "process.exec", map[string]string{"command": "exit 7"})
	if res.ExitCode != 7 {
		t.Fatalf("a command exiting 7 must report 7, got %d (output %q)", res.ExitCode, res.Output)
	}
}

func TestProcessExecReportsSuccessAsZero(t *testing.T) {
	root := t.TempDir()
	exec := newTestExecutor(t, root)

	res := runTool(t, exec, "process.exec", map[string]string{"command": "exit 0"})
	if res.ExitCode != 0 {
		t.Fatalf("a command exiting 0 must report 0, got %d", res.ExitCode)
	}
}

func TestProcessExecCapturesStderrAndKeepsFailingStatus(t *testing.T) {
	root := t.TempDir()
	exec := newTestExecutor(t, root)

	res := runTool(t, exec, "process.exec", map[string]string{
		"command": "echo boom >&2; exit 3",
	})
	if res.ExitCode != 3 {
		t.Fatalf("expected exit 3, got %d", res.ExitCode)
	}
	if !strings.Contains(res.Output, "boom") {
		t.Fatalf("stderr must reach the caller, got %q", res.Output)
	}
}

func TestTestRunReportsFailingTests(t *testing.T) {
	// A module whose only test fails. `test.run` must not report success for it.
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module failing\n\ngo 1.26.6\n")
	writeFile(t, filepath.Join(root, "broken_test.go"), `package failing

import "testing"

func TestDeliberatelyFails(t *testing.T) {
	t.Fatal("this test is meant to fail")
}
`)
	exec := newTestExecutor(t, root)

	res := runTool(t, exec, "test.run", nil)
	if res.ExitCode == 0 {
		t.Fatalf("a failing go test must not report exit 0; output was %q", res.Output)
	}
	if !strings.Contains(res.Output, "DeliberatelyFails") {
		t.Fatalf("the failing test's name must reach the caller, got %q", res.Output)
	}
}

func TestTestRunReportsPassingTests(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module passing\n\ngo 1.26.6\n")
	writeFile(t, filepath.Join(root, "ok_test.go"), `package passing

import "testing"

func TestPasses(t *testing.T) {}
`)
	exec := newTestExecutor(t, root)

	res := runTool(t, exec, "test.run", nil)
	if res.ExitCode != 0 {
		t.Fatalf("a passing go test must report 0, got %d; output %q", res.ExitCode, res.Output)
	}
}

func TestTestRunDoesNotPipeThroughHead(t *testing.T) {
	// The regression guard for the specific pipeline that caused the false
	// green. A shell pipeline's status is its last command's, so `go test |
	// head` reports head's success regardless of the tests.
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module piped\n\ngo 1.26.6\n")
	writeFile(t, filepath.Join(root, "piped_test.go"), `package piped

import "testing"

func TestFailsWithShortOutput(t *testing.T) {
	t.Fatal("nope")
}
`)
	exec := newTestExecutor(t, root)

	res := runTool(t, exec, "test.run", nil)
	if res.ExitCode == 0 {
		t.Fatalf("test.run must not report success when the tests fail; output %q", res.Output)
	}
}

func TestGitStatusReportsFailureOutsideARepository(t *testing.T) {
	// `git status` outside a repository exits non-zero. Reporting 0 here would
	// let a diff-viewer treat an error as "clean tree".
	root := t.TempDir()
	exec := newTestExecutor(t, root)

	res := runTool(t, exec, "git.status", nil)
	if res.ExitCode == 0 {
		t.Fatalf("git status outside a repository must not report 0; output %q", res.Output)
	}
}

func TestCodeDiagnosticsReportsFailingVet(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module vetfail\n\ngo 1.26.6\n")
	writeFile(t, filepath.Join(root, "bad.go"), `package vetfail

import "fmt"

func Bad() { fmt.Println("%s", 42) }
`)
	exec := newTestExecutor(t, root)

	res := runTool(t, exec, "code.diagnostics", nil)
	if res.ExitCode == 0 {
		t.Fatalf("go vet finding a problem must not report 0; output %q", res.Output)
	}
}

func TestProcessExecReportsMissingCommandArgument(t *testing.T) {
	root := t.TempDir()
	exec := newTestExecutor(t, root)

	res := runTool(t, exec, "process.exec", nil)
	if res.ExitCode == 0 {
		t.Fatal("a missing command must be a failure, not success")
	}
	if res.Error == "" {
		t.Fatal("a missing command must carry an explanation")
	}
}

func TestProcessExecReportsUnknownToolAsFailure(t *testing.T) {
	root := t.TempDir()
	exec := newTestExecutor(t, root)

	res := runTool(t, exec, "does.not.exist", nil)
	if res.ExitCode == 0 {
		t.Fatal("an unknown tool must be a failure, not success")
	}
}

func TestCancelledCommandIsNotReportedAsSuccess(t *testing.T) {
	root := t.TempDir()
	exec := newTestExecutor(t, root)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res, err := exec.Execute(ctx, agent.ToolCall{
		ID:        "call-cancel",
		Name:      "process.exec",
		Arguments: map[string]any{"command": "sleep 5"},
	})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if res.ExitCode == 0 {
		t.Fatalf("a cancelled command must not report 0; got output %q", res.Output)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
