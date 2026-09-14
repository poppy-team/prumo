package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLockExclusiveAndStaleTakeover(t *testing.T) {
	dir := t.TempDir()
	release, err := AcquireLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AcquireLock(dir); err == nil {
		t.Fatal("second lock must fail while live")
	}
	release()
	// Stale lock (unparsable pid file is treated as stale-takeable).
	if err := os.WriteFile(PIDFile(dir), []byte("not-a-pid"), 0o644); err != nil {
		t.Fatal(err)
	}
	release2, err := AcquireLock(dir)
	if err != nil {
		t.Fatalf("stale lock must be taken over: %v", err)
	}
	release2()
	// Stop on a dead pid clears the stale lock (never signals self here).
	if err := os.WriteFile(PIDFile(dir), []byte("4000000"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Stop(dir); err == nil {
		t.Fatal("stop of dead pid must report stale clearance")
	}
	if _, err := os.Stat(PIDFile(dir)); !os.IsNotExist(err) {
		t.Fatal("stale lock file must be removed")
	}
}

func TestRotateLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	var b strings.Builder
	for i := 0; i < 100; i++ {
		b.WriteString("{\"n\":1}\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RotateLog(path, 20); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if strings.Count(string(data), "\n") != 10 {
		t.Fatalf("expected newest half kept, got %d lines", strings.Count(string(data), "\n"))
	}
	if err := RotateLog(filepath.Join(t.TempDir(), "missing.jsonl"), 20); err != nil {
		t.Fatal("missing file must be fine")
	}
}
