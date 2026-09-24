package daemon

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/model"
	harnessprotocol "github.com/raillen/prumo/internal/harness/protocol"
)

// The schedule lived in one file read and written whole, with no lock, and every
// write error discarded. The ticker runs on its own goroutine while schedule and
// unschedule arrive on connection handlers, so the slower writer erased the
// faster one's changes; and a full disk or an unwritable directory made every
// operation report success while nothing was stored (GAP-120, GAP-121).

// newScheduleServer builds a server through the real constructor, because the maps
// it initialises are the ones the code under test writes to — a Server literal
// leaves them nil and every write panics before the behaviour is reached.
func newScheduleServer(t *testing.T) *Server {
	t.Helper()
	return New(filepath.Join(t.TempDir(), "d.sock"), t.TempDir(), Deps{})
}

func TestAConcurrentScheduleAndTickDoNotLoseEachOther(t *testing.T) {
	// The lost update, directly: many concurrent writers, then a read, and
	// everything anybody asked for has to still be there.
	s := newScheduleServer(t)
	const writers = 24
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s.opSchedule(map[string]any{
				"goal": "do a thing", "job_id": "job-" + string(rune('a'+n)),
				"every_secs": float64(5 + n),
			})
		}(i)
	}
	wg.Wait()

	jobs := s.loadJobs()
	if len(jobs) != writers {
		names := make([]string, 0, len(jobs))
		for _, j := range jobs {
			names = append(names, j.ID)
		}
		t.Fatalf("%d of %d scheduled jobs survived: %v", len(jobs), writers, names)
	}
}

func TestAJobWhoseWriteFailsIsNotReportedAsScheduled(t *testing.T) {
	// A schedule that was never stored is a job that never existed, and saying
	// otherwise is the gap. A directory where the file should be makes every
	// write fail.
	s := newScheduleServer(t)
	if err := os.MkdirAll(s.jobsPath(), 0o755); err != nil {
		t.Fatal(err)
	}
	result := s.opSchedule(map[string]any{"goal": "x", "job_id": "j1", "every_secs": float64(5)})
	if result["ok"] == true {
		t.Fatalf("a job that could not be written must not be reported as scheduled: %v", result)
	}
	if message, _ := result["error"].(string); message == "" {
		t.Error("a failed write must say why, not only that it failed")
	}
}

func TestUnschedulingAJobThatCannotBeWrittenIsNotReportedAsDone(t *testing.T) {
	s := newScheduleServer(t)
	if res := s.opSchedule(map[string]any{"goal": "x", "job_id": "j1", "every_secs": float64(5)}); res["ok"] != true {
		t.Fatalf("setup failed: %v", res)
	}
	// Replace the file with a directory so the rewrite cannot land.
	if err := os.Remove(s.jobsPath()); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(s.jobsPath(), 0o755); err != nil {
		t.Fatal(err)
	}
	result := s.opUnschedule(map[string]any{"job_id": "j1"})
	if result["ok"] == true {
		t.Fatalf("an unschedule that could not be written must not report success: %v", result)
	}
}

func TestDuplicateSchedulingIsStillRefused(t *testing.T) {
	// The duplicate check moved inside the locked cycle, so it has to still work —
	// and it has to still name the job.
	s := newScheduleServer(t)
	if res := s.opSchedule(map[string]any{"goal": "x", "job_id": "same", "every_secs": float64(5)}); res["ok"] != true {
		t.Fatal(res)
	}
	result := s.opSchedule(map[string]any{"goal": "y", "job_id": "same", "every_secs": float64(5)})
	if result["ok"] == false {
		return
	}
	t.Fatal("a duplicate job id was accepted")
}

func TestUnknownUnscheduleIsStillRefused(t *testing.T) {
	s := newScheduleServer(t)
	result := s.opUnschedule(map[string]any{"job_id": "nope"})
	if result["ok"] != false {
		t.Fatalf("unscheduling an unknown job must fail: %v", result)
	}
}

