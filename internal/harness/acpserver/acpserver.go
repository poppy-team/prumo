// Package acpserver exposes the Harness as an ACP v1 agent over stdio
// JSON-RPC (Content-Length framed), following
// https://agentclientprotocol.com/protocol/v1/ (GAP-010, decision GAP-032:
// v1 stable surface; v2 draft excluded).
//
// Implemented agent methods: initialize, session/new, session/load,
// session/resume, session/list, session/delete, session/close,
// session/prompt (with session/update streaming), session/cancel
// (notification). Out: authenticate (local trust), elicitation, terminals,
// client fs methods (client-side capabilities), MCP transports beyond the
// accepted-and-stored stdio configs (per-session MCP attach is an explicit
// error, never silent).
package acpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// Backend executes sessions. DaemonBackend (below) wires the harness daemon;
// tests use FakeBackend.
type Backend interface {
	NewSession(ctx context.Context, cwd string) (string, error)
	Load(ctx context.Context, id string) ([]ReplayEvent, error)
	Resume(ctx context.Context, id string) error
	List(ctx context.Context) ([]string, error)
	Delete(ctx context.Context, id string) error
	Close(ctx context.Context, id string) error
	Prompt(ctx context.Context, id, text string, emit func(Update)) (string, error)
	Cancel(ctx context.Context, id string) error
}

// ReplayEvent is one replayable timeline unit for session/load.
type ReplayEvent struct {
	Kind string // "user" | "agent"
	Text string
}

// Update is one session/update notification payload.
type Update struct {
	SessionUpdate string `json:"sessionUpdate"`
	MessageID     string `json:"messageId,omitempty"`
	Content       any    `json:"content,omitempty"`
	ToolCallID    string `json:"toolCallId,omitempty"`
	Title         string `json:"title,omitempty"`
	Status        string `json:"status,omitempty"`
}

func textUpdate(kind, messageID, text string) Update {
	return Update{SessionUpdate: kind, MessageID: messageID,
		Content: map[string]any{"type": "text", "text": text}}
}

// Server serves one stdio connection.
type Server struct {
	Backend Backend
	next    int64
	mu      sync.Mutex
	pending map[int64]chan any
}

func NewServer(backend Backend) *Server {
	return &Server{Backend: backend, pending: map[int64]chan any{}}
}

type rpcMsg struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

func writeFrame(w *bufio.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	frame := append([]byte("Content-Length: "+strconv.Itoa(len(data))+"\r\n\r\n"), data...)
	_, err = w.Write(frame)
	if err != nil {
		return err
	}
	return w.Flush()
}

func readFrame(r *bufio.Reader) ([]byte, error) {
	length := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if v, ok := strings.CutPrefix(line, "Content-Length:"); ok {
			length, _ = strconv.Atoi(strings.TrimSpace(v))
		}
	}
	if length < 0 {
		return nil, fmt.Errorf("missing content-length")
	}
	body := make([]byte, length)
	_, err := io.ReadFull(r, body)
	return body, err
}

// Serve blocks on stdin/stdout until EOF or ctx cancel.
func (s *Server) Serve(ctx context.Context) error {
	return s.serveConn(ctx, bufio.NewReader(os.Stdin), bufio.NewWriter(os.Stdout))
}

func (s *Server) serveConn(ctx context.Context, in *bufio.Reader, out *bufio.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-ctx.Done()
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		body, err := readFrame(in)
		if err != nil {
			return nil // EOF closes the session host-side
		}
		var msg rpcMsg
		if err := json.Unmarshal(body, &msg); err != nil {
			continue
		}
		if msg.ID == nil {
			s.handleNotification(ctx, msg, out)
			continue
		}
		go s.handleRequest(ctx, msg, out)
	}
}

func (s *Server) respond(out *bufio.Writer, id int64, result any, rpcErr any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	msg := map[string]any{"jsonrpc": "2.0", "id": id}
	if rpcErr != nil {
		msg["error"] = rpcErr
	} else {
		msg["result"] = result
	}
	_ = writeFrame(out, msg)
}

