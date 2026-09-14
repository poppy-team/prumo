// Package aci implements the Coding ACI: structured native tools executed
// through ToolGateway policy + Environment sandbox. Output is bounded.
package aci

import (
	"context"
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
}

// Catalog is the gradual ACI surface (HA5 baseline).
func Catalog() []Tool {
	return []Tool{
		{"fs.read", "bounded file read", "read-only"},
		{"fs.list", "list directory", "read-only"},
		{"fs.search", "search text", "read-only"},
		{"code.symbols", "list symbols (fallback: grep)", "read-only"},
		{"code.diagnostics", "go vet style diagnostics", "read-only"},
		{"edit.patch", "apply unified patch (guarded)", "side-effecting"},
		{"edit.create", "create file", "side-effecting"},
		{"edit.delete", "delete file", "destructive"},
		{"edit.move", "move file", "side-effecting"},
		{"process.exec", "run command in workspace", "side-effecting"},
		{"test.run", "run go test", "idempotent"},
		{"git.status", "git status --short", "read-only"},
		{"git.diff", "git diff (bounded)", "read-only"},
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
		cmd := exec.CommandContext(ctx, "git", "status", "--short")
		cmd.Dir = e.Root
		data, _ := cmd.CombinedOutput()
		out, trunc := bound(string(data), e.OutputMax)
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: out, Truncated: trunc}, nil
	case "git.diff":
		cmd := exec.CommandContext(ctx, "git", "diff", "--stat", "--", ".")
		cmd.Dir = e.Root
		data, _ := cmd.CombinedOutput()
		out, trunc := bound(string(data), e.OutputMax)
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: out, Truncated: trunc}, nil
	case "test.run", "process.exec":
		bin := arg("command")
		if call.Name == "test.run" {
			bin = "go test ./... 2>&1 | head -c 8000"
		}
		if bin == "" {
			return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: "missing command"}, nil
		}
		if err := CheckEgress(e.Egress, bin); err != nil {
			return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: err.Error()}, nil
		}
		cmd := exec.CommandContext(ctx, "sh", "-c", bin)
		cmd.Dir = e.Root
		data, _ := cmd.CombinedOutput()
		out, trunc := bound(string(data), e.OutputMax)
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: out, Truncated: trunc}, nil
	case "code.diagnostics":
		cmd := exec.CommandContext(ctx, "go", "vet", "./...")
		cmd.Dir = e.Root
		data, _ := cmd.CombinedOutput()
		out, trunc := bound(string(data), e.OutputMax)
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: out, Truncated: trunc}, nil
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