func TestReservingADueJobAdvancesItBeforeDispatch(t *testing.T) {
	// The reservation is the concurrency control. If the next run time is only
	// advanced after dispatch, a slow run is fired again by the next tick — the
	// same defect a cron scheduler has when it records "started" only when the
	// work finishes.
	s := newScheduleServer(t)
	if res := s.opSchedule(map[string]any{"goal": "x", "job_id": "j1", "every_secs": float64(3600)}); res["ok"] != true {
		t.Fatal(res)
	}
	// Make it due.
	jobs := s.loadJobs()
	jobs[0].NextRun = 1
	if err := s.saveJobs(jobs); err != nil {
		t.Fatal(err)
	}

	claimed := s.reserveDueJobs()
	if len(claimed) != 1 {
		t.Fatalf("claimed %d jobs, want 1", len(claimed))
	}
	after := s.loadJobs()
	if after[0].NextRun <= 1 {
		t.Fatalf("next run = %d; the reservation must be persisted before dispatch", after[0].NextRun)
	}
	// A second reservation must find nothing due, which is what stops a double fire.
	if second := s.reserveDueJobs(); len(second) != 0 {
		t.Fatalf("a job was claimed twice: %d", len(second))
	}
}

func TestReservationThatFailsToPersistDispatchesNothing(t *testing.T) {
	// If the reservation did not land, the next tick will legitimately see the
	// job as due. Dispatching now would risk firing it twice, so the honest move
	// is to dispatch nothing and let the next tick try again.
	s := newScheduleServer(t)
	if res := s.opSchedule(map[string]any{"goal": "x", "job_id": "j1", "every_secs": float64(5)}); res["ok"] != true {
		t.Fatal(res)
	}
	jobs := s.loadJobs()
	jobs[0].NextRun = 1
	if err := s.saveJobs(jobs); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(s.jobsPath()); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(s.jobsPath(), 0o755); err != nil {
		t.Fatal(err)
	}
	if claimed := s.reserveDueJobs(); len(claimed) != 0 {
		t.Fatalf("dispatched %d jobs on a reservation that did not persist", len(claimed))
	}
}

func TestAFailedJobBacksOffAndThenIsDeadLettered(t *testing.T) {
	s := newScheduleServer(t)
	// The job has to exist on disk: recordJobFailure edits the persisted schedule,
	// and a job nobody stored has nothing to edit.
	if res := s.opSchedule(map[string]any{"goal": "x", "job_id": "j1", "every_secs": float64(5), "max_retries": float64(1)}); res["ok"] != true {
		t.Fatal(res)
	}
	job := s.loadJobs()[0]
	for attempt := 1; attempt <= 2; attempt++ {
		s.recordJobFailure(job, "boom")
		jobs := s.loadJobs()
		if attempt == 1 {
			if len(jobs) != 1 {
				t.Fatalf("after one failure the job should still be scheduled, got %d", len(jobs))
			}
			if jobs[0].Retries != 1 || jobs[0].LastStatus != "failed" || jobs[0].LastError != "boom" {
				t.Fatalf("state = %+v", jobs[0])
			}
			continue
		}
		if len(jobs) != 0 {
			t.Fatalf("after exhausting retries the job must be dropped, got %+v", jobs)
		}
	}
}

func TestASuccessfulJobClearsItsRetryState(t *testing.T) {
	s := newScheduleServer(t)
	if res := s.opSchedule(map[string]any{"goal": "x", "job_id": "j1", "every_secs": float64(5)}); res["ok"] != true {
		t.Fatal(res)
	}
	s.recordJobFailure(Job{ID: "j1", EverySecs: 5}, "boom")
	s.recordJobOutcome("j1", "started", "", 0)
	jobs := s.loadJobs()
	if len(jobs) != 1 {
		t.Fatalf("jobs = %+v", jobs)
	}
	if jobs[0].Retries != 0 || jobs[0].LastError != "" {
		t.Fatalf("a success must clear the failure state: %+v", jobs[0])
	}
}

