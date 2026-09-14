// MCP client over stdio JSON-RPC (GAP-011 first slice).
//
// Transport decision (GAP-031, recorded): stdlib-only line-delimited
// JSON-RPC behind the ports in this package — no third-party MCP SDK until
// a benchmark justifies one. Swapping the transport later must not change
// the ToolService surface. Servers stay untrusted-by-default: calls pass
// through toolgateway policy before execution.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

// Request is one JSON-RPC request.
type Request struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// Response is one JSON-RPC response.
type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Transport moves framed messages; StdioTransport runs a server process,
// PipeTransport is the in-memory test double.
type Transport interface {
	Send(ctx context.Context, data []byte) error
	Receive(ctx context.Context) ([]byte, error)
	Close() error
}

// PipeTransport connects client and fake server over channels.
type PipeTransport struct {
	toServer  chan []byte
	toClient  chan []byte
	closed    int32
	closeOnce sync.Once
}

func NewPipe() (*PipeTransport, *PipeTransport) {
	a := &PipeTransport{toServer: make(chan []byte, 64), toClient: make(chan []byte, 64)}
	b := &PipeTransport{toServer: a.toClient, toClient: a.toServer}
	return a, b
}

func (p *PipeTransport) Send(_ context.Context, data []byte) error {
	if atomic.LoadInt32(&p.closed) != 0 {
		return fmt.Errorf("transport closed")
	}
	cp := append([]byte{}, data...)
	select {
	case p.toServer <- cp:
		return nil
	default:
		return fmt.Errorf("transport saturated")
	}
}

func (p *PipeTransport) Receive(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case data, ok := <-p.toClient:
		if !ok {
			return nil, fmt.Errorf("transport closed")
		}
		return data, nil
	}
}

func (p *PipeTransport) Close() error {
	p.closeOnce.Do(func() { atomic.StoreInt32(&p.closed, 1) })
	return nil
}

// HTTPTransport speaks Streamable-HTTP-style JSON-RPC (POST per call).
// Session continuity uses the Mcp-Session-Id response header when the
// server provides one; servers without sessions work statelessly. This is
// the documented subset — SSE streams stay future work.
type HTTPTransport struct {
	URL     string
	Headers map[string]string
	Client  *http.Client

	mu      sync.Mutex
	pending [][]byte
	session string
}

func (h *HTTPTransport) timeoutClient() *http.Client {
	if h.Client != nil {
		return h.Client
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (h *HTTPTransport) Send(_ context.Context, data []byte) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pending = append(h.pending, append([]byte{}, data...))
	return nil
}

func (h *HTTPTransport) Receive(ctx context.Context) ([]byte, error) {
	h.mu.Lock()
	if len(h.pending) == 0 {
		h.mu.Unlock()
		return nil, fmt.Errorf("no pending request")
	}
	data := h.pending[0]
	h.pending = h.pending[1:]
	session := h.session
	h.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.URL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range h.Headers {
		req.Header.Set(k, v)
	}
	if session != "" {
		req.Header.Set("Mcp-Session-Id", session)
	}
	resp, err := h.timeoutClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mcp http %d", resp.StatusCode)
	}
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		h.mu.Lock()
		h.session = sid
		h.mu.Unlock()
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (h *HTTPTransport) Close() error { return nil }

// StdioTransport spawns `bin args...` speaking JSONL on stdio.
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  interface{ Write([]byte) (int, error) }
	stdout *bufio.Scanner
	mu     sync.Mutex
}

// Start launches the server process.
func StartStdio(ctx context.Context, bin string, args ...string) (*StdioTransport, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	return &StdioTransport{cmd: cmd, stdin: stdin, stdout: sc}, nil
}

func (s *StdioTransport) Send(_ context.Context, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.stdin.Write(append(append([]byte{}, data...), '\n'))
	return err
}

func (s *StdioTransport) Receive(ctx context.Context) ([]byte, error) {
	type result struct {
		line []byte
		ok   bool
	}
	ch := make(chan result, 1)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.stdout.Scan() {
			ch <- result{append([]byte{}, s.stdout.Bytes()...), true}
			return
		}
		ch <- result{nil, false}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		if !r.ok {
			return nil, fmt.Errorf("server closed stdout")
		}
		return r.line, nil
	}
}

func (s *StdioTransport) Close() error {
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	return nil
}

// Client speaks tools/list + tools/call.
type Client struct {
	Transport Transport
	Timeout   time.Duration
	next      int64
}

// Tool describes one server tool.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Schema      map[string]any `json:"inputSchema,omitempty"`
}

func (c *Client) timeout() time.Duration {
	if c.Timeout <= 0 {
		return 30 * time.Second
	}
	return c.Timeout
}

func (c *Client) roundTrip(ctx context.Context, method string, params any) (any, error) {
	id := atomic.AddInt64(&c.next, 1)
	data, err := json.Marshal(Request{JSONRPC: "2.0", ID: id, Method: method, Params: params})
	if err != nil {
		return nil, err
	}
	rctx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()
	if err := c.Transport.Send(rctx, data); err != nil {
		return nil, err
	}
	raw, err := c.Transport.Receive(rctx)
	if err != nil {
		return nil, err
	}
	var resp Response
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("bad response: %w", err)
	}
	if resp.ID != id {
		return nil, fmt.Errorf("id mismatch: want %d got %d", id, resp.ID)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("server %d: %s", resp.Error.Code, resp.Error.Message)
	}
	return resp.Result, nil
}

// List returns server tools.
func (c *Client) List(ctx context.Context) ([]Tool, error) {
	raw, err := c.roundTrip(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(raw)
	var doc struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc.Tools, nil
}

// Call invokes one tool; result content blocks are concatenated as text.
func (c *Client) Call(ctx context.Context, name string, args map[string]any) (string, error) {
	raw, err := c.roundTrip(ctx, "tools/call", map[string]any{"name": name, "arguments": args})
	if err != nil {
		return "", err
	}
	data, _ := json.Marshal(raw)
	var doc struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", err
	}
	out := ""
	for _, b := range doc.Content {
		out += b.Text
	}
	return out, nil
}
