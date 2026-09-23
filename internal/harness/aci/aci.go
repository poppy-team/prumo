// Package aci implements the Coding ACI: structured native tools executed
// through ToolGateway policy + Environment sandbox. Output is bounded.
package aci

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/raillen/prumo/internal/egress"
	"github.com/raillen/prumo/internal/harness/agent"
)

// Tool identifies one native coding tool.
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Kind        string `json:"kind"` // read-only|idempotent|side-effecting|destructive
	// FileOperation is the change this tool makes to a file: created, modified,
	// moved or deleted. Empty means the tool does not report a file change —
	// either because it does not touch files, or because it can touch anything
	// (`process.exec`), in which case naming one would be a guess.
	FileOperation string `json:"file_operation,omitempty"`
}

// Catalog is the gradual ACI surface (HA5 baseline).
func Catalog() []Tool {
	return []Tool{
		{"fs.read", "bounded file read", "read-only", ""},
		{"fs.list", "list directory", "read-only", ""},
		{"fs.search", "search text", "read-only", ""},
		{"code.symbols", "list symbols (fallback: grep)", "read-only", ""},
		{"code.diagnostics", "go vet style diagnostics", "read-only", ""},
		{"edit.patch", "apply unified patch (guarded)", "side-effecting", "modified"},
		{"edit.create", "create file", "side-effecting", "created"},
		{"edit.delete", "delete file", "destructive", "deleted"},
		{"edit.move", "move file", "side-effecting", "moved"},
		{"process.exec", "run command in workspace", "side-effecting", ""},
		{"test.run", "run go test", "idempotent", ""},
		{"git.status", "git status --short", "read-only", ""},
		{"git.diff", "git diff (bounded)", "read-only", ""},
	}
}

// Executor runs ACI tools inside a workspace root with path containment.
type Executor struct {
	Root      string
	OutputMax int
	// Redact scrubs tool outputs before they reach the model. Nil disables
	// (tests); New() installs the default pattern set. Local command
	// execution is network-unrestricted by posture — use the container
	// executor when network denial is required.
	Redact *egress.Redactor
	// Egress gates network-capable tools. Nil preserves the legacy
	// unrestricted posture (documented); set for fail-closed operation.
	Egress *EgressPolicy
}

func New(root string) *Executor {
	abs, _ := filepath.Abs(root)
	return &Executor{Root: abs, OutputMax: 32 * 1024, Redact: egress.MustNewRedactor()}
}

// OperationOf reports the file change a tool makes, if it makes one.
//
// The catalogue is where a tool is defined, so the answer lives here rather
// than in the caller: a scheduler that matched on tool names would be the first
// place outside this file to know the catalogue.
func (e *Executor) OperationOf(name string) string {
	for _, t := range Catalog() {
		if t.Name == name {
			return t.FileOperation
		}
	}
	return ""
}

func (e *Executor) KindOf(name string) string {
	for _, t := range Catalog() {
		if t.Name == name {
			return t.Kind
		}
	}
	return "side-effecting"
}

func (e *Executor) cleanPath(p string) (string, error) {
	if p == "" {
		return e.Root, nil
	}
	abs := p
	if !filepath.IsAbs(p) {
		abs = filepath.Join(e.Root, p)
	}
	abs = filepath.Clean(abs)
	rel, err := filepath.Rel(e.Root, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path escapes workspace: %s", p)
	}
	return abs, nil
}

func bound(s string, max int) (string, bool) {
	if len(s) <= max {
		return s, false
	}
	return s[:max] + "\n…[truncated]", true
}

// Execute runs one normalized ToolCall.
func (e *Executor) Execute(ctx context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	res, err := e.execute(ctx, call)
	if e.Redact != nil {
		res.Output = e.Redact.Redact(res.Output)
		res.Error = e.Redact.Redact(res.Error)
	}
	return res, err
}

