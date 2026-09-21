// Package evidence defines structured protocol evidence records, validity,
// freshness tracking, and derived completion state machines.
//
// In accordance with Constitución 84.A and 85.A (Total Assurance):
// "implemented != reachable != exercised != evidenced != verified != accepted != released."
// Words like "done", "complete", "fixed", "verified" are strictly derived states,
// never self-declared text by an agent.
package evidence

import (
	"fmt"
	"strings"
	"time"
)

// Known evidence types recognized by schemas/evidence.schema.json.
var ValidTypes = map[string]bool{
	"test":                 true,
	"build":                true,
	"benchmark":            true,
	"lint":                 true,
	"security-scan":        true,
	"screenshot":           true,
	"artifact":             true,
	"review":               true,
	"command":              true,
	"manual-verification":  true,
	"external-reference":   true,
	"harness_run":          true,
}

// Record is a structured evidence item meeting schemas/evidence.schema.json.
type Record struct {
	ID                   string            `json:"id"`
	Type                 string            `json:"type"`
	Summary              string            `json:"summary,omitempty"`
	Producer             string            `json:"producer,omitempty"`
	Timestamp            string            `json:"timestamp,omitempty"`
	Status               string            `json:"status,omitempty"`
	GoalID               string            `json:"goal_id,omitempty"`
	TaskID               string            `json:"task_id,omitempty"`
	RunID                string            `json:"run_id,omitempty"`
	AttemptID            string            `json:"attempt_id,omitempty"`
	Command              string            `json:"command,omitempty"`
	Artifact             string            `json:"artifact,omitempty"`
	Path                 string            `json:"path,omitempty"`
	Hash                 string            `json:"hash,omitempty"`
	SourceRevision       string            `json:"source_revision,omitempty"`
	Environment          map[string]any    `json:"environment,omitempty"`
	Provenance           any               `json:"provenance,omitempty"`
	Confidence           string            `json:"confidence,omitempty"` // low, medium, high, unknown
	RelatedAcceptance    []string          `json:"related_acceptance,omitempty"`
	RelatedGate          []string          `json:"related_gate,omitempty"`
	Metadata             map[string]any    `json:"metadata,omitempty"`
	Stale                bool              `json:"stale,omitempty"`
	InvalidationTriggers []string          `json:"invalidation_triggers,omitempty"`
}

// Validate checks basic map representations for backwards compatibility.
func Validate(record map[string]any) error {
	id := fmt.Sprint(record["id"])
	if id == "" || id == "<nil>" {
		return fmt.Errorf("evidence missing id")
	}
	typ := fmt.Sprint(record["type"])
	if typ == "" || typ == "<nil>" {
		return fmt.Errorf("evidence missing type")
	}
	return nil
}

// ValidateRecord validates a typed Record against canonical rules.
func ValidateRecord(r Record) error {
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("evidence missing id")
	}
	if strings.TrimSpace(r.Type) == "" {
		return fmt.Errorf("evidence missing type")
	}
	if !ValidTypes[r.Type] {
		return fmt.Errorf("evidence type %q is not recognized", r.Type)
	}
	if r.Confidence != "" {
		switch r.Confidence {
		case "low", "medium", "high", "unknown":
		default:
			return fmt.Errorf("invalid confidence level %q", r.Confidence)
		}
	}
	return nil
}

// NewRecord creates a new valid Record with timestamp default.
func NewRecord(id, typ, producer, goalID, status string) Record {
	return Record{
		ID:        id,
		Type:      typ,
		Producer:  producer,
		GoalID:    goalID,
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Metadata:  make(map[string]any),
	}
}

// CheckFreshness tests whether the evidence record remains fresh against the current
// source revision and file hashes. If drift is detected, the record is marked Stale
// and invalidation triggers are recorded.
func (r *Record) CheckFreshness(currentRevision string, currentFileHashes map[string]string) bool {
	// 1. Revision drift
	if r.SourceRevision != "" && currentRevision != "" && r.SourceRevision != currentRevision {
		r.Stale = true
		r.InvalidationTriggers = appendUnique(r.InvalidationTriggers, "source_revision_drift")
	}

	// 2. File hash drift
	if r.Path != "" && r.Hash != "" && currentFileHashes != nil {
		if currHash, exists := currentFileHashes[r.Path]; exists {
			if currHash != r.Hash {
				r.Stale = true
				r.InvalidationTriggers = appendUnique(r.InvalidationTriggers, "file_hash_drift")
			}
		}
	}

	return !r.Stale
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}
