package automation

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestLintRecipeDAG_CycleDetected(t *testing.T) {
	dag := RecipeDAG{
		ID:    "rec-cycle",
		Name:  "Cyclic Recipe",
		Start: "step-1",
		Steps: map[string]RecipeStep{
			"step-1": {ID: "step-1", Actor: "agent-1", Capability: "build", Next: []string{"step-2"}},
			"step-2": {ID: "step-2", Actor: "agent-1", Capability: "test", Next: []string{"step-3"}},
			"step-3": {ID: "step-3", Actor: "agent-1", Capability: "deploy", Next: []string{"step-1"}}, // Cycle!
		},
	}

	findings := LintRecipeDAG(dag)
	hasCycle := false
	for _, f := range findings {
		if f.Kind == "cycle_detected" {
			hasCycle = true
			break
		}
	}
	if !hasCycle {
		t.Fatalf("expected cycle_detected finding, got: %#v", findings)
	}
}

func TestLintRecipeDAG_MissingOwnerAndInfiniteRetry(t *testing.T) {
	dag := RecipeDAG{
		ID:    "rec-invalid",
		Name:  "Invalid Recipe",
		Start: "step-1",
		Steps: map[string]RecipeStep{
			"step-1": {
				ID:           "step-1",
				Actor:        "", // Missing owner!
				Capability:   "build",
				RetryLimit:   99, // Infinite / unbounded retry!
				SideEffects:  "destructive",
				Compensation: "", // Missing compensation!
			},
		},
	}

	findings := LintRecipeDAG(dag)
	kinds := make(map[string]bool)
	for _, f := range findings {
		kinds[f.Kind] = true
	}

	if !kinds["missing_owner"] {
		t.Error("expected missing_owner finding")
	}
	if !kinds["infinite_retry"] {
		t.Error("expected infinite_retry finding")
	}
	if !kinds["missing_compensation"] {
		t.Error("expected missing_compensation finding")
	}
}

func TestExecuteRecipe_SuccessAndCompensation(t *testing.T) {
	dag := RecipeDAG{
		ID:    "rec-exec",
		Name:  "Exec Recipe",
		Start: "step-1",
		Steps: map[string]RecipeStep{
			"step-1": {
				ID:           "step-1",
				Actor:        "agent-1",
				Capability:   "create_tmp",
				SideEffects:  "destructive",
				Compensation: "delete_tmp",
				Next:         []string{"step-2"},
			},
			"step-2": {
				ID:          "step-2",
				Actor:       "agent-1",
				Capability:  "run_fail",
				SideEffects: "read_only",
			},
		},
	}

	compensated := false
	exec := NewExecutor()
	exec.Handlers["create_tmp"] = func(ctx context.Context, step RecipeStep) error {
		return nil
	}
	exec.Handlers["run_fail"] = func(ctx context.Context, step RecipeStep) error {
		return errors.New("simulated failure")
	}
	exec.Compensations["delete_tmp"] = func(ctx context.Context, step RecipeStep) error {
		compensated = true
		return nil
	}

	ctx := context.Background()
	err := exec.Execute(ctx, dag)
	if err == nil || !strings.Contains(err.Error(), "simulated failure") {
		t.Fatalf("expected step failure, got: %v", err)
	}

	if !compensated {
		t.Error("expected compensation delete_tmp to be executed on step failure")
	}
}
