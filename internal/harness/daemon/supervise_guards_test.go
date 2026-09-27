package daemon

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// The lock read the file and then wrote it. Two daemons starting together both
// read nothing, both wrote, and both ran: the sequence is not atomic and
// nothing in it made it one (GAP-104).

func TestOnlyOneOfManyConcurrentAcquiresWins(t *testing.T) {
	dir := t.TempDir()
	const attempts = 24
	var wg sync.WaitGroup
	var mu sync.Mutex
	winners := 0
	losers := 0
	releases := []func(){}

	start := make(chan struct{})
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // release them together, as two daemons would
			release, err := AcquireLock(dir)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				losers++
				return
			}
			winners++
			releases = append(releases, release)
		}()
	}
	close(start)
	wg.Wait()

	if winners != 1 {
		t.Fatalf("%d of %d concurrent acquires succeeded; the lock must admit exactly one", winners, attempts)
	}
	if losers != attempts-1 {
		t.Fatalf("expected %d refusals, got %d", attempts-1, losers)
	}
	for _, release := range releases {
		release()
	}
}

func TestReleaseLeavesNoLockBehind(t *testing.T) {
	dir := t.TempDir()
	release, err := AcquireLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(PIDFile(dir)); err != nil {
		t.Fatalf("the lock file must exist while held: %v", err)
	}
	release()
	if _, err := os.Stat(PIDFile(dir)); !os.IsNotExist(err) {
		t.Fatal("release must remove the lock")
	}
}

func TestReleaseDoesNotRemoveALockSomebodyElseTook(t *testing.T) {
	// A release that deleted a lock another process had since taken would let a
	// second daemon in, so ownership is checked before removing.
	dir := t.TempDir()
	release, err := AcquireLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	other := LockRecord{PID: 424242, StartTime: 99, BootID: "other-boot"}
	data, _ := json.Marshal(other)
	if err := os.WriteFile(PIDFile(dir), data, 0o644); err != nil {
		t.Fatal(err)
	}
	release()
	current, err := ReadLock(PIDFile(dir))
	if err != nil {
		t.Fatalf("a lock owned by another process must survive a stale release: %v", err)
	}
	if current.PID != other.PID {
		t.Fatalf("the lock was removed or overwritten: %+v", current)
	}
}

// A pid is reused. After enough short-lived processes, the number in the file
// belongs to something else, and "kill(pid, 0) succeeded" then reads as a live
// daemon that blocks startup forever.

func TestAReusedPidDoesNotBlockStartup(t *testing.T) {
	dir := t.TempDir()
	// The test process is certainly alive, so this pid is live; the start time
	// is wrong, which is what a reused pid looks like.
	stale := LockRecord{
		PID:        os.Getpid(),
		StartTime:  processStartTime(os.Getpid()) + 12_345,
		BootID:     bootID(),
		Version:    lockVersion,
		AcquiredAt: "2026-01-01T00:00:00Z",
	}
	data, _ := json.Marshal(stale)
	if err := os.WriteFile(PIDFile(dir), data, 0o644); err != nil {
		t.Fatal(err)
	}

	release, err := AcquireLock(dir)
	if err != nil {
		t.Fatalf("a lock naming a reused pid must be taken over, not obeyed: %v", err)
	}
	release()
}

func TestALockFromAnotherBootIsStale(t *testing.T) {
	dir := t.TempDir()
	stale := LockRecord{PID: os.Getpid(), StartTime: processStartTime(os.Getpid()), BootID: "a-previous-boot"}
	data, _ := json.Marshal(stale)
	if err := os.WriteFile(PIDFile(dir), data, 0o644); err != nil {
		t.Fatal(err)
	}
	release, err := AcquireLock(dir)
	if err != nil {
		t.Fatalf("a lock from a previous boot must be stale: %v", err)
	}
	release()
}

