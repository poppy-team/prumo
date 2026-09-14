package team

import "testing"

func TestOwnershipPreventsCollision(t *testing.T) {
	tm := Team{ID: "t1", Delegation: DelegationManual, Roles: []Role{{Name: "a", Workspace: "wt-1"}, {Name: "b", Workspace: "wt-1"}}}
	if err := tm.Validate(); err == nil {
		t.Fatal("expected worktree collision error")
	}
	ok := Team{ID: "t1", Delegation: DelegationSolo, Roles: []Role{{Name: "a", Workspace: "wt-1"}, {Name: "b", Workspace: "wt-2"}}}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
}
