package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestDaemonKillRecovery SIGKILLs a live daemon and proves crash recovery:
// restart takes over the stale lock and the store stays queryable. No
// run is lost beyond its last atomic record by design.
func TestDaemonKillRecovery(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "prumo-test")
	build := exec.Command("go", "build", "-o", bin, "./cmd/prumo")
	build.Dir = "../.."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %s %v", out, err)
	}
	dir := t.TempDir()
	sock := filepath.Join(dir, "agentd.sock")
	start := func() *exec.Cmd {
		cmd := exec.Command(bin, "agent", "serve", "--path", dir, "--socket", sock)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err != nil {
			t.Fatalf("serve start: %v", err)
		}
		return cmd
	}
	waitUp := func() {
		deadline := time.Now().Add(30 * time.Second)
		for {
			out, err := exec.Command(bin, "agent", "ps", "--socket", sock).CombinedOutput()
			_ = out
			if err == nil {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("daemon did not come up: %s %v", out, err)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
	d1 := start()
	waitUp()
	if err := d1.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatalf("kill: %v", err)
	}
	_, _ = d1.Process.Wait()
	d2 := start()
	defer func() {
		_ = exec.Command(bin, "agent", "stop", "--path", dir).Run()
		_, _ = d2.Process.Wait()
	}()
	waitUp() // stale lock takeover + queryable store
	out, err := exec.Command(bin, "agent", "ps", "--socket", sock).CombinedOutput()
	if err != nil {
		t.Fatalf("ps after kill: %s %v", out, err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".prumo", "runtime", "harness", "agentd.pid")); err != nil {
		t.Fatalf("fresh lock missing: %v", err)
	}
}
