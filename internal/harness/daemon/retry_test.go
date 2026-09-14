package daemon

import (
	"errors"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/model"
)

func TestJobRetryDeadLetters(t *testing.T) {
	deps := fakeDeps(false)
	deps.NewProvider = func(string, string, string, string) (model.Provider, error) {
		return nil, errors.New("no such provider")
	}
	srv := New(t.TempDir()+"/s.sock", t.TempDir(), deps)
	srv.saveJobs([]Job{{ID: "bad", Goal: "x", Provider: "no-such-provider", EverySecs: 3600, MaxRetries: 2, NextRun: time.Now().UTC().Unix() - 1}})
	for i := 0; i < 5; i++ {
		// Backoff pushes NextRun out; force due again to simulate time passing.
		jobs := srv.loadJobs()
		for j := range jobs {
			jobs[j].NextRun = time.Now().UTC().Unix() - 1
		}
		srv.saveJobs(jobs)
		srv.tickJobs()
	}
	jobs := srv.loadJobs()
	if len(jobs) != 0 {
		t.Fatalf("exhausted job must dead-letter, got %+v", jobs)
	}
	// No run records leaked from failed starts.
	if list := srv.opList()["runs"].([]any); len(list) != 0 {
		t.Fatalf("failed starts must not pollute runs: %v", list)
	}
}