func TestTheScheduleFileIsOnlyReplacedAtomically(t *testing.T) {
	// A reader that opens jobs.json while it is being rewritten must see either
	// the old schedule or the new one, never half of either.
	s := newScheduleServer(t)
	if res := s.opSchedule(map[string]any{"goal": "x", "job_id": "j1", "every_secs": float64(5)}); res["ok"] != true {
		t.Fatal(res)
	}
	// The temporary file must not be the same path as the target.
	if _, err := os.Stat(filepath.Join(s.StoreDir, "jobs.json")); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 20; attempt++ {
		s.mutateJobs(func(jobs []Job) ([]Job, error) { return jobs, nil })
		data, err := os.ReadFile(s.jobsPath())
		if err != nil {
			t.Fatalf("the schedule was not readable mid-rewrite: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("the schedule was observed empty mid-rewrite")
		}
	}
}

// saveRecord and appendEvent discarded every error, so a run executed with no
// record and an audit trail that stopped mid-run with nothing saying so. A run
// nobody can query and an audit log with a silent hole are both worse than
// nothing, because both read as complete (GAP-120).

func TestARunWhoseRecordCannotBeWrittenIsRefusedRatherThanStarted(t *testing.T) {
	// The record is what makes a run queryable. Losing it means no status, no
	// phase, no stop reason — and a run nobody can query is indistinguishable from
	// one that never happened.
	// The real constructor, because the maps it initialises are what a run is
	// registered in — a Server literal leaves them nil and the test would panic
	// before reaching the thing it is checking.
	s := New(filepath.Join(t.TempDir(), "d.sock"), t.TempDir(), Deps{
		Tools: &countingToolExecutor{},
		NewProvider: func(string, string, string, string) (model.Provider, error) {
			return model.NewFake(map[string][]model.ScriptStep{"*": {{Kind: "complete"}}}), nil
		},
	})
	defer s.rootCancel()
	// Make the record path a directory so the rename cannot land.
	if err := os.MkdirAll(mustRecordPath(t, s, "R-x"), 0o755); err != nil {
		t.Fatal(err)
	}

	result := s.opStart(map[string]any{
		"protocol_version": harnessprotocol.Version,
		"op":               "start", "goal": "x", "provider": "fake", "run_id": "R-x",
	})
	if result["ok"] == true {
		t.Fatalf("a run whose record cannot be persisted must not start: %v", result)
	}
	s.mu.Lock()
	_, stillThere := s.runs["R-x"]
	s.mu.Unlock()
	if stillThere {
		t.Error("the run was left registered even though it refused to start")
	}
}

func TestAWriteFailureInTheEventLogIsCounted(t *testing.T) {
	// An audit log with a silent hole reads as complete, which is worse than no
	// log. The thing that can notice has to remember.
	s := newScheduleServer(t)
	if err := os.MkdirAll(mustEventPath(t, s, "R-y"), 0o755); err != nil {
		t.Fatal(err)
	}
	before := s.EventWriteFailures()
	if err := s.appendEvent("R-y", agent.AgentEvent{ID: "e1", RunID: "R-y", Kind: "test"}); err == nil {
		t.Fatal("expected the append to fail against a directory")
	}
	if s.EventWriteFailures() != before+1 {
		t.Fatalf("failures = %d, want %d", s.EventWriteFailures(), before+1)
	}
}

func TestASuccessfulAppendIsNotCountedAsAFailure(t *testing.T) {
	s := newScheduleServer(t)
	before := s.EventWriteFailures()
	if err := s.appendEvent("R-ok", agent.AgentEvent{ID: "e1", RunID: "R-ok", Kind: "test"}); err != nil {
		t.Fatal(err)
	}
	if s.EventWriteFailures() != before {
		t.Fatalf("a successful append was counted as a failure: %d", s.EventWriteFailures())
	}
}

// mustRecordPath and mustEventPath resolve the paths the daemon will use, so the
// tests make the real one unwritable rather than a guess at its layout.
func mustRecordPath(t *testing.T, s *Server, runID string) string {
	t.Helper()
	path, err := s.recordPath(runID)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func mustEventPath(t *testing.T, s *Server, runID string) string {
	t.Helper()
	path, err := s.eventPath(runID)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

// countingToolExecutor satisfies the tool port without doing anything.
type countingToolExecutor struct{}

func (*countingToolExecutor) Execute(context.Context, agent.ToolCall) (agent.ToolResult, error) {
	return agent.ToolResult{}, nil
}
func (*countingToolExecutor) KindOf(string) string      { return "read-only" }
func (*countingToolExecutor) OperationOf(string) string { return "" }
func (*countingToolExecutor) Specs() []agent.ToolSpec   { return nil }

// A subscriber that stopped reading had its events dropped in silence. A run
// waiting for an approval whose question was dropped waits forever, and the
// client cannot tell a quiet run from a lossy one — so the subscriber is
// disconnected and told to re-read the log, which is the record (GAP-119).

func TestASubscriberThatStopsReadingIsDisconnectedRatherThanSilentlyLosingEvents(t *testing.T) {
	s := newScheduleServer(t)
	stalled := make(chan map[string]any, 2) // never read
	s.subsMu.Lock()
	s.subscribers["R-slow"] = append(s.subscribers["R-slow"], stalled)
	s.subsMu.Unlock()

	// More events than the queue holds: the overflow is where the old code
	// dropped on the floor.
	for i := 0; i < 8; i++ {
		if err := s.appendEvent("R-slow", agent.AgentEvent{ID: "e" + string(rune('a'+i)), RunID: "R-slow", Kind: "test"}); err != nil {
			t.Fatal(err)
		}
	}
	if s.SubscriberDrops() == 0 {
		t.Fatal("a subscriber that never read was left connected; its events were dropped silently")
	}
	s.subsMu.Lock()
	remaining := len(s.subscribers["R-slow"])
	s.subsMu.Unlock()
	if remaining != 0 {
		t.Fatalf("%d subscribers left registered; the stalled one should be gone", remaining)
	}
	// A disconnected channel is closed, so a reader blocked on it wakes rather
	// than waiting for a queue that will never drain.
	select {
	case _, open := <-stalled:
		if open {
			for range stalled {
			}
		}
	case <-time.After(time.Second):
		t.Fatal("the disconnected subscriber's channel was never closed")
	}
}

func TestASubscriberThatKeepsUpStaysConnected(t *testing.T) {
	// The disconnection must not fire for a healthy reader, or the push path
	// would be useless for everyone.
	s := newScheduleServer(t)
	healthy := make(chan map[string]any, 64)
	s.subsMu.Lock()
	s.subscribers["R-ok"] = append(s.subscribers["R-ok"], healthy)
	s.subsMu.Unlock()
	for i := 0; i < 8; i++ {
		if err := s.appendEvent("R-ok", agent.AgentEvent{ID: "e" + string(rune('a'+i)), RunID: "R-ok", Kind: "test"}); err != nil {
			t.Fatal(err)
		}
	}
	if s.SubscriberDrops() != 0 {
		t.Fatalf("a subscriber keeping up was dropped: %d", s.SubscriberDrops())
	}
	s.subsMu.Lock()
	remaining := len(s.subscribers["R-ok"])
	s.subsMu.Unlock()
	if remaining != 1 {
		t.Fatalf("subscribers = %d, want the healthy one still registered", remaining)
	}
}

func TestSendingAndDisconnectingDoNotRace(t *testing.T) {
	// The fan-out runs under the subscriber lock precisely so this cannot happen:
	// copying the channel list and sending afterwards would let a channel be
	// closed mid-send, which panics instead of losing an event.
	s := newScheduleServer(t)
	var wg sync.WaitGroup
	stop := make(chan struct{})
	for writer := 0; writer < 3; writer++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for i := 0; i < 60; i++ {
				_ = s.appendEvent("R-race", agent.AgentEvent{ID: "e", RunID: "R-race", Kind: "test"})
			}
		}(writer)
	}
	for reader := 0; reader < 3; reader++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					ch := make(chan map[string]any, 1)
					s.subsMu.Lock()
					s.subscribers["R-race"] = append(s.subscribers["R-race"], ch)
					s.subsMu.Unlock()
					time.Sleep(time.Millisecond)
				}
			}
		}()
	}
	time.Sleep(200 * time.Millisecond)
	close(stop)
	wg.Wait()
}

