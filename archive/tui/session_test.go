//go:build ignore

package tui

import (
	"context"
	"errors"
	"testing"

	prumo "github.com/raillen/prumo/sdk/prumo"
)

// stubOps is a scripted Agent Protocol endpoint. It is the whole reason `Ops`
// exists: the run lifecycle can be exercised without a daemon process, so the
// vertical slice is covered by `go test` and not only by hand.
type stubOps struct {
	runID     string
	events    []prumo.Event
	status    prumo.RunStatus
	startErr  error
	eventsErr error
	cancelled int
	started   []prumo.StartRequest
	// approved and denied record what the view answered, so a test can assert
	// the decision reached the protocol rather than only changing the screen.
	approved []string
	denied   []string
	// calls records the order the protocol ops were invoked in. Allowing test
	// files to observe ordering without moving either op behind an interface is
	// enough here: order *is* the contract Poll depends on.
	calls []string
}

func (s *stubOps) Start(_ context.Context, req prumo.StartRequest) (string, error) {
	if s.startErr != nil {
		return "", s.startErr
	}
	s.started = append(s.started, req)
	if s.runID == "" {
		s.runID = "R-test-1"
	}
	s.status = prumo.RunStatus{RunID: s.runID, Status: "running", Phase: "model"}
	return s.runID, nil
}

func (s *stubOps) Events(context.Context, string) ([]prumo.Event, error) {
	s.calls = append(s.calls, "events")
	if s.eventsErr != nil {
		return nil, s.eventsErr
	}
	return append([]prumo.Event{}, s.events...), nil
}

func (s *stubOps) Status(context.Context, string) (prumo.RunStatus, error) {
	s.calls = append(s.calls, "status")
	return s.status, nil
}

func (s *stubOps) Cancel(context.Context, string) error {
	s.cancelled++
	s.status.Status = "cancelled"
	return nil
}

func (s *stubOps) Approve(_ context.Context, _, requestID string) error {
	s.approved = append(s.approved, requestID)
	s.status.PendingPermissions = nil
	s.status.Status = "running"
	return nil
}

func (s *stubOps) Deny(_ context.Context, _, requestID, reason string) error {
	s.denied = append(s.denied, requestID+"|"+reason)
	s.status.PendingPermissions = nil
	s.status.Status = "failed"
	s.status.StopReason = "permission denied"
	return nil
}

func TestSessionRequiresAGoal(t *testing.T) {
	if _, err := StartSession(context.Background(), &stubOps{}, StartConfig{}); err == nil {
		t.Fatal("a run without a goal must be refused before it reaches the daemon")
	}
}

