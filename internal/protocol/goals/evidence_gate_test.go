package goals

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/protocol/evidence"
)

// A goal must not reach DONE because its evidence array is non-empty.
//
// The check this replaces was `len(evidence) == 0`, which accepts
// `["does-not-exist"]`, `[{}]` and any other non-empty value. A goal could
// therefore be declared complete with nothing behind it, which inverts the
// framework rule that evidence, not model confidence, determines completion
// (GAP-124).

// writeGoal produces a goal driven through the real state machine to REVIEWING,
// carrying the given evidence value. Walking the real transitions means the lock
// digest and history are produced by the code under test rather than
// hand-written, so the fixture cannot represent an impossible goal.
func writeGoal(t *testing.T, evidenceValue any) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "goal.json")

	goal := map[string]any{
		"id":         "P01-G01",
		"title":      "Evidence gate",
		"phase":      "P01",
		"state":      "DRAFT",
		"objective":  "Finish only with real evidence.",
		"acceptance": []string{"Tests pass."},
		"gates":      map[string]any{"tests": "required"},
		"revision":   1,
		"evidence":   evidenceValue,
		"history":    []any{},
	}
	data, err := json.MarshalIndent(goal, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Evidence is only set after the lock, because the lock digest covers the
	// goal body and the fixture must keep verifying.
	for _, state := range []string{"PLANNED", "LOCKED", "EXECUTING", "VERIFYING", "REVIEWING"} {
		if _, err := TransitionGoal(path, state, "fixture"); err != nil {
			t.Fatalf("fixture could not reach %s: %v", state, err)
		}
	}
	restoreEvidence(t, path, evidenceValue)

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var check map[string]any
	if err := json.Unmarshal(raw, &check); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := strings.ToUpper(fmt.Sprint(check["state"])); got != "REVIEWING" {
		t.Fatalf("fixture must end in REVIEWING, got %s", got)
	}
	return path
}

func restoreEvidence(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var goal map[string]any
	if err := json.Unmarshal(raw, &goal); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	goal["evidence"] = value
	data, err := json.MarshalIndent(goal, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func validRecord(goalID string) evidence.Record {
	return evidence.NewRecord("ev-real-1", "test", "go-test", goalID, "passed")
}

func TestGoalCannotBeDoneWithoutEvidence(t *testing.T) {
	path := writeGoal(t, []any{})
	_, err := TransitionGoalWithEvidence(path, "DONE", "", []evidence.Record{validRecord("P01-G01")})
	if err == nil {
		t.Fatal("a goal with no evidence must not reach DONE")
	}
}

func TestGoalCannotBeDoneWithInventedEvidenceID(t *testing.T) {
	path := writeGoal(t, []any{"ev-does-not-exist"})
	_, err := TransitionGoalWithEvidence(path, "DONE", "", []evidence.Record{validRecord("P01-G01")})
	if err == nil {
		t.Fatal("an evidence id with no matching record must not reach DONE")
	}
	if !strings.Contains(err.Error(), "no such evidence record") {
		t.Fatalf("the error must say the record does not exist, got %q", err.Error())
	}
}

func TestGoalCannotBeDoneWithStaleEvidence(t *testing.T) {
	path := writeGoal(t, []any{"ev-real-1"})
	stale := validRecord("P01-G01")
	stale.Stale = true
	stale.InvalidationTriggers = []string{"source_revision_drift"}
	_, err := TransitionGoalWithEvidence(path, "DONE", "", []evidence.Record{stale})
	if err == nil {
		t.Fatal("stale evidence must not satisfy DONE")
	}
	if !strings.Contains(err.Error(), "stale") {
		t.Fatalf("the error must say the evidence is stale, got %q", err.Error())
	}
}

func TestGoalCannotBeDoneWithIncompleteRecord(t *testing.T) {
	path := writeGoal(t, []any{"ev-real-1"})
	incomplete := evidence.Record{ID: "ev-real-1", Type: "test", GoalID: "P01-G01"}
	_, err := TransitionGoalWithEvidence(path, "DONE", "", []evidence.Record{incomplete})
	if err == nil {
		t.Fatal("a record missing required fields must not satisfy DONE")
	}
}

func TestGoalCannotBeDoneWithAnotherGoalsEvidence(t *testing.T) {
	path := writeGoal(t, []any{"ev-real-1"})
	foreign := validRecord("P99-G99")
	_, err := TransitionGoalWithEvidence(path, "DONE", "", []evidence.Record{foreign})
	if err == nil {
		t.Fatal("evidence belonging to another goal must not satisfy DONE")
	}
	if !strings.Contains(err.Error(), "belongs to goal") {
		t.Fatalf("the error must name the goal mismatch, got %q", err.Error())
	}
}

func TestGoalCannotBeDoneWhenNoStoreIsVisible(t *testing.T) {
	// Fail closed: a caller that cannot see the evidence store cannot certify
	// that the evidence exists.
	path := writeGoal(t, []any{"ev-real-1"})
	_, err := TransitionGoalWithEvidence(path, "DONE", "", nil)
	if err == nil {
		t.Fatal("DONE with no resolvable evidence must fail closed")
	}
	if !strings.Contains(err.Error(), "resolvable") {
		t.Fatalf("the error must explain that evidence could not be resolved, got %q", err.Error())
	}
}

func TestGoalReachesDoneWithValidFreshMatchingEvidence(t *testing.T) {
	path := writeGoal(t, []any{"ev-real-1"})
	record := validRecord("P01-G01")
	if record.Timestamp == "" {
		t.Fatalf("fixture must carry a timestamp: %+v", record)
	}
	goal, err := TransitionGoalWithEvidence(path, "DONE", "criteria met", []evidence.Record{record})
	if err != nil {
		t.Fatalf("valid evidence must allow DONE: %v", err)
	}
	if got := strings.ToUpper(goal["state"].(string)); got != "DONE" {
		t.Fatalf("expected DONE, got %s", got)
	}
}

func TestGoalEvidenceAcceptsObjectShape(t *testing.T) {
	path := writeGoal(t, []any{map[string]any{"id": "ev-real-1"}})
	if _, err := TransitionGoalWithEvidence(path, "DONE", "", []evidence.Record{validRecord("P01-G01")}); err != nil {
		t.Fatalf("an object-shaped evidence entry with a valid id must be accepted: %v", err)
	}
}

func TestGoalEvidenceRejectsObjectWithoutID(t *testing.T) {
	path := writeGoal(t, []any{map[string]any{"note": "trust me"}})
	if _, err := TransitionGoalWithEvidence(path, "DONE", "", []evidence.Record{validRecord("P01-G01")}); err == nil {
		t.Fatal("an evidence entry with no resolvable id must not satisfy DONE")
	}
}

func TestEvidenceRecordTimestampIsAcceptedInRFC3339(t *testing.T) {
	record := validRecord("P01-G01")
	if _, err := time.Parse(time.RFC3339, record.Timestamp); err != nil {
		t.Fatalf("NewRecord must produce an RFC3339 timestamp: %v", err)
	}
}
