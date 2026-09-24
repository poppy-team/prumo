// Package extagent implements the AgentProvider boundary: external runtimes
// (Codex app-server, OpenCode server, ACP-compatible agents) own their loop;
// Prumo owns Run/Goal/Task, budgets, normalized events, review and Handoff.
// External state never becomes canonical Prumo state.
package extagent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
)

// Session is the normalized external session handle.
type Session struct {
	ID        string `json:"id"`
	Provider  string `json:"provider"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// Provider is the canonical AgentProvider contract.
type Provider interface {
	Name() string
	Capabilities(ctx context.Context) ([]string, error)
	CreateSession(ctx context.Context, runID string) (Session, error)
	Send(ctx context.Context, sessionID, message string) error
	Events(ctx context.Context, sessionID string) (<-chan agent.AgentEvent, error)
	Approve(ctx context.Context, sessionID, requestID string, approve bool) error
	Cancel(ctx context.Context, sessionID string) error
	Close(ctx context.Context, sessionID string) error
}

// ---- OpenCode server adapter (HTTP) ----

// OpenCodeServer talks to an `opencode serve` instance using the real
// server API as measured against opencode 1.18.30:
//
//	POST   /session {title?}              → {id,...}
//	GET    /session                       → [{id,...}]
//	GET    /session/:id/message           → [...]
//	POST   /session/:id/message {parts:[{type,text}]} → SSE (model call)
//	POST   /session/:id/abort             → true
//	POST   /session/:id/permissions/:perID {response: once|always|reject}
//	DELETE /session/:id                   → true
//	GET    /event                         → SSE {id,type,properties}
//
// Send() invokes the model (cost/side effects) and is only used with
// explicit approval; lifecycle ops are side-effect free.
type OpenCodeServer struct {
	BaseURL string
	Client  *http.Client
}

func NewOpenCodeServer(baseURL string) *OpenCodeServer {
	return &OpenCodeServer{BaseURL: strings.TrimRight(baseURL, "/"), Client: &http.Client{Timeout: 30 * time.Second}}
}

func (o *OpenCodeServer) Name() string { return "opencode" }

func (o *OpenCodeServer) Capabilities(_ context.Context) ([]string, error) {
	return []string{"session", "events", "permissions", "usage", "cancel", "resume"}, nil
}

func (o *OpenCodeServer) postJSON(ctx context.Context, path string, payload any) (*http.Response, error) {
	var body *strings.Reader
	if payload != nil {
		data, _ := json.Marshal(payload)
		body = strings.NewReader(string(data))
	} else {
		body = strings.NewReader("")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return o.Client.Do(req)
}

func readError(resp *http.Response) error {
	defer resp.Body.Close()
	var doc struct {
		Tag     string `json:"_tag"`
		Message string `json:"message"`
	}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	_ = json.Unmarshal(data, &doc)
	msg := strings.TrimSpace(doc.Message)
	if msg == "" {
		msg = strings.TrimSpace(string(data))
	}
	if doc.Tag != "" {
		return fmt.Errorf("opencode http %d %s: %s", resp.StatusCode, doc.Tag, msg)
	}
	return fmt.Errorf("opencode http %d: %s", resp.StatusCode, msg)
}

func (o *OpenCodeServer) CreateSession(ctx context.Context, runID string) (Session, error) {
	resp, err := o.postJSON(ctx, "/session", map[string]any{"title": "prumo-" + runID})
	if err != nil {
		return Session{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Session{}, readError(resp)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Session{}, err
	}
	if out.ID == "" {
		return Session{}, fmt.Errorf("opencode created session without id")
	}
	return Session{ID: out.ID, Provider: "opencode", Status: "open", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}, nil
}

// ListSessions returns server-side sessions (newest last, as served).
func (o *OpenCodeServer) ListSessions(ctx context.Context) ([]Session, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.BaseURL+"/session", nil)
	if err != nil {
		return nil, err
	}
	resp, err := o.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readError(resp)
	}
	var raw []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]Session, 0, len(raw))
	for _, s := range raw {
		out = append(out, Session{ID: s.ID, Provider: "opencode", Status: "open"})
	}
	return out, nil
}

// Send delivers a user message; the server runs the model (cost). The SSE
// response is drained so the turn completes server-side before returning.
func (o *OpenCodeServer) Send(ctx context.Context, sessionID, message string) error {
	resp, err := o.postJSON(ctx, "/session/"+sessionID+"/message",
		map[string]any{"parts": []any{map[string]any{"type": "text", "text": message}}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readError(resp)
	}
	// Drain the SSE turn stream; individual deltas belong to Events().
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	return sc.Err()
}

// Events streams server events, forwarding those scoped to sessionID.
// Session scoping reads properties.sessionID (server shape); unscopable
// server-level events stay server-side and are not misattributed.
func (o *OpenCodeServer) Events(ctx context.Context, sessionID string) (<-chan agent.AgentEvent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.BaseURL+"/event", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := o.Client.Do(req)
	if err != nil {
		return nil, err
	}
	ch := make(chan agent.AgentEvent, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 1024*1024), 1024*1024)
		for sc.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}
			line := strings.TrimSpace(sc.Text())
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			var ev struct {
				ID         string         `json:"id"`
				Type       string         `json:"type"`
				Properties map[string]any `json:"properties"`
			}
			if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), &ev); err != nil || ev.Type == "" {
				continue
			}
			if ev.Type == "server.connected" {
				continue
			}
			if sid := eventSession(ev.Properties); sid != "" && sid != sessionID {
				continue
			}
			id := ev.ID
			if id == "" {
				id = "evt-" + ev.Type
			}
			ch <- agent.AgentEvent{ID: id, RunID: sessionID, Kind: "external." + ev.Type, Payload: map[string]any{"server_type": ev.Type}, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
		}
	}()
	return ch, nil
}

// eventSession extracts the owning session from server event properties
// across the shapes observed/tolerated (server omits it on global events).
func eventSession(props map[string]any) string {
	for _, k := range []string{"sessionID", "sessionId", "session_id"} {
		if v, _ := props[k].(string); v != "" {
			return v
		}
	}
	if info, ok := props["info"].(map[string]any); ok {
		if v, _ := info["sessionID"].(string); v != "" {
			return v
		}
	}
	return ""
}

func (o *OpenCodeServer) Approve(ctx context.Context, sessionID, requestID string, approve bool) error {
	response := "reject"
	if approve {
		response = "once"
	}
	resp, err := o.postJSON(ctx, "/session/"+sessionID+"/permissions/"+requestID, map[string]any{"response": response})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readError(resp)
	}
	return nil
}

func (o *OpenCodeServer) Cancel(ctx context.Context, sessionID string) error {
	resp, err := o.postJSON(ctx, "/session/"+sessionID+"/abort", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readError(resp)
	}
	return nil
}

func (o *OpenCodeServer) Close(ctx context.Context, sessionID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, o.BaseURL+"/session/"+sessionID, nil)
	if err != nil {
		return err
	}
	resp, err := o.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readError(resp)
	}
	return nil
}

// errUnsupported says an adapter cannot do what was asked, and names the
// capability.
//
// A nil from a method that did nothing is read as success, which is how an
// adapter that cannot cancel a session tells a caller it cancelled one
// (GAP-142).
func errUnsupported(provider, capability string) error {
	return fmt.Errorf("extagent: %s does not implement %q", provider, capability)
}

// SessionLifecycle is the optional rich contract beyond Provider: session
// resume verification and usage accounting. OpenCodeServer implements it
// against live state; adapters without server-side sessions do not.
type SessionLifecycle interface {
	ResumeSession(ctx context.Context, sessionID string) (Session, error)
	Usage(ctx context.Context, sessionID string) (Usage, error)
}

// Usage normalizes token/cost telemetry for one external session.
type Usage struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Reasoning    int     `json:"reasoning_tokens,omitempty"`
	CostUSD      float64 `json:"cost_usd,omitempty"`
}

// ResumeSession verifies a server-side session still exists (no model call).
func (o *OpenCodeServer) ResumeSession(ctx context.Context, sessionID string) (Session, error) {
	list, err := o.ListSessions(ctx)
	if err != nil {
		return Session{}, err
	}
	for _, s := range list {
		if s.ID == sessionID {
			s.Status = "resumed"
			return s, nil
		}
	}
	return Session{}, fmt.Errorf("opencode session not found: %s", sessionID)
}

// Usage reads token/cost accounting off the session list entry.
func (o *OpenCodeServer) Usage(ctx context.Context, sessionID string) (Usage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.BaseURL+"/session", nil)
	if err != nil {
		return Usage{}, err
	}
	resp, err := o.Client.Do(req)
	if err != nil {
		return Usage{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Usage{}, readError(resp)
	}
	var raw []struct {
		ID     string  `json:"id"`
		Cost   float64 `json:"cost"`
		Tokens struct {
			Input     int `json:"input"`
			Output    int `json:"output"`
			Reasoning int `json:"reasoning"`
		} `json:"tokens"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return Usage{}, err
	}
	for _, s := range raw {
		if s.ID == sessionID {
			return Usage{InputTokens: s.Tokens.Input, OutputTokens: s.Tokens.Output, Reasoning: s.Tokens.Reasoning, CostUSD: s.Cost}, nil
		}
	}
	return Usage{}, fmt.Errorf("opencode session not found: %s", sessionID)
}

