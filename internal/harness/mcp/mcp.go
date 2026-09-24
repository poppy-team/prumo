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
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/raillen/prumo/internal/harness/childenv"
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

// Notifier is implemented by transports that can deliver a JSON-RPC
// notification without a reply.
//
// It exists because Send/Receive queue pairs a request with its response, and a
// notification is half of neither: queueing one means the next Receive posts the
// notification instead of the request that was actually made, and the caller
// reads the notification's empty response as if it were its own. The client
// falls back to Send when a transport cannot notify separately, which is correct
// for a stream transport where the server is reading messages in order.
type Notifier interface {
	Notify(ctx context.Context, data []byte) error
}

// PipeTransport connects client and fake server over channels.
type PipeTransport struct {
	toServer  chan []byte
	toClient  chan []byte
	closed    int32
	closeOnce sync.Once
	// done is closed by Close and observed by every blocked Send and Receive.
	//
	// Close used to set a flag that only stopped new sends: a reader already
	// blocked on the channel stayed blocked forever, so closing a transport while
	// a round trip was in flight leaked the goroutine. The channel itself cannot
	// be closed instead, because a concurrent Send would panic on a closed
	// channel — a flag that leaks a goroutine beats a flag that crashes the
	// process.
	done chan struct{}
}

func NewPipe() (*PipeTransport, *PipeTransport) {
	a := &PipeTransport{toServer: make(chan []byte, 64), toClient: make(chan []byte, 64), done: make(chan struct{})}
	b := &PipeTransport{toServer: a.toClient, toClient: a.toServer, done: make(chan struct{})}
	return a, b
}

func (p *PipeTransport) Send(_ context.Context, data []byte) error {
	if atomic.LoadInt32(&p.closed) != 0 {
		return fmt.Errorf("transport closed")
	}
	cp := append([]byte{}, data...)
	select {
	case <-p.done:
		return fmt.Errorf("transport closed")
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
	case <-p.done:
		// A close is reported as io.EOF: the peer is gone, which is not a
		// failure of the caller's context and must not be mistaken for one.
		return nil, io.EOF
	case data := <-p.toClient:
		return data, nil
	}
}

func (p *PipeTransport) Close() error {
	p.closeOnce.Do(func() {
		atomic.StoreInt32(&p.closed, 1)
		close(p.done)
	})
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

// Notify posts a notification and throws the reply away.
//
// It cannot go through the pending queue: a queued notification is dequeued in
// place of the next real request, and the caller then reads the notification's
// response as its own. That is the failure the queue cannot express.
func (h *HTTPTransport) Notify(ctx context.Context, data []byte) error {
	h.mu.Lock()
	session := h.session
	client := h.httpClient()
	h.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.URL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if session != "" {
		req.Header.Set("Mcp-Session-Id", session)
	}
	for k, v := range h.Headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("mcp notification rejected: %s", resp.Status)
	}
	return nil
}

// httpClient returns a usable client. A zero HTTPTransport had none, and every
// call on it panicked at the first request rather than reporting that it was
// not configured.
func (h *HTTPTransport) httpClient() *http.Client {
	if h.Client != nil {
		return h.Client
	}
	return http.DefaultClient
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
	// A spawned server sees only what it needs. Without this it inherited the
	// whole parent environment, model API keys included, whatever its own
	// permission policy says (GAP-144).
	cmd.Env = childenv.Build(childenv.Options{})
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

	// initOnce runs the MCP handshake exactly once. `tools/list` was sent as the
	// first message on the wire, and a server that follows the protocol has no
	// session to answer it: initialization is the first exchange, and tools/list
	// before it is an error or an empty result depending on how forgiving the
	// server is (GAP-143).
	initOnce sync.Once
	initErr  error
}

// protocolVersion is the MCP revision this client speaks.
const protocolVersion = "2025-06-18"

// Tool describes one server tool.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Schema      map[string]any `json:"inputSchema,omitempty"`
	// Annotations are the server's own declarations about what a tool does.
	//
	// They are read because they are the only place a server can say what it
	// means. The adapter used to classify by allowlist: a name on the read-only
	// list was ReadOnly and everything else was SideEffecting, so Destructive —
	// the kind the gateway actually vetoes — was never emitted by anything, and
	// the veto was a branch no input could reach (GAP-157).
	Annotations *ToolAnnotations `json:"annotations,omitempty"`
}

// ToolAnnotations are a server's declared behaviour for one tool.
type ToolAnnotations struct {
	// ReadOnlyHint says the tool does not modify anything.
	ReadOnlyHint bool `json:"readOnlyHint,omitempty"`
	// DestructiveHint says the tool may destroy something. It is the hint that
	// matters here: a tool marked destructive is vetoed rather than gated.
	DestructiveHint bool `json:"destructiveHint,omitempty"`
	// IdempotentHint says repeating the call has the same effect as one.
	IdempotentHint bool `json:"idempotentHint,omitempty"`
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
// initialize performs the MCP handshake: the initialize request, then the
// initialized notification the protocol requires before the client may use any
// other method.
//
// It is run once per client and its result is remembered, including its failure.
// A handshake that failed is not retried on every tools/list, because the second
// failure is the same failure and the server may be wedged rather than slow.
func (c *Client) initialize(ctx context.Context) error {
	c.initOnce.Do(func() {
		if c.Transport == nil {
			c.initErr = errors.New("mcp: no transport")
			return
		}
		raw, err := c.roundTrip(ctx, "initialize", map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "prumo", "version": "0.6"},
		})
		if err != nil {
			c.initErr = fmt.Errorf("mcp initialize: %w", err)
			return
		}
		// The server's own version is informational; a mismatch is the server's
		// call to make. The notification is what it is waiting for.
		_ = raw
		if err := c.notify(ctx, "notifications/initialized", map[string]any{}); err != nil {
			c.initErr = fmt.Errorf("mcp initialized: %w", err)
		}
	})
	return c.initErr
}

// notify sends a JSON-RPC notification: a message with no id, to which the
// server sends no reply.
func (c *Client) notify(ctx context.Context, method string, params any) error {
	data, err := json.Marshal(Request{JSONRPC: "2.0", Method: method, Params: params})
	if err != nil {
		return err
	}
	if notifier, ok := c.Transport.(Notifier); ok {
		return notifier.Notify(ctx, data)
	}
	return c.Transport.Send(ctx, data)
}

func (c *Client) List(ctx context.Context) ([]Tool, error) {
	if err := c.initialize(ctx); err != nil {
		return nil, err
	}
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
	if err := c.initialize(ctx); err != nil {
		return "", err
	}
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
