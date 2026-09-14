package checkpoint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
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

var artifactPrefixes = []string{"events-", "obs-", "permissions-", "knowledge-", "budget-", "evidence-", "context-"}

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
	// Live runs: any run id still owning checkpoints.
	live := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if len(name) < 12 || name[:11] != "checkpoint-" || !strings.HasSuffix(name, ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.Dir, name))
		if err != nil {
			continue
		}
		var cp agent.Checkpoint
		if err := json.Unmarshal(data, &cp); err == nil && cp.RunID != "" {
			live[cp.RunID] = true
		}
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
