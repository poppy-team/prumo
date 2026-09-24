// Nested runs: the default team Work binding executes a full NativeAgent
// Runner per role inside the role workspace, with the role budget enforced,
// checkpoints under the workspace, and a parent↔child Handoff built from
// the final checkpoint. Roles without a workspace cannot nest (explicit
// error, never implicit sharing).
package team

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/raillen/prumo/internal/harness/aci"
	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/checkpoint"
	"github.com/raillen/prumo/internal/harness/handoff"
	"github.com/raillen/prumo/internal/harness/model"
	"github.com/raillen/prumo/internal/harness/perm"
	"github.com/raillen/prumo/internal/harness/runlayer"
	harnessruntime "github.com/raillen/prumo/internal/harness/runtime"
)

// RunnerDeps configures nested runs.
type RunnerDeps struct {
	// NewProvider builds the model provider per role (fake in tests).
	NewProvider func(ctx context.Context, role Role) (model.Provider, error)
	// Goal is the parent goal text seeded as the child run input.
	Goal string
	// MaxTurns caps each child run (default 3).
	MaxTurns int
	// Strict requires a passing test.run for child completion.
	Strict bool
	// Isolation is what contains the child run's tool calls. Nil means none, and
	// that is the default on purpose: a delegated run executing a command has
	// only its root directory unless somebody says otherwise, and the field
	// existing at all is what lets a caller be explicit about the difference
	// (GAP-168).
	Isolation aci.SandboxProvider
	// RequireIsolation refuses to start a child run with no declared boundary
	// when it is set. It is opt-in because a missing boundary is a fact to
	// record, not automatically an error — but a deployment that believes its
	// child runs are sandboxed needs the refusal to be told otherwise.
	RequireIsolation bool
}

// RunWork returns a Work that nests a full agent run per role.
func RunWork(deps RunnerDeps) Work {
	maxTurns := deps.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 3
	}
	return func(ctx context.Context, role Role) ([]string, map[string]float64, error) {
		if role.Workspace == "" {
			return nil, nil, fmt.Errorf("role %s has no workspace: nested runs require explicit ownership", role.Name)
		}
		if deps.RequireIsolation && deps.Isolation == nil {
			return nil, nil, fmt.Errorf("role %s: no sandbox provider declared, so this child run would execute tools with no containment beyond its directory", role.Name)
		}
		isolation := deps.Isolation
		provider, err := deps.NewProvider(ctx, role)
		if err != nil {
			return nil, nil, err
		}
		// The child run's containment is named, not implied. A delegated run
		// executing a tool call has whatever its root directory gives it and
		// nothing more, and the worktree provider — which describes itself as
		// "git isolation only" — was never consulted here at all. Silently
		// building the same executor as an in-process run made a delegated run
		// look as contained as the parent while being no more contained (GAP-168).
		tools := aci.NewWithIsolation(role.Workspace, isolation)
		store := checkpoint.New(filepath.Join(role.Workspace, ".prumo", "runs"))
		tracker := runlayer.NewTracker(budgetOf(role, "tokens"), budgetOf(role, "cost_usd"), budgetOf(role, "tool_calls"))
		counting := &runlayer.CountingTools{Base: tools, Tracker: tracker}
		runID := "R-" + role.Name
		runner := harnessruntime.NewRunner(harnessruntime.Services{
			Models:        provider,
			Tools:         counting,
			Perms:         perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
			Checkpoints:   store,
			EffectJournal: store,
			ContextManifest: func(_ context.Context, _ agent.NativeAgentState) (string, error) {
				return "ctx-" + runID, nil
			},
			ReserveBudget:   tracker.Reserve,
			BudgetExhausted: tracker.Exhausted,
		}, runID, "S-"+role.Name)
		runner.MaxTurns = maxTurns
		if deps.Strict {
			runner.QualityGate = func() error { return runlayer.StrictGate()(counting.ReportsCopy()) }
		}
		goal := deps.Goal
		if goal == "" {
			goal = "team task"
		}
		runner.Messages = []agent.Message{{ID: "m1", Role: agent.RoleUser, Content: goal, CreatedAt: agent.Now()}}
		if err := runner.RunUntilDone(ctx); err != nil {
			return nil, tracker.Snapshot(), err
		}
		if runner.State.Phase != agent.PhaseComplete && runner.State.Phase != agent.PhaseYield {
			return nil, tracker.Snapshot(), fmt.Errorf("child run %s ended %s (%s)", runID, runner.State.Phase, runner.State.StopReason)
		}
		usage := tracker.Snapshot()
		usage["turns"] = float64(runner.TurnsDone)
		return []string{"role " + role.Name + " " + string(runner.State.Phase)}, usage, nil
	}
}

func budgetOf(role Role, key string) float64 {
	if role.Budget == nil {
		return 0
	}
	return role.Budget[key]
}

// ChildHandoff builds the parent↔child continuation from a finished child
// checkpoint state. The parent (or reviewer) resumes from refs, never from
// the child's trajectory.
func ChildHandoff(fromRole, to string, state agent.NativeAgentState, workspaceRev string) (handoff.Bundle, error) {
	return handoff.Build(fromRole, to, state, workspaceRev, "child continuation for "+state.RunID, map[string]string{
		"child_role": fromRole,
	})
}
