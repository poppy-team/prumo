package mcp

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// `tools/list` went out as the first message on the wire. A server that follows
// the protocol has no session to answer it — initialization is the first
// exchange, and a notification tells it the client is ready. A permissive server
// answers anyway; a strict one does not, so a client that works against the
// permissive server and fails against the real one is a client that appears to
// support MCP (GAP-143).

// recordingServer answers the protocol correctly and records what it was sent.
type recordingServer struct {
	mu      sync.Mutex
	methods []string
	refused bool // refuse tools/list before initialize
}

func (s *recordingServer) serve(t *testing.T, client *PipeTransport, halt <-chan struct{}) {
	t.Helper()
	initialized := false
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	// Stop when the client closes, rather than waiting out a timeout: a test
	// that takes twenty seconds to prove a message order is a test nobody runs.
	go func() {
		select {
		case <-ctx.Done():
		case <-halt:
			stop()
		}
		_ = client.Close()
	}()
	recvCtx, recvStop := context.WithTimeout(context.Background(), 20*time.Second)
	defer recvStop()
	for {
		raw, err := client.Receive(recvCtx)
		if err != nil {
			return
		}
		var req struct {
			ID     *int64 `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(raw, &req); err != nil {
			continue
		}
		s.mu.Lock()
		s.methods = append(s.methods, req.Method)
		s.mu.Unlock()

		switch req.Method {
		case "initialize":
			initialized = true
			if req.ID != nil {
				_ = client.Send(ctx, mustJSON(map[string]any{
					"jsonrpc": "2.0", "id": *req.ID,
					"result": map[string]any{
						"protocolVersion": protocolVersion,
						"capabilities":    map[string]any{"tools": map[string]any{}},
						"serverInfo":      map[string]any{"name": "fake", "version": "1"},
					},
				}))
			}
		case "notifications/initialized":
			// A notification has no id and gets no reply.
		case "tools/list":
			if s.refused && !initialized {
				if req.ID != nil {
					_ = client.Send(ctx, mustJSON(map[string]any{
						"jsonrpc": "2.0", "id": *req.ID,
						"error": map[string]any{"code": -32002, "message": "server not initialized"},
					}))
				}
				continue
			}
			if req.ID != nil {
				_ = client.Send(ctx, mustJSON(map[string]any{
					"jsonrpc": "2.0", "id": *req.ID,
					"result": map[string]any{"tools": []map[string]any{{"name": "echo", "description": "echo"}}},
				}))
			}
		default:
			if req.ID != nil {
				_ = client.Send(ctx, mustJSON(map[string]any{
					"jsonrpc": "2.0", "id": *req.ID, "result": map[string]any{},
				}))
			}
		}
	}
}

func (s *recordingServer) seen() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string{}, s.methods...)
}

func mustJSON(v any) []byte {
	data, _ := json.Marshal(v)
	return data
}

func TestTheHandshakeComesBeforeToolsList(t *testing.T) {
	mine, theirs := NewPipe()
	server := &recordingServer{}
	done := make(chan struct{})
	stopServer := make(chan struct{})
	go func() { defer close(done); server.serve(t, theirs, stopServer) }()

	c := &Client{Transport: mine, Timeout: 5 * time.Second}
	tools, err := c.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 || tools[0].Name != "echo" {
		t.Fatalf("tools = %+v, want the server's one tool", tools)
	}
	close(stopServer)
	<-done

	seen := server.seen()
	if len(seen) < 2 || seen[0] != "initialize" || seen[1] != "notifications/initialized" {
		t.Fatalf("the wire order was %v, want initialize then notifications/initialized before tools/list", seen)
	}
	if seen[2] != "tools/list" {
		t.Fatalf("tools/list was sent at position %d, after %v", 2, seen[:2])
	}
}

func TestAServerThatRefusesUninitializedCallsNowWorks(t *testing.T) {
	mine, theirs := NewPipe()
	server := &recordingServer{refused: true}
	done := make(chan struct{})
	stopServer := make(chan struct{})
	go func() { defer close(done); server.serve(t, theirs, stopServer) }()

	c := &Client{Transport: mine, Timeout: 5 * time.Second}
	tools, err := c.List(context.Background())
	if err != nil {
		t.Fatalf("a spec-compliant server refused the client: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("tools = %+v, want one", tools)
	}
	close(stopServer)
	<-done
}

func TestTheHandshakeHappensOnce(t *testing.T) {
	mine, theirs := NewPipe()
	server := &recordingServer{}
	done := make(chan struct{})
	stopServer := make(chan struct{})
	go func() { defer close(done); server.serve(t, theirs, stopServer) }()

	c := &Client{Transport: mine, Timeout: 5 * time.Second}
	for range 3 {
		if _, err := c.List(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	close(stopServer)
	<-done

	initializes := 0
	for _, m := range server.seen() {
		if m == "initialize" {
			initializes++
		}
	}
	if initializes != 1 {
		t.Fatalf("initialize was sent %d times across three calls, want once", initializes)
	}
}

func TestAFailedHandshakeIsNotRetriedIntoTheSameFailure(t *testing.T) {
	mine, _ := NewPipe()
	c := &Client{Transport: mine, Timeout: 100 * time.Millisecond}
	// Nothing is listening: the round trip times out.
	if _, err := c.List(context.Background()); err == nil {
		t.Fatal("expected a handshake failure with no server listening")
	}
	// The remembered failure is returned again rather than re-running a
	// handshake that cannot succeed and burning a timeout on it.
	_, second := c.List(context.Background())
	if second == nil {
		t.Fatal("a second call re-ran a handshake that already failed")
	}
}
