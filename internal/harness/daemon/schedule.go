// Scheduled runs (H16 first slice): cron-like jobs firing headless runs
// inside the daemon. Jobs persist in jobs.json; the serve loop ticks every
// 5s and starts due jobs with deterministic run ids (job-<id>-<unix>).
// Missed ticks fire once (no catch-up storms).
package daemon

import (
	"encoding/json"
	"fmt"
	harnessprotocol "github.com/raillen/prumo/internal/harness/protocol"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Job is one scheduled run template.
type Job struct {
	ID         string `json:"id"`
	Goal       string `json:"goal"`
	Provider   string `json:"provider,omitempty"`
	EverySecs  int64  `json:"every_secs"`
	MaxTurns   int    `json:"max_turns,omitempty"`
	MaxRetries int    `json:"max_retries,omitempty"` // default 3
	Retries    int    `json:"retries,omitempty"`
	LastStatus string `json:"last_status,omitempty"`
	LastError  string `json:"last_error,omitempty"`
	NextRun    int64  `json:"next_run"`
	CreatedAt  string `json:"created_at"`
}

func (s *Server) jobsPath() string { return filepath.Join(s.StoreDir, "jobs.json") }

// jobsMu serialises every read-modify-write of jobs.json.
//
// The file held the whole schedule, so a schedule and a tick each read it, edited
// their own copy and wrote it back. The ticker runs on its own goroutine while
// schedule and unschedule arrive on connection handlers, so the slower writer's
// changes were erased by the faster one — a scheduled job vanishing because
// something else happened to save a moment later (GAP-121).
//
// One mutex around the whole cycle, not around the file operations, because
// holding it only for the write would still let two readers build conflicting
// copies.
var jobsMu sync.Mutex

func (s *Server) loadJobs() []Job {
	data, err := os.ReadFile(s.jobsPath())
	if err != nil {
		return nil
	}
	var jobs []Job
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil
	}
	return jobs
}

// saveJobs persists the schedule, reporting whether it actually landed.
//
// It used to discard every error, so a full disk or an unwritable directory made
// schedule, unschedule and the ticker's own bookkeeping report success while the
// change was lost. A job that was never written is a job that never existed, and
// saying otherwise is the failure this fixes (GAP-120).
func (s *Server) saveJobs(jobs []Job) error {
	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.StoreDir, 0o755); err != nil {
		return err
	}
	tmp := s.jobsPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.jobsPath())
}

// mutateJobs runs one read-modify-write cycle under the lock, and only persists
// when the edit succeeded.
func (s *Server) mutateJobs(edit func([]Job) ([]Job, error)) error {
	jobsMu.Lock()
	defer jobsMu.Unlock()
	jobs := s.loadJobs()
	edited, err := edit(jobs)
	if err != nil {
		return err
	}
	return s.saveJobs(edited)
}

func (s *Server) opSchedule(msg map[string]any) map[string]any {
	goal := str(msg, "goal")
	if goal == "" {
		return map[string]any{"ok": false, "error": "goal required"}
	}
	every, _ := msg["every_secs"].(float64)
	if every < 5 {
		return map[string]any{"ok": false, "error": "every_secs minimum is 5"}
	}
	id := str(msg, "job_id")
	if id == "" {
		id = fmt.Sprintf("job-%d", time.Now().UTC().UnixNano())
	}
	maxTurns := 5
	if v, ok := msg["max_turns"].(float64); ok && v > 0 {
		maxTurns = int(v)
	}
	provider := str(msg, "provider")
	if provider == "" {
		provider = "fake"
	}
	// max_retries is part of the job's own contract and is read here. It was
	// accepted and ignored: a caller that set it got the default instead, with
	// nothing saying so, and the field looked configurable because it is
	// documented on Job.
	maxRetries := 3
	if v, ok := msg["max_retries"].(float64); ok && v > 0 {
		maxRetries = int(v)
	}
	now := time.Now().UTC().Unix()
	created := Job{ID: id, Goal: goal, Provider: provider, EverySecs: int64(every),
		MaxTurns: maxTurns, MaxRetries: maxRetries, NextRun: now + int64(every),
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	// The whole read-append-write runs under one lock, and a persist failure is
	// reported. Writing outside the lock reintroduces the lost update, and
	// reporting success for a schedule that was never stored is the gap (GAP-120).
	err := s.mutateJobs(func(jobs []Job) ([]Job, error) {
		for _, j := range jobs {
			if j.ID == id {
				return nil, fmt.Errorf("job already scheduled: %s", id)
			}
		}
		return append(jobs, created), nil
	})
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true, "job_id": id}
}