func TestSessionDefaultsToTheFakeProvider(t *testing.T) {
	ops := &stubOps{}
	session, err := StartSession(context.Background(), ops, StartConfig{Goal: "g"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(ops.started) != 1 || ops.started[0].Provider != "fake" {
		t.Fatalf("provider not defaulted: %+v", ops.started)
	}
	if session.RunID != "R-test-1" {
		t.Fatalf("run id = %q", session.RunID)
	}
}

func TestSessionReportsStartFailure(t *testing.T) {
	ops := &stubOps{startErr: errors.New("quota exhausted")}
	if _, err := StartSession(context.Background(), ops, StartConfig{Goal: "g"}); err == nil {
		t.Fatal("a refused start must surface as an error")
	}
}

func TestPollReturnsOnlyNewEvents(t *testing.T) {
	ops := &stubOps{}
	session, err := StartSession(context.Background(), ops, StartConfig{Goal: "g"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	ops.events = []prumo.Event{ev("1", "run.started", nil), ev("2", "text_delta", nil)}

	first, err := session.Poll(context.Background())
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if len(first.Events) != 2 {
		t.Fatalf("first poll returned %d events, want 2", len(first.Events))
	}
	second, err := session.Poll(context.Background())
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	// This is the redraw-budget property: a tick costs what is new, not what
	// has accumulated.
	if len(second.Events) != 0 {
		t.Fatalf("second poll replayed %d events", len(second.Events))
	}
}

func TestPollDetectsTerminalStatus(t *testing.T) {
	ops := &stubOps{}
	session, _ := StartSession(context.Background(), ops, StartConfig{Goal: "g"})
	if _, err := session.Poll(context.Background()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if session.Finished() {
		t.Fatal("a running run must not report Finished")
	}
	ops.status = prumo.RunStatus{RunID: "R-test-1", Status: "failed", Phase: "failed", StopReason: "permission denied"}
	result, err := session.Poll(context.Background())
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if !result.Finished || !session.Finished() {
		t.Fatal("a failed run must report Finished")
	}
	evidence := session.Evidence(2, 0)
	if evidence.Status != "failed" || evidence.StopReason != "permission denied" {
		t.Fatalf("evidence lost the terminal accounting: %+v", evidence)
	}
	if evidence.Summary() == "" {
		t.Fatal("evidence summary is empty")
	}
}

// TestPollReadsStatusBeforeEvents pins the ordering that makes a terminal poll
// complete. The daemon appends its final event before recording the terminal
// status, so reading the status first means the events read that follows cannot
// miss anything. Reading events first leaves a window in which a fast run
// finishes between the two calls, and the client stops with a short timeline —
// which is exactly what CI caught, at a speed a developer machine never hit.
func TestPollReadsStatusBeforeEvents(t *testing.T) {
	ops := &stubOps{}
	session, _ := StartSession(context.Background(), ops, StartConfig{Goal: "g"})
	if _, err := session.Poll(context.Background()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if len(ops.calls) != 2 || ops.calls[0] != "status" || ops.calls[1] != "events" {
		t.Fatalf("protocol calls were %v; the terminal poll must read status first", ops.calls)
	}
}

func TestPollRewindsWhenTheLogShrinks(t *testing.T) {
	ops := &stubOps{}
	session, _ := StartSession(context.Background(), ops, StartConfig{Goal: "g"})
	ops.events = []prumo.Event{ev("1", "a", nil), ev("2", "b", nil), ev("3", "c", nil)}
	if _, err := session.Poll(context.Background()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	// A daemon that restarted with a shorter log — H10 criterion 3. Returning
	// nothing forever would leave a reconnected client permanently blank.
	ops.events = []prumo.Event{ev("9", "z", nil)}
	result, err := session.Poll(context.Background())
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if len(result.Events) != 1 || result.Events[0].ID != "9" {
		t.Fatalf("replay after shrink returned %+v", result.Events)
	}
}

func TestCancelReachesTheDaemon(t *testing.T) {
	ops := &stubOps{}
	session, _ := StartSession(context.Background(), ops, StartConfig{Goal: "g"})
	if err := session.Cancel(context.Background()); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if ops.cancelled != 1 {
		t.Fatalf("cancel calls = %d, want 1", ops.cancelled)
	}
}

// TestPollTreatsAwaitingApprovalAsInFlight pins the distinction the approval
// surface depends on: a run waiting for a decision is not finished. Reading it
// as finished would drop the view out of the run panel exactly when it has a
// question to ask.
func TestPollTreatsAwaitingApprovalAsInFlight(t *testing.T) {
	ops := &stubOps{}
	session, _ := StartSession(context.Background(), ops, StartConfig{Goal: "g"})
	ops.status = prumo.RunStatus{RunID: "R-test-1", Status: "awaiting_approval", Phase: "yield", PendingPermissions: []string{"perm-c1"}}
	result, err := session.Poll(context.Background())
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if result.Finished || session.Finished() {
		t.Fatal("awaiting approval must not report Finished")
	}
	requestID, ok := session.PendingPermission()
	if !ok || requestID != "perm-c1" {
		t.Fatalf("pending request = %q %v", requestID, ok)
	}
	// A parked run (no pending request) is still an ending.
	ops.status = prumo.RunStatus{RunID: "R-test-1", Status: "yielded", Phase: "yield"}
	if _, err := session.Poll(context.Background()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if !session.Finished() {
		t.Fatal("a yielded run with nothing pending is finished")
	}
	if _, ok := session.PendingPermission(); ok {
		t.Fatal("a parked run has no pending request")
	}
}

func TestApproveSendsThePendingRequest(t *testing.T) {
	ops := &stubOps{}
	session, _ := StartSession(context.Background(), ops, StartConfig{Goal: "g"})
	if err := session.Approve(context.Background()); err == nil {
		t.Fatal("approving with nothing pending must be refused")
	}
	ops.status = prumo.RunStatus{RunID: "R-test-1", Status: "awaiting_approval", Phase: "yield", PendingPermissions: []string{"perm-c1"}}
	if _, err := session.Poll(context.Background()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if err := session.Approve(context.Background()); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if len(ops.approved) != 1 || ops.approved[0] != "perm-c1" {
		t.Fatalf("approve did not reach the protocol: %v", ops.approved)
	}
}

func TestDenySendsThePendingRequest(t *testing.T) {
	ops := &stubOps{}
	session, _ := StartSession(context.Background(), ops, StartConfig{Goal: "g"})
	ops.status = prumo.RunStatus{RunID: "R-test-1", Status: "awaiting_approval", Phase: "yield", PendingPermissions: []string{"perm-c9"}}
	if _, err := session.Poll(context.Background()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if err := session.Deny(context.Background(), "outside the workspace"); err != nil {
		t.Fatalf("deny: %v", err)
	}
	if len(ops.denied) != 1 || ops.denied[0] != "perm-c9|outside the workspace" {
		t.Fatalf("deny did not reach the protocol: %v", ops.denied)
	}
}
