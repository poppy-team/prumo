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
}

func (a Adapter) isReadOnly(name string) bool {
	for _, n := range a.ReadOnly {
		if n == name {
			return true
		}
	}
	return false
}

func (a Adapter) descriptorFor(name, desc string) toolgateway.Descriptor {
	kind := toolgateway.SideEffecting
	if a.isReadOnly(name) {
		kind = toolgateway.ReadOnly
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
func (a Adapter) KindOf(name string) string {
	return string(a.descriptorFor(name, "").Kind)
}

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

func (f Fanout) Specs(ctx context.Context) ([]agent.ToolSpec, error) {
	return f.MCP.Specs(ctx)
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
	dec := toolgateway.EvaluateMCP(a.Server, a.descriptorFor(name, ""), target, a.SafeMode)
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
