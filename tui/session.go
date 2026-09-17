package tui

import (
	"context"
	"errors"
	"fmt"
	"time"

	prumo "github.com/raillen/prumo/sdk/prumo"
)

// Ops is the slice of the Agent Protocol a session needs.
//
// `prumo.Client` satisfies it, so production code passes the SDK client
// unchanged. Naming the slice matters for the reason interfaces are usually
// named: it is what lets the whole palette → run → stream → evidence flow be
// driven in a test without a daemon process, so the vertical slice is verified
// by `go test` rather than only by hand.
type Ops interface {
	Start(ctx context.Context, req prumo.StartRequest) (string, error)
	Events(ctx context.Context, runID string) ([]prumo.Event, error)
	Status(ctx context.Context, runID string) (prumo.RunStatus, error)
	Cancel(ctx context.Context, runID string) error
	// Approve and Deny answer a permission request the run stopped on. They
	// take the request id explicitly because that is the protocol's shape; the
	// view reaches them through Session, which already knows which request is
	// pending.
	Approve(ctx context.Context, runID, requestID string) error
	Deny(ctx context.Context, runID, requestID, reason string) error
}

// Session is one run observed over the Agent Protocol.
//
// It owns a cursor instead of replaying the whole timeline on every tick. That
// matters for the H10 redraw budget (criterion 5): the amount of work per poll
// is proportional to *new* events, not to the length of the run so far, so a
// 200-line stream does not cost 200 lines per frame.
type Session struct {
	ops   Ops
	Goal  string
	RunID string

	provider string
	cursor   int
	status   prumo.RunStatus
	finished bool
	// pending is the permission request the daemon reports the run is waiting
	// on. The status is authoritative: the daemon reads it off the live run, so
	// a client that reconnected still learns what to answer.
	pending string
}

// StartConfig is what the palette's "run goal" action needs to launch a run.
type StartConfig struct {
	Goal      string
	Provider  string
	Model     string
	MaxTurns  int
	Workspace string
	RunID     string
}

// StartSession launches a run and returns the session observing it.
//
// It fails loudly when the daemon refuses the run: a palette that closes as if
// the run had started, while the daemon answered `ok:false`, is worse than one
// that stays open showing the refusal.
func StartSession(ctx context.Context, ops Ops, cfg StartConfig) (*Session, error) {
	if cfg.Goal == "" {
		return nil, errors.New("tui: goal is required")
	}
	providerName := cfg.Provider
	if providerName == "" {
		providerName = "fake"
	}
	runID, err := ops.Start(ctx, prumo.StartRequest{
		Goal:      cfg.Goal,
		Provider:  providerName,
		Model:     cfg.Model,
		MaxTurns:  cfg.MaxTurns,
		Workspace: cfg.Workspace,
		RunID:     cfg.RunID,
	})
	if err != nil {
		return nil, err
	}
	return &Session{ops: ops, Goal: cfg.Goal, RunID: runID, provider: providerName}, nil
}

// StartRun is StartSession for callers that already hold the client type; it
// exists so `cmd/prumo` does not have to name the interface explicitly.
func StartRun(ctx context.Context, client prumo.Client, cfg StartConfig) (*Session, error) {
	return StartSession(ctx, client, cfg)
}

// PollResult is what one tick observed: the events that appeared since the
// previous tick, and the run's current status.
type PollResult struct {
	Events   []prumo.Event
	Status   prumo.RunStatus
	Finished bool
}

