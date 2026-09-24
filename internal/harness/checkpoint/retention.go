package checkpoint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RetentionPolicy bounds store growth without dangling provenance
// (GAP-020): checkpoints prune to Keep per run; non-checkpoint artifacts
// older than MaxAgeDays are collected only when their run keeps no
// checkpoints (orphaned partial runs). Daemon run records are queryable
// history and never collected here.
type RetentionPolicy struct {
	KeepCheckpoints int
	MaxAgeDays      int
}

// DefaultRetention keeps 5 checkpoints and collects 30-day orphans.
func DefaultRetention() RetentionPolicy { return RetentionPolicy{KeepCheckpoints: 5, MaxAgeDays: 30} }

// GCReport counts what collection did.
type GCReport struct {
	CheckpointsPruned int      `json:"checkpoints_pruned"`
	ArtifactsRemoved  []string `json:"artifacts_removed,omitempty"`
}

// artifactPrefixes names the per-run artifacts the collector owns. A run's
// diffs and its record were missing, so both survived every GC: a diff is the
// largest artifact a run produces and it was the one thing guaranteed never to
// be collected (GAP-163).
var artifactPrefixes = []string{
	"events-", "obs-", "permissions-", "knowledge-", "budget-",
	"evidence-", "context-", "diffs-", "daemon-run-",
}

func artifactRunID(name string) (string, bool) {
	for _, prefix := range artifactPrefixes {
		if len(name) > len(prefix) && name[:len(prefix)] == prefix {
			rest := name[len(prefix):]
			if i := strings.LastIndex(rest, "."); i > 0 {
				return rest[:i], true
			}
		}
	}
	return "", false
}

// GC enforces the retention policy on the store dir.
func (s *Store) GC(policy RetentionPolicy) (GCReport, error) {
	var rep GCReport
	if policy.KeepCheckpoints < 1 {
		policy.KeepCheckpoints = 1
	}
	pruned, err := s.Prune(policy.KeepCheckpoints)
	if err != nil {
		return rep, err
	}
	rep.CheckpointsPruned = pruned
	if policy.MaxAgeDays <= 0 {
		return rep, nil
	}
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return rep, nil
		}
		return rep, err
	}
	// Which runs are still live, according to the run's own record.
	//
	// It used to be "owns a checkpoint", and a completed run keeps its pruned
	// checkpoint forever — so every run that ever finished was permanently live,
	// and the collector collected nothing at all. Liveness is a property of the
	// run's status, not of which files it left behind (GAP-163).
	live := map[string]bool{}
	for _, e := range entries {
		runID, ok := artifactRunID(e.Name())
		if !ok || !strings.HasPrefix(e.Name(), "daemon-run-") {
			continue
		}
		record, err := s.readRunRecord(filepath.Join(s.Dir, e.Name()))
		if err != nil {
			// A record that cannot be read is not evidence that the run is
			// finished, so the run is treated as live and kept. Collecting the
			// artifacts of a run nobody can account for is how a resumable run
			// becomes unresumable.
			live[runID] = true
			continue
		}
		if record == nil {
			continue
		}
		live[runID] = record.Status == "" || isInFlightStatus(record.Status)
	}
	cutoff := time.Now().AddDate(0, 0, -policy.MaxAgeDays)
	for _, e := range entries {
		runID, ok := artifactRunID(e.Name())
		if !ok || live[runID] {
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(s.Dir, e.Name())); err == nil {
			rep.ArtifactsRemoved = append(rep.ArtifactsRemoved, e.Name())
		}
	}
	sort.Strings(rep.ArtifactsRemoved)
	return rep, nil
}

// runRecord is the part of a daemon run record the collector needs.
type runRecord struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
}

// readRunRecord reads one run record, returning nil when the file is not one.
func (s *Store) readRunRecord(path string) (*runRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var record runRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}
	if record.RunID == "" {
		return nil, nil
	}
	return &record, nil
}

// isInFlightStatus reports whether a run in this state is still going to write
// more artifacts.
func isInFlightStatus(status string) bool {
	switch status {
	case "running", "awaiting_approval", "interrupted", "yielded":
		return true
	default:
		return false
	}
}
