package lsp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/repomap"
)

// fakeServer answers initialize + workspace/symbol with framing.
func fakeServer(t *testing.T, server *PipeTransport) {
	t.Helper()
	go func() {
		for {
			raw, err := server.Receive(context.Background())
			if err != nil {
				return
			}
			// Pipe transport is unfamed JSON; wrap in frame for the client.
			var msg struct {
				ID     int64  `json:"id"`
				Method string `json:"method"`
			}
			body := raw
			if i := indexOfDoubleCRLF(raw); i >= 0 {
				body = raw[i:]
			}
			if err := json.Unmarshal(body, &msg); err != nil {
				return
			}
			var result any
			switch msg.Method {
			case "initialize":
				result = map[string]any{"capabilities": map[string]any{}}
			case "textDocument/hover":
				result = map[string]any{"contents": map[string]any{"value": "func NewRunner() *Runner"}}
			case "textDocument/definition":
				result = map[string]any{"uri": "file:///work/runtime.go", "range": map[string]any{"start": map[string]any{"line": 55}}}
			case "workspace/symbol":
				result = []any{map[string]any{
					"name": "NewRunner", "kind": 12,
					"location": map[string]any{
						"uri":   "file:///work/runtime.go",
						"range": map[string]any{"start": map[string]any{"line": 41}}},
				}}
			default:
				result = nil
			}
			resp, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": msg.ID, "result": result})
			_ = server.Send(context.Background(), resp)
		}
	}()
}

func indexOfDoubleCRLF(b []byte) int {
	for i := 0; i+3 < len(b); i++ {
		if b[i] == '\r' && b[i+1] == '\n' && b[i+2] == '\r' && b[i+3] == '\n' {
			return i + 4
		}
	}
	return -1
}

func TestWorkspaceSymbolsOverPipe(t *testing.T) {
	clientSide, serverSide := NewPipe()
	fakeServer(t, serverSide)
	c := &Client{Transport: clientSide, Root: "/work", Timeout: 5 * time.Second}
	syms, err := c.WorkspaceSymbols(context.Background(), "runner")
	if err != nil {
		t.Fatal(err)
	}
	if len(syms) != 1 || syms[0].Name != "NewRunner" || syms[0].Line != 42 || syms[0].Provider != "lsp" {
		t.Fatalf("bad symbols: %+v", syms)
	}
}

func TestFallbackProvider(t *testing.T) {
	idx := repomap.Map{Modules: []repomap.Module{{Path: ".", Symbols: []repomap.Symbol{
		{Name: "NewRunner", Kind: "func", File: "runtime.go", Line: 42},
	}}}}
	fb := FallbackProvider{Index: idx}
	if !fb.Available() {
		t.Fatal("fallback always available")
	}
	got, err := fb.WorkspaceSymbols(context.Background(), "runner")
	if err != nil || len(got) != 1 || got[0].Provider != "repomap" {
		t.Fatalf("fallback failed: %+v %v", got, err)
	}
}

func TestSelectFallsBackWithoutServer(t *testing.T) {
	p := Select(t.TempDir(), "cobol", repomap.Map{})
	if !p.Available() || !strings.Contains(p.Describe(), "repomap") {
		t.Fatalf("must fall back: %s", p.Describe())
	}
}
