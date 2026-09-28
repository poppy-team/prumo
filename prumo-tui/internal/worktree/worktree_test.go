package worktree

import (
	"path/filepath"
	"testing"
)

func TestParseList(t *testing.T) {
	raw := `worktree /home/user/project
HEAD aabbccddeeff00112233445566778899aabbccdd
branch refs/heads/main

worktree /home/user/project/.prumo/worktrees/feat-auth
HEAD 112233445566778899aabbccddeeff0011223344
branch refs/heads/feat/auth

worktree /home/user/project/.git/bare
HEAD 99887766554433221100ffeeddccbbaa99887766
bare
`
	list := ParseList(raw)
	if len(list) != 3 {
		t.Fatalf("expected 3 worktrees, got %d", len(list))
	}

	if list[0].Path != "/home/user/project" || list[0].Branch != "main" {
		t.Errorf("worktree 0 mismatch: %+v", list[0])
	}
	if list[1].Path != "/home/user/project/.prumo/worktrees/feat-auth" || list[1].Branch != "feat/auth" {
		t.Errorf("worktree 1 mismatch: %+v", list[1])
	}
	if !list[2].Bare {
		t.Errorf("worktree 2 expected bare: %+v", list[2])
	}
}

func TestDefaultWorktreePath(t *testing.T) {
	base := "/tmp/repo"
	got := DefaultWorktreePath(base, "feature/login")
	want := filepath.Join(base, ".prumo", "worktrees", "feature-login")
	if got != want {
		t.Errorf("DefaultWorktreePath() = %q, want %q", got, want)
	}
}
