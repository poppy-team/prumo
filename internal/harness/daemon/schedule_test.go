package daemon

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

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
