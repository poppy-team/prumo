package mcp

import (
	"context"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/toolgateway"
)

func adapterWithFake(t *testing.T, server toolgateway.MCPServerDescriptor) (Adapter, func()) {
	t.Helper()
	clientSide, serverSide := NewPipe()
	fakeServer(t, serverSide)
	c := &Client{Transport: clientSide}
	c.Timeout = 5000000000
	return Adapter{Client: c, Server: server}, func() { clientSide.Close(); serverSide.Close() }
}

func TestAdapterSpecsAndCall(t *testing.T) {
	a, done := adapterWithFake(t, toolgateway.MCPServerDescriptor{ID: "s1", Trust: "untrusted"})
	defer done()
	specs, err := a.Specs(context.Background())
	if err != nil || len(specs) != 1 || specs[0].Name != "mcp.echo" {
		t.Fatalf("specs failed: %+v %v", specs, err)
	}
	res, err := a.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "mcp.echo"})
	if err != nil || res.ExitCode != 0 || res.Output != "echo-ok" {
		t.Fatalf("call failed: %+v %v", res, err)
	}
}

func TestAdapterPolicyDenies(t *testing.T) {
	// Allowlist excludes echo.
	a, done := adapterWithFake(t, toolgateway.MCPServerDescriptor{ID: "s1", AllowedTools: []string{"other"}})
	defer done()
	specs, err := a.Specs(context.Background())
	if err != nil || len(specs) != 0 {
		t.Fatalf("allowlist must filter specs: %+v", err)
	}
	res, _ := a.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "mcp.echo"})
	if res.ExitCode == 0 {
		t.Fatalf("allowlist must deny execution: %+v", res)
	}
	// Scoped server allows matching targets only.
	a2, done2 := adapterWithFake(t, toolgateway.MCPServerDescriptor{ID: "s2", RootScopes: []string{"/work"}})
	defer done2()
	res, _ = a2.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "mcp.echo", Arguments: map[string]any{"path": "/elsewhere/x"}})
	if res.ExitCode == 0 {
		t.Fatalf("out-of-scope target must be denied: %+v", res)
	}
}
