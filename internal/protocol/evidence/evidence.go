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
	"sort"
	"strings"
	"time"
)

// Known evidence types recognized by schemas/evidence.schema.json.
var ValidTypes = map[string]bool{
	"test":                true,
	"build":               true,
	"benchmark":           true,
	"lint":                true,
	"security-scan":       true,
	"screenshot":          true,
	"artifact":            true,
	"review":              true,
	"command":             true,
	"manual-verification": true,
	"external-reference":  true,
	"harness_run":         true,
}

// Record is a structured evidence item meeting schemas/evidence.schema.json.
type Record struct {
	ID                   string         `json:"id"`
	Type                 string         `json:"type"`
	Summary              string         `json:"summary,omitempty"`
	Producer             string         `json:"producer,omitempty"`
	Timestamp            string         `json:"timestamp,omitempty"`
	Status               string         `json:"status,omitempty"`
	GoalID               string         `json:"goal_id,omitempty"`
	TaskID               string         `json:"task_id,omitempty"`
	RunID                string         `json:"run_id,omitempty"`
	AttemptID            string         `json:"attempt_id,omitempty"`
	Command              string         `json:"command,omitempty"`
	Artifact             string         `json:"artifact,omitempty"`
	Path                 string         `json:"path,omitempty"`
	Hash                 string         `json:"hash,omitempty"`
	SourceRevision       string         `json:"source_revision,omitempty"`
	Environment          map[string]any `json:"environment,omitempty"`
	Provenance           any            `json:"provenance,omitempty"`
	Confidence           string         `json:"confidence,omitempty"` // low, medium, high, unknown
	RelatedAcceptance    []string       `json:"related_acceptance,omitempty"`
	RelatedGate          []string       `json:"related_gate,omitempty"`
	Metadata             map[string]any `json:"metadata,omitempty"`
	Stale                bool           `json:"stale,omitempty"`
	InvalidationTriggers []string       `json:"invalidation_triggers,omitempty"`
}

// Validate checks a map representation against the canonical rules. It exists
// for callers holding untyped JSON, and it applies the same checks as
// ValidateRecord rather than a weaker subset.
//
// Before the 2026-09-23 audit this only checked that `id` and `type` were
// present, so a record missing producer, timestamp, status and goal_id — all of
// them required by schemas/evidence.schema.json — was accepted. The harness
// wrote exactly such records, and the strict gate accepted them.
func Validate(record map[string]any) error {
	// optionalString reads a field that may legitimately be absent. A missing
	// optional field is empty, not the string "<nil>": fmt.Sprint on an absent
	// key produces that, and feeding it to a validator rejects a valid record.
	optionalString := func(key string) string {
		value, present := record[key]
		if !present || value == nil {
			return ""
		}
		return fmt.Sprint(value)
	}
	requiredString := func(key string) string {
		value := optionalString(key)
		return strings.TrimSpace(value)
	}

	fields := map[string]string{
		"id":        requiredString("id"),
		"type":      requiredString("type"),
		"producer":  requiredString("producer"),
		"timestamp": requiredString("timestamp"),
		"status":    requiredString("status"),
		"goal_id":   requiredString("goal_id"),
	}
	var missing []string
	for name, value := range fields {
		if value == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("evidence missing required field(s): %s", strings.Join(missing, ", "))
	}
	return ValidateRecord(Record{
		ID:         fields["id"],
		Type:       fields["type"],
		Producer:   fields["producer"],
		Timestamp:  fields["timestamp"],
		Status:     fields["status"],
		GoalID:     fields["goal_id"],
		Confidence: optionalString("confidence"),
	})
}

// ValidateRecord validates a typed Record against canonical rules, including the
// required fields schemas/evidence.schema.json declares.
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
	if strings.TrimSpace(r.Producer) == "" {
		return fmt.Errorf("evidence %q missing producer", r.ID)
	}
	if strings.TrimSpace(r.Timestamp) == "" {
		return fmt.Errorf("evidence %q missing timestamp", r.ID)
	}
	if strings.TrimSpace(r.Status) == "" {
		return fmt.Errorf("evidence %q missing status", r.ID)
	}
	if strings.TrimSpace(r.GoalID) == "" {
		return fmt.Errorf("evidence %q missing goal_id", r.ID)
	}
	if _, err := time.Parse(time.RFC3339, r.Timestamp); err != nil {
		return fmt.Errorf("evidence %q timestamp is not RFC3339: %v", r.ID, err)
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
