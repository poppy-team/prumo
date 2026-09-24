// Package prumo is the public typed SDK for the Prumo Harness daemon.
//
// Boundary invariant: this package imports stdlib only — never
// prumo/internal. It is the client surface the future prumo-code repo (and
// any third-party client) builds against. Wire shape follows
// schemas/protocol-manifest.json v0.3.0.
package prumo

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// ProtocolVersion is the IDL this SDK speaks.
//
// It is a copy of the engine's, because the SDK must not import the server's
// internals — a public client that compiles against internal packages cannot be
// used outside the module. A copy is a drift risk, so TestSDKVersionMatchesEngine
// pins the two together and fails when the engine moves and this does not.
const ProtocolVersion = "0.4.0"

// Client talks to a harness daemon over its Unix socket.
type Client struct {
	SocketPath string
	Timeout    time.Duration
	// Remote addressing (empty = Unix socket). Token authenticates the
	// first line of every connection; TLSConf secures it.
	RemoteAddr  string
	RemoteToken string
	TLSConf     *tls.Config
	// Actor names who is using this client, and is recorded on every
	// permission decision it makes.
	//
	// Without it the daemon recorded the literal actor "client" — a transport
	// rather than a person — so the permissions trail could not say who allowed
	// a shell command, only that something did (GAP-160). Set it to whatever
	// identifies the operator: a user name, a service account, an agent id.
	Actor string
}

// DialRemote builds a client for a TCP+TLS daemon endpoint.
func DialRemote(addr, token string, tlsConf *tls.Config) *Client {
	return &Client{RemoteAddr: addr, RemoteToken: token, TLSConf: tlsConf}
}

func (c Client) dial(ctx context.Context) (net.Conn, error) {
	if c.RemoteAddr == "" {
		dialer := net.Dialer{Timeout: 5 * time.Second}
		conn, err := dialer.DialContext(ctx, "unix", c.SocketPath)
		if err != nil {
			return nil, fmt.Errorf("dial %s: %w", c.SocketPath, err)
		}
		return conn, nil
	}
	conf := c.TLSConf
	if conf == nil {
		conf = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	dialer := &tls.Dialer{NetDialer: &net.Dialer{Timeout: 5 * time.Second}, Config: conf.Clone()}
	conn, err := dialer.DialContext(ctx, "tcp", c.RemoteAddr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", c.RemoteAddr, err)
	}
	auth, _ := json.Marshal(map[string]any{"auth": c.RemoteToken})
	if _, err := conn.Write(append(auth, '\n')); err != nil {
		conn.Close()
		return nil, err
	}
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 64*1024), 64*1024)
	if !sc.Scan() {
		conn.Close()
		return nil, fmt.Errorf("no auth response from daemon")
	}
	var out map[string]any
	if err := json.Unmarshal(sc.Bytes(), &out); err != nil {
		conn.Close()
		return nil, err
	}
	if ok, _ := out["ok"].(bool); !ok {
		conn.Close()
		return nil, &Error{Op: "auth", Message: fmt.Sprint(out["error"])}
	}
	return conn, nil
}

func (c Client) timeout() time.Duration {
	if c.Timeout <= 0 {
		return 30 * time.Second
	}
	return c.Timeout
}

// Error is a typed daemon-side failure (ok:false payload).
type Error struct {
	Op      string
	Message string
}

func (e *Error) Error() string { return fmt.Sprintf("daemon op %s: %s", e.Op, e.Message) }

func (c Client) call(ctx context.Context, msg map[string]any) (map[string]any, error) {
	op, _ := msg["op"].(string)
	// The daemon refuses a request that does not declare its protocol, so the SDK
	// declares it on every one rather than making each call site remember.
	msg["protocol_version"] = ProtocolVersion
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	} else {
		_ = conn.SetDeadline(time.Now().Add(c.timeout()))
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Write(append(data, '\n')); err != nil {
		return nil, err
	}
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("no response from daemon")
	}
	var out map[string]any
	if err := json.Unmarshal(sc.Bytes(), &out); err != nil {
		return nil, err
	}
	if ok, _ := out["ok"].(bool); !ok {
		msg, _ := out["error"].(string)
		return nil, &Error{Op: op, Message: msg}
	}
	return out, nil
}

