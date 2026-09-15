// ACPClient drives an external ACP v1 agent over stdio Content-Length-framed
// JSON-RPC (the wire format of internal/harness/acpserver). It is the
// no-API-key external path: the child process owns its own credentials and
// model loop; Prumo owns Run state, budgets and the normalized event stream.
//
// Scope honesty: one prompt in flight per client; session/update
// notifications arriving during a prompt are collected and available to the
// caller through Events buffered history. There is no async read loop; a
// hung child is bounded by the child's own prompt timeout (the Prumo ACP
// server bounds prompts at 5 minutes) or by context cancellation, which
// sends session/cancel.
package extagent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
)

// ACPClient is the stdio ACP v1 AgentProvider.
type ACPClient struct {
	Command string   // binary to spawn (e.g. "prumo")
	Args    []string // arguments (e.g. ["agent","acp"])
	// Start overrides process creation for tests. It must return a framed
	// JSON-RPC connection and a cleanup func.
	Start func(ctx context.Context) (io.ReadWriteCloser, func(), error)

	mu      sync.Mutex
	rwc     io.ReadWriteCloser
	br      *bufio.Reader
	cleanup func()
	nextID  int
	updates []map[string]any
}

func NewACPClient(command string, args ...string) *ACPClient {
	return &ACPClient{Command: command, Args: args}
}

func (c *ACPClient) Name() string { return "acp" }

func (c *ACPClient) Capabilities(_ context.Context) ([]string, error) {
	return []string{"session"}, nil
}

func (c *ACPClient) start(ctx context.Context) error {
	if c.rwc != nil {
		return nil
	}
	if c.Start != nil {
		rwc, cleanup, err := c.Start(ctx)
		if err != nil {
			return err
		}
		c.rwc, c.cleanup, c.br = rwc, cleanup, bufio.NewReader(rwc)
		return nil
	}
	cmd := exec.CommandContext(ctx, c.Command, c.Args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = errBuffer{}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("acp spawn %s: %w", c.Command, err)
	}
	c.rwc = stdioRWC{r: stdout, w: stdin, close: func() error {
		_ = stdin.Close()
		_ = stdout.Close()
		return cmd.Process.Kill()
	}}
	c.cleanup = func() { _ = cmd.Wait() }
	c.br = bufio.NewReader(c.rwc)
	return nil
}

func (c *ACPClient) CreateSession(ctx context.Context, runID string) (Session, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.start(ctx); err != nil {
		return Session{}, err
	}
	if _, err := c.request(ctx, "initialize", map[string]any{"protocolVersion": 1}); err != nil {
		return Session{}, err
	}
	res, err := c.request(ctx, "session/new", map[string]any{"cwd": "."})
	if err != nil {
		return Session{}, err
	}
	sid, _ := res["sessionId"].(string)
	if sid == "" {
		return Session{}, fmt.Errorf("acp session/new returned no sessionId")
	}
	return Session{ID: sid, Provider: "acp", Status: "open", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}, nil
}

// Send performs one ACP prompt turn. The child runs its loop; completion is
// the prompt response stopReason.
func (c *ACPClient) Send(ctx context.Context, sessionID, message string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rwc == nil {
		return fmt.Errorf("acp send before CreateSession")
	}
	params := map[string]any{
		"sessionId": sessionID,
		"prompt":    []any{map[string]any{"type": "text", "text": message}},
	}
	res, err := c.request(ctx, "session/prompt", params)
	if err != nil {
		if ctx.Err() != nil {
			// Best-effort cancel notification, per ACP v1. Async: a blocked
			// child pipe must never wedge the caller (Send already failed).
			go c.bestEffortNotify("session/cancel", map[string]any{"sessionId": sessionID})
			return ctx.Err()
		}
		return err
	}
	stop, _ := res["stopReason"].(string)
	if stop == "cancelled" {
		return context.Canceled
	}
	return nil
}

