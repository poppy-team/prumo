package goals

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/protocol/evidence"
)

func loadGoal(path string) (Goal, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var goal Goal
	if err := json.Unmarshal(data, &goal); err != nil {
		return nil, err
	}
	return goal, nil
}

func saveGoal(path string, goal Goal) error {
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return fmt.Errorf("Legacy YAML Goal is read-only in v0.2; migrate it to .goal.json before changing state.")
	}
	data, err := json.MarshalIndent(goal, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func TransitionGoal(path, target, reason string) (Goal, error) {
	return TransitionGoalWithEvidence(path, target, reason, nil)
}

// TransitionGoalWithEvidence moves a goal between states and, when the target is
// DONE, requires that the goal's evidence resolves to real records.
//
// The check before 2026-09-23 was `len(evidence) == 0`. That accepts
// `["does-not-exist"]`, `[{}]`, or any non-empty array, so a goal could reach
// DONE with nothing behind it. FRAMEWORK.md requires that evidence, not model
// confidence, determines completion.
//
// A nil records slice fails closed rather than passing: a caller that cannot see
// the evidence store cannot certify that evidence exists. That is the
// uncomfortable direction and the correct one.
func TransitionGoalWithEvidence(path, target, reason string, records []evidence.Record) (Goal, error) {
	goal, err := loadGoal(path)
	if err != nil {
		return nil, err
	}
	current := strings.ToUpper(fmt.Sprint(goal["state"]))
	target = strings.ToUpper(target)
	if !allowedTransition(current, target) {
		return nil, fmt.Errorf("Invalid goal transition: %s -> %s", current, target)
	}
	if target == "DONE" {
		if err := verifyGoalEvidence(goal, records); err != nil {
			return nil, err
		}
	}
	if target == "LOCKED" {
		revision := 1
		if value, ok := goal["revision"].(float64); ok {
			revision = int(value)
		}
		goal["revision"] = revision
		goal["lock"] = map[string]any{
			"revision":  revision,
			"digest":    ComputeDigest(goal),
			"locked_at": time.Now().UTC().Format(time.RFC3339Nano),
		}
	} else if (current == "LOCKED" || current == "EXECUTING" || current == "VERIFYING" || current == "REVIEWING") && goal["lock"] != nil {
		if valid, msg := VerifyLock(goal); !valid {
			return nil, fmt.Errorf("Illegal goal mutation detected: %s", msg)
		}
	}
	goal["state"] = target
	history, _ := goal["history"].([]any)
	goal["history"] = append(history, map[string]any{
		"at": time.Now().UTC().Format(time.RFC3339Nano), "event": "state-transition",
		"from": current, "to": target, "reason": reason,
	})
	if err := saveGoal(path, goal); err != nil {
		return nil, err
	}
	return goal, nil
}

// verifyGoalEvidence requires that every evidence entry the goal names resolves
// to a valid, non-stale record. An entry may be a bare id or an object carrying
// an id; anything that yields no id is rejected rather than skipped.
func verifyGoalEvidence(goal Goal, records []evidence.Record) error {
	entries := goalEvidenceIDs(goal)
	if len(entries) == 0 {
		return fmt.Errorf("A goal cannot be marked DONE without evidence.")
	}
	if records == nil {
		return fmt.Errorf("A goal cannot be marked DONE without resolvable evidence: no evidence records were supplied for %d reference(s).", len(entries))
	}

	byID := make(map[string]evidence.Record, len(records))
	for _, record := range records {
		byID[record.ID] = record
	}

	var problems []string
	for _, id := range entries {
		record, found := byID[id]
		if !found {
			problems = append(problems, fmt.Sprintf("%s: no such evidence record", id))
			continue
		}
		if err := evidence.ValidateRecord(record); err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", id, err))
			continue
		}
		if record.Stale {
			problems = append(problems, fmt.Sprintf("%s: evidence is stale", id))
			continue
		}
		if goalID := strings.TrimSpace(fmt.Sprint(goal["id"])); goalID != "" && record.GoalID != goalID {
			problems = append(problems, fmt.Sprintf("%s: belongs to goal %q, not %q", id, record.GoalID, goalID))
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("A goal cannot be marked DONE: %s", strings.Join(problems, "; "))
	}
	return nil
}

// goalEvidenceIDs extracts the referenced evidence ids from a goal, accepting
// both the bare-string and the object-with-id shapes the schema permits.
func goalEvidenceIDs(goal Goal) []string {
	raw, present := goal["evidence"]
	if !present || raw == nil {
		return nil
	}
	ids := []string{}
	switch items := raw.(type) {
	case []string:
		for _, id := range items {
			if id = strings.TrimSpace(id); id != "" {
				ids = append(ids, id)
			}
		}
	case []any:
		for _, item := range items {
			switch value := item.(type) {
			case string:
				if value = strings.TrimSpace(value); value != "" {
					ids = append(ids, value)
				}
			case map[string]any:
				if id := strings.TrimSpace(fmt.Sprint(value["id"])); id != "" && id != "<nil>" {
					ids = append(ids, id)
				}
			}
		}
	case []map[string]any:
		for _, item := range items {
			if id := strings.TrimSpace(fmt.Sprint(item["id"])); id != "" && id != "<nil>" {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

func AmendGoal(path string, amendment map[string]any) (Goal, error) {
	goal, err := loadGoal(path)
	if err != nil {
		return nil, err
	}
	current := strings.ToUpper(fmt.Sprint(goal["state"]))
	switch current {
	case "LOCKED", "EXECUTING", "VERIFYING", "REVIEWING", "BLOCKED":
	default:
		return nil, fmt.Errorf("Cannot amend goal in %s state (must be locked/active).", current)
	}
	revision := 1
	if value, ok := goal["revision"].(float64); ok {
		revision = int(value)
	}
	newRev := revision + 1
	goal["revision"] = newRev
	changes, _ := amendment["changes"].(map[string]any)
	if changes == nil {
		changes = map[string]any{}
		if raw, ok := amendment["changes"]; ok && raw != nil {
			_ = raw
		}
	}
	for _, key := range []string{"objective", "acceptance", "constraints", "non_goals", "gates"} {
		if value, ok := changes[key]; ok {
			goal[key] = value
		}
	}
	id, _ := amendment["id"].(string)
	if id == "" {
		id = fmt.Sprintf("AMD-%03d", newRev)
	}
	reason, _ := amendment["reason"].(string)
	if reason == "" {
		reason = "Formal amendment"
	}
	approvedBy, _ := amendment["approved_by"].(string)
	if approvedBy == "" {
		approvedBy = "human"
	}
	approvedAt, _ := amendment["approved_at"].(string)
	if approvedAt == "" {
		approvedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	digest := ComputeDigest(goal)
	record := map[string]any{
		"id": id, "goal_id": goal["id"], "revision": newRev, "reason": reason,
		"approved_by": approvedBy, "approved_at": approvedAt, "changes": changes, "digest": digest,
	}
	amendments, _ := goal["amendments"].([]any)
	goal["amendments"] = append(amendments, record)
	goal["lock"] = map[string]any{"revision": newRev, "digest": digest, "locked_at": approvedAt}
	history, _ := goal["history"].([]any)
	goal["history"] = append(history, map[string]any{
		"at": approvedAt, "event": "amended", "revision": newRev, "reason": reason, "approved_by": approvedBy,
	})
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return nil, fmt.Errorf("Legacy YAML Goal is read-only; migrate to JSON first.")
	}
	if err := saveGoal(path, goal); err != nil {
		return nil, err
	}
	return goal, nil
}
