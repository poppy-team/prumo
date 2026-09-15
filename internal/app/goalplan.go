package app

import (
	"fmt"
	"sort"
	"strings"

	docengine "github.com/raillen/prumo/internal/documentation"
	"github.com/raillen/prumo/internal/planning"
	"github.com/raillen/prumo/internal/protocol/goals"
	"github.com/raillen/prumo/internal/protocol/plans"
)

// GoalPlanInput describes what the Goal/Plan integration needs from planning
// and documentation before proposing output.
type GoalPlanInput struct {
	Scope     string
	Preview   planning.Preview
	Readiness docengine.ReadinessReport
	// Plan is the Goal documentation preflight (W17.7). An empty plan means the
	// Goal has not been through preflight, which the output reports instead of
	// silently assuming there is no documentation impact.
	Plan docengine.DocumentationPlan
}

// GoalPlanOutput is the Goal/Plan integration result. A simple Goal that does
// not need task decomposition produces Goal without Tasks (no inflation).
type GoalPlanOutput struct {
	Goal             goals.Goal         `json:"goal"`
	PlanRequired     bool               `json:"plan_required"`
	Tasks            []map[string]any   `json:"tasks,omitempty"`
	GammaRationale   string             `json:"rationale"`
	DocumentationGap []docengine.Impact `json:"documentation_gap,omitempty"`
	// DocumentationPlan is the preflight plan this Goal was proposed with.
	DocumentationPlan *docengine.DocumentationPlan `json:"documentation_plan,omitempty"`
	// PlanFindings are the plan's own findings (silent N/A, duplicates, missing
	// reasons). They are surfaced, never swallowed.
	PlanFindings []docengine.SemanticFinding `json:"plan_findings,omitempty"`
}

// ProposeGoalPlan produces or proposes Goal intent/acceptance criteria for a
// ready scope and, only when appropriate or explicitly requested, a Plan/Task
// DAG. Task decomposition is not inflated: a single-contract scope with all
// questions resolved yields a Goal alone.
//
// The Goal is proposed in DRAFT state; locking, execution and applicability
// remain responsibilities of the Goal protocol and M5 coverage/readiness.
func ProposeGoalPlan(input GoalPlanInput, explicitlyRequested bool) (GoalPlanOutput, error) {
	goal, err := buildGoal(input)
	if err != nil {
		return GoalPlanOutput{}, err
	}
	planRequired, rationale := planNeeded(input, explicitlyRequested)
	out := GoalPlanOutput{
		Goal:           goal,
		PlanRequired:   planRequired,
		GammaRationale: rationale,
	}
	if len(input.Plan.Obligations) > 0 || input.Plan.Goal != "" {
		plan := input.Plan
		out.DocumentationPlan = &plan
		out.DocumentationGap = plan.PlanImpacts()
		if len(plan.Findings) == 0 {
			out.PlanFindings = plan.Validate()
		} else {
			out.PlanFindings = plan.Findings
		}
	}
	if planRequired {
		out.Tasks = buildTasks(input)
		if cycles := plans.CheckDAGCycles(out.Tasks); len(cycles) > 0 {
			return GoalPlanOutput{}, fmt.Errorf("proposed task DAG is invalid: %s", strings.Join(cycles, "; "))
		}
	}
	return out, nil
}

