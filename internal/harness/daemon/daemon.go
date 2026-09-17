// Package daemon hosts headless Harness runs behind a local Unix-socket
// server (start/status/list/events/cancel/steer/approve/deny/protocol), or
// over TCP+TLS+token when a remote address is configured. Run records and the
// JSONL timeline persist under the store dir, so a client can disconnect, the
// daemon can restart, and runs remain observable (reconnect baseline).
package daemon

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/checkpoint"
	"github.com/raillen/prumo/internal/harness/contextv2"
	"github.com/raillen/prumo/internal/harness/knowledge"
	"github.com/raillen/prumo/internal/harness/model"
	"github.com/raillen/prumo/internal/harness/perm"
	harnessprotocol "github.com/raillen/prumo/internal/harness/protocol"
	"github.com/raillen/prumo/internal/harness/runlayer"
	harnessruntime "github.com/raillen/prumo/internal/harness/runtime"
)

// StartRequest asks the daemon to run a goal headlessly.
type StartRequest struct {
	Goal      string `json:"goal"`
	Provider  string `json:"provider,omitempty"`
	Model     string `json:"model,omitempty"`
	BaseURL   string `json:"base_url,omitempty"`
	APIKey    string `json:"api_key,omitempty"`
	MaxTurns  int    `json:"max_turns,omitempty"`
	RunID     string `json:"run_id,omitempty"`
	Workspace string `json:"workspace,omitempty"`
}

