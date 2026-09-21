package automation

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrInvalidDAG   = errors.New("recipe: invalid DAG")
	ErrCycleDetected = errors.New("recipe: cycle detected in recipe DAG")
	ErrMissingOwner = errors.New("recipe: step missing owner actor")
	ErrInfiniteRetry = errors.New("recipe: unbounded or infinite retry detected")
	ErrMissingCompensation = errors.New("recipe: destructive step missing compensation procedure")
	ErrStepFailed   = errors.New("recipe: step execution failed")
)

// RecipeStep declares one node in the Recipe workflow.
type RecipeStep struct {
	ID             string   `json:"id"`
	Actor          string   `json:"actor"` // step owner (required)
	Capability     string   `json:"capability"`
	Inputs         []string `json:"inputs"`
	Outputs        []string `json:"outputs"`
	Preconditions  []string `json:"preconditions,omitempty"`
	SideEffects    string   `json:"side_effects"` // read_only, idempotent, stateful, destructive
	RetryLimit     int      `json:"retry_limit"`  // must be >= 0 and <= 10
	FailurePolicy  string   `json:"failure_policy,omitempty"` // stop, compensate, continue
	Compensation   string   `json:"compensation,omitempty"`   // required if destructive
	Evidence       string   `json:"evidence,omitempty"`
	Gate           string   `json:"gate,omitempty"`
	Next           []string `json:"next,omitempty"`
}

// RecipeDAG is the directed acyclic graph representing a complete recipe.
type RecipeDAG struct {
	ID    string                `json:"id"`
	Name  string                `json:"name"`
	Steps map[string]RecipeStep `json:"steps"`
	Start string                `json:"start"`
}

// LintFinding describes one validation problem found in a Recipe DAG.
type LintFinding struct {
	StepID  string `json:"step_id,omitempty"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// LintRecipeDAG validates all structural invariants per Rule 32.
func LintRecipeDAG(dag RecipeDAG) []LintFinding {
	findings := []LintFinding{}

	if len(dag.Steps) == 0 {
		return []LintFinding{{Kind: "empty_dag", Message: "recipe DAG has no steps"}}
	}

	if dag.Start == "" || dag.Steps[dag.Start].ID == "" {
		findings = append(findings, LintFinding{
			Kind:    "invalid_start",
			Message: fmt.Sprintf("start step %q does not exist in DAG", dag.Start),
		})
	}

	// 1. Step-level validation
	for id, step := range dag.Steps {
		if step.Actor == "" {
			findings = append(findings, LintFinding{
				StepID:  id,
				Kind:    "missing_owner",
				Message: fmt.Sprintf("step %q has no owner actor", id),
			})
		}

		if step.RetryLimit < 0 || step.RetryLimit > 10 {
			findings = append(findings, LintFinding{
				StepID:  id,
				Kind:    "infinite_retry",
				Message: fmt.Sprintf("step %q has invalid retry limit %d (must be 0-10)", id, step.RetryLimit),
			})
		}

		if step.SideEffects == "destructive" && step.Compensation == "" {
			findings = append(findings, LintFinding{
				StepID:  id,
				Kind:    "missing_compensation",
				Message: fmt.Sprintf("destructive step %q requires explicit compensation", id),
			})
		}

		for _, nextID := range step.Next {
			if _, exists := dag.Steps[nextID]; !exists {
				findings = append(findings, LintFinding{
					StepID:  id,
					Kind:    "dangling_transition",
					Message: fmt.Sprintf("step %q transitions to nonexistent step %q", id, nextID),
				})
			}
		}
	}

	// 2. Cycle detection using DFS with 3-color states (0=unvisited, 1=visiting, 2=visited)
	visited := make(map[string]int)
	var hasCycle bool
	var cyclePath string

	var dfs func(u string, path []string)
	dfs = func(u string, path []string) {
		visited[u] = 1 // visiting
		currPath := append(path, u)

		step, ok := dag.Steps[u]
		if ok {
			for _, v := range step.Next {
				if visited[v] == 1 {
					hasCycle = true
					cyclePath = fmt.Sprintf("%v -> %s", currPath, v)
					return
				}
				if visited[v] == 0 {
					dfs(v, currPath)
					if hasCycle {
						return
					}
				}
			}
		}
		visited[u] = 2 // visited
	}

	for id := range dag.Steps {
		if visited[id] == 0 {
			dfs(id, []string{})
			if hasCycle {
				findings = append(findings, LintFinding{
					Kind:    "cycle_detected",
					Message: fmt.Sprintf("cycle detected in recipe DAG: %s", cyclePath),
				})
				break
			}
		}
	}

	return findings
}

// StepHandler is the function executed for a recipe step.
type StepHandler func(ctx context.Context, step RecipeStep) error

// Executor executes steps in a validated Recipe DAG.
type Executor struct {
	Handlers      map[string]StepHandler
	Compensations map[string]StepHandler
}

// NewExecutor creates a new recipe workflow executor.
func NewExecutor() *Executor {
	return &Executor{
		Handlers:      make(map[string]StepHandler),
		Compensations: make(map[string]StepHandler),
	}
}

// Execute runs the DAG from the start step to terminal nodes.
func (e *Executor) Execute(ctx context.Context, dag RecipeDAG) error {
	findings := LintRecipeDAG(dag)
	if len(findings) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalidDAG, findings[0].Message)
	}

	executedSteps := []RecipeStep{}
	currID := dag.Start

	for currID != "" {
		step := dag.Steps[currID]
		handler := e.Handlers[step.Capability]
		if handler == nil {
			handler = func(_ context.Context, _ RecipeStep) error { return nil }
		}

		var lastErr error
		attempts := step.RetryLimit + 1
		for a := 0; a < attempts; a++ {
			lastErr = handler(ctx, step)
			if lastErr == nil {
				break
			}
		}

		if lastErr != nil {
			// Trigger compensation for previously executed destructive steps in reverse
			for i := len(executedSteps) - 1; i >= 0; i-- {
				prev := executedSteps[i]
				if prev.SideEffects == "destructive" && prev.Compensation != "" {
					if compHandler := e.Compensations[prev.Compensation]; compHandler != nil {
						_ = compHandler(ctx, prev)
					}
				}
			}
			return fmt.Errorf("%w at step %s: %v", ErrStepFailed, step.ID, lastErr)
		}

		executedSteps = append(executedSteps, step)

		// Transition to next step
		if len(step.Next) > 0 {
			currID = step.Next[0] // sequential branch
		} else {
			currID = ""
		}
	}

	return nil
}
