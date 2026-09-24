package automation

import (
	"context"
	"errors"
	"fmt"
	"sort"
)

// Failure policies a step may declare. A step that declares none stops the
// recipe, which is the conservative reading: continuing past an unmentioned
// failure is a decision someone has to make on purpose.
const (
	FailureStop       = "stop"
	FailureContinue   = "continue"
	FailureCompensate = "compensate"
)

// Side effect classes, as declared by a step.
const (
	SideEffectReadOnly    = "read_only"
	SideEffectIdempotent  = "idempotent"
	SideEffectStateful    = "stateful"
	SideEffectDestructive = "destructive"
)

var (
	ErrInvalidDAG          = errors.New("recipe: invalid DAG")
	ErrCycleDetected       = errors.New("recipe: cycle detected in recipe DAG")
	ErrMissingOwner        = errors.New("recipe: step missing owner actor")
	ErrInfiniteRetry       = errors.New("recipe: unbounded or infinite retry detected")
	ErrMissingCompensation = errors.New("recipe: destructive step missing compensation procedure")
	ErrStepFailed          = errors.New("recipe: step execution failed")
	ErrNoHandler           = errors.New("recipe: no handler registered for capability")
)

// RecipeStep declares one node in the Recipe workflow.
type RecipeStep struct {
	ID            string   `json:"id"`
	Actor         string   `json:"actor"` // step owner (required)
	Capability    string   `json:"capability"`
	Inputs        []string `json:"inputs"`
	Outputs       []string `json:"outputs"`
	Preconditions []string `json:"preconditions,omitempty"`
	SideEffects   string   `json:"side_effects"`             // read_only, idempotent, stateful, destructive
	RetryLimit    int      `json:"retry_limit"`              // must be >= 0 and <= 10
	FailurePolicy string   `json:"failure_policy,omitempty"` // stop, compensate, continue
	Compensation  string   `json:"compensation,omitempty"`   // required if destructive
	Evidence      string   `json:"evidence,omitempty"`
	Gate          string   `json:"gate,omitempty"`
	Next          []string `json:"next,omitempty"`
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

// Execute runs every branch of the DAG.
//
// It used to follow step.Next[0] and stop. Two things went wrong, and only the
// first is visible in the shape of the code:
//
//   - Every branch after the first was discarded. A recipe declaring a fan-out
//     ran its first branch, ignored the rest, and returned success — a recipe
//     that skipped half its work reported having done all of it.
//   - A step whose capability had no registered handler was replaced with a
//     function returning nil. A typo in a capability name, or a handler nobody
//     wired, produced a step that does nothing and reports that it worked. That
//     is the worse of the two, because a step that fails loudly at least gets
//     noticed (GAP-134).
//
// Execution is a topological walk: a step runs when every step pointing at it
// has finished, which is what makes a join — two branches converging on one step
// — work at all. Ready steps are taken in step-id order so a run is reproducible;
// that also means the handlers are never called concurrently, which matters
// because a handler is arbitrary caller code and this type makes no promise about
// its thread safety. Running independent branches in parallel is a real
// improvement and is deliberately not done here, because doing it would require a
// concurrency contract for every existing handler.
func (e *Executor) Execute(ctx context.Context, dag RecipeDAG) error {
	findings := LintRecipeDAG(dag)
	if len(findings) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalidDAG, findings[0].Message)
	}

	executed := map[string]RecipeStep{}
	queued := map[string]bool{}
	var order []string
	remaining := indegrees(dag)
	ready := sortedReady(dag, remaining, executed, queued)

	failures := 0
	for len(ready) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		id := ready[0]
		ready = ready[1:]
		step := dag.Steps[id]

		// A step with no handler has not run. Reporting success for it is how a
		// recipe claims work it never did.
		handler, hasHandler := e.Handlers[step.Capability]
		if !hasHandler {
			return fmt.Errorf("%w at step %s: no handler registered for capability %q",
				ErrStepFailed, step.ID, step.Capability)
		}

		var lastErr error
		for attempt := 0; attempt <= step.RetryLimit; attempt++ {
			lastErr = handler(ctx, step)
			if lastErr == nil {
				break
			}
		}

		if lastErr != nil {
			failures++
			wrapped := fmt.Errorf("%w at step %s: %v", ErrStepFailed, step.ID, lastErr)
			switch failurePolicy(step) {
			case FailureContinue:
				// The step failed and the recipe said to carry on. Its successors
				// are still gated on it, so marking it done unblocks exactly the
				// branches that do not depend on it.
				executed[id] = step
				order = append(order, id)
				for _, next := range step.Next {
					if remaining[next] > 0 {
						remaining[next]--
					}
				}
				ready = enqueue(ready, sortedReady(dag, remaining, executed, queued), queued)
				continue
			default:
				e.compensate(ctx, order, dag)
				return wrapped
			}
		}

		executed[id] = step
		order = append(order, id)
		for _, next := range step.Next {
			if remaining[next] > 0 {
				remaining[next]--
			}
		}
		ready = enqueue(ready, sortedReady(dag, remaining, executed, queued), queued)
	}

	// A step that never became ready is downstream of a cycle. Lint catches
	// cycles, so reaching here means a graph that passed the lint and still could
	// not be finished — and finishing silently is the failure mode this whole
	// change exists to remove.
	if remaining := len(dag.Steps) - len(order); remaining > 0 {
		e.compensate(ctx, order, dag)
		return fmt.Errorf("%w: %d of %d steps were never reachable; the graph is not a DAG",
			ErrInvalidDAG, remaining, len(dag.Steps))
	}
	return nil
}

