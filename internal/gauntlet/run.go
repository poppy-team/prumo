// Deterministic Docs Gauntlet run records (W19). Deterministic checks run
// always; the model critic is optional and its skip is recorded explicitly,
// never implied by silence. A model critic can never override a deterministic
// failure.
package gauntlet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunRoot is the derived directory holding gauntlet run records.
const RunRoot = ".prumo/runtime/docs/gauntlet"

// Finding is one gauntlet finding with its critic provenance.
type Finding struct {
	ID        string   `json:"id"`
	Dimension string   `json:"dimension"`
	Severity  string   `json:"severity"`
	Status    string   `json:"status"`
	Critic    string   `json:"critic"`
	Detail    string   `json:"detail,omitempty"`
	Evidence  []string `json:"evidence,omitempty"`
}

// Deterministic records which deterministic checks ran and which failed.
type Deterministic struct {
	Passed bool     `json:"passed"`
	Checks []string `json:"checks"`
	Failed []string `json:"failed"`
}

// Round is one gauntlet round.
type Round struct {
	Round         int           `json:"round"`
	Deterministic Deterministic `json:"deterministic"`
	Findings      []Finding     `json:"findings"`
	ModelCritic   string        `json:"model_critic"`
}

// RunRecord matches schemas/gauntlet-run.schema.json.
type RunRecord struct {
	RunID      string    `json:"run_id"`
	Mode       string    `json:"mode"`
	Goal       string    `json:"goal,omitempty"`
	Rounds     []Round   `json:"rounds"`
	StopReason string    `json:"stop_reason"`
	Findings   []Finding `json:"findings"`
	Evidence   []string  `json:"evidence,omitempty"`
}

// DimensionFor maps a documentation check kind to a gauntlet dimension.
func DimensionFor(kind string) string {
	base := strings.TrimPrefix(kind, "authority:")
	switch base {
	case "broken-link", "region-marker-mismatch", "region-marker-unclosed", "missing_file", "unclassified":
		return "reference_integrity"
	case "claim-drift", "version_drift":
		return "factual_grounding"
	case "token-budget-exceeded", "surface-stale":
		return "token_efficiency"
	case "dead-internal-link":
		return "information_architecture"
	}
	return "cross_document_consistency"
}

// SeverityFor maps a check kind to a severity. A claim contradicted by the
// implementation is critical: it is exactly the failure this gauntlet exists
// to catch.
func SeverityFor(kind string) string {
	base := strings.TrimPrefix(kind, "authority:")
	switch base {
	case "claim-drift":
		return "critical"
	case "broken-link", "region-marker-unclosed", "region-marker-mismatch", "dead-internal-link":
		return "high"
	}
	return "medium"
}

// BuildRun assembles a deterministic run record. checks are the check names
// that ran; problems are the failing check kinds with their location.
func BuildRun(policy Policy, goal, runID string, checks []string, problems []Finding) RunRecord {
	record := RunRecord{
		RunID: runID, Mode: string(normalizeMode(policy.Mode)), Goal: goal,
		Rounds: []Round{}, Findings: []Finding{}, Evidence: []string{},
	}
	failed := []string{}
	for _, p := range problems {
		failed = append(failed, p.ID)
	}
	sort.Strings(failed)
	passed := len(problems) == 0
	critic := "skipped:policy-mode-" + string(normalizeMode(policy.Mode))
	if normalizeMode(policy.Mode) != ModeOff {
		critic = "skipped:deterministic-first-no-model-critic-invoked"
	}
	record.Rounds = append(record.Rounds, Round{
		Round: 1,
		Deterministic: Deterministic{
			Passed: passed, Checks: append([]string{}, checks...), Failed: failed,
		},
		Findings:    append([]Finding{}, problems...),
		ModelCritic: critic,
	})
	record.Findings = append(record.Findings, problems...)
	switch {
	case passed:
		record.StopReason = string(StopGatesPassed)
	case normalizeMode(policy.Mode) == ModeOff:
		// The gauntlet is off by default: a failing deterministic gate stops
		// the run and returns to a human, it never self-iterates.
		record.StopReason = string(StopHuman)
	default:
		// An auto run is bounded: report the first configured stopping
		// condition that applies, in a fixed order.
		switch {
		case policy.Stopping.BudgetTokens > 0:
			record.StopReason = string(StopBudget)
		case policy.Stopping.NoImprovementRounds > 0:
			record.StopReason = string(StopNoImprovement)
		default:
			record.StopReason = string(StopMaxRounds)
		}
	}
	for _, p := range problems {
		record.Evidence = append(record.Evidence, p.Evidence...)
	}
	sort.Strings(record.Evidence)
	return record
}

func normalizeMode(mode Mode) Mode {
	switch mode {
	case ModeAuto, ModeForce:
		return mode
	default:
		return ModeOff
	}
}

// Save writes the run record as derived runtime state and returns its path.
func Save(root string, record RunRecord) (string, error) {
	if strings.TrimSpace(record.RunID) == "" {
		return "", fmt.Errorf("gauntlet run requires an id")
	}
	dir := filepath.Join(root, filepath.FromSlash(RunRoot))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, record.RunID+".json")
	return filepath.ToSlash(filepath.Join(RunRoot, record.RunID+".json")), os.WriteFile(path, append(data, '\n'), 0o644)
}

// Passed reports whether the last round passed its deterministic gate.
func (r RunRecord) Passed() bool {
	if len(r.Rounds) == 0 {
		return false
	}
	return r.Rounds[len(r.Rounds)-1].Deterministic.Passed
}