func (s *Server) opUnschedule(msg map[string]any) map[string]any {
	id := str(msg, "job_id")
	if id == "" {
		return map[string]any{"ok": false, "error": "job_id required"}
	}
	found := false
	err := s.mutateJobs(func(jobs []Job) ([]Job, error) {
		kept := make([]Job, 0, len(jobs))
		for _, j := range jobs {
			if j.ID == id {
				found = true
				continue
			}
			kept = append(kept, j)
		}
		if !found {
			return nil, fmt.Errorf("unknown job %s", id)
		}
		return kept, nil
	})
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true, "unscheduled": true}
}

func (s *Server) opJobs() map[string]any {
	jobs := s.loadJobs()
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].ID < jobs[j].ID })
	out := make([]any, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, map[string]any{"job_id": j.ID, "goal": j.Goal, "every_secs": j.EverySecs, "next_run": j.NextRun})
	}
	return map[string]any{"ok": true, "jobs": out}
}

// tickJobs fires due jobs once each. Start failures back off linearly and drop
// the job after MaxRetries (the job leaves the file; the failed run record, if
// any, stays for audit).
//
// Dispatching happens outside the lock because starting a run is the slow part
// and holding the schedule lock across it would make every schedule and
// unschedule request wait for it. What keeps the two consistent is that the next
// run time is advanced and persisted *before* the dispatch: a concurrent tick
// therefore sees a job that is no longer due and cannot fire it a second time,
// and a dispatch that fails has its reservation rolled back by the backoff.
func (s *Server) tickJobs() {
	due := s.reserveDueJobs()
	for _, reserved := range due {
		res := s.dispatch(map[string]any{
			"protocol_version": harnessprotocol.Version,
			"op":               "start", "goal": reserved.job.Goal, "provider": reserved.job.Provider,
			"run_id": reserved.runID, "max_turns": reserved.job.MaxTurns,
		})
		if res["ok"] == true {
			s.recordJobOutcome(reserved.job.ID, "started", "", 0)
			continue
		}
		message, _ := res["error"].(string)
		s.recordJobFailure(reserved.job, message)
	}
}

// reservedJob is a job that has claimed its slot and is about to be dispatched.
type reservedJob struct {
	job   Job
	runID string
}

// reserveDueJobs advances every due job's next run and persists that, returning
// what was claimed.
//
// The reservation is the concurrency control. Advancing before dispatching means
// a second tick cannot see the same job as due, so a slow run does not get fired
// again by the next tick — which is the same defect a cron scheduler has when it
// records "started" only after the work finishes.
func (s *Server) reserveDueJobs() []reservedJob {
	jobsMu.Lock()
	defer jobsMu.Unlock()

	jobs := s.loadJobs()
	if len(jobs) == 0 {
		return nil
	}
	now := time.Now().UTC().Unix()
	var claimed []reservedJob
	changed := false
	for i := range jobs {
		if jobs[i].NextRun > now {
			continue
		}
		runID := fmt.Sprintf("job-%s-%d", jobs[i].ID, now)
		jobs[i].NextRun = now + jobs[i].EverySecs
		changed = true
		claimed = append(claimed, reservedJob{job: jobs[i], runID: runID})
	}
	if changed {
		if err := s.saveJobs(jobs); err != nil {
			// The reservation did not land, so nothing may be dispatched: the next
			// tick would legitimately see these jobs as due again, and dispatching
			// now would risk firing them twice.
			return nil
		}
	}
	return claimed
}

// recordJobOutcome writes a successful result, clearing the retry state.
func (s *Server) recordJobOutcome(id, status, message string, retries int) {
	_ = s.mutateJobs(func(jobs []Job) ([]Job, error) {
		for i := range jobs {
			if jobs[i].ID != id {
				continue
			}
			jobs[i].Retries = retries
			jobs[i].LastStatus = status
			jobs[i].LastError = message
		}
		return jobs, nil
	})
}

// recordJobFailure backs a failed job off, or dead-letters it once its retries are
// spent.
func (s *Server) recordJobFailure(job Job, message string) {
	_ = s.mutateJobs(func(jobs []Job) ([]Job, error) {
		kept := make([]Job, 0, len(jobs))
		for _, j := range jobs {
			if j.ID != job.ID {
				kept = append(kept, j)
				continue
			}
			j.Retries++
			j.LastStatus = "failed"
			if message != "" {
				j.LastError = message
			}
			maxRetries := j.MaxRetries
			if maxRetries <= 0 {
				maxRetries = 3
			}
			if j.Retries > maxRetries {
				continue
			}
			j.NextRun = time.Now().UTC().Unix() + j.EverySecs*int64(j.Retries)
			kept = append(kept, j)
		}
		return kept, nil
	})
}