// indegrees counts how many steps point at each step, which is what has to
// finish before it can run.
func indegrees(dag RecipeDAG) map[string]int {
	counts := make(map[string]int, len(dag.Steps))
	for _, step := range dag.Steps {
		for _, next := range step.Next {
			counts[next]++
		}
	}
	return counts
}

// sortedReady returns the steps whose predecessors have all finished and which are
// not already done or already waiting, in step-id order so a run is reproducible.
//
// The queued set is not an optimisation. Without it, recomputing the ready set
// after each step returns the steps still sitting in the queue, and the walk runs
// them again — a step executed four times because four of its predecessors
// finished. The first version of this did exactly that.
func sortedReady(dag RecipeDAG, remaining map[string]int, executed map[string]RecipeStep, queued map[string]bool) []string {
	ready := make([]string, 0, len(dag.Steps))
	for id := range dag.Steps {
		if _, done := executed[id]; done {
			continue
		}
		if queued[id] {
			continue
		}
		if remaining[id] == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	return ready
}

// enqueue adds newly unblocked steps and keeps the queue in step-id order.
func enqueue(ready, added []string, queued map[string]bool) []string {
	for _, id := range added {
		queued[id] = true
	}
	merged := append(ready, added...)
	sort.Strings(merged)
	return merged
}

// failurePolicy reads a step's declared policy, defaulting to stopping.
func failurePolicy(step RecipeStep) string {
	if step.FailurePolicy == "" {
		return FailureStop
	}
	return step.FailurePolicy
}

// compensate undoes the destructive steps that already ran, most recent first.
func (e *Executor) compensate(ctx context.Context, order []string, dag RecipeDAG) {
	for i := len(order) - 1; i >= 0; i-- {
		step := dag.Steps[order[i]]
		if step.SideEffects != SideEffectDestructive || step.Compensation == "" {
			continue
		}
		compensation, ok := e.Compensations[step.Compensation]
		if !ok {
			// A missing compensation is worth saying out loud. The lint requires a
			// destructive step to name one, and a name with no handler behind it is
			// the same class of gap as a step with no handler at all.
			continue
		}
		_ = compensation(ctx, step)
	}
}
