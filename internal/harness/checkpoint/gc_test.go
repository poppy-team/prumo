package checkpoint

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
)

func age(t *testing.T, path string, days int) {
	t.Helper()
	ts := time.Now().AddDate(0, 0, -days)
	if err := os.Chtimes(path, ts, ts); err != nil {
		t.Fatal(err)
	}
}

func TestGCCollectsOnlyOrphans(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	// Live run: an in-flight record + aged artifacts (must survive:
	// provenance). Liveness is read from the record, not from the checkpoint
	// being present — a finished run keeps its checkpoint, so "has a
	// checkpoint" meant nothing ever was collected (GAP-163).
	live := agent.Checkpoint{ID: "cp-live", RunID: "R-live", State: agent.NativeAgentState{RunID: "R-live"}}
	if err := s.Save(live); err != nil {
		t.Fatal(err)
	}
	writeRunRecord(t, dir, "R-live", "running")
	liveEv := filepath.Join(dir, "evidence-R-live.json")
	if err := os.WriteFile(liveEv, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	age(t, liveEv, 60)
	// Orphan run: aged artifacts, no checkpoints (collectible).
	orphEv := filepath.Join(dir, "events-R-ghost.jsonl")
	if err := os.WriteFile(orphEv, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	age(t, orphEv, 60)
	// Fresh orphan (too young to collect).
	fresh := filepath.Join(dir, "events-R-young.jsonl")
	if err := os.WriteFile(fresh, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := s.GC(RetentionPolicy{KeepCheckpoints: 5, MaxAgeDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.ArtifactsRemoved) != 1 || rep.ArtifactsRemoved[0] != "events-R-ghost.jsonl" {
		t.Fatalf("only the aged orphan collects: %+v", rep)
	}
	for _, p := range []string{liveEv, fresh} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("must survive: %s", p)
		}
	}
	if _, err := s.Load("cp-live"); err != nil {
		t.Fatal("live checkpoint must survive")
	}
}

// writeRunRecord writes the daemon's per-run record, which is what the
// collector reads to decide whether a run is still going.
func writeRunRecord(t *testing.T, dir, runID, status string) {
	t.Helper()
	body := `{"run_id":"` + runID + `","status":"` + status + `"}`
	if err := os.WriteFile(filepath.Join(dir, "daemon-run-"+runID+".json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A completed run keeps its checkpoint forever, so the old rule — live means
// "owns a checkpoint" — made every finished run permanently live. The
// collector therefore collected nothing, ever, and the diffs prefix was not
// even in the list of artifacts it knew about (GAP-163).
func TestGCCollectsAFinishedRunThatStillHasItsCheckpoint(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.Save(agent.Checkpoint{
		ID: "cp-done", RunID: "R-done", State: agent.NativeAgentState{RunID: "R-done"},
	}); err != nil {
		t.Fatal(err)
	}
	writeRunRecord(t, dir, "R-done", "complete")

	for _, name := range []string{
		"events-R-done.jsonl", "evidence-R-done.json", "diffs-R-done.jsonl",
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		age(t, path, 60)
	}

	rep, err := s.GC(RetentionPolicy{KeepCheckpoints: 5, MaxAgeDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	removed := map[string]bool{}
	for _, name := range rep.ArtifactsRemoved {
		removed[name] = true
	}
	for _, name := range []string{
		"events-R-done.jsonl", "evidence-R-done.json", "diffs-R-done.jsonl",
	} {
		if !removed[name] {
			t.Errorf("a finished run's %s survived GC; removed=%v", name, rep.ArtifactsRemoved)
		}
	}
}

func TestGCKeepsARunWaitingForAHuman(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.Save(agent.Checkpoint{
		ID: "cp-wait", RunID: "R-wait", State: agent.NativeAgentState{RunID: "R-wait"},
	}); err != nil {
		t.Fatal(err)
	}
	// Awaiting approval is not finished: the run will write more, and the answer
	// it is waiting for may arrive tomorrow.
	writeRunRecord(t, dir, "R-wait", "awaiting_approval")
	path := filepath.Join(dir, "events-R-wait.jsonl")
	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	age(t, path, 60)

	if _, err := s.GC(RetentionPolicy{KeepCheckpoints: 5, MaxAgeDays: 30}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("a run waiting for a human had its events collected: %v", err)
	}
}

func TestGCKeepsARunWhoseRecordCannotBeRead(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	path := filepath.Join(dir, "daemon-run-R-broken.json")
	if err := os.WriteFile(path, []byte("not json at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	events := filepath.Join(dir, "events-R-broken.jsonl")
	if err := os.WriteFile(events, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	age(t, events, 60)

	if _, err := s.GC(RetentionPolicy{KeepCheckpoints: 5, MaxAgeDays: 30}); err != nil {
		t.Fatal(err)
	}
	// An unaccountable run is not a finished run. Collecting its artifacts makes
	// a resumable run unresumable.
	if _, err := os.Stat(events); err != nil {
		t.Fatalf("a run with an unreadable record had its events collected: %v", err)
	}
}
