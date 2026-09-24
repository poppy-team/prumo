package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/toolgateway"
)

// The adapter classified by allowlist: a name on the read-only list was
// ReadOnly and everything else was SideEffecting. Destructive — the kind the
// gateway vetoes outright for an untrusted server — was emitted by nothing, so
// the veto was a branch no input could reach. A server exposing
// `delete_repository` was gated like a tool that renames a file (GAP-157).

// withAnnotations serves one tool list carrying the given annotations.
func withAnnotations(t *testing.T, tool Tool) *PipeTransport {
	t.Helper()
	mine, theirs := NewPipe()
	go func() {
		for {
			raw, err := theirs.Receive(context.Background())
			if err != nil {
				return
			}
			var req Request
			if json.Unmarshal(raw, &req) != nil {
				continue
			}
			if req.ID == 0 {
				continue
			}
			resp := Response{JSONRPC: "2.0", ID: req.ID}
			switch req.Method {
			case "initialize":
				resp.Result = map[string]any{
					"protocolVersion": protocolVersion,
					"capabilities":    map[string]any{"tools": map[string]any{}},
					"serverInfo":      map[string]any{"name": "fake", "version": "1"},
				}
			case "tools/list":
				resp.Result = map[string]any{"tools": []Tool{tool}}
			default:
				resp.Result = map[string]any{"content": []any{map[string]any{"type": "text", "text": "did it"}}}
			}
			data, _ := json.Marshal(resp)
			_ = theirs.Send(context.Background(), data)
		}
	}()
	t.Cleanup(func() { _ = mine.Close() })
	return mine
}

func TestAServerDeclaredDestructiveToolIsClassifiedDestructive(t *testing.T) {
	mine := withAnnotations(t, Tool{
		Name: "delete_repository", Description: "delete everything",
		Annotations: &ToolAnnotations{DestructiveHint: true},
	})
	a := Adapter{
		Client: &Client{Transport: mine},
		Server: toolgateway.MCPServerDescriptor{ID: "srv", Trust: "untrusted"},
	}
	if got := a.KindOf("delete_repository"); got != string(toolgateway.Destructive) {
		t.Fatalf("a tool the server declared destructive was classified %q", got)
	}
}

func TestTheDestructiveVetoIsReachable(t *testing.T) {
	mine := withAnnotations(t, Tool{
		Name: "drop_everything", Description: "delete everything",
		Annotations: &ToolAnnotations{DestructiveHint: true},
	})
	a := Adapter{
		Client: &Client{Transport: mine},
		Server: toolgateway.MCPServerDescriptor{ID: "srv", Trust: "untrusted"},
	}
	res, err := a.Execute(context.Background(), agent.ToolCall{
		ID: "c1", Name: "drop_everything", Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK() {
		t.Fatalf("an untrusted server ran a destructive tool it declared destructive: %+v", res)
	}
	if res.Error == "" {
		t.Fatal("the refusal says nothing about why")
	}
}

func TestAToolWithNoHintsIsNotAssumedSafe(t *testing.T) {
	mine := withAnnotations(t, Tool{Name: "do_something", Description: "no hints"})
	a := Adapter{
		Client:   &Client{Transport: mine},
		Server:   toolgateway.MCPServerDescriptor{ID: "srv", Trust: "untrusted"},
		ReadOnly: []string{},
	}
	// No hint is not a claim of safety. SideEffecting is the honest reading and
	// ReadOnly would be the permissive direction in which to be wrong.
	if got := a.KindOf("do_something"); got != string(toolgateway.SideEffecting) {
		t.Fatalf("a tool with no annotations was classified %q", got)
	}
}

func TestAServerDeclaredReadOnlyToolIsClassifiedReadOnly(t *testing.T) {
	mine := withAnnotations(t, Tool{
		Name: "list_files", Annotations: &ToolAnnotations{ReadOnlyHint: true},
	})
	a := Adapter{
		Client: &Client{Transport: mine},
		Server: toolgateway.MCPServerDescriptor{ID: "srv", Trust: "untrusted"},
	}
	if got := a.KindOf("list_files"); got != string(toolgateway.ReadOnly) {
		t.Fatalf("a tool the server declared read-only was classified %q", got)
	}
}

func TestAnOperatorCanNameADestructiveToolForAServerThatDeclaresNothing(t *testing.T) {
	mine := withAnnotations(t, Tool{Name: "rm_rf"})
	a := Adapter{
		Client:      &Client{Transport: mine},
		Server:      toolgateway.MCPServerDescriptor{ID: "srv", Trust: "untrusted"},
		Destructive: []string{"rm_rf"},
	}
	if got := a.KindOf("rm_rf"); got != string(toolgateway.Destructive) {
		t.Fatalf("an operator-named destructive tool was classified %q", got)
	}
}
