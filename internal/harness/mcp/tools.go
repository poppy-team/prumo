// MCP tools adapter (GAP-011 remainder): exposes one MCP server's tools
// as a ToolExecutor behind toolgateway policy. Server-declared tools map
// conservatively to side-effecting unless explicitly listed read-only;
// untrusted servers never run destructive tools (policy, not trust).
package mcp

import (
	"context"

	"github.com/raillen/prumo/internal/harness/agent"
	harnessruntime "github.com/raillen/prumo/internal/harness/runtime"
	"github.com/raillen/prumo/internal/toolgateway"
)

// Adapter binds a Client to gateway policy.
type Adapter struct {
	Client   *Client
	Server   toolgateway.MCPServerDescriptor
	ReadOnly []string // tool names safe to mark read-only
	SafeMode bool
	// Destructive names tools that must be vetoed rather than gated, for
	// operators whose server does not declare annotations. It is not the
	// default because a name typed into a config file is a weaker claim than
	// one the server makes about itself.
	Destructive []string
}

// isDestructive reports whether the operator named this tool destructive.
func (a Adapter) isDestructive(name string) bool {
	for _, n := range a.Destructive {
		if n == name {
			return true
		}
	}
	return false
}

func (a Adapter) isReadOnly(name string) bool {
	for _, n := range a.ReadOnly {
		if n == name {
			return true
		}
	}
	return false
}

// descriptorFor classifies one server tool.
//
// The classification is the server's, not a guess. A tool that declares itself
// read-only is read-only; a tool that declares itself destructive is
// destructive, which is the kind the gateway vetoes. Neither hint is taken on
// faith in the permissive direction: a server that claims read-only on a tool
// that writes is the server's problem to detect, not ours to prevent, but a tool
// with no hint at all is treated as side-effecting rather than as safe
// (GAP-157).
func (a Adapter) descriptorFor(name, desc string, annotations *ToolAnnotations) toolgateway.Descriptor {
	kind := toolgateway.SideEffecting
	switch {
	case a.isDestructive(name), annotations != nil && annotations.DestructiveHint:
		kind = toolgateway.Destructive
	case a.isReadOnly(name), annotations != nil && annotations.ReadOnlyHint:
		kind = toolgateway.ReadOnly
	case annotations != nil && annotations.IdempotentHint:
		kind = toolgateway.Idempotent
	}
	return toolgateway.Descriptor{ID: name, Version: 1, Description: desc, Kind: kind, Trust: "untrusted"}
}

// Specs advertises server tools to the model (allowlist-filtered).
func (a Adapter) Specs(ctx context.Context) ([]agent.ToolSpec, error) {
	tools, err := a.Client.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]agent.ToolSpec, 0, len(tools))
	for _, t := range tools {
		if len(a.Server.AllowedTools) > 0 && !allowedName(a.Server.AllowedTools, t.Name) {
			continue
		}
		out = append(out, agent.ToolSpec{Name: "mcp." + t.Name, Description: t.Description, Schema: t.Schema})
	}
	return out, nil
}

func allowedName(list []string, name string) bool {
	for _, n := range list {
		if n == name || n == "*" {
			return true
		}
	}
	return false
}

// KindOf implements ToolExecutor introspection.
//
// It asks the server what the tool is, because the allowlist cannot answer:
// a destructive tool the operator never allowlisted was SideEffecting, and the
// veto the gateway performs on destructive tools had no input that could reach
// it (GAP-157). A server that cannot be asked falls back to the allowlist, and
// the fallback is SideEffecting — not ReadOnly, which would be the permissive
// direction to fail in.
func (a Adapter) KindOf(name string) string {
	return string(a.descriptorFor(name, "", a.annotationsFor(name)).Kind)
}

// annotationsFor asks the server what it declared about one tool.
//
// The list is fetched per call rather than cached: an Adapter is a value, and
// caching would need shared state and a way to invalidate it. tools/list is
// cheap next to a tool call, and a wrong answer here is a wrong veto.
func (a Adapter) annotationsFor(name string) *ToolAnnotations {
	if a.Client == nil {
		return nil
	}
	tools, err := a.Client.List(context.Background())
	if err != nil {
		return nil
	}
	for i := range tools {
		if tools[i].Name == name {
			return tools[i].Annotations
		}
	}
	return nil
}

// OperationOf reports no file change.
//
// An MCP server is somebody else's process: it may write files, but it never
// told us which, and inferring one from a tool name would be a guess dressed as
// a fact. A server that wants its changes reported needs to say so in the
// descriptor, at which point this returns it.
func (a Adapter) OperationOf(string) string { return "" }

// Fanout routes mcp.* calls to the MCP adapter, everything else to Base.
type Fanout struct {
	Base harnessruntime.ToolExecutor
	MCP  Adapter
}

func (f Fanout) Execute(ctx context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	if len(call.Name) > 4 && call.Name[:4] == "mcp." {
		return f.MCP.Execute(ctx, call)
	}
	return f.Base.Execute(ctx, call)
}

func (f Fanout) KindOf(name string) string {
	if len(name) > 4 && name[:4] == "mcp." {
		return f.MCP.KindOf(name)
	}
	return f.Base.KindOf(name)
}

func (f Fanout) OperationOf(name string) string {
	if len(name) > 4 && name[:4] == "mcp." {
		return f.MCP.OperationOf(name)
	}
	return f.Base.OperationOf(name)
}

// Specs reports every tool this fanout can run: the native ones and the MCP
// ones.
//
// It used to return only the MCP specs, so a run wired through the fanout told
// the model about the servers and said nothing about the tools it could actually
// edit the workspace with (GAP-114).
func (f Fanout) Specs(ctx context.Context) ([]agent.ToolSpec, error) {
	out := []agent.ToolSpec{}
	if f.Base != nil {
		if specer, ok := f.Base.(interface{ Specs() []agent.ToolSpec }); ok {
			out = append(out, specer.Specs()...)
		}
	}
	// No client means no server is configured, which is not a failure: a fanout
	// over native tools alone is a normal thing to build. Listing a nil client
	// panics, so it is checked before it is asked.
	if f.MCP.Client == nil {
		return out, nil
	}
	mcpSpecs, err := f.MCP.Specs(ctx)
	if err != nil {
		// A server that cannot list its tools does not take the native ones down
		// with it: the model should still be told what it can do.
		return out, nil
	}
	return append(out, mcpSpecs...), nil
}

// Execute policy-checks then calls through. Target scoping is best-effort
// (args path/dir keys); unscoped calls are evaluated without a target.
func (a Adapter) Execute(ctx context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	name := call.Name
	if len(name) > 4 && name[:4] == "mcp." {
		name = name[4:]
	}
	args := map[string]any{}
	for k, v := range call.Arguments {
		args[k] = v
	}
	target := ""
	for _, k := range []string{"path", "dir", "file", "target"} {
		if v, _ := args[k].(string); v != "" {
			target = v
			break
		}
	}
	dec := toolgateway.EvaluateMCP(a.Server, a.descriptorFor(name, "", a.annotationsFor(name)), target, a.SafeMode)
	if !dec.Allowed {
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: "mcp policy denied: " + dec.Reason}, nil
	}
	out, err := a.Client.Call(ctx, name, args)
	if err != nil {
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: err.Error()}, nil
	}
	if len(out) > 32*1024 {
		out = out[:32*1024] + "\n…[truncated]"
	}
	return agent.ToolResult{ToolCallID: call.ID, ExitCode: 0, Output: out}, nil
}
