package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// fakeServer answers JSON-RPC over the paired pipe.
func fakeServer(t *testing.T, server *PipeTransport) {
	t.Helper()
	go func() {
		for {
			raw, err := server.Receive(context.Background())
			if err != nil {
				return
			}
			var req Request
			if err := json.Unmarshal(raw, &req); err != nil {
				return
			}
			var resp Response
			resp.JSONRPC = "2.0"
			resp.ID = req.ID
			switch req.Method {
			case "tools/list":
				resp.Result = map[string]any{"tools": []any{map[string]any{"name": "echo", "description": "echoes"}}}
			case "tools/call":
				resp.Result = map[string]any{"content": []any{map[string]any{"type": "text", "text": "echo-ok"}}}
			default:
				resp.Error = &struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				}{Code: -32601, Message: "unknown method"}
			}
			data, _ := json.Marshal(resp)
			_ = server.Send(context.Background(), data)
		}
	}()
}

func TestListAndCall(t *testing.T) {
	clientSide, serverSide := NewPipe()
	defer clientSide.Close()
	defer serverSide.Close()
	fakeServer(t, serverSide)
	c := &Client{Transport: clientSide, Timeout: 5 * time.Second}
	tools, err := c.List(context.Background())
	if err != nil || len(tools) != 1 || tools[0].Name != "echo" {
		t.Fatalf("list failed: %+v %v", tools, err)
	}
	out, err := c.Call(context.Background(), "echo", map[string]any{"x": 1})
	if err != nil || out != "echo-ok" {
		t.Fatalf("call failed: %q %v", out, err)
	}
}

func TestUnknownMethodErrors(t *testing.T) {
	clientSide, serverSide := NewPipe()
	defer clientSide.Close()
	defer serverSide.Close()
	fakeServer(t, serverSide)
	c := &Client{Transport: clientSide, Timeout: 5 * time.Second}
	_, err := c.roundTrip(context.Background(), "nope", nil)
	if err == nil || !strings.Contains(err.Error(), "unknown method") {
		t.Fatalf("expected method error, got %v", err)
	}
}

func TestTimeout(t *testing.T) {
	clientSide, _ := NewPipe()
	defer clientSide.Close()
	c := &Client{Transport: clientSide, Timeout: 50 * time.Millisecond}
	// Nobody answers on the paired pipe: must time out, not hang.
	if _, err := c.List(context.Background()); err == nil {
		t.Fatal("expected timeout")
	}
}