// A positional cursor stopped meaning the same thing the moment the log rotated,
// because rotation keeps the second half: index 40 named a different event than
// it did a moment ago. A reconnecting client silently missed everything between
// where it left off and where the cut fell, and re-read what it had already seen.
// An id survives rotation because the events it names are the ones that were kept
// (GAP-119).

func TestAnEventIdCursorSurvivesRotation(t *testing.T) {
	events := []map[string]any{
		{"id": "e1"}, {"id": "e2"}, {"id": "e3"},
		{"id": "e4"}, {"id": "e5"}, {"id": "e6"},
	}
	// A positional cursor at index 1 would be honoured as index 1 whatever the
	// log now contains, which after rotation is a different event.
	if start, resumed := positionAfterID(events, "e3"); start != 3 || !resumed {
		t.Fatalf("start = %d resumed = %v, want 3 and true", start, resumed)
	}
	// Rotation kept the second half. The id still resolves to the same event's
	// successor.
	rotated := events[3:]
	if _, resumed := positionAfterID(rotated, "e3"); resumed {
		t.Fatal("an id that was rotated away must be reported as not found, not silently accepted")
	}
	// An id that was kept still resolves, which is the whole point.
	if start, resumed := positionAfterID(rotated, "e4"); start != 1 || !resumed {
		t.Fatalf("a kept id must still resolve: start=%d resumed=%v", start, resumed)
	}
}

