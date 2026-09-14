package daemon

import (
	"testing"
	"time"
)

func TestScheduleTickFiresOnce(t *testing.T) {
	srv := New(t.TempDir()+"/s.sock", t.TempDir(), fakeDeps(false))
	res := srv.dispatch(map[string]any{"op": "schedule", "goal": "tick", "job_id": "j1", "every_secs": float64(60)})
	if res["ok"] != true {
		t.Fatalf("schedule failed: %v", res)
	}
	if res := srv.dispatch(map[string]any{"op": "schedule", "goal": "x", "job_id": "j1", "every_secs": float64(60)}); res["ok"] != false {
		t.Fatal("duplicate job must fail")
	}
	if res := srv.dispatch(map[string]any{"op": "schedule", "goal": "x", "every_secs": float64(1)}); res["ok"] != false {
		t.Fatal("sub-minimum interval must fail")
	}
	// Force due and tick twice: exactly one run (no catch-up storm).
	srv.saveJobs([]Job{{ID: "j1", Goal: "tick", Provider: "fake", EverySecs: 3600, MaxTurns: 1, NextRun: time.Now().UTC().Unix() - 1}})
	srv.tickJobs()
	srv.tickJobs()
	runs := 0
	for _, r := range srv.opList()["runs"].([]any) {
		if m, _ := r.(map[string]any); m != nil {
			if id, _ := m["run_id"].(string); len(id) > 4 && id[:4] == "job-" {
				runs++
			}
		}
	}
	if runs != 1 {
		t.Fatalf("expected exactly one fired run, got %d", runs)
	}
	if res := srv.dispatch(map[string]any{"op": "unschedule", "job_id": "j1"}); res["ok"] != true {
		t.Fatalf("unschedule failed: %v", res)
	}
	if res := srv.dispatch(map[string]any{"op": "unschedule", "job_id": "j1"}); res["ok"] != false {
		t.Fatal("double unschedule must fail")
	}
	jobs := srv.dispatch(map[string]any{"op": "jobs"})
	if list, _ := jobs["jobs"].([]any); len(list) != 0 {
		t.Fatalf("jobs must be empty: %v", list)
	}
	// Drain background runs before TempDir cleanup (no writer may race it).
	deadline := time.Now().Add(5 * time.Second)
	for {
		srv.mu.Lock()
		active := len(srv.runs)
		srv.mu.Unlock()
		if active == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("background runs did not drain")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
