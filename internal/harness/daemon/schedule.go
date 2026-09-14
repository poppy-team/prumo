// Scheduled runs (H16 first slice): cron-like jobs firing headless runs
// inside the daemon. Jobs persist in jobs.json; the serve loop ticks every
// 5s and starts due jobs with deterministic run ids (job-<id>-<unix>).
// Missed ticks fire once (no catch-up storms).
package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

func (s *Server) saveJobs(jobs []Job) {
	data, _ := json.MarshalIndent(jobs, "", "  ")
	_ = os.MkdirAll(s.StoreDir, 0o755)
	tmp := s.jobsPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, s.jobsPath())
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
	jobs := s.loadJobs()
	for _, j := range jobs {
		if j.ID == id {
			return map[string]any{"ok": false, "error": "job already scheduled: " + id}
		}
	}
	now := time.Now().UTC().Unix()
	jobs = append(jobs, Job{ID: id, Goal: goal, Provider: provider, EverySecs: int64(every),
		MaxTurns: maxTurns, NextRun: now + int64(every), CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)})
	s.saveJobs(jobs)
	return map[string]any{"ok": true, "job_id": id}
}

func (s *Server) opUnschedule(msg map[string]any) map[string]any {
	id := str(msg, "job_id")
	if id == "" {
		return map[string]any{"ok": false, "error": "job_id required"}
	}
	jobs := s.loadJobs()
	kept := jobs[:0]
	found := false
	for _, j := range jobs {
		if j.ID == id {
			found = true
			continue
		}
		kept = append(kept, j)
	}
	if !found {
		return map[string]any{"ok": false, "error": "unknown job " + id}
	}
	s.saveJobs(kept)
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

// tickJobs fires due jobs once each. Start failures back off linearly and
// drop the job after MaxRetries (dead-lettered in the jobs file removal;
// the failed run record, if any, stays for audit).
func (s *Server) tickJobs() {
	jobs := s.loadJobs()
	if len(jobs) == 0 {
		return
	}
	now := time.Now().UTC().Unix()
	changed := false
	kept := jobs[:0]
	for i := range jobs {
		if jobs[i].NextRun > now {
			kept = append(kept, jobs[i])
			continue
		}
		maxRetries := jobs[i].MaxRetries
		if maxRetries <= 0 {
			maxRetries = 3
		}
		runID := fmt.Sprintf("job-%s-%d", jobs[i].ID, now)
		res := s.dispatch(map[string]any{
			"op": "start", "goal": jobs[i].Goal, "provider": jobs[i].Provider,
			"run_id": runID, "max_turns": jobs[i].MaxTurns,
		})
		if res["ok"] == true {
			jobs[i].Retries = 0
			jobs[i].LastStatus = "started"
			jobs[i].LastError = ""
			jobs[i].NextRun = now + jobs[i].EverySecs
			kept = append(kept, jobs[i])
			changed = true
			continue
		}
		jobs[i].Retries++
		jobs[i].LastStatus = "failed"
		if msg, _ := res["error"].(string); msg != "" {
			jobs[i].LastError = msg
		}
		if jobs[i].Retries > maxRetries {
			continue // dead-letter: drop, record stays queryable via list
		}
		jobs[i].NextRun = now + jobs[i].EverySecs*int64(jobs[i].Retries)
		kept = append(kept, jobs[i])
		changed = true
	}
	if changed || len(kept) != len(jobs) {
		s.saveJobs(kept)
	}
}
