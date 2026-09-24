package team

import (
	"context"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/aci"
	"github.com/raillen/prumo/internal/harness/model"
)

// A delegated run was built with the same executor constructor as an in-process
// run, so it looked as contained as its parent while being no more contained: a
// tool call in a child run ran a command directly in the worktree, and the
// worktree provider — which describes itself as "git isolation only" — was never
// consulted at all. Nothing recorded which of the two had happened (GAP-168).
//
// The fix here is to make the boundary sayable and to let a deployment insist on
// it. It is deliberately not a container: a worktree is a git checkout, and
// claiming otherwise is the bug.

func isolationDeps() RunnerDeps {
	return RunnerDeps{
		NewProvider: func(context.Context, Role) (model.Provider, error) {
			return model.NewFake(map[string][]model.ScriptStep{
				"*": {{Kind: "text", Text: "done"}, {Kind: "complete"}},
			}), nil
		},
		Goal: "do the thing",
	}
}

func TestAChildRunWithNoDeclaredIsolationIsRefusedWhenRequired(t *testing.T) {
	deps := isolationDeps()
	deps.RequireIsolation = true
	work := RunWork(deps)
	_, _, err := work(context.Background(), Role{Name: "executor", Workspace: t.TempDir()})
	if err == nil {
		t.Fatal("a child run with no containment must be refused when the deployment requires isolation")
	}
	if !strings.Contains(err.Error(), "no sandbox provider") {
		t.Fatalf("the error must name what is missing: %v", err)
	}
}

func TestADeclaredIsolationSatisfiesTheRequirement(t *testing.T) {
	deps := isolationDeps()
	deps.RequireIsolation = true
	deps.Isolation = aci.WorktreeProvider{Path: t.TempDir()}
	work := RunWork(deps)
	if _, _, err := work(context.Background(), Role{Name: "executor", Workspace: t.TempDir()}); err != nil {
		t.Fatalf("a declared provider must satisfy the requirement: %v", err)
	}
}

func TestWithoutTheRequirementAChildRunStillStarts(t *testing.T) {
	// A missing boundary is a fact to record, not automatically an error. A
	// deployment that has not asked for isolation should not be broken by this
	// change; it should just be able to see the truth.
	work := RunWork(isolationDeps())
	if _, _, err := work(context.Background(), Role{Name: "executor", Workspace: t.TempDir()}); err != nil {
		t.Fatalf("a child run with no declared isolation must still start: %v", err)
	}
}

func TestAnExecutorBuiltFromAPathAloneClaimsNoIsolation(t *testing.T) {
	// The constructor takes only a root, so a caller cannot have chosen
	// containment here. Reporting anything reassuring would be asserting a
	// boundary that does not exist.
	executor := aci.New(t.TempDir())
	if executor.Isolation != aci.SandboxNone {
		t.Fatalf("isolation = %q, want none: this constructor cannot have provided one", executor.Isolation)
	}
}

func TestAnExecutorRecordsTheProviderItWasGiven(t *testing.T) {
	// Read from the provider rather than passed separately: a caller naming
	// "container" for a local provider would be recording a boundary it lacks,
	// and a recorded boundary is what a later reader trusts.
	root := t.TempDir()
	executor := aci.NewWithIsolation(root, aci.WorktreeProvider{Path: root})
	if executor.Isolation != aci.SandboxWorktree {
		t.Fatalf("isolation = %q, want the provider's own kind", executor.Isolation)
	}
	container := aci.NewWithIsolation(root, aci.LocalProvider{Root: root})
	if container.Isolation != aci.SandboxLocalTrusted {
		t.Fatalf("isolation = %q, want local-trusted for a local provider", container.Isolation)
	}
}

func TestANilProviderLeavesTheExecutorAtNoIsolation(t *testing.T) {
	executor := aci.NewWithIsolation(t.TempDir(), nil)
	if executor.Isolation != aci.SandboxNone {
		t.Fatalf("isolation = %q, want none when no provider was given", executor.Isolation)
	}
}
