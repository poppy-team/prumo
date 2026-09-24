// Package daemon hosts headless Harness runs behind a local Unix-socket
// server (start/status/list/events/cancel/steer/approve/deny/protocol), or
// over TCP+TLS+token when a remote address is configured. Run records and the
// JSONL timeline persist under the store dir, so a client can disconnect, the
// daemon can restart, and runs remain observable (reconnect baseline).
package daemon

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/raillen/prumo/internal/harness/aci"
	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/checkpoint"
	"github.com/raillen/prumo/internal/harness/contextv2"
	"github.com/raillen/prumo/internal/harness/knowledge"
	"github.com/raillen/prumo/internal/harness/model"
	"github.com/raillen/prumo/internal/harness/perm"
	harnessprotocol "github.com/raillen/prumo/internal/harness/protocol"
	"github.com/raillen/prumo/internal/harness/runlayer"
	harnessruntime "github.com/raillen/prumo/internal/harness/runtime"
	"github.com/raillen/prumo/internal/harness/safepath"
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
	Status     string `json:"status"` // running|complete|failed|cancelled|yielded|awaiting_approval|interrupted
	Phase      string `json:"phase,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
	UpdatedAt  string `json:"updated_at"`
	// PendingPermissions is what makes a run that stopped for approval still
	// answerable after the daemon restarts. The reconnect contract says a
	// reconnected client "still knows what to answer", which is impossible if the
	// pending ids live only in a process that no longer exists (GAP-164).
	PendingPermissions []string `json:"pending_permissions,omitempty"`
	// ResumeCheckpoint names the checkpoint an interrupted run continues from.
	// A run that was in flight when the process died is not resumed on its own:
	// continuing it spends money nobody asked to spend. It is made resumable
	// instead, and the client decides.
	ResumeCheckpoint string `json:"resume_checkpoint,omitempty"`
	// Interrupted marks a record reconciled at startup rather than written by the
	// run that owned it. Without it a record saying "running" is indistinguishable
	// from a live one, and a client is told a run is in flight when nothing is.
	Interrupted bool `json:"interrupted,omitempty"`
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
	// Budget is the ceiling one run gets. The zero value means
	// DefaultRunBudget, because a run with no ceiling at all is the failure
	// this field exists to remove (GAP-098): the daemon used to build every
	// tracker with three zeroes, and zero means unlimited in the envelope.
	Budget Budget
}

// Budget is a per-run ceiling in the three dimensions the envelope tracks.
type Budget struct {
	Tokens    float64
	CostUSD   float64
	ToolCalls float64
}

// DefaultRunBudget is what a daemon runs with when nothing is configured.
//
// These are not aspirations. The old tracker had all three limits at zero, and
// the envelope reads zero as unlimited, so every run through the daemon could
// spend without limit until the provider declined. The numbers here are sized
// for an ordinary agent task: enough that normal work is not interrupted, low
// enough that a runaway loop or a pathological tool result is stopped.
func DefaultRunBudget() Budget {
	return Budget{Tokens: 2_000_000, CostUSD: 5, ToolCalls: 500}
}

func (b Budget) limits() (tokens, usd, tools float64) {
	def := DefaultRunBudget()
	tokens, usd, tools = b.Tokens, b.CostUSD, b.ToolCalls
	if tokens <= 0 {
		tokens = def.Tokens
	}
	if usd <= 0 {
		usd = def.CostUSD
	}
	if tools <= 0 {
		tools = def.ToolCalls
	}
	return tokens, usd, tools
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
	// diffsMu guards the diff store only. It is deliberately not s.mu: the run
	// map and the disk are different resources, and holding one lock across both
	// made every recorded diff block the run list (GAP-154).
	diffsMu sync.Mutex
	// eventsMu guards eventLines, the per-file line count that decides when a
	// log needs trimming.
	eventsMu   sync.Mutex
	eventLines map[string]int

	mu   sync.Mutex
	runs map[string]*activeRun
	ln   net.Listener
	rln  net.Listener
	seq  int

	// rootCtx is what every run context derives from. Runs used to derive from
	// context.Background(), so nothing the daemon did could reach them: a
	// SIGTERM closed the listener and the process exited with its runs still
	// mid-flight, and their work was simply gone (GAP-105).
	rootCtx    context.Context
	rootCancel context.CancelFunc

	subsMu      sync.Mutex
	subscribers map[string][]chan map[string]any
}

// DiffRecord records what a run changed in one file (ADR 014).
type DiffRecord struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Content string `json:"content"`
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

	// goal is the request's goal text, recorded so the run's evidence can name
	// the goal it speaks to. The schema requires goal_id on every record.
	goal string
}

// New creates a server; call Serve to block.
func New(socketPath, storeDir string, deps Deps) *Server {
	if deps.NewProvider == nil {
		deps.NewProvider = model.ForName
	}
	if deps.PermPolicy.DefaultAction == "" {
		deps.PermPolicy = DefaultPermPolicy()
	}
	rootCtx, rootCancel := context.WithCancel(context.Background())
	srv := &Server{
		SocketPath:  socketPath,
		StoreDir:    storeDir,
		Deps:        deps,
		policy:      deps.PermPolicy,
		runs:        map[string]*activeRun{},
		subscribers: map[string][]chan map[string]any{},
		eventLines:  map[string]int{},
		rootCtx:     rootCtx,
		rootCancel:  rootCancel,
	}
	srv.reconcileStore()
	return srv
}

// recordPath and eventPath place a run id into a filename, so the id is
// validated first. A run id is accepted straight from a request, and before
// this a value like "../../etc/cron.d/evil" reached Join untouched and wrote
// outside the store (GAP-112).
func (s *Server) recordPath(runID string) (string, error) {
	if err := safepath.ValidateID("run_id", runID); err != nil {
		return "", err
	}
	return filepath.Join(s.StoreDir, "daemon-run-"+runID+".json"), nil
}

func (s *Server) eventPath(runID string) (string, error) {
	if err := safepath.ValidateID("run_id", runID); err != nil {
		return "", err
	}
	return filepath.Join(s.StoreDir, "events-"+runID+".jsonl"), nil
}

// saveRecord writes the run record, or does nothing if the id is unusable. A
// record that cannot be named safely is not written somewhere unsafe.
func (s *Server) saveRecord(r RunRecord) {
	path, err := s.recordPath(r.RunID)
	if err != nil {
		return
	}
	r.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	data, _ := json.MarshalIndent(r, "", "  ")
	_ = os.MkdirAll(s.StoreDir, 0o755)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

// appendEvent appends one event, or drops it if the id is unusable.
func (s *Server) appendEvent(runID string, ev agent.AgentEvent) {
	path, err := s.eventPath(runID)
	if err != nil {
		return
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	_ = os.MkdirAll(s.StoreDir, 0o755)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, writeErr := f.Write(append(data, '\n'))
	_ = f.Close()
	if writeErr != nil {
		return
	}
	s.noteEventWritten(path)

	s.subsMu.Lock()
	chans := append([]chan map[string]any{}, s.subscribers[runID]...)
	s.subsMu.Unlock()
	if len(chans) > 0 {
		var evMap map[string]any
		_ = json.Unmarshal(data, &evMap)
		push := map[string]any{"op": "event", "run_id": runID, "event": evMap}
		for _, ch := range chans {
			select {
			case ch <- push:
			default:
			}
		}
	}
}

// eventLogMax is the line count at which a run's event log is trimmed.
const eventLogMax = 2000

// noteEventWritten counts a line and rotates when the cap is passed.
//
// The rotation used to run on every append, and RotateLog opens and reads the
// whole log to find out whether it needs trimming. It did that for every single
// event, including the overwhelming majority that needed nothing (GAP-153).
//
// To be precise about the shape of it: the log is capped, so each read is
// bounded and the total is linear — not quadratic, as the gap was written. What
// made it expensive is that a full-file read was paid per event. Measured on
// 20k events with everything else identical, reading per event costs 12.6s and
// counting costs 0.6s, for the same 240KB of log either way.
//
// Counting is O(N) amortised: the rewrite happens once every eventLogMax/2
// appends, and the count is reset to the half that was kept, which is exactly
// what the file then holds. Resetting it to the cap instead makes every
// following append rewrite too, which is the behaviour this removes.
func (s *Server) noteEventWritten(path string) {
	s.eventsMu.Lock()
	count := s.eventLines[path] + 1
	if count < eventLogMax {
		s.eventLines[path] = count
		s.eventsMu.Unlock()
		return
	}
	s.eventsMu.Unlock()

	// Rotate outside the lock: it reads and rewrites the file, and holding the
	// counter's lock across that would serialise every run's event loop behind
	// one file's disk work.
	trimmed, _ := RotateLog(path, eventLogMax)
	s.eventsMu.Lock()
	// Only resume from half the cap if the file was actually trimmed. A file at
	// exactly the cap is not over it, so nothing is written; assuming otherwise
	// leaves the count permanently behind the file, and the log grows past its
	// cap by a cap's worth before the next trim.
	if trimmed {
		s.eventLines[path] = eventLogMax / 2
	} else {
		s.eventLines[path] = eventLogMax
	}
	s.eventsMu.Unlock()
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
				// Stop accepting first, then let the work in flight finish. The
				// listener closing is not the end of the daemon's job: a run mid
				// model call has a checkpoint and a record to write, and exiting
				// here lost both (GAP-105).
				s.drain()
				return nil
			default:
				continue
			}
		}
		go s.handle(conn)
	}
}

// drainGrace is how long runs in flight are given to reach a stopping point on
// their own before they are cancelled. It is long enough for a model call to
// return, which is the thing a run is usually waiting on.
const drainGrace = 10 * time.Second

// drain ends the daemon's runs in the order that loses least.
//
// A run that is already advancing is left alone for up to drainGrace, so a
// SIGTERM during a model call ends with the call finished and the checkpoint
// written rather than with the call abandoned. A run still inside that window
// is then cancelled, which unwinds it through the same path a client-side
// cancel takes, so its record is written either way.
//
// Runs parked waiting for a permission are cancelled immediately: nobody is
// coming to answer them, and a drain that waited for an answer would wait
// forever.
func (s *Server) drain() {
	deadline := time.Now().Add(drainGrace)
	for {
		if !s.waitForRuns(deadline) {
			break
		}
		if time.Now().After(deadline) {
			break
		}
	}
	s.cancelAllRuns()
}

// waitForRuns reports whether any run is still advancing, waiting until they
// have all finished or the deadline passes.
func (s *Server) waitForRuns(deadline time.Time) bool {
	for {
		s.mu.Lock()
		active := 0
		for _, ar := range s.runs {
			if ar == nil {
				continue
			}
			// A run with no runner yet is starting up; it is still work the
			// daemon owns, so it counts.
			if ar.runner == nil {
				active++
				continue
			}
			// The snapshot must not block. A runner holds its lock for the whole
			// of a step, so a model call or a tool call can hold it for seconds,
			// and a drain that waits on it waits for the run it is meant to be
			// draining. A busy runner counts as advancing, which it is.
			state, read := ar.runner.TryStateCopy()
			if !read {
				active++
				continue
			}
			if len(state.PendingPerms) > 0 {
				// Parked on an answer nobody is going to give during a shutdown.
				continue
			}
			active++
		}
		s.mu.Unlock()
		if active == 0 {
			return false
		}
		if time.Now().After(deadline) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// cancelAllRuns stops every run still present, so each one unwinds and writes
// its record rather than being dropped.
//
// Cancelling is not finishing. A cancelled run is still inside its persist step
// when this returns, so the daemon would exit with a checkpoint half-written and
// the caller would be told the drain completed. The second phase waits, bounded,
// for the runs to actually leave the map.
func (s *Server) cancelAllRuns() {
	s.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(s.runs))
	for _, ar := range s.runs {
		if ar != nil && ar.cancel != nil {
			cancels = append(cancels, ar.cancel)
		}
	}
	s.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
	if s.rootCancel != nil {
		s.rootCancel()
	}
	s.releaseParked()
	s.awaitUnwind(cancelGrace)
}

// releaseParked drops runs that are waiting for an answer nobody is coming to
// give.
//
// Cancelling one does not remove it: its observe call already returned when it
// parked, so there is no goroutine left to notice the cancellation and clear
// its entry. Without this the drain would wait out its whole grace for a run
// that was never going to move, and the daemon would exit still holding it.
//
// Nothing is lost. A parked run persisted its record and its checkpoint before
// it parked; that is what a client reconnects to.
func (s *Server) releaseParked() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for runID, ar := range s.runs {
		if ar == nil || ar.runner == nil {
			continue
		}
		state, read := ar.runner.TryStateCopy()
		if read && len(state.PendingPerms) > 0 {
			delete(s.runs, runID)
		}
	}
}

// cancelGrace is how long cancelled runs are given to unwind and persist. It
// covers the persist step — a checkpoint write, a knowledge save, a budget save
// and an evidence record — which is fast but is real work.
const cancelGrace = 5 * time.Second

// awaitUnwind waits, bounded, for the run map to empty.
func (s *Server) awaitUnwind(grace time.Duration) {
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		remaining := len(s.runs)
		s.mu.Unlock()
		if remaining == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	w := bufio.NewWriter(conn)
	var writeMu sync.Mutex
	safeWrite := func(v map[string]any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return writeMsg(w, v)
	}

	// The connection owns a context that dies when handle returns, and every
	// subscription derives from it. A subscription used to hang off
	// context.Background(), so closing the socket left its goroutine running and
	// writing into a dead connection until the next subscribe arrived.
	//
	// Replacing a subscription stops the one it replaces, and the last one stops
	// with the connection. A cancel func held in a variable assigned inside a
	// loop is the shape a static check reads as a leak, so the replacement signal
	// is a channel instead: closing it ends that subscription, and the
	// connection context is what guarantees the last one ends too.
	connCtx, connCancel := context.WithCancel(context.Background())
	defer connCancel()
	var replaced chan struct{}

	for sc.Scan() {
		var msg map[string]any
		if err := json.Unmarshal(sc.Bytes(), &msg); err != nil {
			_ = safeWrite(map[string]any{"ok": false, "error": "invalid json"})
			continue
		}
		if str(msg, "op") == "subscribe" {
			if replaced != nil {
				close(replaced)
			}
			done := make(chan struct{})
			replaced = done
			ctx, stop := context.WithCancel(connCtx)
			go func() {
				defer stop()
				defer close(done)
				s.handleSubscribe(ctx, msg, safeWrite)
			}()
			continue
		}
		_ = safeWrite(s.dispatch(msg))
	}
}

func writeMsg(w *bufio.Writer, v map[string]any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := w.Write(append(data, '\n')); err != nil {
		return err
	}
	return w.Flush()
}

func str(m map[string]any, k string) string {
	v, _ := m[k].(string)
	return v
}

// dispatch routes one request, and refuses to route it at all if the client and
// the engine do not speak the same protocol.
//
// Negotiation existed and nothing called it: the version the client sent was
// never read, so a client built for a different protocol was served until an op
// behaved differently than it expected. The reconnect contract says a version
// mismatch fails closed and does not guess, and that a reconnect begins with a
// handshake (GAP-165).
//
// The "protocol" op itself is exempt: it is how a client asks what is supported,
// and answering that cannot require already knowing the answer.
func (s *Server) dispatch(msg map[string]any) map[string]any {
	if str(msg, "op") != "protocol" {
		if refusal := checkProtocol(msg); refusal != nil {
			return *refusal
		}
	}
	switch str(msg, "op") {
	case "protocol":
		return map[string]any{"ok": true, "version": harnessprotocol.Version, "min_compatible": harnessprotocol.MinCompatible, "schemas": harnessprotocol.Schemas, "ops": harnessprotocol.Ops}
	case "models":
		return s.opModels(msg)
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
		return s.opPermission(str(msg, "run_id"), str(msg, "request_id"), str(msg, "fingerprint"), str(msg, "reason"), true)
	case "deny":
		return s.opPermission(str(msg, "run_id"), str(msg, "request_id"), str(msg, "fingerprint"), str(msg, "reason"), false)
	case "schedule":
		return s.opSchedule(msg)
	case "unschedule":
		return s.opUnschedule(msg)
	case "jobs":
		return s.opJobs()
	case "diff":
		return s.opDiff(msg)
	case "subscribe":
		if str(msg, "run_id") == "" {
			return map[string]any{"ok": false, "error": "run_id required"}
		}
		return map[string]any{"ok": false, "error": "subscribe requires a streaming connection"}
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
	// The workspace is resolved before the provider is built, because the provider
	// is told where to work. It used to be told the daemon's workspace, and the
	// request's own workspace was only read afterwards — so a run naming a
	// different tree got tools rooted there and a provider rooted somewhere else,
	// and a delegated provider edits whichever it was given (GAP-125).
	workspace := str(msg, "workspace")
	if workspace == "" {
		workspace = s.Deps.Workspace
	}
	if workspace == "" {
		workspace = s.StoreDir
	}
	provider, err := s.Deps.NewProvider(providerName, str(msg, "base_url"), str(msg, "api_key"), str(msg, "model"))
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	// A provider that runs work of its own needs to know where: a delegated turn
	// launched in the daemon's directory would edit the wrong tree. The factory
	// builds a provider from a name, a URL and a key — none of which is a
	// workspace — so it is handed over here, and only to providers that ask.
	if aware, ok := provider.(interface{ SetWorkspace(string) }); ok {
		aware.SetWorkspace(workspace)
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
	} else if err := safepath.ValidateID("run_id", runID); err != nil {
		// Rejected here so the caller learns why, rather than getting a run that
		// starts and then silently records nothing.
		return map[string]any{"ok": false, "error": err.Error()}
	}
	tools := s.Deps.Tools
	if tools == nil {
		return map[string]any{"ok": false, "error": "no tool executor configured"}
	}
	runCtx, cancel := context.WithCancel(s.rootCtx)
	s.mu.Lock()
	if _, dup := s.runs[runID]; dup {
		s.mu.Unlock()
		cancel()
		return map[string]any{"ok": false, "error": "run already active: " + runID}
	}
	ar := &activeRun{cancel: cancel, ctx: runCtx, busy: true, goal: goal}
	s.runs[runID] = ar
	s.mu.Unlock()

	modelName := str(msg, "model")
	s.saveRecord(RunRecord{RunID: runID, Status: "running"})
	s.appendEvent(runID, agent.AgentEvent{ID: runID + "-started", RunID: runID, Kind: "run.started", Payload: map[string]any{"goal": goal, "provider": providerName}, CreatedAt: agent.Now()})

	go s.execute(runCtx, runID, goal, modelName, provider, tools, workspace, maxTurns, ar)
	return map[string]any{"ok": true, "run_id": runID}
}

// execute builds one run's collaborators and hands them to observe. Every
// artifact the run produces is written by observe, so a run that stops for
// approval and then continues still ends with exactly one coherent record.
func (s *Server) execute(ctx context.Context, runID, goal, modelName string, provider model.Provider, tools harnessruntime.ToolExecutor, workspace string, maxTurns int, ar *activeRun) {
	dir := s.StoreDir
	budgetTokens, budgetUSD, budgetTools := s.Deps.Budget.limits()
	tracker := runlayer.NewTracker(budgetTokens, budgetUSD, budgetTools)
	// What a previous life of this run spent is part of its budget. Without this
	// a restart hands back a full allowance, and the cheapest way past a ceiling
	// is to crash (GAP-001).
	if err := runlayer.Load(filepath.Join(dir, "budget-"+runID+".json"), tracker); err != nil {
		s.appendEvent(runID, agent.AgentEvent{
			ID: runID + "-budget-unreadable", RunID: runID, Kind: "budget_envelope_unreadable",
			Payload: map[string]any{"error": err.Error()}, CreatedAt: agent.Now(),
		})
	}
	counting := &runlayer.CountingTools{Base: tools, Tracker: tracker}
	engine := perm.New(s.policy)
	// Decisions are read back before the run starts. Without this an approval
	// given to a previous process is unknown here, and a run resumed after a
	// daemon restart asks the same human the same question again (GAP-106).
	if err := runlayer.LoadPermissions(filepath.Join(dir, "permissions-"+runID+".jsonl"), engine); err != nil {
		// A trail that cannot be read is not a reason to start a run that would
		// then re-ask; it is a reason to say so.
		s.appendEvent(runID, agent.AgentEvent{
			ID: runID + "-perm-load-failed", RunID: runID, Kind: "permission_trail_unreadable",
			Payload: map[string]any{"error": err.Error()}, CreatedAt: agent.Now(),
		})
	}
	checkpoints := checkpoint.New(filepath.Join(dir, "checkpoints"))
	hasVision := false
	if declared, err := model.LoadDeclarations(workspace); err == nil {
		if caps, ok := declared.Models[modelName]; ok {
			hasVision = caps.Vision
		}
	}
	runner := harnessruntime.NewRunner(harnessruntime.Services{
		Models:        provider,
		Tools:         counting,
		Perms:         engine,
		Checkpoints:   checkpoints,
		EffectJournal: checkpoints,
		Workspace:     workspace,
		HasVision:     hasVision,
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
		RecordDiff: func(runID, path, kind, content string) {
			s.saveDiff(runID, path, kind, content)
		},
	}, runID, "S-daemon")
	runner.MaxTurns = maxTurns
	// Preflight: a run stops before making a call it cannot afford, and
	// reserves what the call may cost so the ceiling binds the last turn too
	// rather than only being checked after the money is spent (GAP-098).
	runner.Svc.ReserveBudget = tracker.Reserve
	runner.Svc.BudgetExhausted = tracker.Exhausted
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
	if _, evErr := runlayer.WriteEvidence(filepath.Join(dir, "evidence-"+runID+".json"),
		runID, ar.goal, string(phase), runner.State.StopReason, ar.tracker.Snapshot(), ar.counting.ReportsCopy()); evErr != nil {
		// Evidence that cannot be written must be visible. Reporting the run
		// as successful while its evidence is missing is the false green this
		// audit closed, so the failure becomes part of the run's own timeline.
		s.appendEvent(runID, agent.AgentEvent{
			ID: runID + "-evidence-failed", RunID: runID, Kind: "evidence.write_failed",
			Payload: map[string]any{"error": evErr.Error()}, CreatedAt: agent.Now(),
		})
	}
	_ = runlayer.BridgeToObservability(filepath.Join(dir, "obs-"+runID+".jsonl"), timeline)
	_, _ = checkpoint.New(filepath.Join(dir, "checkpoints")).Prune(5)
	// "finished" is for a run that ended. A run that stopped for a decision has
	// not ended, and calling that finished would tell every client the opposite
	// of what the status says.
	if status == "complete" || status == "failed" || status == "cancelled" {
		s.appendEvent(runID, agent.AgentEvent{ID: runID + "-finished", RunID: runID, Kind: "run.finished",
			Payload: map[string]any{"status": status, "phase": string(phase)}, CreatedAt: agent.Now()})
	} else {
		s.appendEvent(runID, agent.AgentEvent{ID: runID + "-paused", RunID: runID, Kind: "run.paused",
			Payload: map[string]any{"status": status, "phase": string(phase)}, CreatedAt: agent.Now()})
	}
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

// opModels asks a provider what it can serve.
//
// The answer belongs to the harness: which models exist is a property of the
// provider, and a client that carried its own list would be asserting something
// it cannot verify. A provider that cannot enumerate says so by returning what
// it was configured with rather than an empty list.
func (s *Server) opModels(msg map[string]any) map[string]any {
	providerName := str(msg, "provider")
	if providerName == "" {
		providerName = "fake"
	}
	provider, err := s.Deps.NewProvider(providerName, str(msg, "base_url"), str(msg, "api_key"), str(msg, "model"))
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	models, err := provider.Models(ctx)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}

	// What each model can do is declared by the workspace, not probed: no
	// provider publishes its models' features in a form a client can read, so
	// the daemon reports what somebody wrote down and marks the rest as
	// undeclared. `models` keeps carrying bare ids for clients that only pick
	// one; `model_info` carries what is known about each.
	workspace := s.Deps.Workspace
	if workspace == "" {
		workspace = s.StoreDir
	}
	declared, err := model.LoadDeclarations(workspace)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}

	return map[string]any{
		"ok": true, "provider": providerName,
		"models":     models,
		"model_info": model.Describe(models, declared),
	}
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
	path, err := s.recordPath(runID)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]any{"ok": false, "error": "unknown run " + runID}
	}
	var rec RunRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return map[string]any{"ok": false, "error": "corrupt record " + runID}
	}
	pending := append([]string{}, rec.PendingPermissions...)
	fingerprints := map[string]any{}
	if active && runner != nil {
		// A non-blocking read: a client polling status during a long model call
		// must get an answer, not wait for the call to return before it can be
		// told the run is busy.
		live, read := runner.TryStateCopy()
		if !read {
			rec.Status = "running"
			return map[string]any{"ok": true, "run_id": rec.RunID, "status": rec.Status, "phase": rec.Phase, "active": active, "pending_permissions": pending, "permission_fingerprints": fingerprints}
		}
		rec.Phase = string(live.Phase)
		pending = append(pending, live.PendingPerms...)
		// The fingerprint travels with the request so an approver can quote what
		// they were shown. Without it the answer identifies a request by id
		// alone, which is a label, not the content (GAP-107).
		for _, id := range live.PendingPerms {
			fingerprints[id] = runner.PendingFingerprint(id)
		}
		rec.Status = "awaiting_approval"
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
	return map[string]any{
		"ok": true, "run_id": rec.RunID, "status": rec.Status, "phase": rec.Phase,
		"stop_reason": rec.StopReason, "active": active,
		"pending_permissions": pending, "permission_fingerprints": fingerprints,
		"interrupted": rec.Interrupted, "resume_checkpoint": rec.ResumeCheckpoint,
	}
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
	prints := map[string]map[string]any{}
	for id, ar := range s.runs {
		if ar != nil && ar.runner != nil {
			if st := ar.runner.StateCopy(); len(st.PendingPerms) > 0 {
				pending[id] = append([]string{}, st.PendingPerms...)
				// The fingerprint goes in the listing too. An answer has to quote
				// it, so a listing that omits it makes the gate unanswerable from
				// the only view an operator has (GAP-107).
				prints[id] = map[string]any{}
				for _, requestID := range st.PendingPerms {
					prints[id][requestID] = ar.runner.PendingFingerprint(requestID)
				}
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
		row := map[string]any{
			"run_id": rec.RunID, "status": rec.Status, "phase": rec.Phase,
			"pending_permissions": []string{}, "active": false,
			"interrupted": rec.Interrupted, "resume_checkpoint": rec.ResumeCheckpoint,
		}
		if ids, ok := pending[rec.RunID]; ok {
			row["status"] = "awaiting_approval"
			row["pending_permissions"] = ids
			row["permission_fingerprints"] = prints[rec.RunID]
			row["active"] = true
		} else if len(rec.PendingPermissions) > 0 {
			// Recovered from a restart: waiting, answerable, but no process owns
			// it here yet. The ids are the contract's step 6.
			row["pending_permissions"] = rec.PendingPermissions
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
	path, err := s.eventPath(runID)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	data, err := os.ReadFile(path)
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

// diffPath names a run's diff store. The run id is validated because it becomes
// a filename, the same reason recordPath and eventPath validate theirs.
func (s *Server) diffPath(runID string) (string, error) {
	if err := safepath.ValidateID("run_id", runID); err != nil {
		return "", err
	}
	return filepath.Join(s.StoreDir, "diffs-"+runID+".json"), nil
}

// saveDiff records one file change.
//
// It takes diffMu, not s.mu. The global lock guards the run map, and holding it
// across a file read, a marshal, a write and a rename meant every recorded diff
// blocked opList, opStatus and every run start for the duration of the disk
// write (GAP-154). The diff store has its own lock because it is its own
// resource; nothing that touches the run map waits on it.
func (s *Server) saveDiff(runID, path, kind, content string) {
	target, err := s.diffPath(runID)
	if err != nil {
		return
	}
	s.diffsMu.Lock()
	defer s.diffsMu.Unlock()
	diffs := s.loadDiffs(target)
	diffs[path] = DiffRecord{Path: path, Kind: kind, Content: content}
	data, err := json.MarshalIndent(diffs, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(s.StoreDir, 0o755)
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err == nil {
		_ = os.Rename(tmp, target)
	}
}

func (s *Server) loadDiffs(target string) map[string]DiffRecord {
	data, err := os.ReadFile(target)
	if err != nil {
		return map[string]DiffRecord{}
	}
	var m map[string]DiffRecord
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]DiffRecord{}
	}
	if m == nil {
		return map[string]DiffRecord{}
	}
	return m
}

// opDiff returns what a run changed in one file (ADR 014).
func (s *Server) opDiff(msg map[string]any) map[string]any {
	runID := str(msg, "run_id")
	if runID == "" {
		return map[string]any{"ok": false, "error": "run_id required"}
	}
	path := str(msg, "path")
	if path == "" {
		return map[string]any{"ok": false, "error": "path required"}
	}
	target, err := s.diffPath(runID)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	// The diff store's own lock, for the same reason saveDiff uses it: a read
	// here must not block the run map (GAP-154).
	s.diffsMu.Lock()
	defer s.diffsMu.Unlock()
	diffs := s.loadDiffs(target)
	entry, ok := diffs[path]
	if !ok {
		clean := filepath.Clean(path)
		for k, v := range diffs {
			if filepath.Clean(k) == clean {
				entry = v
				ok = true
				break
			}
		}
	}
	if !ok {
		return map[string]any{"ok": false, "error": "file not modified by run: " + path}
	}
	return map[string]any{
		"ok":      true,
		"path":    entry.Path,
		"kind":    entry.Kind,
		"content": entry.Content,
	}
}

func (s *Server) readRawEvents(runID string) []map[string]any {
	path, err := s.eventPath(runID)
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var evs []map[string]any
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(line, &m); err == nil {
			evs = append(evs, m)
		}
	}
	return evs
}

// handleSubscribe streams live and replayed events over JSONL (ADR 014).
func (s *Server) handleSubscribe(ctx context.Context, msg map[string]any, writeMsg func(map[string]any) error) {
	runID := str(msg, "run_id")
	if runID == "" {
		_ = writeMsg(map[string]any{"ok": false, "error": "run_id required"})
		return
	}
	from := 0
	if f, ok := msg["from"].(float64); ok {
		from = int(f)
	} else if i, ok := msg["from"].(int); ok {
		from = i
	}

	// Register subscriber first so no live events emitted during catch-up are dropped.
	subCh := make(chan map[string]any, 128)
	s.subsMu.Lock()
	s.subscribers[runID] = append(s.subscribers[runID], subCh)
	s.subsMu.Unlock()

	defer func() {
		s.subsMu.Lock()
		cur := s.subscribers[runID]
		for i, ch := range cur {
			if ch == subCh {
				s.subscribers[runID] = append(cur[:i], cur[i+1:]...)
				break
			}
		}
		s.subsMu.Unlock()
	}()

	existing := s.readRawEvents(runID)
	if from < 0 || from > len(existing) {
		_ = writeMsg(map[string]any{"ok": false, "error": fmt.Sprintf("cursor beyond event log: %d > %d", from, len(existing))})
		return
	}

	// First line: subscribed acknowledgement carrying the honoured cursor.
	if err := writeMsg(map[string]any{"op": "subscribed", "run_id": runID, "from": from}); err != nil {
		return
	}

	// A subscriber that has already been sent an id must not receive it twice,
	// and must not lose a new event for sharing one. See alreadySent.
	sent := make(map[string]string)
	for i := from; i < len(existing); i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		ev := existing[i]
		if id, ok := ev["id"].(string); ok && id != "" {
			sent[id] = describeEvent(ev)
		}
		if err := writeMsg(map[string]any{"op": "event", "run_id": runID, "event": ev}); err != nil {
			return
		}
	}

	// Check if run was already finished.
	s.mu.Lock()
	_, active := s.runs[runID]
	s.mu.Unlock()
	if !active {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case evMsg, ok := <-subCh:
			if !ok {
				return
			}
			if ev, ok := evMsg["event"].(map[string]any); ok {
				if id, ok := ev["id"].(string); ok && id != "" {
					if alreadySent(sent, id, ev) {
						continue
					}
					sent[id] = describeEvent(ev)
				}
				if err := writeMsg(evMsg); err != nil {
					return
				}
				kind, _ := ev["kind"].(string)
				if kind == "run.finished" || kind == "run.completed" || kind == "run.failed" || kind == "run.cancelled" {
					return
				}
			}
		}
	}
}

// alreadySent reports whether this exact event has already gone to the
// subscriber, and records it otherwise.
//
// The comparison is on the whole event, not the id alone. While event ids could
// collide — they were derived from len(payload) — a live event sharing an id
// with a replayed one was dropped as if it were a repeat, and the client missed
// it (GAP-118). Losing an event is worse than showing one twice, so a shared id
// with different content is sent: at worst the subscriber sees a collision, and
// the id scheme is what should be fixed.
func alreadySent(sent map[string]string, id string, ev map[string]any) bool {
	// An event with no id cannot be compared with anything, so it is always
	// sent. The caller checks too; owning the whole rule here means a second
	// caller cannot get it wrong by forgetting.
	if id == "" {
		return false
	}
	shape := describeEvent(ev)
	if previous, seen := sent[id]; seen && previous == shape {
		return true
	}
	sent[id] = shape
	return false
}

// describeEvent renders an event so two deliveries of it compare equal. A value
// that cannot be marshalled yields the empty string, which makes two such events
// look alike; that is the permissive direction, and losing an event is the
// direction to err in.
func describeEvent(ev map[string]any) string {
	data, err := json.Marshal(ev)
	if err != nil {
		return ""
	}
	return string(data)
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
func (s *Server) opPermission(runID, requestID, fingerprint, reason string, allow bool) map[string]any {
	if runID == "" {
		return map[string]any{"ok": false, "error": "run_id required"}
	}
	if requestID == "" {
		return map[string]any{"ok": false, "error": "permission request required"}
	}
	// The approver must quote the fingerprint it was shown. The runner recomputes
	// it from the tool call actually pending and compares, so an approval cannot
	// be aimed at a different call carrying the same request id (GAP-107).
	if fingerprint == "" {
		return map[string]any{"ok": false, "error": "fingerprint required: answer with the fingerprint from the permission_wait event"}
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

	if err := ar.runner.ResolvePermission(requestID, fingerprint, allow, "client", reason); err != nil {
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
	// Every request declares the protocol it speaks, because the daemon now
	// refuses the ones that do not. Setting it here means no caller can forget.
	msg["protocol_version"] = harnessprotocol.Version
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
// Approve allows a pending permission request. The fingerprint is the one the
// approver was shown; the daemon recomputes it from the pending call and
// refuses a mismatch.
func (c Client) Approve(runID, requestID, fingerprint string) (map[string]any, error) {
	return c.call(map[string]any{"op": "approve", "run_id": runID, "request_id": requestID, "fingerprint": fingerprint})
}

// Deny refuses a pending permission request.
func (c Client) Deny(runID, requestID, fingerprint, reason string) (map[string]any, error) {
	return c.call(map[string]any{"op": "deny", "run_id": runID, "request_id": requestID, "fingerprint": fingerprint, "reason": reason})
}

// Models asks a provider what it can serve.
func (c Client) Models(provider, baseURL, model string) (map[string]any, error) {
	return c.call(map[string]any{"op": "models", "provider": provider, "base_url": baseURL, "model": model})
}

// Diff reads what a run changed in one file (ADR 014).
func (c Client) Diff(runID, path string) (map[string]any, error) {
	return c.call(map[string]any{"op": "diff", "run_id": runID, "path": path})
}

// Protocol negotiates versions.
func (c Client) Protocol() (map[string]any, error) {
	return c.call(map[string]any{"op": "protocol"})
}

// checkProtocol negotiates the client's version against this engine and returns
// a refusal when they cannot work together.
//
// A request with no version is refused, not assumed current. Assuming is the
// exact failure the contract rules out: a client too old to send a version is
// also too old to know which ops exist, so its idea of any of them may be wrong.
// The refusal names both versions, so the client can be fixed rather than
// guessing what went wrong.
func checkProtocol(msg map[string]any) *map[string]any {
	declared, _ := msg["protocol_version"].(string)
	if strings.TrimSpace(declared) == "" {
		return refusal(
			fmt.Sprintf("protocol_version required: this daemon speaks %s (compatible from %s); send it on every request",
				harnessprotocol.Version, harnessprotocol.MinCompatible))
	}
	server, compatible, err := harnessprotocol.Negotiate(declared)
	if err != nil {
		return refusal(fmt.Sprintf("invalid protocol_version %q: %v", declared, err))
	}
	if !compatible {
		return refusal(fmt.Sprintf("protocol version mismatch: client %s, daemon %s (compatible from %s)",
			declared, server, harnessprotocol.MinCompatible))
	}
	return nil
}

func refusal(message string) *map[string]any {
	return &map[string]any{
		"ok":               false,
		"error":            message,
		"version":          harnessprotocol.Version,
		"min_compatible":   harnessprotocol.MinCompatible,
		"protocol_version": harnessprotocol.Version,
	}
}

// reconcileStore brings the records left by a previous process into line with
// what is actually true.
//
// A record saying "running" was written by a process that no longer exists, so
// nothing is running. Before this the daemon started with an empty run map and
// read those records as-is: status reported active:false while the record still
// claimed running, and a client had no way to tell a live run from a dead one
// except by looking for something that would never arrive (GAP-164).
//
// A run that stopped for approval is the case the reconnect contract calls out.
// It is not interrupted: it is waiting, and the request it is waiting on is
// recorded in its checkpoint. It is restored to awaiting_approval with its
// pending ids, so a reconnected client still knows what to answer.
func (s *Server) reconcileStore() {
	entries, err := os.ReadDir(s.StoreDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if len(name) < 12 || name[:11] != "daemon-run-" || !strings.HasSuffix(name, ".json") {
			continue
		}
		// The filename carries the run id behind a prefix; the id is what the
		// checkpoint store is keyed by, and looking it up with the prefix still
		// attached finds nothing.
		runID := strings.TrimSuffix(name[len("daemon-run-"):], ".json")
		s.reconcileRun(runID, filepath.Join(s.StoreDir, name))
	}
}

func (s *Server) reconcileRun(runID, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var rec RunRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return
	}
	// A terminal run said everything it had to say; the record is not ours to
	// reinterpret. The reconnect contract is about replay, and replay reads the
	// event log, not this.
	switch rec.Status {
	case "complete", "failed", "cancelled", "interrupted":
		return
	}

	cp, cpErr := checkpoint.New(filepath.Join(s.StoreDir, "checkpoints")).Latest(runID)
	pending := []string{}
	phase := rec.Phase
	if cpErr == nil {
		pending = append(pending, cp.State.PendingPerms...)
		if phase == "" {
			phase = string(cp.State.Phase)
		}
	}
	rec.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	rec.Interrupted = true

	switch {
	case len(pending) > 0:
		// Waiting, not interrupted: the request is still answerable.
		rec.Status = "awaiting_approval"
		rec.PendingPermissions = pending
		rec.StopReason = "daemon restarted while this run was waiting for approval"
	case cpErr == nil:
		// Resumable, but not resumed: continuing spends what nobody authorised.
		rec.Status = "interrupted"
		rec.ResumeCheckpoint = cp.ID
		rec.StopReason = "daemon restarted while this run was in flight; resume from checkpoint " + cp.ID
	default:
		rec.Status = "interrupted"
		rec.StopReason = "daemon restarted while this run was in flight; no checkpoint survived to resume from"
	}
	rec.Phase = phase

	out, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err == nil {
		_ = os.Rename(tmp, path)
	}
}

// restoreRun rebuilds a run's runner from its checkpoint so a run that was
// waiting for approval can be answered after a restart.
//
// The contract calls a reconnected run "still answerable". A pending id without
// a runner behind it is decoration: the approve op needs a live runner to
// resolve against, and the permission engine needs the decisions already taken so
// it does not ask again for something a person has already answered.
//
// It returns nil when there is no checkpoint to restore from. A run that cannot
// be restored stays interrupted, which is the truth about it.
func (s *Server) restoreRun(runID string) *harnessruntime.Runner {
	store := checkpoint.New(filepath.Join(s.StoreDir, "checkpoints"))
	cp, err := store.Latest(runID)
	if err != nil {
		return nil
	}
	workspace := s.Deps.Workspace
	if workspace == "" {
		workspace = s.StoreDir
	}
	providerName, providerMDL, apiKey, baseURL := s.runProviderFor(cp)
	provider, err := s.Deps.NewProvider(providerName, baseURL, apiKey, providerMDL)
	if err != nil {
		return nil
	}
	engine := perm.New(s.policy)
	// The decisions this run already collected, so answering a second request
	// does not re-ask the first.
	_ = runlayer.LoadPermissions(filepath.Join(s.StoreDir, "permissions-"+runID+".jsonl"), engine)

	budgetTokens, budgetUSD, budgetTools := s.Deps.Budget.limits()
	tracker := runlayer.NewTracker(budgetTokens, budgetUSD, budgetTools)
	_ = runlayer.Load(filepath.Join(s.StoreDir, "budget-"+runID+".json"), tracker)
	tools := s.Deps.Tools
	if tools == nil {
		tools = aci.New(workspace)
	}
	runner := harnessruntime.NewRunner(harnessruntime.Services{
		Models:          provider,
		Tools:           tools,
		Perms:           engine,
		Checkpoints:     store,
		EffectJournal:   store,
		Workspace:       workspace,
		ReserveBudget:   tracker.Reserve,
		BudgetExhausted: tracker.Exhausted,
	}, runID, cp.State.SessionID)
	runner.RestoreFrom(cp)
	return runner
}

// runProviderFor recovers the provider a checkpointed run was using. A run that
// cannot name one is restored with the daemon's default, which is what a fresh
// run would use.
func (s *Server) runProviderFor(cp agent.Checkpoint) (name, model, apiKey, baseURL string) {
	// The route is recorded as a string; a run that named one is restored with
	// that provider, and one that did not falls back to the default.
	if cp.State.ModelRoute != "" {
		name = cp.State.ModelRoute
	}
	if name == "" {
		name = "fake"
	}
	return name, model, os.Getenv("PRUMO_MODEL_API_KEY"), os.Getenv("PRUMO_MODEL_BASE_URL")
}
