package acpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func ioPipe() (*io.PipeReader, *io.PipeWriter) { return io.Pipe() }

func itoa(n int) string { return strconv.Itoa(n) }

// fakeBackend answers from scripted sessions.
type fakeBackend struct {
	mu       sync.Mutex
	sessions map[string][]ReplayEvent
	prompts  []string
}

func newFake() *fakeBackend {
	return &fakeBackend{sessions: map[string][]ReplayEvent{}}
}

func (f *fakeBackend) NewSession(_ context.Context, _ string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := fmt.Sprintf("sess_%d", len(f.sessions)+1)
	f.sessions[id] = nil
	return id, nil
}

func (f *fakeBackend) Load(_ context.Context, id string) ([]ReplayEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	evs, ok := f.sessions[id]
	if !ok {
		return nil, fmt.Errorf("unknown session %s", id)
	}
	return evs, nil
}

func (f *fakeBackend) Resume(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.sessions[id]; !ok {
		return fmt.Errorf("unknown session %s", id)
	}
	return nil
}

func (f *fakeBackend) List(_ context.Context) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for id := range f.sessions {
		out = append(out, id)
	}
	return out, nil
}

func (f *fakeBackend) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.sessions[id]; !ok {
		return fmt.Errorf("unknown session %s", id)
	}
	delete(f.sessions, id)
	return nil
}

func (f *fakeBackend) Close(_ context.Context, id string) error {
	return f.Delete(context.Background(), id)
}

func (f *fakeBackend) Prompt(_ context.Context, id, text string, emit func(Update)) (string, error) {
	f.mu.Lock()
	if _, ok := f.sessions[id]; !ok {
		f.mu.Unlock()
		return "", fmt.Errorf("unknown session %s", id)
	}
	f.prompts = append(f.prompts, text)
	f.sessions[id] = append(f.sessions[id], ReplayEvent{Kind: "user", Text: text}, ReplayEvent{Kind: "agent", Text: "done: " + text})
	f.mu.Unlock()
	emit(textUpdate("agent_message_chunk", "", "done: "+text))
	return "end_turn", nil
}

func (f *fakeBackend) Cancel(_ context.Context, _ string) error { return nil }

// client drives the server over in-memory pipes and collects responses.
type testClient struct {
	t       *testing.T
	toSrv   *bufio.Writer
	fromSrv *bufio.Reader
	next    int64
}

func newTestClient(t *testing.T, srv *Server) *testClient {
	t.Helper()
	srvInR, srvInW := ioPipe()
	srvOutR, srvOutW := ioPipe()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = srv.serveConn(ctx, bufio.NewReader(srvInR), bufio.NewWriter(srvOutW)) }()
	return &testClient{t: t, toSrv: bufio.NewWriter(srvInW), fromSrv: bufio.NewReader(srvOutR)}
}

func (c *testClient) call(method string, params any) map[string]any {
	c.t.Helper()
	c.next++
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": c.next, "method": method, "params": params})
	_, _ = c.toSrv.WriteString("Content-Length: " + itoa(len(body)) + "\r\n\r\n")
	_, _ = c.toSrv.Write(body)
	_ = c.toSrv.Flush()
	for {
		raw, err := readFrame(c.fromSrv)
		if err != nil {
			c.t.Fatalf("read: %v", err)
		}
		var msg map[string]any
		if err := json.Unmarshal(raw, &msg); err != nil {
			c.t.Fatal(err)
		}
		if _, isNotif := msg["method"]; isNotif {
			continue // notifications interleave; caller drains separately
		}
		return msg
	}
}

func TestInitializeAndSessionFlow(t *testing.T) {
	srv := NewServer(newFake())
	c := newTestClient(t, srv)
	res := c.call("initialize", map[string]any{"protocolVersion": 1})
	result, _ := res["result"].(map[string]any)
	if result["protocolVersion"] != float64(1) {
		t.Fatalf("bad initialize: %v", res)
	}
	caps, _ := result["agentCapabilities"].(map[string]any)
	if caps["loadSession"] != true {
		t.Fatalf("must advertise loadSession: %v", res)
	}
	nr := c.call("session/new", map[string]any{"cwd": "/work", "mcpServers": []any{}})
	sid, _ := nr["result"].(map[string]any)["sessionId"].(string)
	if sid == "" {
		t.Fatalf("no session id: %v", nr)
	}
	pr := c.call("session/prompt", map[string]any{"sessionId": sid, "prompt": []any{map[string]any{"type": "text", "text": "hi"}}})
	if pr["result"].(map[string]any)["stopReason"] != "end_turn" {
		t.Fatalf("bad stop: %v", pr)
	}
	lr := c.call("session/load", map[string]any{"sessionId": sid})
	if lr["result"] != nil {
		t.Fatalf("load must return null result: %v", lr)
	}
	if r := c.call("session/resume", map[string]any{"sessionId": sid}); r["error"] != nil {
		t.Fatalf("resume failed: %v", r)
	}
	if r := c.call("session/close", map[string]any{"sessionId": sid}); r["error"] != nil {
		t.Fatalf("close failed: %v", r)
	}
	if r := c.call("bogus/method", nil); r["error"] == nil {
		t.Fatal("unknown method must error")
	}
	if r := c.call("session/new", map[string]any{"cwd": "/w", "mcpServers": []any{map[string]any{"name": "x"}}}); r["error"] == nil {
		t.Fatal("per-session mcp must fail loudly, not silently")
	}
}

func TestPromptStreamsUpdates(t *testing.T) {
	fb := newFake()
	srv := NewServer(fb)
	c := newTestClient(t, srv)
	nr := c.call("session/new", map[string]any{"cwd": "/w"})
	sid := nr["result"].(map[string]any)["sessionId"].(string)
	// Drain the prompt response plus interleaved notifications manually.
	c.next++
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": c.next,
		"method": "session/prompt",
		"params": map[string]any{"sessionId": sid, "prompt": []any{map[string]any{"type": "text", "text": "ping"}}}})
	_, _ = c.toSrv.WriteString("Content-Length: " + itoa(len(body)) + "\r\n\r\n")
	_, _ = c.toSrv.Write(body)
	_ = c.toSrv.Flush()
	var sawChunk, sawResult bool
	for !(sawChunk && sawResult) {
		raw, err := readFrame(c.fromSrv)
		if err != nil {
			t.Fatal(err)
		}
		var msg map[string]any
		_ = json.Unmarshal(raw, &msg)
		if msg["method"] == "session/update" {
			params, _ := msg["params"].(map[string]any)
			update, _ := params["update"].(map[string]any)
			if update["sessionUpdate"] == "agent_message_chunk" {
				content, _ := update["content"].(map[string]any)
				if strings.Contains(content["text"].(string), "done: ping") {
					sawChunk = true
				}
			}
			continue
		}
		if res, _ := msg["result"].(map[string]any); res["stopReason"] == "end_turn" {
			sawResult = true
		}
	}
	if !sawChunk || !sawResult {
		t.Fatal("expected chunk + result")
	}
}