func TestALegacyBarePidLockIsStillHonouredWhenLive(t *testing.T) {
	// The previous format wrote a bare integer. Reading one as corrupt would
	// take it over, which is how two daemons end up running.
	dir := t.TempDir()
	if err := os.WriteFile(PIDFile(dir), []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := AcquireLock(dir); err == nil {
		t.Fatal("a live daemon's lock in the old format must still exclude")
	}
}

func TestALegacyBarePidLockIsStaleWhenTheProcessIsGone(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(PIDFile(dir), []byte("4000000"), 0o644); err != nil {
		t.Fatal(err)
	}
	release, err := AcquireLock(dir)
	if err != nil {
		t.Fatalf("a lock for a process that is gone must be taken over: %v", err)
	}
	release()
}

func TestACorruptLockIsTakenOverRatherThanBlockingForever(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(PIDFile(dir), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	release, err := AcquireLock(dir)
	if err != nil {
		t.Fatalf("an unreadable lock must not be permanent: %v", err)
	}
	release()
}

func TestTheLockRecordsEnoughToIdentifyItsHolder(t *testing.T) {
	dir := t.TempDir()
	release, err := AcquireLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	record, err := ReadLock(PIDFile(dir))
	if err != nil {
		t.Fatal(err)
	}
	if record.PID != os.Getpid() {
		t.Fatalf("the lock must name this process, got %d", record.PID)
	}
	if record.StartTime == 0 {
		t.Error("the lock must carry a start time; a bare pid cannot identify a reused number")
	}
	if record.AcquiredAt == "" {
		t.Error("the lock should say when it was taken, for a human reading it")
	}
}

// Stop sent SIGTERM and returned. Every caller went on believing the daemon was
// gone while it was still draining runs (GAP-104).

func TestStopWaitsForTheDaemonToActuallyExit(t *testing.T) {
	dir := t.TempDir()
	// A process that traps SIGTERM and exits after a delay, standing in for a
	// daemon draining a run.
	script := filepath.Join(dir, "slow-exit.sh")
	// A trap that sleeps is not dependable here: the shell may run it as a
	// single short step and exit at once. Resetting a counter on TERM and
	// requiring several more loop iterations before exiting is what actually
	// holds the process open, so a stop that does not wait returns too early.
	body := "#!/bin/sh\ntrap 'n=0' TERM\nn=0\nwhile true; do\n  n=$((n+1))\n  if [ \"$n\" -ge 8 ]; then exit 0; fi\n  sleep 0.05\ndone\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(script)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	// The script installs its trap on its second line. Signalling before that
	// reaches the default handler, which terminates at once, and the test would
	// measure a helper that never installed one.
	waitUntilRunning(t, cmd.Process.Pid)
	writeLockFor(t, dir, cmd.Process.Pid)

	begin := time.Now()
	if err := Stop(dir); err != nil {
		t.Fatalf("Stop must report success once the daemon is gone: %v", err)
	}
	// The helper takes 400ms to exit after the signal. Returning sooner would
	// mean Stop reported a daemon as stopped while it was still running.
	if elapsed := time.Since(begin); elapsed < 300*time.Millisecond {
		t.Fatalf("Stop returned after %s, before the daemon could have exited", elapsed)
	}
	_ = cmd.Wait()
}

func TestStopReportsWhenTheDaemonDoesNotExit(t *testing.T) {
	// A daemon that ignores SIGTERM must be reported, not waited on forever and
	// not reported as stopped.
	dir := t.TempDir()
	script := filepath.Join(dir, "ignore-term.sh")
	body := "#!/bin/sh\ntrap '' TERM\nwhile true; do sleep 0.05; done\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(script)
	if err := cmd.Start(); err != nil {
		t.Skipf("helper unavailable: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	waitUntilRunning(t, cmd.Process.Pid)
	writeLockFor(t, dir, cmd.Process.Pid)

	done := make(chan error, 1)
	go func() { done <- Stop(dir) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Stop must report that the daemon did not exit")
		}
		if !containsAny(err.Error(), "did not exit", "still running") {
			t.Fatalf("the report must say the daemon is still there, got %q", err.Error())
		}
	case <-time.After(stopGracePeriod + 5*time.Second):
		t.Fatalf("Stop must give up after its grace period, not block forever")
	}
}

func TestStopDoesNotRemoveTheLockOfADaemonItCouldNotSignal(t *testing.T) {
	// Removing the lock here would let a second daemon start beside a running
	// one, which is the worst outcome in this file.
	dir := t.TempDir()
	if os.Geteuid() == 0 {
		t.Skip("running as root; the EPERM path cannot be exercised")
	}
	writeLockFor(t, dir, 1) // init: exists, belongs to another user
	err := Stop(dir)
	if err == nil {
		t.Skip("this session can signal pid 1")
	}
	if _, statErr := os.Stat(PIDFile(dir)); statErr != nil {
		t.Fatalf("the lock of a daemon that could not be signalled must be left in place: %v", statErr)
	}
}

func writeLockFor(t *testing.T, dir string, pid int) {
	t.Helper()
	record := LockRecord{PID: pid, StartTime: processStartTime(pid), BootID: bootID(), Version: lockVersion}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(PIDFile(dir), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

// waitUntilRunning gives a freshly spawned helper time to install its signal
// handling before a test signals it.
func waitUntilRunning(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		switch processState(pid) {
		case 'S', 'R':
			time.Sleep(150 * time.Millisecond) // let the script finish its prologue
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("helper (pid %d) never reached a running state", pid)
}
