// Daemon supervision: a single-instance lock that actually excludes, a stop
// that waits for the daemon to be gone, and JSONL log rotation. Rotation keeps
// the newest lines; provenance lives in records and knowledge, never only in
// trimmed logs.
package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// LockRecord is what the lock file holds.
//
// A bare pid is not enough to decide whether a lock is live. Pids are reused:
// after a reboot, or after enough short-lived processes, the number in the
// file can belong to something else entirely. Treating that as "already
// running" blocks startup permanently, and treating it as stale lets two
// daemons run. StartTime and BootID name the process, not just the number
// (GAP-104).
type LockRecord struct {
	PID int `json:"pid"`
	// StartTime is the kernel's start-time for the process, in clock ticks since
	// boot. It never changes for a live process and is not reused within a boot.
	StartTime uint64 `json:"start_time"`
	// BootID changes on reboot, which is what makes a pid from a previous boot
	// identifiable as stale.
	BootID string `json:"boot_id"`
	// Version lets a future lock format be recognised rather than misread.
	Version int `json:"version"`
	// AcquiredAt is for a human reading the file, not for correctness.
	AcquiredAt string `json:"acquired_at"`
}

const lockVersion = 1

// PIDFile is the single-instance lock path for a store dir.
func PIDFile(storeDir string) string { return filepath.Join(storeDir, "agentd.pid") }

// AcquireLock claims the single-instance lock.
//
// Exclusion comes from O_EXCL, which is atomic: the kernel creates the file or
// reports that it exists. The previous implementation read the file and then
// wrote it, which two daemons starting together both pass — they both read
// nothing, both write, and both run.
//
// A lock whose process is gone is taken over. A lock whose process is alive and
// whose identity matches is refused. A lock file that cannot be parsed is
// treated as stale and reported, because refusing forever is the one outcome
// with no way forward.
func AcquireLock(storeDir string) (func(), error) {
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		return nil, err
	}
	path := PIDFile(storeDir)
	record := LockRecord{
		PID:        os.Getpid(),
		StartTime:  processStartTime(os.Getpid()),
		BootID:     bootID(),
		Version:    lockVersion,
		AcquiredAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	data, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}

	// The content is written to a temporary and linked into place. Creating the
	// lock with O_EXCL and then writing into it leaves a window in which the file
	// exists and is empty, and a second daemon arriving in that window reads an
	// unreadable lock and takes it over — so two can win. link is atomic and
	// fails when the target exists, so the lock never exists without its content.
	staging := path + ".claim." + strconv.Itoa(os.Getpid())
	if err := os.WriteFile(staging, data, 0o644); err != nil {
		return nil, err
	}
	defer os.Remove(staging)

	if err := os.Link(staging, path); err == nil {
		return func() { releaseLock(path, record) }, nil
	} else if !os.IsExist(err) {
		return nil, err
	}

	// The lock exists. It is ours only if the process it names is gone.
	existing, readErr := ReadLock(path)
	if readErr != nil {
		// Unreadable or corrupt: a stale lock must not be permanent. Take it
		// over and say so, because silently overwriting a live daemon's lock
		// would be the worse failure.
		if takeoverErr := takeOver(path, data); takeoverErr != nil {
			return nil, fmt.Errorf("daemon lock at %s is unreadable (%v) and could not be replaced: %w", path, readErr, takeoverErr)
		}
		return func() { releaseLock(path, record) }, nil
	}
	if existing.PID > 0 && processAlive(existing) && sameIdentity(existing, record) {
		// Including when the holder is this very process. A second acquire from
		// one process is a programming error, and letting it succeed would mean
		// the lock does not exclude — which is the one property it exists for.
		return nil, fmt.Errorf("daemon already running (pid %d)", existing.PID)
	}
	if err := takeOver(path, data); err != nil {
		return nil, err
	}
	return func() { releaseLock(path, record) }, nil
}

