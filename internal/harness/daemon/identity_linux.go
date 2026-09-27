//go:build linux

package daemon

import (
	"os"
	"strconv"
	"strings"
)

// processStartTime returns the kernel's start-time for a pid, in clock ticks
// since boot. It is stable for the life of a process and, within a boot, is not
// reused by another one — which is exactly the property a pid lacks.
func processStartTime(pid int) uint64 {
	fields := statFields(pid)
	if fields == nil {
		return 0
	}
	// After the command, field 3 is state; starttime is field 22 overall, which
	// is index 19 of this slice.
	const startTimeIndex = 19
	if len(fields) <= startTimeIndex {
		return 0
	}
	value, err := strconv.ParseUint(fields[startTimeIndex], 10, 64)
	if err != nil {
		return 0
	}
	return value
}

// processState returns the single-letter process state, or 0 when unknown.
//
// It exists because kill(pid, 0) succeeds for a zombie. A process that has
// exited but has not been reaped still has an entry in the process table, so a
// liveness probe built on kill reports it as running — and a stop that waits
// for it waits forever for something that already stopped.
func processState(pid int) byte {
	fields := statFields(pid)
	if fields == nil || len(fields) == 0 {
		return 0
	}
	if len(fields[0]) == 0 {
		return 0
	}
	return fields[0][0]
}

func statFields(pid int) []string {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return nil
	}
	// The second field is the command name in parentheses and may itself contain
	// spaces or parentheses, so the fields after it are located from the last
	// ')' rather than by splitting the whole line.
	closing := strings.LastIndexByte(string(data), ')')
	if closing < 0 {
		return nil
	}
	return strings.Fields(string(data[closing+1:]))
}

// bootID identifies this boot. It changes on reboot, which is what makes a pid
// recorded before one identifiable as stale.
func bootID() string {
	data, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
