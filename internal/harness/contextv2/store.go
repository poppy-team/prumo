package contextv2

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/raillen/prumo/internal/harness/safepath"
)

// ManifestDir is where compiled manifests live. They are runtime state: a
// manifest records a compilation and is never a canonical artifact.
func ManifestDir(root string) string {
	return filepath.Join(root, ".prumo", "runtime", "harness")
}

// ManifestPath is where one compilation's manifest is stored. The filename keeps
// the run-scoped spelling so existing artifacts and retention rules keep
// working, while the manifest's own ID is the canonical `CTX-` form.
//
// The run id is validated because it becomes a filename. A run id is accepted
// from a request, and before this a value carrying a separator reached Join
// untouched and wrote outside the runtime directory (GAP-112).
func ManifestPath(root, runID string) (string, error) {
	if err := safepath.ValidateID("run_id", runID); err != nil {
		return "", err
	}
	return filepath.Join(ManifestDir(root), "context-"+runID+".json"), nil
}

// SaveManifest writes a manifest atomically under runtime state.
func SaveManifest(root string, m Manifest) (string, error) {
	path, err := ManifestPath(root, RunIDOf(m))
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	return path, nil
}

// RunIDOf recovers the run identifier a manifest belongs to, accepting either
// the canonical `CTX-` id or the run id itself.
func RunIDOf(m Manifest) string {
	if m.RunID != "" {
		return m.RunID
	}
	return NormalizeManifestRef(m.ID)
}

// NormalizeManifestRef strips the manifest prefix so both `CTX-R-1` and the
// legacy `ctx-R-1` resolve to the same compilation.
func NormalizeManifestRef(id string) string {
	trimmed := strings.TrimSpace(id)
	if len(trimmed) >= 4 && strings.EqualFold(trimmed[:4], "ctx-") {
		return trimmed[4:]
	}
	return trimmed
}

// LoadManifest reads a previously compiled manifest by id.
func LoadManifest(root, id string) (Manifest, error) {
	runID := NormalizeManifestRef(id)
	if runID == "" {
		return Manifest{}, fmt.Errorf("context manifest id is required")
	}
	path, err := ManifestPath(root, runID)
	if err != nil {
		return Manifest{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("no compiled context %s: %w", id, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, err
	}
	if m.RunID == "" {
		m.RunID = runID
	}
	if m.ID == "" {
		m.ID = ManifestID(runID)
	}
	return m, nil
}

// Explanation is the operator-facing view of one compilation (W4.8). It answers
// three questions: what was included and why, what was left out and why, and
// whether the result can be trusted to be sufficient.
type Explanation struct {
	ID                string         `json:"id"`
	RunID             string         `json:"run_id"`
	Level             Level          `json:"level"`
	Policy            string         `json:"policy,omitempty"`
	EstimatedTokens   int            `json:"estimated_tokens"`
	InstructionTokens int            `json:"instruction_tokens"`
	Pressure          string         `json:"pressure"`
	Budget            int            `json:"budget,omitempty"`
	IncludedCount     int            `json:"included_count"`
	Included          []ExplainEntry `json:"included"`
	ExcludedCount     int            `json:"excluded_count"`
	Excluded          []Exclusion    `json:"excluded"`
	ExclusionReasons  map[string]int `json:"exclusion_reasons"`
	Sufficiency       Sufficiency    `json:"sufficiency"`
	Notes             []string       `json:"notes"`
}

// ExplainEntry is one included item with the reason it was selected.
type ExplainEntry struct {
	Ref       string `json:"ref"`
	Reason    string `json:"reason"`
	Tokens    int    `json:"tokens"`
	Level     Level  `json:"level,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
	Authority string `json:"authority,omitempty"`
}

// Sufficiency is the W4.9 verdict: a compilation is sufficient when every
// obligation it was asked to satisfy is present.
type Sufficiency struct {
	Required   []string `json:"required"`
	Missing    []string `json:"missing"`
	Sufficient bool     `json:"sufficient"`
}

// EvaluateSufficiency checks that every required reference is present in the
// compilation. An empty requirement set is trivially satisfied but says so
// explicitly, so an unexplained "sufficient" cannot be mistaken for a check.
func EvaluateSufficiency(m Manifest, required []string) Sufficiency {
	present := map[string]bool{}
	for _, it := range m.Included {
		present[it.Ref] = true
	}
	verdict := Sufficiency{Required: []string{}, Missing: []string{}, Sufficient: true}
	for _, req := range required {
		req = strings.TrimSpace(req)
		if req == "" {
			continue
		}
		verdict.Required = append(verdict.Required, req)
		if !present[req] {
			verdict.Missing = append(verdict.Missing, req)
			verdict.Sufficient = false
		}
	}
	sort.Strings(verdict.Required)
	sort.Strings(verdict.Missing)
	return verdict
}

// Explain renders the explicit explanation of a compilation.
func Explain(m Manifest, budget int, required []string) Explanation {
	entry := Explanation{
		ID: m.ID, RunID: m.RunID, Level: m.Level, Policy: m.Policy,
		EstimatedTokens: m.EstimatedTokens, InstructionTokens: m.InstructionTokens,
		Pressure: m.Pressure, Budget: budget,
		Included: []ExplainEntry{}, Excluded: []Exclusion{},
		ExclusionReasons: map[string]int{}, Notes: []string{},
	}
	if entry.ID == "" {
		entry.ID = ManifestID(m.RunID)
	}
	for _, it := range m.Included {
		reason := it.Reason
		if reason == "" {
			// An included item without a stated reason is a defect; surface it
			// rather than inventing an explanation.
			reason = "unexplained selection"
			entry.Notes = append(entry.Notes, "item "+it.Ref+" was included without a selection reason")
		}
		entry.Included = append(entry.Included, ExplainEntry{
			Ref: it.Ref, Reason: reason, Tokens: it.TokenCost,
			Level: it.Level, Truncated: it.Truncated, Authority: it.Authority,
		})
	}
	entry.IncludedCount = len(entry.Included)
	entry.Excluded = append(entry.Excluded, m.Excluded...)
	entry.ExcludedCount = len(entry.Excluded)
	for _, ex := range entry.Excluded {
		entry.ExclusionReasons[ex.Reason]++
	}
	sort.SliceStable(entry.Included, func(i, j int) bool { return entry.Included[i].Ref < entry.Included[j].Ref })
	sort.SliceStable(entry.Excluded, func(i, j int) bool {
		if entry.Excluded[i].Reason != entry.Excluded[j].Reason {
			return entry.Excluded[i].Reason < entry.Excluded[j].Reason
		}
		return entry.Excluded[i].Ref < entry.Excluded[j].Ref
	})
	entry.Sufficiency = EvaluateSufficiency(m, required)
	if budget > 0 {
		entry.Notes = append(entry.Notes,
			fmt.Sprintf("used %d of %d tokens (%.0f%%) at level %s",
				entry.EstimatedTokens, budget, 100*float64(entry.EstimatedTokens)/float64(budget), entry.Level))
	}
	if entry.InstructionTokens > 0 {
		entry.Notes = append(entry.Notes,
			fmt.Sprintf("%d tokens (%.0f%% of the compilation) are agent instruction surfaces",
				entry.InstructionTokens, 100*float64(entry.InstructionTokens)/float64(maxInt(entry.EstimatedTokens, 1))))
	}
	if len(entry.Sufficiency.Missing) > 0 {
		entry.Notes = append(entry.Notes,
			"the compilation is not sufficient: "+strings.Join(entry.Sufficiency.Missing, ", ")+" absent")
	}
	return entry
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
