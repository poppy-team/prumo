package daemon

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// saveDiff took the global Server.mu and held it across a file read, a marshal,
// a write and a rename. opList takes the same lock to read the run map, so every
// recorded file change blocked listing, status and run start for the duration of
// the disk write (GAP-154).

func TestRecordingADiffDoesNotBlockTheRunMap(t *testing.T) {
	srv := New("test.sock", t.TempDir(), Deps{Workspace: t.TempDir()})
	srv.runs["R-1"] = &activeRun{}

	// Hold the run-map lock the way a long operation would, and check that a
	// diff can still be recorded. With saveDiff on s.mu this deadlocks until the
	// test times out.
	srv.mu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.saveDiff("R-1", "a.txt", "modified", "content")
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		srv.mu.Unlock()
		t.Fatal("recording a diff blocked on the run-map lock; the two are separate resources")
	}
	srv.mu.Unlock()

	if _, err := srv.diffPath("R-1"); err != nil {
		t.Fatalf("diffPath: %v", err)
	}
}

func TestConcurrentDiffWritesDoNotLoseRecords(t *testing.T) {
	// The lock that was moved must still serialise the writers, or the
	// read-modify-write loses records.
	srv := New("test.sock", t.TempDir(), Deps{Workspace: t.TempDir()})
	var wg sync.WaitGroup
	const writers = 20
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			srv.saveDiff("R-conc", filepath.Join("dir", string(rune('a'+n))+".txt"), "modified", "x")
		}(i)
	}
	wg.Wait()

	target, err := srv.diffPath("R-conc")
	if err != nil {
		t.Fatalf("diffPath: %v", err)
	}
	srv.diffsMu.Lock()
	diffs := srv.loadDiffs(target)
	srv.diffsMu.Unlock()
	if len(diffs) != writers {
		t.Fatalf("concurrent writes lost records: %d of %d", len(diffs), writers)
	}
}

func TestDiffPathRejectsARunIDThatEscapes(t *testing.T) {
	srv := New("test.sock", t.TempDir(), Deps{Workspace: t.TempDir()})
	if _, err := srv.diffPath("../escape"); err == nil {
		t.Fatal("a run id that escapes the store must be refused")
	}
	// A refused id must not create anything.
	srv.saveDiff("../escape", "a.txt", "modified", "x")
	if entries, err := filepath.Glob(filepath.Join(srv.StoreDir, "diffs-*")); err == nil && len(entries) != 0 {
		t.Fatalf("a refused run id created %v", entries)
	}
}
