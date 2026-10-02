package goals

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestComputeDigestMatchesPython(t *testing.T) {
	data, err := os.ReadFile("../../../conformance/goals/valid_locked_goal.json")
	if err != nil {
		t.Skipf("conformance fixture missing: %v", err)
	}
	var goal Goal
	if err := json.Unmarshal(data, &goal); err != nil {
		t.Fatalf("failed to parse fixture: %v", err)
	}
	lock, _ := goal["lock"].(map[string]any)
	if ComputeDigest(goal) != lock["digest"] {
		t.Fatalf("digest mismatch: expected %v got %v", lock["digest"], ComputeDigest(goal))
	}
	if valid, _ := VerifyLock(goal); !valid {
		t.Fatalf("expected valid lock")
	}
}

func TestTransitionAndAmend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "P00-G01.goal.json")
	goal := NewGoal("P00-G01", "Foundation", "P00", "Ship foundation.")
	data, _ := json.Marshal(goal)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := TransitionGoal(path, "PLANNED", ""); err != nil {
		t.Fatalf("transition: %v", err)
	}
	if _, err := TransitionGoal(path, "LOCKED", ""); err != nil {
		t.Fatalf("lock: %v", err)
	}
	if _, err := TransitionGoal(path, "DONE", ""); err == nil {
		t.Fatalf("expected DONE without evidence to fail")
	}
	if _, err := AmendGoal(path, map[string]any{"changes": map[string]any{"objective": "Updated."}}); err != nil {
		t.Fatalf("amend: %v", err)
	}
}

// TestAmendFlatPayloadAndFlagPreservation covers the finding that an amendment
// written without the wrapper "changes" key used to be silently ignored, and
// flags like --reason and --approved-by were discarded when loaded alongside a
// file. The Goal must apply the new fields and recalculate the lock against them.
func TestAmendFlatPayloadAndFlagPreservation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "P00-G01.goal.json")
	initial := NewGoal("P00-G01", "Foundation", "P00", "Ship foundation.")
	data, _ := json.Marshal(initial)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	_, _ = TransitionGoal(path, "PLANNED", "")
	locked, err := TransitionGoal(path, "LOCKED", "")
	if err != nil {
		t.Fatal(err)
	}
	initialDigest := locked["lock"].(map[string]any)["digest"].(string)

	// Flat amendment payload: acceptance and objective directly at the top level.
	flatPayload := map[string]any{
		"acceptance":  []string{"New strict criterion A", "New strict criterion B"},
		"objective":   "Hardened objective after security review",
		"reason":      "File default reason (must be overridden by flag)",
		"approved_by": "cto-lead",
	}

	amended, err := AmendGoal(path, flatPayload)
	if err != nil {
		t.Fatalf("AmendGoal with flat payload: %v", err)
	}

	// 1. Acceptance criteria must be updated.
	acc := amended["acceptance"].([]string)
	if len(acc) != 2 || acc[0] != "New strict criterion A" {
		t.Fatalf("acceptance was not updated from flat payload: %#v", amended["acceptance"])
	}
	if amended["objective"] != "Hardened objective after security review" {
		t.Fatalf("objective was not updated: %v", amended["objective"])
	}

	// 2. Lock digest must be recomputed and valid for the new content.
	newLock := amended["lock"].(map[string]any)
	newDigest := newLock["digest"].(string)
	if newDigest == initialDigest {
		t.Fatalf("lock digest did not change after amendment: %s", newDigest)
	}
	if valid, msg := VerifyLock(amended); !valid {
		t.Fatalf("lock integrity invalid after amendment: %s", msg)
	}

	// 3. History must record the specified approver.
	history := amended["history"].([]any)
	lastEvent := history[len(history)-1].(map[string]any)
	if lastEvent["approved_by"] != "cto-lead" {
		t.Fatalf("expected approved_by 'cto-lead', got: %v", lastEvent["approved_by"])
	}
}