// runCommand executes cmd in the workspace root and reports the process's real
// exit status.
//
// The exit code is the command's own, taken from *exec.ExitError. It is not
// derived from whether output was produced and it is not replaced by a constant:
// a gate that trusts this value can only be as honest as the command that
// produced it. GAP-097 recorded that this used to be hardcoded to 0, which let a
// failing `go test ./...` satisfy the strict quality gate.
//
// A command that could not be started (missing binary, permission denied) is
// reported as 127, the conventional shell code for "not found", and a context
// cancellation is 130, matching the shell's convention for SIGINT. Both are
// failures; neither is silently success.
func (e *Executor) runCommand(ctx context.Context, name string, args ...string) (output string, exitCode int) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = e.Root
	data, err := cmd.CombinedOutput()
	if err == nil {
		return string(data), 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return string(data), exitErr.ExitCode()
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return string(data), 124
	}
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return string(data), 130
	}
	return string(data), 127
}

// runShell executes a shell command in the workspace root and reports the real
// exit status, with the same contract as runCommand.
func (e *Executor) runShell(ctx context.Context, script string) (output string, exitCode int) {
	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.Dir = e.Root
	data, err := cmd.CombinedOutput()
	if err == nil {
		return string(data), 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return string(data), exitErr.ExitCode()
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return string(data), 124
	}
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return string(data), 130
	}
	return string(data), 127
}

func (e *Executor) execute(ctx context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	arg := func(k string) string {
		if call.Arguments == nil {
			return ""
		}
		v, _ := call.Arguments[k].(string)
		return v
	}
	switch call.Name {
	case "fs.read":
		p, err := e.cleanPath(arg("path"))
		if err != nil {
			return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: err.Error()}, nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: err.Error()}, nil
		}
		out, trunc := bound(string(data), e.OutputMax)
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: out, Truncated: trunc}, nil
	case "fs.list":
		p, err := e.cleanPath(arg("path"))
		if err != nil {
			return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: err.Error()}, nil
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: err.Error()}, nil
		}
		names := []string{}
		for _, en := range entries {
			names = append(names, en.Name())
		}
		out, trunc := bound(strings.Join(names, "\n"), e.OutputMax)
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: out, Truncated: trunc}, nil
	case "fs.search", "code.symbols":
		pattern := arg("pattern")
		if pattern == "" {
			pattern = arg("query")
		}
		out, trunc := e.searchText(pattern)
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: out, Truncated: trunc}, nil
	case "git.status":
		data, code := e.runCommand(ctx, "git", "status", "--short")
		out, trunc := bound(data, e.OutputMax)
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: code, Output: out, Truncated: trunc}, nil
	case "git.diff":
		data, code := e.runCommand(ctx, "git", "diff", "--stat", "--", ".")
		out, trunc := bound(data, e.OutputMax)
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: code, Output: out, Truncated: trunc}, nil
	case "test.run", "process.exec":
		bin := arg("command")
		if call.Name == "test.run" {
			// No pipe to `head`: a pipeline reports the exit status of its last
			// command, so truncating the output this way replaced the test
			// result with head's. The output is bounded by bound() instead, which
			// leaves the test's own status intact.
			bin = "go test ./... 2>&1"
		}
		if bin == "" {
			return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: "missing command"}, nil
		}
		if err := CheckEgress(e.Egress, bin); err != nil {
			return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: err.Error()}, nil
		}
		data, code := e.runShell(ctx, bin)
		out, trunc := bound(data, e.OutputMax)
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: code, Output: out, Truncated: trunc}, nil
	case "code.diagnostics":
		data, code := e.runCommand(ctx, "go", "vet", "./...")
		out, trunc := bound(data, e.OutputMax)
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: code, Output: out, Truncated: trunc}, nil
	case "edit.patch":
		return e.editPatch(call, arg), nil
	case "edit.delete":
		return e.editDelete(call, arg), nil
	case "edit.move":
		return e.editMove(call, arg), nil
	case "edit.create":
		p, err := e.cleanPath(arg("path"))
		if err != nil {
			return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: err.Error()}, nil
		}
		if err := os.WriteFile(p, []byte(arg("content")), 0o644); err != nil {
			return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: err.Error()}, nil
		}
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: "created " + arg("path")}, nil
	default:
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: "unknown tool " + call.Name}, nil
	}
}