// RunStatus is the observable state of one run.
type RunStatus struct {
	RunID      string `json:"run_id"`
	Status     string `json:"status"`
	Phase      string `json:"phase"`
	StopReason string `json:"stop_reason"`
	Active     bool   `json:"active"`
	// PendingPermissions are the request ids awaiting a client decision. A run
	// waiting for approval reports status "awaiting_approval" and names what it
	// is waiting on, so a client that reconnected can still answer.
	PendingPermissions []string `json:"pending_permissions"`
}

// StartRequest launches a headless run.
type StartRequest struct {
	Goal      string
	Provider  string
	Model     string
	BaseURL   string
	MaxTurns  int
	RunID     string
	Workspace string
}

// Start launches a run and returns its id.
func (c Client) Start(ctx context.Context, r StartRequest) (string, error) {
	maxTurns := r.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 5
	}
	out, err := c.call(ctx, map[string]any{
		"op": "start", "goal": r.Goal, "provider": r.Provider, "model": r.Model,
		"base_url": r.BaseURL, "max_turns": maxTurns, "run_id": r.RunID, "workspace": r.Workspace,
	})
	if err != nil {
		return "", err
	}
	id, _ := out["run_id"].(string)
	return id, nil
}

// Status queries one run.
func (c Client) Status(ctx context.Context, runID string) (RunStatus, error) {
	var st RunStatus
	out, err := c.call(ctx, map[string]any{"op": "status", "run_id": runID})
	if err != nil {
		return st, err
	}
	st.RunID, _ = out["run_id"].(string)
	st.Status, _ = out["status"].(string)
	st.Phase, _ = out["phase"].(string)
	st.StopReason, _ = out["stop_reason"].(string)
	st.Active, _ = out["active"].(bool)
	st.PendingPermissions = toStrSlice(out["pending_permissions"])
	return st, nil
}

// List returns all runs known to the daemon.
func (c Client) List(ctx context.Context) ([]RunStatus, error) {
	out, err := c.call(ctx, map[string]any{"op": "list"})
	if err != nil {
		return nil, err
	}
	raw, _ := out["runs"].([]any)
	states := make([]RunStatus, 0, len(raw))
	for _, item := range raw {
		m, _ := item.(map[string]any)
		states = append(states, RunStatus{
			RunID:  strOf(m, "run_id"),
			Status: strOf(m, "status"),
			Phase:  strOf(m, "phase"),
		})
	}
	return states, nil
}

// Event is one timeline unit.
type Event struct {
	ID      string         `json:"id"`
	RunID   string         `json:"run_id"`
	Kind    string         `json:"kind"`
	Payload map[string]any `json:"payload"`
}

// Events replays a run timeline (server-capped).
func (c Client) Events(ctx context.Context, runID string) ([]Event, error) {
	out, err := c.call(ctx, map[string]any{"op": "events", "run_id": runID})
	if err != nil {
		return nil, err
	}
	raw, _ := out["events"].([]any)
	evs := make([]Event, 0, len(raw))
	for _, item := range raw {
		m, _ := item.(map[string]any)
		ev := Event{ID: strOf(m, "id"), RunID: strOf(m, "run_id"), Kind: strOf(m, "kind")}
		if p, ok := m["payload"].(map[string]any); ok {
			ev.Payload = p
		}
		evs = append(evs, ev)
	}
	return evs, nil
}

// Cancel stops an active run.
func (c Client) Cancel(ctx context.Context, runID string) error {
	_, err := c.call(ctx, map[string]any{"op": "cancel", "run_id": runID})
	return err
}

// Steer injects follow-up input into an active run (refused when terminal).
func (c Client) Steer(ctx context.Context, runID, message string) error {
	_, err := c.call(ctx, map[string]any{"op": "steer", "run_id": runID, "message": message})
	return err
}

// Approve answers a pending permission request, letting the run continue.
func (c Client) Approve(ctx context.Context, runID, requestID string) error {
	_, err := c.call(ctx, map[string]any{
		"op": "approve", "run_id": runID, "request_id": requestID, "actor": c.Actor,
	})
	return err
}

// Deny refuses a pending permission request. The run then fails the way a
// policy denial fails; nothing executes.
func (c Client) Deny(ctx context.Context, runID, requestID, reason string) error {
	_, err := c.call(ctx, map[string]any{
		"op": "deny", "run_id": runID, "request_id": requestID,
		"reason": reason, "actor": c.Actor,
	})
	return err
}

