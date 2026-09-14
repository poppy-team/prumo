package team

import (
	"context"
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
			// Always wants more; the cap must terminate the loop.
			return []Role{{Name: "extra"}}, nil
		},
	}
	sum, err := r.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// lead + extra run, then the cap stops further delegation: the
	// suggester is consulted exactly twice (no unbounded loop).
	if len(sum.Results) != 2 || calls != 2 {
		t.Fatalf("cap not enforced: results=%d calls=%d", len(sum.Results), calls)
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
