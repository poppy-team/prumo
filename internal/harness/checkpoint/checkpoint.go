// Package checkpoint implements crash-safe checkpoint/resume with a
// file-backed store and a side-effect journal with idempotency keys.
// Goal: duplicate observable side effect rate = zero.
package checkpoint

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/safepath"
)

// Store persists checkpoints + effects under a directory.
type Store struct {
	Dir string
}

func New(dir string) *Store { return &Store{Dir: dir} }

// path names a checkpoint file. The id is validated because it comes from a
// run and becomes a filename: a value containing a separator previously walked
// out of the store (GAP-112).
func (s *Store) path(id string) (string, error) {
	if err := safepath.ValidateID("checkpoint_id", id); err != nil {
		return "", err
	}
	return filepath.Join(s.Dir, "checkpoint-"+id+".json"), nil
}

func (s *Store) fxPath() string { return filepath.Join(s.Dir, "side-effects.json") }

// Save writes atomically (tmp + rename) with a SHA-256 fingerprint.
func (s *Store) Save(cp agent.Checkpoint) error {
	if cp.ID == "" || cp.RunID == "" {
		return fmt.Errorf("checkpoint requires id and run_id")
	}
	if cp.CreatedAt == "" {
		cp.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	_ = sum
	path, err := s.path(cp.ID)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Load reads a checkpoint by id.
func (s *Store) Load(id string) (agent.Checkpoint, error) {
	var cp agent.Checkpoint
	path, err := s.path(id)
	if err != nil {
		return cp, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cp, err
	}
	if err := json.Unmarshal(data, &cp); err != nil {
		return cp, err
	}
	return cp, nil
}

// Latest returns the most recently created checkpoint for a run.
func (s *Store) Latest(runID string) (agent.Checkpoint, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return agent.Checkpoint{}, err
	}
	var best agent.Checkpoint
	found := false
	for _, e := range entries {
		if len(e.Name()) < 12 || e.Name()[:11] != "checkpoint-" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.Dir, e.Name()))
		if err != nil {
			continue
		}
		var cp agent.Checkpoint
		if err := json.Unmarshal(data, &cp); err != nil {
			continue
		}
		if cp.RunID != runID {
			continue
		}
		if !found || cp.CreatedAt > best.CreatedAt {
			best, found = cp, true
		}
	}
	if !found {
		return agent.Checkpoint{}, fmt.Errorf("no checkpoint for run %s", runID)
	}
	return best, nil
}

// Prune keeps the newest keep checkpoints per run, deleting older files.
// Retention without provenance loss: records, events and knowledge stay;
// only superseded intermediate checkpoints are collected.
func (s *Store) Prune(keep int) (int, error) {
	if keep < 1 {
		keep = 1
	}
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	byRun := map[string][]string{}
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
		if err := json.Unmarshal(data, &cp); err != nil {
			continue
		}
		byRun[cp.RunID] = append(byRun[cp.RunID], name)
	}
	removed := 0
	for runID, names := range byRun {
		if len(names) <= keep {
			continue
		}
		type stamped struct {
			name string
			at   string
		}
		var ordered []stamped
		for _, name := range names {
			data, _ := os.ReadFile(filepath.Join(s.Dir, name))
			var cp agent.Checkpoint
			_ = json.Unmarshal(data, &cp)
			ordered = append(ordered, stamped{name, cp.CreatedAt})
		}
		sort.Slice(ordered, func(i, j int) bool { return ordered[i].at < ordered[j].at })
		for _, old := range ordered[:len(ordered)-keep] {
			if err := os.Remove(filepath.Join(s.Dir, old.name)); err == nil {
				removed++
			}
		}
		_ = runID
	}
	return removed, nil
}

func (s *Store) loadEffects() []agent.PendingEffect {
	var out []agent.PendingEffect
	data, err := os.ReadFile(s.fxPath())
	if err != nil {
		return out
	}
	_ = json.Unmarshal(data, &out)
	return out
}

func (s *Store) saveEffects(fx []agent.PendingEffect) error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	sort.Slice(fx, func(i, j int) bool { return fx[i].ID < fx[j].ID })
	data, _ := json.MarshalIndent(fx, "", "  ")
	tmp := s.fxPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.fxPath())
}

// RecordIntent persists intent BEFORE the side effect, and reports whether the
// caller should go ahead.
//
// A pending entry is a promise whose outcome was never recorded — the process
// died between writing the intent and learning the result. What to do about that
// depends on whether the effect is safe to repeat, so the decision is made here,
// from the recovery policy, and not left to the caller. The old behaviour
// returned true and executed, which re-ran an unknown-outcome effect on every
// resume (GAP-126).
//
//	applied already, any policy      → false, skip: the effect is known done
//	pending, policy retry            → true,  go ahead: repeating is the plan
//	pending, policy skip             → false, skip: a human decides
//	pending, policy fail / unknown   → error: fail closed
//	failed or rolled_back             → true,  go ahead: it did not take
func (s *Store) RecordIntent(fx agent.PendingEffect) (bool, error) {
	if fx.IdempotencyKey == "" {
		return false, fmt.Errorf("pending effect requires idempotency_key")
	}
	all := s.loadEffects()
	for _, e := range all {
		if e.IdempotencyKey == fx.IdempotencyKey && e.Status == agent.EffectApplied {
			return false, nil
		}
		if e.ID == fx.ID {
			switch e.Status {
			case agent.EffectApplied:
				return false, nil
			case agent.EffectFailed, agent.EffectRolledBack:
				// It did not take. Recording the intent again is the right move.
				return true, nil
			case agent.EffectPending:
				switch recoveryPolicy(e) {
				case "retry":
					return true, nil
				case "skip":
					return false, nil
				case "fail":
					return false, fmt.Errorf("effect %s has an unknown outcome and its recovery policy is fail", e.ID)
				default:
					// No policy is not permission to repeat. An effect with an
					// unknown outcome that nobody planned for is the case where
					// guessing is worst.
					return false, fmt.Errorf("effect %s has an unknown outcome and no recovery policy", e.ID)
				}
			}
		}
	}
	if fx.Status == "" {
		fx.Status = agent.EffectPending
	}
	if fx.CreatedAt == "" {
		fx.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if fx.RecoveryPolicy == "" {
		fx.RecoveryPolicy = "fail"
	}
	all = append(all, fx)
	return true, s.saveEffects(all)
}

func recoveryPolicy(fx agent.PendingEffect) string {
	return strings.ToLower(strings.TrimSpace(fx.RecoveryPolicy))
}

// RecordOutcome persists outcome AFTER execution.
func (s *Store) RecordOutcome(id string, status agent.EffectStatus) error {
	all := s.loadEffects()
	for i := range all {
		if all[i].ID == id {
			all[i].Status = status
			return s.saveEffects(all)
		}
	}
	return fmt.Errorf("unknown effect %s", id)
}

// Fingerprint returns a stable hash of a checkpoint file for evidence.
func Fingerprint(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Effects returns the recorded effects, oldest first by creation time. It is
// how a caller inspects what a crashed run may have left behind.
func (s *Store) Effects() []agent.PendingEffect {
	all := s.loadEffects()
	sort.SliceStable(all, func(i, j int) bool { return all[i].CreatedAt < all[j].CreatedAt })
	return all
}
