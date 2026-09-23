//go:build !linux

package daemon

// processStartTime has no portable source, so it reports 0. The lock then falls
// back to pid liveness alone, which is weaker: a reused pid reads as a live
// daemon. The consequence is that a stale lock on a platform without /proc has
// to be removed by hand after a crash. The behaviour is stated rather than
// approximated, because a plausible-looking start time would be a guess.
func processStartTime(int) uint64 { return 0 }

// bootID is unknown off Linux, for the same reason.
func bootID() string { return "" }
