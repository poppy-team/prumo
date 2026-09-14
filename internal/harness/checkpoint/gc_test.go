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
	// Live run: checkpoint + aged artifacts (must survive: provenance).
	live := agent.Checkpoint{ID: "cp-live", RunID: "R-live", State: agent.NativeAgentState{RunID: "R-live"}}
	if err := s.Save(live); err != nil {
		t.Fatal(err)
	}
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
