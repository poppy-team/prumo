// Daemon supervision: single-instance PID lock with stale takeover,
// graceful stop, and JSONL log rotation. Rotation keeps the newest lines;
// provenance lives in records/knowledge, never only in trimmed logs.
package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// PIDFile is the single-instance lock path for a store dir.
func PIDFile(storeDir string) string { return filepath.Join(storeDir, "agentd.pid") }

// AcquireLock claims the single-instance lock, taking over stale locks
// (pid not running). Returns a release func removing the lock.
func AcquireLock(storeDir string) (func(), error) {
	path := PIDFile(storeDir)
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		return nil, err
	}
	if data, err := os.ReadFile(path); err == nil {
		if pid, perr := strconv.Atoi(strings.TrimSpace(string(data))); perr == nil {
			if err := syscall.Kill(pid, 0); err == nil {
				return nil, fmt.Errorf("daemon already running (pid %d)", pid)
			}
		}
	}
	if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		return nil, err
	}
	return func() { _ = os.Remove(path) }, nil
}

// ReadPID returns the locked pid, if any.
func ReadPID(storeDir string) (int, error) {
	data, err := os.ReadFile(PIDFile(storeDir))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// Stop signals a running daemon (SIGTERM) and clears a stale lock.
func Stop(storeDir string) error {
	pid, err := ReadPID(storeDir)
	if err != nil {
		return fmt.Errorf("no daemon lock: %w", err)
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		_ = os.Remove(PIDFile(storeDir))
		return fmt.Errorf("stale lock cleared (pid %d unreachable): %w", pid, err)
	}
	return nil
}

// RotateLog caps a JSONL file at maxLines, keeping the newest half.
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
	// Drop trailing empty segment.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) <= maxLines {
		return nil
	}
	keep := lines[len(lines)-maxLines/2:]
	return os.WriteFile(path, []byte(strings.Join(keep, "\n")+"\n"), 0o644)
}
