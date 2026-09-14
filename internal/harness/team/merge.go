// Explicit merge (GAP-008): converge one role workspace into a base tree
// as an auditable operation. Three-way against the pre-work baseline:
// unchanged→take overlay, overlay-unchanged→keep base, both changed
// differently→conflict (reported, never auto-resolved). Content-addressed
// by SHA-256; bounded file counts/sizes.
package team

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// MergeResult is the auditable outcome of one merge.
type MergeResult struct {
	Merged    []string `json:"merged"`
	Conflicts []string `json:"conflicts"`
	Deleted   []string `json:"deleted,omitempty"`
	Skipped   []string `json:"skipped,omitempty"`
}

// MergeWorkspaces merges overlay into base given the pre-work baseline
// snapshot of base (see SnapshotWorkspace). Base files are overwritten only
// on clean fast-forwards; conflicts leave base untouched. Deletions
// propagate only when the base file is unchanged since baseline; otherwise
// the deletion conflicts (resurrect-vs-edit is a human call).
func MergeWorkspaces(base, overlay string, baseline map[string]string) (MergeResult, error) {
	var res MergeResult
	after := SnapshotWorkspace(overlay)
	keys := make([]string, 0, len(after))
	for k := range after {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, rel := range keys {
		baseSum := hashFile(filepath.Join(base, rel))
		overSum := after[rel]
		baseLine, ok := baseline[rel]
		switch {
		case !ok && baseSum == "":
			// Added in overlay: copy in.
			if err := copyFile(filepath.Join(overlay, rel), filepath.Join(base, rel)); err != nil {
				return res, err
			}
			res.Merged = append(res.Merged, "A "+rel)
		case baseSum == overSum:
			res.Skipped = append(res.Skipped, rel)
		case ok && baseSum == baseLine:
			// Base untouched since baseline: fast-forward.
			if err := copyFile(filepath.Join(overlay, rel), filepath.Join(base, rel)); err != nil {
				return res, err
			}
			res.Merged = append(res.Merged, "M "+rel)
		default:
			res.Conflicts = append(res.Conflicts, rel)
		}
	}
	// Deletions: in baseline and base, gone from overlay.
	for rel, baseLine := range baseline {
		if _, ok := after[rel]; ok {
			continue
		}
		baseSum := hashFile(filepath.Join(base, rel))
		switch {
		case baseSum == "":
			res.Skipped = append(res.Skipped, rel) // already gone both sides
		case baseSum == baseLine:
			if err := os.Remove(filepath.Join(base, rel)); err != nil {
				return res, err
			}
			res.Deleted = append(res.Deleted, "D "+rel)
		default:
			res.Conflicts = append(res.Conflicts, rel)
		}
	}
	sort.Strings(res.Merged)
	sort.Strings(res.Conflicts)
	sort.Strings(res.Deleted)
	sort.Strings(res.Skipped)
	return res, nil
}

func hashFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 1<<20 {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}

func copyFile(from, to string) error {
	data, err := os.ReadFile(from)
	if err != nil {
		return fmt.Errorf("read overlay: %w", err)
	}
	if len(data) > 1<<20 {
		return fmt.Errorf("refusing large file %s", to)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	return os.WriteFile(to, data, 0o644)
}