func (s *Server) notify(out *bufio.Writer, method string, params any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = writeFrame(out, map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func rpcError(code int, message string) map[string]any {
	return map[string]any{"code": code, "message": message}
}

func (s *Server) handleNotification(ctx context.Context, msg rpcMsg, out *bufio.Writer) {
	if msg.Method != "session/cancel" {
		return
	}
	var p struct {
		SessionID string `json:"sessionId"`
	}
	_ = json.Unmarshal(msg.Params, &p)
	_ = s.Backend.Cancel(ctx, p.SessionID)
}

func (s *Server) handleRequest(ctx context.Context, msg rpcMsg, out *bufio.Writer) {
	id := *msg.ID
	var params map[string]any
	if len(msg.Params) > 0 {
		_ = json.Unmarshal(msg.Params, &params)
	}
	str := func(k string) string {
		v, _ := params[k].(string)
		return v
	}
	switch msg.Method {
	case "initialize":
		s.respond(out, id, map[string]any{
			"protocolVersion": 1,
			"agentCapabilities": map[string]any{
				"loadSession": true,
				"sessionCapabilities": map[string]any{
					"resume": map[string]any{},
					"close":  map[string]any{},
				},
			},
		}, nil)
	case "session/new":
		cwd, _ := params["cwd"].(string)
		if servers, ok := params["mcpServers"].([]any); ok && len(servers) > 0 {
			s.respond(out, id, nil, rpcError(-32602, "per-session mcpServers pending: start the daemon with --mcp instead"))
			return
		}
		sid, err := s.Backend.NewSession(ctx, cwd)
		if err != nil {
			s.respond(out, id, nil, rpcError(-32000, err.Error()))
			return
		}
		s.respond(out, id, map[string]any{"sessionId": sid}, nil)
	case "session/load":
		sid := str("sessionId")
		evs, err := s.Backend.Load(ctx, sid)
		if err != nil {
			s.respond(out, id, nil, rpcError(-32000, err.Error()))
			return
		}
		for i, ev := range evs {
			kind := "agent_message_chunk"
			if ev.Kind == "user" {
				kind = "user_message_chunk"
			}
			s.notify(out, "session/update", map[string]any{
				"sessionId": sid,
				"update":    textUpdate(kind, fmt.Sprintf("msg_%d", i), ev.Text),
			})
		}
		s.respond(out, id, nil, nil)
	case "session/resume":
		if err := s.Backend.Resume(ctx, str("sessionId")); err != nil {
			s.respond(out, id, nil, rpcError(-32000, err.Error()))
			return
		}
		s.respond(out, id, map[string]any{}, nil)
	case "session/list":
		ids, err := s.Backend.List(ctx)
		if err != nil {
			s.respond(out, id, nil, rpcError(-32000, err.Error()))
			return
		}
		out_list := make([]any, 0, len(ids))
		for _, sid := range ids {
			out_list = append(out_list, map[string]any{"sessionId": sid})
		}
		s.respond(out, id, out_list, nil)
	case "session/delete", "session/close":
		if err := s.Backend.Delete(ctx, str("sessionId")); err != nil {
			// close falls back to cancel semantics when no history exists
			if msg.Method == "session/close" {
				_ = s.Backend.Close(ctx, str("sessionId"))
			} else {
				s.respond(out, id, nil, rpcError(-32000, err.Error()))
				return
			}
		}
		s.respond(out, id, map[string]any{}, nil)
	case "session/prompt":
		sid := str("sessionId")
		text := extractText(params["prompt"])
		seq := atomic.AddInt64(&s.next, 1)
		mid := fmt.Sprintf("msg_agent_%d", seq)
		stop, err := s.Backend.Prompt(ctx, sid, text, func(u Update) {
			if u.MessageID == "" {
				u.MessageID = mid
			}
			s.notify(out, "session/update", map[string]any{"sessionId": sid, "update": u})
		})
		if err != nil {
			// Cancellation surfaces as the cancelled stop reason, per spec.
			if ctx.Err() != nil {
				s.respond(out, id, map[string]any{"stopReason": "cancelled"}, nil)
				return
			}
			s.respond(out, id, nil, rpcError(-32000, err.Error()))
			return
		}
		if stop == "" {
			stop = "end_turn"
		}
		s.respond(out, id, map[string]any{"stopReason": stop}, nil)
	default:
		s.respond(out, id, nil, rpcError(-32601, "unknown method "+msg.Method))
	}
}

// extractText concatenates text content blocks of a prompt.
func extractText(prompt any) string {
	blocks, _ := prompt.([]any)
	var parts []string
	for _, b := range blocks {
		m, _ := b.(map[string]any)
		if m["type"] == "text" {
			if t, _ := m["text"].(string); t != "" {
				parts = append(parts, t)
			}
		}
	}
	return strings.Join(parts, "\n")
}