func TestAnEventIdThatWasRotatedAwayIsReportedNotGuessed(t *testing.T) {
	events := []map[string]any{{"id": "e9"}, {"id": "e10"}}
	start, resumed := positionAfterID(events, "e3")
	if resumed {
		t.Fatal("an id that is not in the log must not be reported as found")
	}
	if start != 0 {
		t.Fatalf("start = %d; a cursor that cannot be resolved must fall back to reading the retained log, not to guessing a position", start)
	}
}

func TestAnEmptyCursorResumesFromTheBeginning(t *testing.T) {
	events := []map[string]any{{"id": "e1"}}
	start, resumed := positionAfterID(events, "")
	if resumed {
		t.Fatal("an empty cursor names no event and cannot be resolved")
	}
	if start != 0 {
		t.Fatalf("start = %d, want 0", start)
	}
}

func TestRotationKeepsTheIdsTheCursorResolves(t *testing.T) {
	// The property that makes an id cursor work at all, checked against the real
	// rotation rather than a simulated one.
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	var b strings.Builder
	for i := 1; i <= 40; i++ {
		b.WriteString(`{"id":"e` + strconv.Itoa(i) + `"}` + "\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	rotated, err := RotateLog(path, 10)
	if err != nil || !rotated {
		t.Fatalf("rotation did not happen: %v %v", rotated, err)
	}
	var kept []map[string]any
	data, _ := os.ReadFile(path)
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil {
			kept = append(kept, m)
		}
	}
	if len(kept) == 0 {
		t.Fatal("rotation left nothing")
	}
	// An id from before the rotation that was kept still resolves.
	if _, resumed := positionAfterID(kept, kept[0]["id"].(string)); !resumed {
		t.Fatal("a kept id must still resolve after rotation")
	}
}