// ---- Codex CLI adapter (structured exec) ----

// CodexCLI drives `codex exec --json` as a structured external runtime.
type CodexCLI struct {
	Bin string
}

func NewCodexCLI(bin string) *CodexCLI {
	if bin == "" {
		bin = "codex"
	}
	return &CodexCLI{Bin: bin}
}

func (c *CodexCLI) Name() string { return "codex" }

// Capabilities declares what this adapter can actually do.
//
// It declared six — session, events, permissions, usage, cancel, resume — and
// implemented one. `usage` and `resume` had no method at all; `events` emitted
// a single synthetic event and closed; `permissions` and `cancel` returned nil
// from a body that did nothing. A capability list is a contract a caller plans
// against: a run told it can resume a Codex session will try, and get a nil
// that reads as "it worked" (GAP-142).
func (c *CodexCLI) Capabilities(_ context.Context) ([]string, error) {
	return []string{"session"}, nil
}

func (c *CodexCLI) CreateSession(_ context.Context, runID string) (Session, error) {
	return Session{ID: "codex-" + runID, Provider: "codex", Status: "open", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}, nil
}

func (c *CodexCLI) Send(ctx context.Context, sessionID, message string) error {
	cmd := exec.CommandContext(ctx, c.Bin, "exec", "--json", message)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("codex exec failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// Events reports that this adapter has no event stream.
//
// It used to manufacture one: a single "session open" event, then a closed
// channel. A caller reading that channel sees a stream that begins and ends,
// and a run waiting for a tool result waits forever (GAP-142).
func (c *CodexCLI) Events(_ context.Context, _ string) (<-chan agent.AgentEvent, error) {
	return nil, errUnsupported("codex", "events")
}

// The methods below return an explicit refusal rather than nil.
//
// A nil here is read as "done". The caller closes a session, gets no error,
// and believes it — and the process it meant to stop is still running.
func (c *CodexCLI) Approve(_ context.Context, _, _ string, _ bool) error {
	return errUnsupported("codex", "permissions")
}
func (c *CodexCLI) Cancel(_ context.Context, _ string) error {
	return errUnsupported("codex", "cancel")
}
func (c *CodexCLI) Close(_ context.Context, _ string) error { return errUnsupported("codex", "close") }

// ---- Cursor CLI adapter (print mode) ----

// CursorCLI drives `cursor-agent -p` as an external runtime. Authentication
// is the user's Cursor login handled by the binary; no API key crosses the
// Prumo boundary. The turn is process-scoped: the CLI owns its model loop
// and Prumo observes completion via exit status + JSON result envelope.
type CursorCLI struct {
	Bin    string
	Trust  bool
	Runner func(ctx context.Context, bin string, args ...string) (string, error) // injectable for tests
}

func NewCursorCLI(bin string) *CursorCLI {
	if bin == "" {
		bin = "cursor-agent"
	}
	return &CursorCLI{Bin: bin, Trust: true}
}

func (c *CursorCLI) Name() string { return "cursor" }

func (c *CursorCLI) Capabilities(_ context.Context) ([]string, error) {
	return []string{"session"}, nil
}

func (c *CursorCLI) CreateSession(_ context.Context, runID string) (Session, error) {
	return Session{ID: "cursor-" + runID, Provider: "cursor", Status: "open", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}, nil
}

func (c *CursorCLI) run(ctx context.Context, args ...string) (string, error) {
	if c.Runner != nil {
		return c.Runner(ctx, c.Bin, args...)
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, c.Bin, args...).CombinedOutput()
	return string(out), err
}

// Send performs one turn via print mode. Trust is explicit: the operator
// approved this workspace for headless execution (non-interactive runs
// refuse without it, so it is passed deliberately, never silently).
func (c *CursorCLI) Send(ctx context.Context, sessionID, message string) error {
	args := []string{"-p", message, "--output-format", "json"}
	if c.Trust {
		args = append(args, "--trust")
	}
	out, err := c.run(ctx, args...)
	if err != nil {
		return fmt.Errorf("cursor exec failed: %s: %w", strings.TrimSpace(out), err)
	}
	var result struct {
		Result  string `json:"result"`
		IsError bool   `json:"is_error"`
	}
	if jsonErr := json.Unmarshal([]byte(lastJSONLine(out)), &result); jsonErr == nil && result.IsError {
		return fmt.Errorf("cursor turn failed: %s", strings.TrimSpace(result.Result))
	}
	return nil
}

// lastJSONLine returns the last non-empty line (JSON envelopes arrive on
// stdout while warnings may precede them).
func lastJSONLine(s string) string {
	s = strings.TrimRight(s, "\r\n \t")
	if i := strings.LastIndex(s, "\n"); i >= 0 {
		s = s[i+1:]
	}
	return strings.TrimSpace(s)
}

func (c *CursorCLI) Events(_ context.Context, sessionID string) (<-chan agent.AgentEvent, error) {
	ch := make(chan agent.AgentEvent, 8)
	go func() {
		defer close(ch)
		ch <- agent.AgentEvent{ID: "ev-cursor-open", RunID: sessionID, Kind: "external.session.open", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	}()
	return ch, nil
}

func (c *CursorCLI) Approve(_ context.Context, _, _ string, _ bool) error { return nil }
func (c *CursorCLI) Cancel(_ context.Context, _ string) error             { return nil }
func (c *CursorCLI) Close(_ context.Context, _ string) error              { return nil }

// ---- FakeAgentProvider for conformance without external binaries ----

type FakeAgent struct{ Sessions int }

func (f *FakeAgent) Name() string { return "fake-agent" }
func (f *FakeAgent) Capabilities(_ context.Context) ([]string, error) {
	return []string{"session", "events", "permissions", "usage", "cancel", "resume"}, nil
}
func (f *FakeAgent) CreateSession(_ context.Context, runID string) (Session, error) {
	f.Sessions++
	return Session{ID: "fake-" + runID, Provider: "fake-agent", Status: "open", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}, nil
}
func (f *FakeAgent) Send(_ context.Context, _, _ string) error { return nil }
func (f *FakeAgent) Events(_ context.Context, sessionID string) (<-chan agent.AgentEvent, error) {
	ch := make(chan agent.AgentEvent, 4)
	go func() {
		defer close(ch)
		ch <- agent.AgentEvent{ID: "ev-1", RunID: sessionID, Kind: "external.session.open", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
		ch <- agent.AgentEvent{ID: "ev-2", RunID: sessionID, Kind: "external.session.completed", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	}()
	return ch, nil
}
func (f *FakeAgent) Approve(_ context.Context, _, _ string, _ bool) error { return nil }
func (f *FakeAgent) Cancel(_ context.Context, _ string) error             { return nil }
func (f *FakeAgent) Close(_ context.Context, _ string) error              { return nil }
