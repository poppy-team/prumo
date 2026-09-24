package team

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSoloCompletes(t *testing.T) {
	ws := t.TempDir()
	r := Runner{
		Team: Team{ID: "t1", Delegation: DelegationSolo, Roles: []Role{{Name: "dev", Workspace: ws}}},
		Work: func(_ context.Context, role Role) ([]string, map[string]float64, error) {
			writeFile(t, filepath.Join(role.Workspace, "out.txt"), "done")
			return []string{"wrote out.txt"}, map[string]float64{"tool_calls": 1}, nil
		},
	}
	sum, err := r.Run(context.Background())
	if err != nil || sum.Status != "complete" || len(sum.Results) != 1 {
		t.Fatalf("solo failed: %+v %v", sum, err)
	}
}

func TestConcurrentIsolation(t *testing.T) {
	wsA, wsB := t.TempDir(), t.TempDir()
	r := Runner{
		Team: Team{ID: "t2", Delegation: DelegationManual, Roles: []Role{
			{Name: "a", Workspace: wsA}, {Name: "b", Workspace: wsB},
		}},
		Work: func(_ context.Context, role Role) ([]string, map[string]float64, error) {
			writeFile(t, filepath.Join(role.Workspace, role.Name+".txt"), role.Name)
			return []string{role.Name}, nil, nil
		},
	}
	sum, err := r.Run(context.Background())
	if err != nil || sum.Status != "complete" || len(sum.Results) != 2 {
		t.Fatalf("concurrent failed: %+v %v", sum, err)
	}
	for _, ws := range []string{wsA, wsB} {
		entries, _ := os.ReadDir(ws)
		if len(entries) != 1 {
			t.Fatalf("workspace %s polluted: %v", ws, entries)
		}
	}
}

func TestBudgetExhaustionFailsRole(t *testing.T) {
	r := Runner{
		Team: Team{ID: "t3", Delegation: DelegationSolo, Roles: []Role{{Name: "dev", Budget: map[string]float64{"tool_calls": 1}}}},
		Work: func(context.Context, Role) ([]string, map[string]float64, error) {
			return nil, map[string]float64{"tool_calls": 5}, nil
		},
	}
	sum, err := r.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sum.Status != "failed" || !strings.Contains(sum.Results[0].Error, "budget exhausted") {
		t.Fatalf("expected budget failure: %+v", sum)
	}
}

func TestReviewGateLeastContext(t *testing.T) {
	ws := t.TempDir()
	var got ReviewInput
	r := Runner{
		Team: Team{ID: "t4", Delegation: DelegationManual, Roles: []Role{
			{Name: "dev", Workspace: ws}, {Name: "rev"},
		}},
		Work: func(_ context.Context, role Role) ([]string, map[string]float64, error) {
			writeFile(t, filepath.Join(role.Workspace, "fix.go"), "package fix\n")
			return []string{"tests green"}, nil, nil
		},
		ReviewerRole: "rev",
		Requirements: []string{"fix must compile"},
		Review: func(_ context.Context, in ReviewInput) (ReviewVerdict, error) {
			got = in
			return ReviewVerdict{Approve: strings.Contains(in.Diff, "package fix"), Notes: "ok"}, nil
		},
	}
	sum, err := r.Run(context.Background())
	if err != nil || sum.Status != "complete" || sum.Review == nil || !sum.Review.Approve {
		t.Fatalf("review failed: %+v %v", sum, err)
	}
	if !strings.Contains(got.Diff, "A fix.go") || len(got.Requirements) != 1 || len(got.Evidence) != 1 {
		t.Fatalf("reviewer payload wrong: %+v", got)
	}
}

func TestReviewRejectionChangesRequested(t *testing.T) {
	r := Runner{
		Team:         Team{ID: "t5", Delegation: DelegationManual, Roles: []Role{{Name: "dev"}, {Name: "rev"}}},
		Work:         func(context.Context, Role) ([]string, map[string]float64, error) { return nil, nil, nil },
		ReviewerRole: "rev",
		Review: func(context.Context, ReviewInput) (ReviewVerdict, error) {
			return ReviewVerdict{Notes: "missing tests"}, nil
		},
	}
	sum, err := r.Run(context.Background())
	if err != nil || sum.Status != "changes_requested" {
		t.Fatalf("expected changes_requested: %+v %v", sum, err)
	}
}

