package mcp

import (
	"context"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

type baseTools struct{ calls []string }

func (b *baseTools) Execute(_ context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	b.calls = append(b.calls, call.Name)
	return agent.ToolResult{ToolCallID: call.ID, Output: "base"}, nil
}
func (b *baseTools) KindOf(string) string { return "read-only" }

func TestFanoutRoutes(t *testing.T) {
	clientSide, serverSide := NewPipe()
	defer clientSide.Close()
	defer serverSide.Close()
	fakeServer(t, serverSide)
	base := &baseTools{}
	f := Fanout{Base: base, MCP: Adapter{
		Client: &Client{Transport: clientSide},
		Server: serverDescriptorForTest(),
	}}
	res, err := f.Execute(context.Background(), agent.ToolCall{ID: "c1", Name: "mcp.echo"})
	if err != nil || res.Output != "echo-ok" || len(base.calls) != 0 {
		t.Fatalf("mcp call misrouted: %+v %v", res, err)
	}
	res, err = f.Execute(context.Background(), agent.ToolCall{ID: "c2", Name: "fs.read"})
	if err != nil || res.Output != "base" || len(base.calls) != 1 {
		t.Fatalf("base call misrouted: %+v %v", res, err)
	}
	if f.KindOf("mcp.echo") != "side-effecting" {
		t.Fatalf("mcp default kind must be conservative: %s", f.KindOf("mcp.echo"))
	}
	specs, err := f.Specs(context.Background())
	if err != nil || len(specs) != 1 || specs[0].Name != "mcp.echo" {
		t.Fatalf("fanout specs failed: %+v %v", specs, err)
	}
}