// buildGoal constructs the goal intent, acceptance criteria, constraints,
// non-goals and evidence expectations from accepted decisions and the
// readiness report. Decisions classified as constraints/requirements become
// constraints; non-goal classifications become non_goals; each accepted
// decision yields an acceptance criterion.
func buildGoal(input GoalPlanInput) (goals.Goal, error) {
	scope := strings.TrimSpace(input.Scope)
	if scope == "" {
		return nil, fmt.Errorf("goal/plan proposal requires a scope")
	}
	id := scopeSlug(scope)
	title := "Implementation-ready: " + scope
	objective := fmt.Sprintf("Make scope %q implementation-ready: all blocking questions closed and supported contracts covered.", scope)

	g := goals.NewGoal(id, title, "M6", objective)
	g["state"] = "DRAFT"
	acceptance := []any{}

	var constraints, nonGoals []string
	for _, d := range input.Preview.Extracted {
		switch {
		case d.Classification == planning.ClassificationNonGoal:
			nonGoals = append(nonGoals, d.Statement)
		case d.Classification == planning.ClassificationConstraint || d.Classification == planning.ClassificationRequirement:
			constraints = append(constraints, d.Statement)
		case d.Status == planning.StatusAccepted:
			acceptance = append(acceptance, "Acceptance: "+d.Statement)
		}
	}

	// Blockers that were open and are now resolved close the readiness gap.
	if len(input.Readiness.BlockingContracts) == 0 && input.Preview.BlockersResolved == 0 {
		acceptance = append(acceptance, "Scope has no remaining blockers and is implementation-ready.")
		acceptance = append(acceptance, "Evidence: "+strings.Join(input.Readiness.ApplicableContracts, ", "))
	}
	if len(input.Readiness.BlockingQuestions) > 0 {
		acceptance = append(acceptance, "Blocking questions still open: "+strings.Join(input.Readiness.BlockingQuestions, ", "))
	}

	g["acceptance"] = acceptance
	g["constraints"] = toAny(constraints)
	g["non_goals"] = toAny(nonGoals)
	g["context"] = map[string]any{
		"budget_profile": "medium", "max_delegation_depth": 1,
		"applicable_contracts": input.Readiness.ApplicableContracts,
	}
	if len(input.Readiness.BlockingContracts) > 0 {
		g["dependencies"] = toAny(input.Readiness.BlockingContracts)
	}
	return g, nil
}

// planNeeded decides whether the Goal needs a task DAG. Decomposition is not
// inflated: it is required only when the caller explicitly asks for a plan or
// the scope complexity warrants it (multiple affected contracts or still-open
// blockers that require sequencing).
func planNeeded(input GoalPlanInput, explicitlyRequested bool) (bool, string) {
	if explicitlyRequested {
		return true, "plan explicitly requested"
	}
	contracts := affectedContracts(input.Preview)
	switch {
	case len(contracts) > 1:
		return true, fmt.Sprintf("%d affected contracts require sequencing", len(contracts))
	case input.Preview.OpenRemaining > 0:
		return true, fmt.Sprintf("%d open questions remain and require follow-up tasks", input.Preview.OpenRemaining)
	case input.Preview.BlockersResolved > 0:
		return true, "resolved blockers require verification tasks"
	default:
		return false, "simple scope: Goal without task decomposition"
	}
}

// AffectedContracts lists the contracts the scope's accepted decisions touch,
// in deterministic order. It is exported so the preflight plan can be built
// from the same set the task DAG uses (W17.7).
func AffectedContracts(p planning.Preview) []string { return affectedContracts(p) }

func affectedContracts(p planning.Preview) []string {
	seen := map[string]bool{}
	contracts := []string{}
	for _, d := range p.Extracted {
		for _, c := range d.Affected {
			if c != "" && !seen[c] {
				seen[c] = true
				contracts = append(contracts, c)
			}
		}
	}
	sort.Strings(contracts)
	return contracts
}

// buildTasks produces a proposed task DAG: one task per affected contract
// (verify coverage) plus one final review task. Tasks are chained so the DAG
// is finite and acyclic by construction.
func buildTasks(input GoalPlanInput) []map[string]any {
	contracts := affectedContracts(input.Preview)
	tasks := []map[string]any{}
	previous := ""
	for i, contract := range contracts {
		id := taskID(contract, i)
		steps := []string{"verify " + contract}
		if input.Preview.BlockersResolved > 0 {
			steps = append(steps, "re-run readiness")
		}
		deps := []string{}
		if previous != "" {
			deps = append(deps, previous)
		}
		tasks = append(tasks, map[string]any{
			"id": id, "label": strings.Join(steps, ", "), "dependencies": deps,
		})
		previous = id
	}
	if len(tasks) > 0 {
		last := taskID("review", len(tasks))
		tasks = append(tasks, map[string]any{
			"id": last, "label": "governance review and documentation delta",
			"dependencies": []string{previous},
		})
	}
	return tasks
}

func taskID(contract string, i int) string {
	return fmt.Sprintf("T-%d-%s", i+1, scopeSlug(contract))
}

func scopeSlug(scope string) string {
	s := strings.ToLower(strings.TrimSpace(scope))
	replacer := strings.NewReplacer(" ", "-", "/", "-", ":", "-", ".", "-")
	s = replacer.Replace(s)
	if s == "" {
		return "scope"
	}
	return s
}

func toAny(values []string) []string {
	sort.Strings(values)
	return values
}
