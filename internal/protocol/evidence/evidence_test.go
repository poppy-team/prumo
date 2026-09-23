package evidence

import (
	"strings"
	"testing"
)

// A record carrying only id and type is not valid evidence.
// schemas/evidence.schema.json requires id, type, producer, timestamp, status
// and goal_id. This test used to assert the opposite, which is how the harness
// came to write records that named neither a producer, a time, a status nor a
// goal and still had them accepted (GAP-099).
func completeMap() map[string]any {
	return map[string]any{
		"id":        "EV-001",
		"type":      "test",
		"producer":  "go-test",
		"timestamp": "2026-09-23T10:00:00Z",
		"status":    "passed",
		"goal_id":   "P00-G01",
	}
}

func TestValidateAcceptsACompleteRecord(t *testing.T) {
	if err := Validate(completeMap()); err != nil {
		t.Fatalf("a complete record must validate: %v", err)
	}
}

func TestValidateRejectsARecordMissingRequiredFields(t *testing.T) {
	cases := map[string][]string{
		"missing id":         {"id"},
		"missing type":       {"type"},
		"missing producer":   {"producer"},
		"missing timestamp":  {"timestamp"},
		"missing status":     {"status"},
		"missing goal_id":    {"goal_id"},
		"missing everything": {"goal_id", "id", "producer", "status", "timestamp", "type"},
	}
	for name, expected := range cases {
		record := completeMap()
		for _, field := range expected {
			delete(record, field)
		}
		err := Validate(record)
		if err == nil {
			t.Errorf("%s: expected rejection, got acceptance", name)
			continue
		}
		for _, field := range expected {
			if !strings.Contains(err.Error(), field) {
				t.Errorf("%s: error must name %q, got %q", name, field, err.Error())
			}
		}
	}
}

func TestValidateRejectsABadTimestamp(t *testing.T) {
	record := completeMap()
	record["timestamp"] = "last tuesday"
	if err := Validate(record); err == nil {
		t.Fatal("a non-RFC3339 timestamp must be rejected")
	}
}

func TestValidateRejectsAnUnknownType(t *testing.T) {
	record := completeMap()
	record["type"] = "unregistered_type"
	if err := Validate(record); err == nil {
		t.Fatal("a type outside the schema enum must be rejected")
	}
}

func TestValidateRecord(t *testing.T) {
	rec := NewRecord("EV-100", "test", "tester", "goal-1", "pass")
	if err := ValidateRecord(rec); err != nil {
		t.Fatalf("expected valid record: %v", err)
	}

	invalidType := NewRecord("EV-101", "unregistered_type", "tester", "goal-1", "pass")
	if err := ValidateRecord(invalidType); err == nil {
		t.Fatalf("expected error for unregistered type")
	}

	invalidConfidence := NewRecord("EV-102", "benchmark", "tester", "goal-1", "pass")
	invalidConfidence.Confidence = "super_high"
	if err := ValidateRecord(invalidConfidence); err == nil {
		t.Fatalf("expected error for invalid confidence")
	}
}

func TestCheckFreshness(t *testing.T) {
	rec := NewRecord("EV-200", "build", "ci", "goal-1", "pass")
	rec.SourceRevision = "rev-abc123"
	rec.Path = "src/main.go"
	rec.Hash = "sha256:1111"

	hashes := map[string]string{
		"src/main.go": "sha256:1111",
	}

	// 1. Fresh case
	if !rec.CheckFreshness("rev-abc123", hashes) {
		t.Fatalf("expected record to be fresh")
	}

	// 2. Revision drift
	driftRec := rec
	if driftRec.CheckFreshness("rev-def456", hashes) {
		t.Fatalf("expected record to be stale due to revision drift")
	}
	if !driftRec.Stale {
		t.Fatalf("expected Stale flag to be set")
	}

	// 3. File hash drift
	hashDriftRec := rec
	changedHashes := map[string]string{
		"src/main.go": "sha256:2222",
	}
	if hashDriftRec.CheckFreshness("rev-abc123", changedHashes) {
		t.Fatalf("expected record to be stale due to file hash drift")
	}
	if !hashDriftRec.Stale {
		t.Fatalf("expected Stale flag to be set")
	}
}