// request sends one framed JSON-RPC request and reads frames until its
// response arrives, collecting session/update notifications on the way.
// Caller holds mu.
func (c *ACPClient) request(ctx context.Context, method string, params any) (map[string]any, error) {
	c.nextID++
	id := c.nextID
	if err := c.writeMsg(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
		return nil, err
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		raw, err := readACPPacket(c.br)
		if err != nil {
			return nil, fmt.Errorf("acp read %s: %w", method, err)
		}
		var msg struct {
			ID     *int           `json:"id"`
			Result map[string]any `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			return nil, fmt.Errorf("acp decode: %w", err)
		}
		if msg.Method == "session/update" {
			c.updates = append(c.updates, msg.Params)
			continue
		}
		if msg.ID != nil && *msg.ID == id {
			if msg.Error != nil {
				return nil, fmt.Errorf("acp %s error %d: %s", method, msg.Error.Code, msg.Error.Message)
			}
			return msg.Result, nil
		}
		// Notifications/requests from the agent other than updates are
		// tolerated (ACP permits server→client requests we have not
		// negotiated; ignoring keeps scope honest).
	}
}

func (c *ACPClient) notify(method string, params any) error {
	return c.writeMsg(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

// bestEffortNotify fires a notification without wedging the caller if the
// child stopped reading.
func (c *ACPClient) bestEffortNotify(method string, params any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rwc == nil {
		return
	}
	_ = c.notify(method, params)
}

func (c *ACPClient) writeMsg(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	frame := append([]byte("Content-Length: "+strconv.Itoa(len(data))+"\r\n\r\n"), data...)
	_, err = c.rwc.Write(frame)
	return err
}

func readACPPacket(r *bufio.Reader) ([]byte, error) {
	length := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "Content-Length:"); ok {
			length, err = strconv.Atoi(strings.TrimSpace(v))
			if err != nil {
				return nil, err
			}
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}
	if length <= 0 {
		return nil, fmt.Errorf("acp frame without Content-Length")
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func (c *ACPClient) Events(_ context.Context, sessionID string) (<-chan agent.AgentEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := make(chan agent.AgentEvent, 8+len(c.updates))
	now := time.Now().UTC().Format(time.RFC3339Nano)
	ch <- agent.AgentEvent{ID: "ev-acp-open", RunID: sessionID, Kind: "external.session.open", CreatedAt: now}
	for i, u := range c.updates {
		text, _ := extractACPText(u)
		ch <- agent.AgentEvent{ID: fmt.Sprintf("ev-acp-update-%d", i), RunID: sessionID, Kind: "external.acp.update", Payload: map[string]any{"text": text}, CreatedAt: now}
	}
	close(ch)
	return ch, nil
}

func extractACPText(update map[string]any) (string, bool) {
	u, ok := update["update"].(map[string]any)
	if !ok {
		return "", false
	}
	if content, ok := u["content"].(map[string]any); ok {
		if t, ok := content["text"].(string); ok {
			return t, true
		}
	}
	return "", false
}

func (c *ACPClient) Approve(_ context.Context, _, _ string, _ bool) error { return nil }

func (c *ACPClient) Cancel(ctx context.Context, sessionID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rwc == nil {
		return nil
	}
	return c.notify("session/cancel", map[string]any{"sessionId": sessionID})
}

// Close deletes the session (best effort, async: the child may have stopped
// reading) and terminates the child.
func (c *ACPClient) Close(ctx context.Context, sessionID string) error {
	c.mu.Lock()
	if c.rwc == nil {
		c.mu.Unlock()
		return nil
	}
	rwc := c.rwc
	cleanup := c.cleanup
	c.rwc, c.cleanup, c.br = nil, nil, nil
	c.updates = nil
	c.mu.Unlock()
	go func(w io.Writer) {
		body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "session/delete", "params": map[string]any{"sessionId": sessionID}})
		if err != nil {
			return
		}
		_, _ = w.Write(append([]byte("Content-Length: "+strconv.Itoa(len(body))+"\r\n\r\n"), body...))
	}(rwc)
	if rwc != nil {
		_ = rwc.Close()
	}
	if cleanup != nil {
		cleanup()
	}
	return nil
}

// stdioRWC joins process pipes into one ReadWriteCloser.
type stdioRWC struct {
	r     io.Reader
	w     io.Writer
	close func() error
}

func (s stdioRWC) Read(p []byte) (int, error)  { return s.r.Read(p) }
func (s stdioRWC) Write(p []byte) (int, error) { return s.w.Write(p) }
func (s stdioRWC) Close() error                { return s.close() }

// errBuffer swallows child stderr without allocation pressure.
type errBuffer struct{}

func (errBuffer) Write(p []byte) (int, error) { return len(p), nil }
