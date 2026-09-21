package evidence

import (
	"testing"
)

func TestValidate(t *testing.T) {
	if err := Validate(map[string]any{"id": "EV-001", "type": "test"}); err != nil {
		t.Fatalf("expected valid evidence: %v", err)
	}
	if err := Validate(map[string]any{"type": "test"}); err == nil {
		t.Fatalf("expected missing id error")
	}
	if err := Validate(map[string]any{"id": "EV-001"}); err == nil {
		t.Fatalf("expected missing type error")
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