// takeOver replaces a lock that is no longer held. The rename is atomic, so a
// competing takeover cannot interleave into a half-written file.
func takeOver(path string, data []byte) error {
	tmp := path + ".takeover"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// releaseLock removes the lock, but only when it is still ours. A release that
// deleted a lock another process had since taken would let a second daemon in.
func releaseLock(path string, mine LockRecord) {
	current, err := ReadLock(path)
	if err != nil {
		return
	}
	if current.PID != mine.PID || current.StartTime != mine.StartTime {
		return
	}
	_ = os.Remove(path)
}

// ReadLock reads and parses a lock record.
//
// The bare-integer form written by earlier versions is still accepted. A lock
// file left on disk by the previous format is a real lock, and reading it as
// corrupt would take it over — which is how two daemons end up running. An
// unreadable record is weaker evidence than an old one, so the old one wins.
func ReadLock(path string) (LockRecord, error) {
	var record LockRecord
	data, err := os.ReadFile(path)
	if err != nil {
		return record, err
	}
	if err := json.Unmarshal(data, &record); err != nil {
		legacy, legacyErr := strconv.Atoi(strings.TrimSpace(string(data)))
		if legacyErr != nil || legacy <= 0 {
			return record, err
		}
		return LockRecord{PID: legacy}, nil
	}
	if record.PID <= 0 {
		return record, fmt.Errorf("lock at %s names no process", path)
	}
	return record, nil
}

// ReadPID returns the locked pid, if any.
func ReadPID(storeDir string) (int, error) {
	record, err := ReadLock(PIDFile(storeDir))
	if err != nil {
		return 0, err
	}
	return record.PID, nil
}

// sameIdentity reports whether two records name the same process. Comparing the
// pid alone is the check that let a reused pid pass as a live daemon.
func sameIdentity(a, b LockRecord) bool {
	if a.PID != b.PID {
		return false
	}
	// Where the kernel gives us a start time, it is the discriminator. A lock
	// written before this field existed cannot be matched, and a pid without a
	// start time is treated as live rather than silently taken over.
	if a.StartTime == 0 || b.StartTime == 0 {
		return true
	}
	return a.StartTime == b.StartTime && a.BootID == b.BootID
}

func processAlive(record LockRecord) bool {
	// A zombie is not running. kill(pid, 0) succeeds for one, so a probe built
	// only on that reports an exited process as alive and a stop waits forever
	// for something that already stopped.
	if state := processState(record.PID); state == 'Z' || state == 'X' {
		return false
	}
	err := syscall.Kill(record.PID, 0)
	if err == nil {
		return true
	}
	// EPERM means the process exists and belongs to someone else, which is
	// exactly the case that must not be read as "gone".
	return err == syscall.EPERM
}

// Stop signals a running daemon and waits for it to be gone.
//
// It returns when the daemon has actually exited, or reports that it did not.
// The previous version sent SIGTERM and returned at once, so every caller went
// on believing the daemon was gone while it was still draining runs (GAP-104).
//
// A lock whose process cannot be signalled at all is left in place. Removing it
// would let a second daemon start beside one that is still running, which is
// the single worst outcome here; a lock nobody can clear is merely inconvenient.
func Stop(storeDir string) error {
	record, err := ReadLock(PIDFile(storeDir))
	if err != nil {
		return fmt.Errorf("no daemon lock: %w", err)
	}
	if !processAlive(record) {
		_ = os.Remove(PIDFile(storeDir))
		return fmt.Errorf("stale lock cleared (pid %d unreachable)", record.PID)
	}
	if err := syscall.Kill(record.PID, syscall.SIGTERM); err != nil {
		return fmt.Errorf("cannot signal daemon (pid %d): %w", record.PID, err)
	}
	return waitForExit(record, stopGracePeriod, stopPollInterval)
}

const (
	// stopGracePeriod is how long a daemon gets to drain its runs. It is long
	// enough for a model call to return, which is the thing draining waits on.
	stopGracePeriod  = 10 * time.Second
	stopPollInterval = 25 * time.Millisecond
)

func waitForExit(record LockRecord, grace, poll time.Duration) error {
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if !processAlive(record) {
			return nil
		}
		time.Sleep(poll)
	}
	return fmt.Errorf("daemon (pid %d) did not exit within %s of SIGTERM; it is still running", record.PID, grace)
}

// RotateLog caps a JSONL file at maxLines, keeping the newest half.
//
// It rewrites the file, which is O(n) per call. The event path called it once
// per event, so a run's log cost O(n²) to produce (GAP-153); the fix there is
// to stop calling it per event. This stays a rewrite because rotating in place
// is not something a plain append-only file can do, and capping by rewriting is
// the honest implementation of "keep the newest half".
func RotateLog(path string, maxLines int) error {
	if maxLines < 10 {
		maxLines = 10
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) <= maxLines {
		return nil
	}
	keep := lines[len(lines)-maxLines/2:]
	return os.WriteFile(path, []byte(strings.Join(keep, "\n")+"\n"), 0o644)
}
