package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/planning"
	"github.com/raillen/prumo/internal/validation"
)

// A generated task carried id, label and dependencies, where the schema
// requires eight fields and calls the title "title". So the plan Prumo proposed
// failed Prumo's own validation, and the DAG that passed CheckDAGCycles was not
// a task graph the framework could load (GAP-150).

func TestProposedTasksSatisfyTheTaskSchema(t *testing.T) {
	schemaDir := filepath.Clean(filepath.Join("..", "..", "schemas"))
	raw, err := os.ReadFile(filepath.Join(schemaDir, "task.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	registry, err := validation.LoadRegistry(schemaDir)
	if err != nil {
		t.Fatal(err)
	}

	tasks := buildTasks(GoalPlanInput{
		Scope:   "GAP-150",
		Preview: previewTouching("prumo.schema.json", "task.schema.json"),
	})
	if len(tasks) == 0 {
		t.Skip("this scope produced no tasks")
	}
	for _, task := range tasks {
		if problems := validation.ValidateSchema(task, schema, registry); len(problems) > 0 {
			t.Errorf("task %v fails the task schema:\n  %s", task["id"], strings.Join(problems, "\n  "))
		}
	}
}

func TestATaskSaysWhichGoalAndPlanItBelongsTo(t *testing.T) {
	tasks := buildTasks(GoalPlanInput{
		Scope:   "GAP-150",
		Preview: previewTouching("a.schema.json", "b.schema.json"),
	})
	if len(tasks) == 0 {
		t.Skip("this scope produced no tasks")
	}
	for _, task := range tasks {
		if task["goal_id"] != "GAP-150" {
			t.Errorf("task %v is not attached to the goal it was built for: %v", task["id"], task["goal_id"])
		}
		if plan, _ := task["plan_id"].(string); plan == "" || plan == "plan-" {
			t.Errorf("task %v has no usable plan id: %v", task["id"], task["plan_id"])
		}
		if status, _ := task["status"].(string); status == "" {
			t.Errorf("task %v has no status", task["id"])
		}
	}
}

// previewTouching builds the smallest preview that makes buildTasks produce
// work, so the test exercises the real code path rather than a fixture.
func previewTouching(contracts ...string) planning.Preview {
	preview := planning.Preview{}
	for _, c := range contracts {
		preview.Extracted = append(preview.Extracted, planning.DecisionProposal{Affected: []string{c}})
	}
	return preview
}