// Poll fetches the run's status and then any new events.
//
// The order is the whole point. A terminal status ends the session, so the event
// read that accompanies it must be the last one — and that only holds if the
// status is read *first*. Reading events first leaves a window in which the run
// completes between the two calls: the client sees a stale log, a terminal
// status, and stops, having silently dropped the events that are already on
// disk. On a fast run the whole run fits in that window, which is exactly how a
// fast CI machine exposed this.
//
// The daemon's own write order is what makes the reverse order sound: it appends
// the final event before recording the terminal status, so a terminal status
// implies the log is complete.
func (s *Session) Poll(ctx context.Context) (PollResult, error) {
	status, err := s.ops.Status(ctx, s.RunID)
	if err != nil {
		return PollResult{}, err
	}
	events, err := s.ops.Events(ctx, s.RunID)
	if err != nil {
		return PollResult{}, err
	}
	// A shorter list than the cursor means the daemon restarted with a
	// smaller log (reconnect with replay, or rotation); rewind rather than
	// return nothing forever.
	if s.cursor > len(events) {
		s.cursor = 0
	}
	fresh := append([]prumo.Event{}, events[s.cursor:]...)
	s.cursor = len(events)

	s.status = status
	if len(status.PendingPermissions) > 0 {
		s.pending = status.PendingPermissions[0]
	} else {
		s.pending = ""
	}
	// Waiting for approval is not an ending. Treating it as one would drop the
	// view out of the run panel at the exact moment it has a question to ask.
	s.finished = status.Status != "running" && status.Status != "awaiting_approval"
	return PollResult{Events: fresh, Status: status, Finished: s.finished}, nil
}

// PendingPermission is the request the run is waiting on, if any.
func (s *Session) PendingPermission() (string, bool) {
	if s.pending == "" {
		return "", false
	}
	return s.pending, true
}

// Approve answers the pending request, and the run continues.
func (s *Session) Approve(ctx context.Context) error {
	requestID, ok := s.PendingPermission()
	if !ok {
		return errors.New("tui: no permission is pending")
	}
	return s.ops.Approve(ctx, s.RunID, requestID)
}

// Deny refuses the pending request. The run then fails the way a policy denial
// fails; nothing executes.
func (s *Session) Deny(ctx context.Context, reason string) error {
	requestID, ok := s.PendingPermission()
	if !ok {
		return errors.New("tui: no permission is pending")
	}
	return s.ops.Deny(ctx, s.RunID, requestID, reason)
}

// Status is the last status observed.
func (s *Session) Status() prumo.RunStatus { return s.status }

// Finished reports whether a terminal status has been observed.
func (s *Session) Finished() bool { return s.finished }

// Provider is the provider the run was started with, for the statusline.
func (s *Session) Provider() string { return s.provider }

// Cancel asks the daemon to stop the run.
func (s *Session) Cancel(ctx context.Context) error {
	return s.ops.Cancel(ctx, s.RunID)
}

// Ops returns the transport this session runs over.
func (s *Session) Ops() Ops { return s.ops }

// PollInterval is how often the view asks for new events. Polling is the shape
// the protocol supports today (the daemon has no push channel); the interval is
// a named constant so the cost of that decision is visible in one place rather
// than buried in the update loop.
const PollInterval = 250 * time.Millisecond

// Evidence is the terminal accounting for a run: what the protocol can state
// about how it ended.
//
// This is deliberately *only* what the Agent Protocol exposes. The daemon writes
// richer artifacts (`evidence-<run>.json`, budget, permissions) into its store
// directory, and the TUI reports that path rather than parsing files the
// protocol does not define — a client that reaches into the daemon's private
// files is not a protocol client.
type Evidence struct {
	RunID      string
	Status     string
	Phase      string
	StopReason string
	Events     int
	Dropped    int
}

// Evidence summarises how the run ended.
func (s *Session) Evidence(events, dropped int) Evidence {
	return Evidence{
		RunID:      s.RunID,
		Status:     s.status.Status,
		Phase:      s.status.Phase,
		StopReason: s.status.StopReason,
		Events:     events,
		Dropped:    dropped,
	}
}

// Summary is the one-line form shown in the evidence panel.
func (e Evidence) Summary() string {
	line := fmt.Sprintf("%s · %s/%s · %d events", e.RunID, e.Status, e.Phase, e.Events)
	if e.StopReason != "" {
		line += " · " + e.StopReason
	}
	if e.Dropped > 0 {
		line += fmt.Sprintf(" · %d aged out of view", e.Dropped)
	}
	return line
}
