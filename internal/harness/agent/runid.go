package agent

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

// runCounter disambiguates two ids minted in the same process within the same
// millisecond. The random suffix is the real disambiguator across processes;
// this only keeps two ids in one process from being identical when the clock
// and the entropy both fail to move.
var runCounter atomic.Uint64

// NewRunID mints an id for a run that nobody named.
//
// A generated id used to be a constant in one place (`R-agent-1`, so every
// `prumo agent run` without `--run` was the same run) and a counter from zero in
// another (`R-daemon-1`, restarting at one on every boot, so today's daemon
// named a run exactly as last week's did). Both collide with records already on
// disk, which is the one thing a run id has to be able to avoid (GAP-136).
//
// The shape is prefix-UTC-timestamp-process-random-counter. The timestamp makes
// the id sortable by eye; the process and random parts make it unique even when
// two processes start in the same millisecond, which is what a counter in a
// daemon that a supervisor restarts immediately does.
func NewRunID(prefix string) string {
	var random [4]byte
	// A failure to read entropy is not a reason to hand back a predictable id:
	// the counter and the process are still there, and the timestamp narrows the
	// window to a millisecond of one process.
	_, _ = rand.Read(random[:])
	return fmt.Sprintf("R-%s-%s-%d-%s-%d",
		prefix,
		time.Now().UTC().Format("20060102T150405"),
		os.Getpid(),
		hex.EncodeToString(random[:]),
		runCounter.Add(1),
	)
}
