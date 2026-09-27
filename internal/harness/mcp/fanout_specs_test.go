package mcp

import (
	"context"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

// Fanout.Specs returned only the MCP tools, so a run wired through it told the
// model about the servers and nothing about the tools that edit the workspace
// (GAP-114, and the same class of drift as GAP-158 on the registry side).

type listingExecutor struct {
	specs []agent.ToolSpec
}

func (l *listingExecutor) Execute(_ context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	return agent.ToolResult{ToolCallID: call.ID}, nil
}
func (l *listingExecutor) KindOf(string) string      { return "read-only" }
func (l *listingExecutor) OperationOf(string) string { return "" }
func (l *listingExecutor) Specs() []agent.ToolSpec   { return l.specs }

func TestTheFanoutDescribesTheNativeTools(t *testing.T) {
	executor := &listingExecutor{specs: []agent.ToolSpec{{Name: "fs.read"}, {Name: "edit.create"}}}
	fanout := Fanout{Base: executor}

	specs, err := fanout.Specs(context.Background())
	if err != nil {
		t.Fatalf("a fanout with no server must still describe its base: %v", err)
	}
	if len(specs) != 2 {
		t.Fatalf("the native tools must be described, got %+v", specs)
	}
}

func TestAFailingServerDoesNotTakeTheNativeToolsDown(t *testing.T) {
	// The client is nil, so listing it fails. A run must still be told what it can
	// do: failing the whole surface because one server is down would leave a model
	// unable to edit anything while the workspace tools are perfectly available.
	executor := &listingExecutor{specs: []agent.ToolSpec{{Name: "fs.read"}}}
	fanout := Fanout{Base: executor, MCP: Adapter{}}

	specs, err := fanout.Specs(context.Background())
	if err != nil {
		t.Fatalf("a server that cannot list must not fail the surface: %v", err)
	}
	if len(specs) != 1 || specs[0].Name != "fs.read" {
		t.Fatalf("the native tools must survive a failing server listing, got %+v", specs)
	}
}

func TestAFanoutWithNothingToDescribeReportsNothing(t *testing.T) {
	fanout := Fanout{}
	specs, err := fanout.Specs(context.Background())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(specs) != 0 {
		t.Fatalf("expected nothing, got %+v", specs)
	}
}