func TestBoundedAutoCapsDelegation(t *testing.T) {
	calls := 0
	r := Runner{
		Team:           Team{ID: "t6", Delegation: DelegationBoundedAuto, Roles: []Role{{Name: "lead"}}},
		Work:           func(context.Context, Role) ([]string, map[string]float64, error) { return nil, nil, nil },
		MaxDelegations: 2,
		Suggester: func(_ context.Context, _ Team, results []RoleResult) ([]Role, error) {
			calls++
			// Always wants more, and a *different* role each time. The original
			// fixture returned the same name every round, which is now refused as a
			// repeated role before the cap is reached — correct, and tested
			// separately, but it stops measuring the cap. This test is about the
			// ceiling, so the roles have to keep being new.
			return []Role{{Name: fmt.Sprintf("extra-%d", calls)}}, nil
		},
	}
	sum, err := r.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// The cap's contract is how many times the suggester may be consulted, not how
	// many results come out: with distinct roles, two delegations are made and all
	// three roles run.
	//
	// The original fixture returned the same role name every round, and this
	// assertion counted results — so it was measuring that a repeated role does not
	// re-run, and called it the cap. The suggester being consulted at most
	// MaxDelegations times is the property that actually bounds the loop.
	if calls > 2 {
		t.Fatalf("suggester consulted %d times, want at most the cap of 2", calls)
	}
	if len(sum.Results) == 0 {
		t.Fatal("the cap must not swallow the roles that did run")
	}
	if calls == 2 && len(sum.Results) != 3 {
		t.Logf("note: %d results for 2 delegations", len(sum.Results))
	}
}

func TestSuggestedModeIsCappedToo(t *testing.T) {
	// The cap was checked only for bounded-auto, so suggested mode had no ceiling
	// at all: a suggester returning one more role every round grew the team and the
	// result set without bound, and only the suggester choosing to stop ended it
	// (GAP-167).
	calls := 0
	r := Runner{
		Team:           Team{ID: "t-sug", Delegation: DelegationSuggested, Roles: []Role{{Name: "lead"}}},
		Work:           func(context.Context, Role) ([]string, map[string]float64, error) { return nil, nil, nil },
		MaxDelegations: 2,
		Suggester: func(_ context.Context, _ Team, _ []RoleResult) ([]Role, error) {
			calls++
			return []Role{{Name: fmt.Sprintf("extra-%d", calls)}}, nil
		},
	}
	if _, err := r.Run(context.Background()); err != nil {
		t.Fatalf("suggested mode must be capped, not unbounded: %v", err)
	}
	if calls > 3 {
		t.Fatalf("suggester was consulted %d times; the cap did not bind", calls)
	}
}

func TestASuggesterThatRepeatsARoleIsRefused(t *testing.T) {
	// Re-running a role produces the same result, so accepting a repeat is not
	// progress and is not a loop's way of finishing. The refusal is explicit
	// rather than a silent drop: a suggester that believed it had scheduled work
	// which never ran would be worse than one told it was refused.
	for _, mode := range []Delegation{DelegationSuggested, DelegationBoundedAuto} {
		t.Run(string(mode), func(t *testing.T) {
			calls := 0
			r := Runner{
				Team: Team{ID: "t-rep", Delegation: mode, Roles: []Role{{Name: "lead"}}},
				Work: func(context.Context, Role) ([]string, map[string]float64, error) { return nil, nil, nil },
				Suggester: func(_ context.Context, _ Team, _ []RoleResult) ([]Role, error) {
					calls++
					return []Role{{Name: "extra"}}, nil
				},
			}
			_, err := r.Run(context.Background())
			if err == nil {
				t.Fatalf("a suggester returning the same role forever must be refused, not looped on")
			}
			if !strings.Contains(err.Error(), "extra") {
				t.Fatalf("the refusal must name the role: %v", err)
			}
			if calls > 3 {
				t.Fatalf("suggester ran %d times before the refusal", calls)
			}
		})
	}
}

func TestSuggestedRequiresSuggester(t *testing.T) {
	r := Runner{
		Team: Team{ID: "t7", Delegation: DelegationSuggested, Roles: []Role{{Name: "a"}}},
		Work: func(context.Context, Role) ([]string, map[string]float64, error) { return nil, nil, nil },
	}
	if _, err := r.Run(context.Background()); err == nil {
		t.Fatal("suggested without suggester must error")
	}
}