// RunRecord is the persisted observable state of one run.
type RunRecord struct {
	RunID      string `json:"run_id"`
	Status     string `json:"status"` // running|complete|failed|cancelled|yielded
	Phase      string `json:"phase,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
	UpdatedAt  string `json:"updated_at"`
}

// Deps injects provider construction and tool execution (no globals).
type Deps struct {
	NewProvider func(name, baseURL, apiKey, mdl string) (model.Provider, error)
	Tools       harnessruntime.ToolExecutor
	// Workspace is the default context/tool root for runs that do not
	// carry their own (CLI serve sets it to the project root).
	Workspace string
	// PermPolicy decides tool permissions. The zero value means "the default
	// policy" (allow, ask for destructive), so leaving it unset does not
	// silently turn every tool call into an approval request.
	PermPolicy perm.Policy
}

// DefaultPermPolicy is the policy a daemon runs with when none is configured.
func DefaultPermPolicy() perm.Policy {
	return perm.Policy{
		DefaultAction: agent.PermissionAllow,
		DenyPrefixes:  []string{"/etc", ".."},
		AskKinds:      []string{"destructive"},
	}
}

// Server hosts runs on a Unix socket.
type Server struct {
	SocketPath string
	StoreDir   string
	Deps       Deps
	policy     perm.Policy

	mu   sync.Mutex
	runs map[string]*activeRun
	ln   net.Listener
	rln  net.Listener
	seq  int
}

// activeRun tracks a live run: cancel stops it, runner accepts steering and
// permission decisions.
//
// A run waiting for approval stays here rather than being discarded: it is not
// finished, and a run a client cannot reach is a run a client cannot answer.
type activeRun struct {
	cancel context.CancelFunc
	ctx    context.Context
	runner *harnessruntime.Runner

	engine   *perm.Engine
	tracker  *runlayer.Tracker
	counting *runlayer.CountingTools
	kstore   *knowledge.Store

	// mu guards timeline, which the runner's event callback appends to while
	// the persistence step reads it.
	mu       sync.Mutex
	timeline []agent.AgentEvent

	// busy is true while the run loop is advancing, so an approval cannot
	// start a second loop over the same state. Guarded by Server.mu.
	busy bool
}

// New creates a server; call Serve to block.
func New(socketPath, storeDir string, deps Deps) *Server {
	if deps.NewProvider == nil {
		deps.NewProvider = model.ForName
	}
	if deps.PermPolicy.DefaultAction == "" {
		deps.PermPolicy = DefaultPermPolicy()
	}
	return &Server{SocketPath: socketPath, StoreDir: storeDir, Deps: deps, policy: deps.PermPolicy, runs: map[string]*activeRun{}}
}

func (s *Server) recordPath(runID string) string {
	return filepath.Join(s.StoreDir, "daemon-run-"+runID+".json")
}

func (s *Server) eventPath(runID string) string {
	return filepath.Join(s.StoreDir, "events-"+runID+".jsonl")
}

func (s *Server) saveRecord(r RunRecord) {
	r.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	data, _ := json.MarshalIndent(r, "", "  ")
	_ = os.MkdirAll(s.StoreDir, 0o755)
	tmp := s.recordPath(r.RunID) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, s.recordPath(r.RunID))
}

func (s *Server) appendEvent(runID string, ev agent.AgentEvent) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	_ = os.MkdirAll(s.StoreDir, 0o755)
	path := s.eventPath(runID)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(data, '\n'))
	_ = RotateLog(path, 2000)
}

// Serve blocks until ctx is cancelled.
func (s *Server) Serve(ctx context.Context) error {
	_ = os.Remove(s.SocketPath)
	if err := os.MkdirAll(filepath.Dir(s.SocketPath), 0o755); err != nil {
		return err
	}
	ln, err := net.Listen("unix", s.SocketPath)
	if err != nil {
		return err
	}
	s.ln = ln
	go func() {
		<-ctx.Done()
		ln.Close()
	}()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.tickJobs()
			}
		}
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				continue
			}
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	w := bufio.NewWriter(conn)
	for sc.Scan() {
		var msg map[string]any
		if err := json.Unmarshal(sc.Bytes(), &msg); err != nil {
			writeMsg(w, map[string]any{"ok": false, "error": "invalid json"})
			continue
		}
		writeMsg(w, s.dispatch(msg))
	}
}

func writeMsg(w *bufio.Writer, v map[string]any) {
	data, _ := json.Marshal(v)
	_, _ = w.Write(append(data, '\n'))
	_ = w.Flush()
}

func str(m map[string]any, k string) string {
	v, _ := m[k].(string)
	return v
}

func (s *Server) dispatch(msg map[string]any) map[string]any {
	switch str(msg, "op") {
	case "protocol":
		return map[string]any{"ok": true, "version": harnessprotocol.Version, "min_compatible": harnessprotocol.MinCompatible, "schemas": harnessprotocol.Schemas, "ops": harnessprotocol.Ops}
	case "start":
		return s.opStart(msg)
	case "status":
		return s.opStatus(str(msg, "run_id"))
	case "list":
		return s.opList()
	case "events":
		return s.opEvents(str(msg, "run_id"))
	case "cancel":
		return s.opCancel(str(msg, "run_id"))
	case "steer":
		return s.opSteer(str(msg, "run_id"), str(msg, "message"))
	case "approve":
		return s.opPermission(str(msg, "run_id"), str(msg, "request_id"), str(msg, "reason"), true)
	case "deny":
		return s.opPermission(str(msg, "run_id"), str(msg, "request_id"), str(msg, "reason"), false)
	case "schedule":
		return s.opSchedule(msg)
	case "unschedule":
		return s.opUnschedule(msg)
	case "jobs":
		return s.opJobs()
	default:
		return map[string]any{"ok": false, "error": "unknown op"}
	}
}

func (s *Server) opStart(msg map[string]any) map[string]any {
	goal := str(msg, "goal")
	if goal == "" {
		return map[string]any{"ok": false, "error": "goal required"}
	}
	providerName := str(msg, "provider")
	if providerName == "" {
		providerName = "fake"
	}
	provider, err := s.Deps.NewProvider(providerName, str(msg, "base_url"), str(msg, "api_key"), str(msg, "model"))
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	maxTurns := 5
	if v, ok := msg["max_turns"].(float64); ok && v > 0 {
		maxTurns = int(v)
	}
	runID := str(msg, "run_id")
	if runID == "" {
		s.mu.Lock()
		s.seq++
		runID = fmt.Sprintf("R-daemon-%d", s.seq)
		s.mu.Unlock()
	}
	workspace := str(msg, "workspace")
	if workspace == "" {
		workspace = s.Deps.Workspace
	}
	if workspace == "" {
		workspace = s.StoreDir
	}
	tools := s.Deps.Tools
	if tools == nil {
		return map[string]any{"ok": false, "error": "no tool executor configured"}
	}
	runCtx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	if _, dup := s.runs[runID]; dup {
		s.mu.Unlock()
		cancel()
		return map[string]any{"ok": false, "error": "run already active: " + runID}
	}
	ar := &activeRun{cancel: cancel, ctx: runCtx, busy: true}
	s.runs[runID] = ar
	s.mu.Unlock()

	s.saveRecord(RunRecord{RunID: runID, Status: "running"})
	s.appendEvent(runID, agent.AgentEvent{ID: runID + "-started", RunID: runID, Kind: "run.started", Payload: map[string]any{"goal": goal, "provider": providerName}, CreatedAt: agent.Now()})

	go s.execute(runCtx, runID, goal, provider, tools, workspace, maxTurns, ar)
	return map[string]any{"ok": true, "run_id": runID}
}

// execute builds one run's collaborators and hands them to observe. Every
// artifact the run produces is written by observe, so a run that stops for
// approval and then continues still ends with exactly one coherent record.
func (s *Server) execute(ctx context.Context, runID, goal string, provider model.Provider, tools harnessruntime.ToolExecutor, workspace string, maxTurns int, ar *activeRun) {
	dir := s.StoreDir
	tracker := runlayer.NewTracker(0, 0, 0)
	counting := &runlayer.CountingTools{Base: tools, Tracker: tracker}
	engine := perm.New(s.policy)
	checkpoints := checkpoint.New(filepath.Join(dir, "checkpoints"))
	runner := harnessruntime.NewRunner(harnessruntime.Services{
		Models:      provider,
		Tools:       counting,
		Perms:       engine,
		Checkpoints: checkpoints,
		Events: func(ev agent.AgentEvent) {
			s.appendEvent(runID, ev)
			ar.mu.Lock()
			ar.timeline = append(ar.timeline, ev)
			ar.mu.Unlock()
		},
		ContextManifest: func(_ context.Context, _ agent.NativeAgentState) (string, error) {
			m := contextv2.CompileWorkspace(runID, goal, workspace, 8000, "L1")
			if data, err := json.MarshalIndent(m, "", "  "); err == nil {
				_ = os.WriteFile(filepath.Join(dir, "context-"+runID+".json"), data, 0o644)
			}
			s.appendEvent(runID, agent.AgentEvent{ID: runID + "-ctx", RunID: runID, Kind: "context.compiled",
				Payload: map[string]any{"included": len(m.Included), "tokens": m.EstimatedTokens, "pressure": m.Pressure}, CreatedAt: agent.Now()})
			return "ctx-" + runID, nil
		},
	}, runID, "S-daemon")
	runner.MaxTurns = maxTurns
	runner.Svc.ConsumeBudget = tracker.ConsumeUsage
	runner.SeedMessages([]agent.Message{{ID: "m1", Role: agent.RoleUser, Content: goal, CreatedAt: agent.Now()}})
	kstore := knowledge.New()
	knowledge.SeedRequirement(kstore, runID, goal)

	s.mu.Lock()
	ar.runner = runner
	ar.engine = engine
	ar.tracker = tracker
	ar.counting = counting
	ar.kstore = kstore
	s.mu.Unlock()

	s.observe(ctx, runID, ar)
}

// observe advances a run to its next stopping point and persists the result.
// It runs once per start, and once per answered permission request.
func (s *Server) observe(ctx context.Context, runID string, ar *activeRun) {
	runner := ar.runner
	err := runner.RunUntilDone(ctx)
	phase := runner.State.Phase
	status := "complete"
	switch {
	case err != nil && ctx.Err() != nil:
		status = "cancelled"
	case err != nil:
		status = "failed"
	case phase == agent.PhaseYield && len(runner.State.PendingPerms) > 0:
		// Waiting for a client decision is its own status: "yielded" means the
		// run parked, and conflating them would make a client either wait
		// forever on a parked run or leave the panel on one that needs it.
		status = "awaiting_approval"
	case phase == agent.PhaseYield:
		status = "yielded"
	}
	dir := s.StoreDir
	ar.mu.Lock()
	timeline := append([]agent.AgentEvent{}, ar.timeline...)
	ar.mu.Unlock()
	knowledge.SeedEvidence(ar.kstore, runID, string(phase), runner.State.StopReason, runID+"-latest")
	_ = ar.kstore.Save(filepath.Join(dir, "knowledge-"+runID+".json"))
	_ = ar.tracker.Save(filepath.Join(dir, "budget-"+runID+".json"))
	_ = runlayer.SavePermissions(filepath.Join(dir, "permissions-"+runID+".jsonl"), ar.engine)
	_, _ = runlayer.WriteEvidence(filepath.Join(dir, "evidence-"+runID+".json"),
		runID, string(phase), runner.State.StopReason, ar.tracker.Snapshot(), ar.counting.ReportsCopy())
	_ = runlayer.BridgeToObservability(filepath.Join(dir, "obs-"+runID+".jsonl"), timeline)
	_, _ = checkpoint.New(filepath.Join(dir, "checkpoints")).Prune(5)
	s.appendEvent(runID, agent.AgentEvent{ID: runID + "-finished", RunID: runID, Kind: "run.finished",
		Payload: map[string]any{"status": status, "phase": string(phase)}, CreatedAt: agent.Now()})
	s.saveRecord(RunRecord{RunID: runID, Status: status, Phase: string(phase), StopReason: runner.State.StopReason})

	s.mu.Lock()
	defer s.mu.Unlock()
	if status == "awaiting_approval" {
		ar.busy = false
		return
	}
	// Anything else is done: a parked run must not leak in the map.
	delete(s.runs, runID)
}

func (s *Server) opStatus(runID string) map[string]any {
	if runID == "" {
		return map[string]any{"ok": false, "error": "run_id required"}
	}
	s.mu.Lock()
	ar, active := s.runs[runID]
	var runner *harnessruntime.Runner
	busy := false
	if active && ar != nil {
		runner = ar.runner
		busy = ar.busy
	}
	s.mu.Unlock()
	data, err := os.ReadFile(s.recordPath(runID))
	if err != nil {
		return map[string]any{"ok": false, "error": "unknown run " + runID}
	}
	var rec RunRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return map[string]any{"ok": false, "error": "corrupt record " + runID}
	}
	pending := []string{}
	if active && runner != nil {
		live := runner.StateCopy()
		rec.Phase = string(live.Phase)
		pending = append(pending, live.PendingPerms...)
		// The record is written when a run stops. While the loop is advancing
		// it is stale, and while the run waits it is the live state that tells
		// a client there is something to answer.
		switch {
		case busy:
			rec.Status = "running"
		case live.Phase == agent.PhaseYield && len(live.PendingPerms) > 0:
			rec.Status = "awaiting_approval"
		}
	}
	return map[string]any{"ok": true, "run_id": rec.RunID, "status": rec.Status, "phase": rec.Phase, "stop_reason": rec.StopReason, "active": active, "pending_permissions": pending}
}

func (s *Server) opList() map[string]any {
	entries, err := os.ReadDir(s.StoreDir)
	if err != nil {
		return map[string]any{"ok": true, "runs": []any{}}
	}
	// A run waiting for approval is listed as such and names its request, so an
	// operator can find what to answer without reading the event log.
	s.mu.Lock()
	pending := map[string][]string{}
	for id, ar := range s.runs {
		if ar != nil && ar.runner != nil {
			if st := ar.runner.StateCopy(); len(st.PendingPerms) > 0 {
				pending[id] = append([]string{}, st.PendingPerms...)
			}
		}
	}
	s.mu.Unlock()
	out := []any{}
	for _, e := range entries {
		name := e.Name()
		if len(name) < 12 || name[:11] != "daemon-run-" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.StoreDir, name))
		if err != nil {
			continue
		}
		var rec RunRecord
		if err := json.Unmarshal(data, &rec); err != nil {
			continue
		}
		row := map[string]any{"run_id": rec.RunID, "status": rec.Status, "phase": rec.Phase, "pending_permissions": []string{}}
		if ids, ok := pending[rec.RunID]; ok {
			row["status"] = "awaiting_approval"
			row["pending_permissions"] = ids
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		a, _ := out[i].(map[string]any)
		b, _ := out[j].(map[string]any)
		ra, _ := a["run_id"].(string)
		rb, _ := b["run_id"].(string)
		return ra < rb
	})
	return map[string]any{"ok": true, "runs": out}
}

func (s *Server) opEvents(runID string) map[string]any {
	if runID == "" {
		return map[string]any{"ok": false, "error": "run_id required"}
	}
	data, err := os.ReadFile(s.eventPath(runID))
	if err != nil {
		return map[string]any{"ok": false, "error": "no events for run " + runID}
	}
	evs := []any{}
	start := 0
	flush := func(end int) {
		if line := data[start:end]; len(line) > 0 {
			var m map[string]any
			if err := json.Unmarshal(line, &m); err == nil {
				evs = append(evs, m)
			}
		}
	}
	for i, b := range data {
		if b == '\n' {
			flush(i)
			start = i + 1
		}
	}
	if start < len(data) {
		flush(len(data))
	}
	if len(evs) > 500 {
		evs = evs[len(evs)-500:]
	}
	return map[string]any{"ok": true, "run_id": runID, "events": evs}
}

func (s *Server) opCancel(runID string) map[string]any {
	if runID == "" {
		return map[string]any{"ok": false, "error": "run_id required"}
	}
	s.mu.Lock()
	ar, ok := s.runs[runID]
	s.mu.Unlock()
	if !ok {
		return map[string]any{"ok": false, "error": "run not active: " + runID}
	}
	ar.cancel()
	return map[string]any{"ok": true, "cancelled": true}
}

func (s *Server) opSteer(runID, message string) map[string]any {
	if runID == "" {
		return map[string]any{"ok": false, "error": "run_id required"}
	}
	if message == "" {
		return map[string]any{"ok": false, "error": "message required"}
	}
	s.mu.Lock()
	ar, ok := s.runs[runID]
	var runner *harnessruntime.Runner
	if ok && ar != nil {
		runner = ar.runner
	}
	s.mu.Unlock()
	if !ok || runner == nil {
		return map[string]any{"ok": false, "error": "run not active: " + runID}
	}
	if err := runner.Inject(agent.Message{ID: "steer-" + runID, Role: agent.RoleUser, Content: message, CreatedAt: agent.Now()}); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true, "steered": true}
}

// opPermission answers a pending permission request and lets the run continue.
// Approve and deny are the same op with a different decision, so the two share
// one code path and one set of validations.
func (s *Server) opPermission(runID, requestID, reason string, allow bool) map[string]any {
	if runID == "" {
		return map[string]any{"ok": false, "error": "run_id required"}
	}
	if requestID == "" {
		return map[string]any{"ok": false, "error": "permission request required"}
	}
	s.mu.Lock()
	ar, ok := s.runs[runID]
	if !ok || ar == nil || ar.runner == nil {
		s.mu.Unlock()
		return map[string]any{"ok": false, "error": "run not active: " + runID}
	}
	if ar.busy {
		s.mu.Unlock()
		return map[string]any{"ok": false, "error": "run is advancing: " + runID}
	}
	ar.busy = true
	ctx := ar.ctx
	s.mu.Unlock()

	if err := ar.runner.ResolvePermission(requestID, allow, "client", reason); err != nil {
		s.mu.Lock()
		ar.busy = false
		s.mu.Unlock()
		return map[string]any{"ok": false, "error": err.Error()}
	}
	go s.observe(ctx, runID, ar)
	return map[string]any{"ok": true, "run_id": runID, "request_id": requestID, "approved": allow}
}

// ---- Client ----

// Client speaks to a daemon over its Unix socket, or over TLS when
// RemoteAddr is set (token-authenticated first line, fail-closed TLS:
// system roots unless CACertFile pins the server cert).
type Client struct {
	SocketPath string
	RemoteAddr string
	Token      string
	CACertFile string
}

func (c Client) call(msg map[string]any) (map[string]any, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	data, _ := json.Marshal(msg)
	if _, err := conn.Write(append(data, '\n')); err != nil {
		return nil, err
	}
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	if !sc.Scan() {
		return nil, fmt.Errorf("no response from daemon")
	}
	var out map[string]any
	if err := json.Unmarshal(sc.Bytes(), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Call sends one raw op (used by thin CLI paths and tests).
func (c Client) Call(msg map[string]any) (map[string]any, error) {
	return c.call(msg)
}

// dial opens the socket, or a token-authenticated TLS connection.
func (c Client) dial() (net.Conn, error) {
	if c.RemoteAddr == "" {
		return net.Dial("unix", c.SocketPath)
	}
	if c.Token == "" {
		return nil, fmt.Errorf("remote %s requires a token (--token/--token-file/PRUMO_DAEMON_TOKEN)", c.RemoteAddr)
	}
	tlsConf := &tls.Config{MinVersion: tls.VersionTLS12}
	if c.CACertFile != "" {
		pem, err := os.ReadFile(c.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("ca cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("ca cert: no certificates parsed")
		}
		tlsConf.RootCAs = pool
	}
	conn, err := tls.Dial("tcp", c.RemoteAddr, tlsConf)
	if err != nil {
		return nil, err
	}
	auth, _ := json.Marshal(map[string]any{"auth": c.Token})
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
		return nil, fmt.Errorf("auth: %v", out["error"])
	}
	return conn, nil
}

// Start launches a run; MaxTurns<=0 defaults to 5.
func (c Client) Start(goal, provider, runID string, maxTurns int) (map[string]any, error) {
	return c.call(map[string]any{"op": "start", "goal": goal, "provider": provider, "run_id": runID, "max_turns": maxTurns})
}

// Status queries one run.
func (c Client) Status(runID string) (map[string]any, error) {
	return c.call(map[string]any{"op": "status", "run_id": runID})
}

// List returns all known runs.
func (c Client) List() (map[string]any, error) {
	return c.call(map[string]any{"op": "list"})
}

// Events replays a run timeline (last 500, capped server-side).
func (c Client) Events(runID string) (map[string]any, error) {
	return c.call(map[string]any{"op": "events", "run_id": runID})
}

// Cancel stops an active run.
func (c Client) Cancel(runID string) (map[string]any, error) {
	return c.call(map[string]any{"op": "cancel", "run_id": runID})
}

// Approve answers a pending permission request, letting the run continue.
func (c Client) Approve(runID, requestID string) (map[string]any, error) {
	return c.call(map[string]any{"op": "approve", "run_id": runID, "request_id": requestID})
}

// Deny refuses a pending permission request.
func (c Client) Deny(runID, requestID, reason string) (map[string]any, error) {
	return c.call(map[string]any{"op": "deny", "run_id": runID, "request_id": requestID, "reason": reason})
}

// Protocol negotiates versions.
func (c Client) Protocol() (map[string]any, error) {
	return c.call(map[string]any{"op": "protocol"})
}