// ModelsRequest identifies the provider to ask about.
//
// Empty fields mean "the daemon's default", which is the common case: a client
// usually wants to know what the harness it is attached to can serve, not what
// some other endpoint could.
type ModelsRequest struct {
	Provider string
	BaseURL  string
	Model    string
}

// Models asks the harness what a provider can serve.
//
// The client does not keep its own catalogue: which models exist is a property
// of the provider, and only the process that talks to it can answer.
func (c Client) Models(ctx context.Context, r ModelsRequest) ([]string, error) {
	msg := map[string]any{"op": "models"}
	if r.Provider != "" {
		msg["provider"] = r.Provider
	}
	if r.BaseURL != "" {
		msg["base_url"] = r.BaseURL
	}
	if r.Model != "" {
		msg["model"] = r.Model
	}
	out, err := c.call(ctx, msg)
	if err != nil {
		return nil, err
	}
	return toStrSlice(out["models"]), nil
}

// Job is one scheduled run template.
type Job struct {
	ID        string `json:"job_id"`
	Goal      string `json:"goal"`
	EverySecs int64  `json:"every_secs"`
	NextRun   int64  `json:"next_run"`
}

// Schedule registers a recurring run (everySecs minimum 5, server-side).
func (c Client) Schedule(ctx context.Context, goal, provider string, everySecs int64, maxTurns int) (string, error) {
	out, err := c.call(ctx, map[string]any{"op": "schedule", "goal": goal, "provider": provider, "every_secs": everySecs, "max_turns": maxTurns})
	if err != nil {
		return "", err
	}
	id, _ := out["job_id"].(string)
	return id, nil
}

// Unschedule removes a job.
func (c Client) Unschedule(ctx context.Context, jobID string) error {
	_, err := c.call(ctx, map[string]any{"op": "unschedule", "job_id": jobID})
	return err
}

// CapabilitySet is what a model can do, as the workspace declared it.
//
// Absent means undeclared, never denied: the daemon reports what somebody wrote
// in `.prumo/models.json` and nothing more.
type CapabilitySet struct {
	Text          bool `json:"text,omitempty"`
	Vision        bool `json:"vision,omitempty"`
	Reasoning     bool `json:"reasoning,omitempty"`
	Tools         bool `json:"tools,omitempty"`
	Audio         bool `json:"audio,omitempty"`
	ContextTokens int  `json:"context_tokens,omitempty"`
}

// ModelInfo is one model with what is known about it.
type ModelInfo struct {
	ID           string        `json:"id"`
	Declared     bool          `json:"declared"`
	Capabilities CapabilitySet `json:"capabilities"`
}

// ModelInfo asks what the provider serves *and* what each model can do.
//
// The models operation answers both: the ids come from the provider, the
// capabilities from the workspace's own declaration, and a model nobody declared
// arrives with Declared false rather than with an empty set of denials.
func (c Client) ModelInfo(ctx context.Context, r ModelsRequest) ([]ModelInfo, error) {
	out, err := c.call(ctx, map[string]any{"op": "models", "provider": r.Provider, "base_url": r.BaseURL})
	if err != nil {
		return nil, err
	}
	raw, _ := out["model_info"].([]any)
	infos := make([]ModelInfo, 0, len(raw))
	for _, item := range raw {
		m, _ := item.(map[string]any)
		info := ModelInfo{ID: strOf(m, "id")}
		if declared, ok := m["declared"].(bool); ok {
			info.Declared = declared
		}
		if caps, ok := m["capabilities"].(map[string]any); ok {
			info.Capabilities = CapabilitySet{
				Text:          boolOf(caps, "text"),
				Vision:        boolOf(caps, "vision"),
				Reasoning:     boolOf(caps, "reasoning"),
				Tools:         boolOf(caps, "tools"),
				Audio:         boolOf(caps, "audio"),
				ContextTokens: intOf(caps, "context_tokens"),
			}
		}
		if info.ID != "" {
			infos = append(infos, info)
		}
	}
	return infos, nil
}

func boolOf(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}

