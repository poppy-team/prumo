// File mutations: unified-diff patching (via git apply, atomic with
// context validation), contained delete and move. Patching requires a git
// workspace; an optional base_rev guard refuses application over silently
// changed trees. Re-applying an already-applied patch fails instead of
// duplicating — callers needing idempotency must gate on diff/status first.
package aci

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/raillen/prumo/internal/harness/agent"
)

func gitHead(root string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git workspace: %s", root)
	}
	return strings.TrimSpace(string(out)), nil
}

func (e *Executor) editPatch(call agent.ToolCall, arg func(string) string) agent.ToolResult {
	fail := func(format string, a ...any) agent.ToolResult {
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: fmt.Sprintf(format, a...)}
	}
	patch := arg("patch")
	if strings.TrimSpace(patch) == "" {
		return fail("missing patch")
	}
	if _, err := gitHead(e.Root); err != nil {
		return fail("%s", err.Error())
	}
	if want := strings.TrimSpace(arg("base_rev")); want != "" {
		head, err := gitHead(e.Root)
		if err != nil {
			return fail("%s", err.Error())
		}
		if head != want {
			return fail("base revision moved: want %s, have %s", want, head)
		}
	}
	tmp, err := os.CreateTemp("", "prumo-patch-*.diff")
	if err != nil {
		return fail("temp file: %s", err.Error())
	}
	tmpName := tmp.Name()
	_, _ = tmp.WriteString(patch)
	_ = tmp.Close()
	defer os.Remove(tmpName)

	check := exec.Command("git", "apply", "--check", tmpName)
	check.Dir = e.Root
	if out, err := check.CombinedOutput(); err != nil {
		return fail("patch does not apply cleanly: %s", strings.TrimSpace(string(out)))
	}
	apply := exec.Command("git", "apply", "--stat", tmpName)
	apply.Dir = e.Root
	stat, _ := apply.CombinedOutput()
	do := exec.Command("git", "apply", tmpName)
	do.Dir = e.Root
	if out, err := do.CombinedOutput(); err != nil {
		return fail("patch apply failed: %s", strings.TrimSpace(string(out)))
	}
	out, trunc := bound("patched\n"+string(stat), e.OutputMax)
	return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: out, Truncated: trunc}
}

func (e *Executor) editDelete(call agent.ToolCall, arg func(string) string) agent.ToolResult {
	fail := func(format string, a ...any) agent.ToolResult {
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: fmt.Sprintf(format, a...)}
	}
	p, err := e.cleanPath(arg("path"))
	if err != nil {
		return fail("%s", err.Error())
	}
	st, err := os.Stat(p)
	if err != nil {
		return fail("stat: %s", err.Error())
	}
	if st.IsDir() {
		return fail("refusing to delete directory %s (files only)", arg("path"))
	}
	if err := os.Remove(p); err != nil {
		return fail("delete: %s", err.Error())
	}
	return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: "deleted " + arg("path")}
}

func (e *Executor) editMove(call agent.ToolCall, arg func(string) string) agent.ToolResult {
	fail := func(format string, a ...any) agent.ToolResult {
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: fmt.Sprintf(format, a...)}
	}
	from, err := e.cleanPath(arg("from"))
	if err != nil {
		return fail("from: %s", err.Error())
	}
	to, err := e.cleanPath(arg("to"))
	if err != nil {
		return fail("to: %s", err.Error())
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return fail("mkdir: %s", err.Error())
	}
	if err := os.Rename(from, to); err != nil {
		return fail("move: %s", err.Error())
	}
	return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: "moved " + arg("from") + " -> " + arg("to")}
}

// searchText prefers rg, falling back to grep so hermetic hosts without
// ripgrep still search (slower, same bounded contract).
func (e *Executor) searchText(pattern string) (string, bool) {
	if _, err := exec.LookPath("rg"); err == nil {
		cmd := exec.Command("rg", "--no-heading", "--line-number", "--max-count", "50", pattern, e.Root)
		data, _ := cmd.CombinedOutput()
		return bound(string(data), e.OutputMax)
	}
	cmd := exec.Command("grep", "-rn", "-m", "50", "--exclude-dir=.git", "--exclude-dir=node_modules", pattern, e.Root)
	data, _ := cmd.CombinedOutput()
	return bound(string(data), e.OutputMax)
}
