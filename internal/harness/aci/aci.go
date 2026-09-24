// Package aci implements the Coding ACI: structured native tools executed
// through ToolGateway policy + Environment sandbox. Output is bounded.
package aci

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/raillen/prumo/internal/egress"
	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/safepath"
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
	// Schema is the JSON Schema of this tool's arguments. It is what a model is
	// shown, so it has to describe what the handler actually reads: a schema that
	// invents an argument the handler ignores teaches the model to send input
	// that goes nowhere, and one that omits a required argument produces a tool
	// call that fails for a reason the model was never told (GAP-114).
	Schema map[string]any `json:"schema,omitempty"`
}

// object builds an object schema with the given required names and properties.
func object(required []string, props map[string]any) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": required}
}

func stringProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

// Catalog is the gradual ACI surface (HA5 baseline).
//
// The schemas here are the same contract the handlers implement. A tool whose
// handler reads an argument has it in the schema; a tool that requires an
// argument lists it as required.
func Catalog() []Tool {
	return []Tool{
		{
			"fs.read", "read a file as text", "read-only", "",
			object([]string{"path"}, map[string]any{
				"path": stringProp("file to read, relative to the workspace root or absolute inside it"),
			}),
		},
		{
			"fs.list", "list the entries of a directory", "read-only", "",
			object(nil, map[string]any{
				"path": stringProp("directory to list; the workspace root when omitted"),
			}),
		},
		{
			"fs.search", "search the workspace for text", "read-only", "",
			object(nil, map[string]any{
				"pattern": stringProp("text or regular expression to search for"),
				"query":   stringProp("accepted as an alias for pattern"),
			}),
		},
		{
			"code.symbols", "list symbols in the workspace", "read-only", "",
			object(nil, map[string]any{}),
		},
		{
			"code.diagnostics", "run static diagnostics over the workspace", "read-only", "",
			object(nil, map[string]any{}),
		},
		{
			"edit.patch", "apply a unified diff to the workspace", "side-effecting", "modified",
			object([]string{"patch"}, map[string]any{
				"patch":    stringProp("the unified diff to apply"),
				"base_rev": stringProp("optional git revision the patch is expected to apply onto; refuses if HEAD has moved"),
			}),
		},
		{
			"edit.create", "create or overwrite a file", "side-effecting", "created",
			object([]string{"path", "content"}, map[string]any{
				"path":    stringProp("file to create, inside the workspace root"),
				"content": stringProp("full contents of the file"),
			}),
		},
		{
			"edit.delete", "delete a file", "destructive", "deleted",
			object([]string{"path"}, map[string]any{
				"path": stringProp("file to delete; directories are refused"),
			}),
		},
		{
			"edit.move", "move or rename a file", "side-effecting", "moved",
			object([]string{"from", "to"}, map[string]any{
				"from": stringProp("existing file to move"),
				"to":   stringProp("destination path; parent directories are created"),
			}),
		},
		{
			"process.exec", "run a shell command in the workspace", "side-effecting", "",
			object([]string{"command"}, map[string]any{
				"command": stringProp("the command line to run in the workspace root"),
			}),
		},
		{
			"test.run", "run the workspace test suite", "idempotent", "",
			object(nil, map[string]any{
				"command": stringProp("optional override; the workspace test command is used when omitted"),
			}),
		},
		{
			"git.status", "show the working tree status", "read-only", "",
			object(nil, map[string]any{}),
		},
		{
			"git.diff", "show a summary of the working tree diff", "read-only", "",
			object(nil, map[string]any{}),
		},
	}
}

// Specs renders the catalogue as provider-facing tool specifications.
//
// This is the only place a native tool becomes something a model can be told
// about. Without it the request carries no tools at all, whatever the provider
// would do with them (GAP-114).
func Specs() []agent.ToolSpec {
	catalog := Catalog()
	out := make([]agent.ToolSpec, 0, len(catalog))
	for _, t := range catalog {
		out = append(out, agent.ToolSpec{
			Name:        t.Name,
			Description: t.Description,
			Schema:      t.Schema,
		})
	}
	return out
}

// Executor.Specs reports the native tools this executor can run.
func (e *Executor) Specs() []agent.ToolSpec { return Specs() }

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
	// Isolation records what actually contains this executor's tool calls.
	//
	// It exists because nothing else said. A caller that handed a worktree to a
	// child run had no way to tell, later, whether the child ran inside a
	// container or directly in the worktree, and the worktree provider described
	// itself as "git isolation only" while never being consulted at all. A
	// recorded value cannot be mistaken for a boundary nobody checked (GAP-168).
	Isolation SandboxKind
}

// New builds an executor rooted at root with no isolation beyond the root path.
//
// The default is SandboxNone rather than something reassuring. This constructor
// takes only a path, so a caller cannot have chosen containment here, and an
// executor that reported "local-trusted" or "worktree" would be asserting a
// boundary that does not exist. A root directory is a scoping convenience: it
// decides which files a relative path resolves inside, and nothing about what
// happens when a tool runs a command (GAP-168).
func New(root string) *Executor {
	abs, _ := filepath.Abs(root)
	return &Executor{Root: abs, OutputMax: 32 * 1024, Redact: egress.MustNewRedactor(), Isolation: SandboxNone}
}

// NewWithIsolation builds an executor and records what actually contains it.
//
// The kind is the provider's own claim, read from the provider rather than passed
// in separately: a caller naming "container" for a local provider would be able to
// record a boundary it does not have, and a recorded boundary is exactly what a
// later reader trusts.
func NewWithIsolation(root string, provider SandboxProvider) *Executor {
	executor := New(root)
	if provider != nil {
		executor.Isolation = provider.Kind()
	}
	return executor
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

// cleanPath confines p to the workspace root, resolving symlinks before the
// containment test. mustExist distinguishes a read from a create: edit.create
// legitimately targets a path that does not exist yet, but its existing
// ancestors must still be inside the root.
//
// The empty path means the root itself, which is what the directory tools pass.
func (e *Executor) cleanPath(p string, mustExist bool) (string, error) {
	if p == "" {
		return e.Root, nil
	}
	return safepath.Resolve(e.Root, p, mustExist)
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
		p, err := e.cleanPath(arg("path"), true)
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
		p, err := e.cleanPath(arg("path"), true)
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
		p, err := e.cleanPath(arg("path"), false)
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