func intOf(m map[string]any, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

// Jobs lists scheduled jobs.
func (c Client) Jobs(ctx context.Context) ([]Job, error) {
	out, err := c.call(ctx, map[string]any{"op": "jobs"})
	if err != nil {
		return nil, err
	}
	raw, _ := out["jobs"].([]any)
	jobs := make([]Job, 0, len(raw))
	for _, item := range raw {
		m, _ := item.(map[string]any)
		var secs, next int64
		if v, ok := m["every_secs"].(float64); ok {
			secs = int64(v)
		}
		if v, ok := m["next_run"].(float64); ok {
			next = int64(v)
		}
		jobs = append(jobs, Job{ID: strOf(m, "job_id"), Goal: strOf(m, "goal"), EverySecs: secs, NextRun: next})
	}
	return jobs, nil
}

// ProtocolInfo describes the daemon's IDL.
type ProtocolInfo struct {
	Version       string   `json:"version"`
	MinCompatible string   `json:"min_compatible"`
	Schemas       []string `json:"schemas"`
	Ops           []string `json:"ops"`
}

// Protocol fetches the daemon's IDL description.
func (c Client) Protocol(ctx context.Context) (ProtocolInfo, error) {
	var info ProtocolInfo
	out, err := c.call(ctx, map[string]any{"op": "protocol"})
	if err != nil {
		return info, err
	}
	info.Version, _ = out["version"].(string)
	info.MinCompatible, _ = out["min_compatible"].(string)
	for _, s := range toStrSlice(out["schemas"]) {
		info.Schemas = append(info.Schemas, s)
	}
	for _, s := range toStrSlice(out["ops"]) {
		info.Ops = append(info.Ops, s)
	}
	return info, nil
}

// Wait polls Status until the run leaves "running" or ctx expires.
func (c Client) Wait(ctx context.Context, runID string, poll time.Duration) (RunStatus, error) {
	if poll <= 0 {
		poll = 100 * time.Millisecond
	}
	t := time.NewTicker(poll)
	defer t.Stop()
	for {
		st, err := c.Status(ctx, runID)
		if err != nil {
			return st, err
		}
		if st.Status != "running" {
			return st, nil
		}
		select {
		case <-ctx.Done():
			return st, ctx.Err()
		case <-t.C:
		}
	}
}

func strOf(m map[string]any, k string) string {
	v, _ := m[k].(string)
	return v
}

func toStrSlice(v any) []string {
	raw, _ := v.([]any)
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// DiffResponse is what a run changed in one file (ADR 014).
type DiffResponse struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Content string `json:"content"`
}

// Diff asks the harness what a run changed in one file.
func (c Client) Diff(ctx context.Context, runID, path string) (DiffResponse, error) {
	out, err := c.call(ctx, map[string]any{"op": "diff", "run_id": runID, "path": path})
	if err != nil {
		return DiffResponse{}, err
	}
	return DiffResponse{
		Path:    strOf(out, "path"),
		Kind:    strOf(out, "kind"),
		Content: strOf(out, "content"),
	}, nil
}

// Subscribe opens a push stream for a run's events starting at from (ADR 014).
func (c Client) Subscribe(ctx context.Context, runID string, from int) (<-chan Event, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	req, _ := json.Marshal(map[string]any{"op": "subscribe", "run_id": runID, "from": from})
	if _, err := conn.Write(append(req, '\n')); err != nil {
		conn.Close()
		return nil, err
	}
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	if !sc.Scan() {
		conn.Close()
		return nil, fmt.Errorf("no response from daemon for subscribe")
	}
	var ack map[string]any
	if err := json.Unmarshal(sc.Bytes(), &ack); err != nil {
		conn.Close()
		return nil, fmt.Errorf("invalid subscribe ack: %w", err)
	}
	if ok, _ := ack["ok"].(bool); !ok && ack["op"] != "subscribed" {
		conn.Close()
		return nil, fmt.Errorf("subscribe failed: %v", ack["error"])
	}

	events := make(chan Event, 64)
	go func() {
		defer conn.Close()
		defer close(events)

		ctxDone := make(chan struct{})
		defer close(ctxDone)
		go func() {
			select {
			case <-ctx.Done():
				conn.Close()
			case <-ctxDone:
			}
		}()

		for sc.Scan() {
			var line map[string]any
			if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
				continue
			}
			if strOf(line, "op") != "event" {
				continue
			}
			evRaw, ok := line["event"].(map[string]any)
			if !ok {
				continue
			}
			evBytes, err := json.Marshal(evRaw)
			if err != nil {
				continue
			}
			var ev Event
			if err := json.Unmarshal(evBytes, &ev); err != nil {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case events <- ev:
			}
			if ev.Kind == "run.finished" || ev.Kind == "run.completed" || ev.Kind == "run.failed" || ev.Kind == "run.cancelled" {
				return
			}
		}
	}()

	return events, nil
}
