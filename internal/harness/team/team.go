// Package team implements the Workforce multi-agent baseline: teams, role
// bindings, delegation modes, per-role budgets, least-context review and
// explicit worktree ownership with merge/review/conflict handling.
package team

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Delegation modes: solo|manual|suggested|bounded-auto.
type Delegation string

const (
	DelegationSolo        Delegation = "solo"
	DelegationManual      Delegation = "manual"
	DelegationSuggested   Delegation = "suggested"
	DelegationBoundedAuto Delegation = "bounded_auto"
)

// Role is one agent binding.
type Role struct {
	Name      string             `json:"name"`
	Binding   string             `json:"binding"` // provider/agent id
	Budget    map[string]float64 `json:"budget,omitempty"`
	Workspace string             `json:"workspace,omitempty"`
}

// Team groups roles with explicit ownership.
type Team struct {
	ID         string     `json:"id"`
	Delegation Delegation `json:"delegation"`
	Roles      []Role     `json:"roles"`
}

// Validate enforces separate workspaces for concurrent writers.
func (t Team) Validate() error {
	if len(t.Roles) == 0 {
		return fmt.Errorf("team requires roles")
	}
	seen := map[string]string{}
	for _, r := range t.Roles {
		if r.Workspace != "" {
			if owner, ok := seen[r.Workspace]; ok {
				return fmt.Errorf("worktree %s owned by both %s and %s without policy", r.Workspace, owner, r.Name)
			}
			seen[r.Workspace] = r.Name
		}
	}
	return nil
}

// AllocateWorktree creates an isolated git worktree for a role.
func AllocateWorktree(repoRoot, branch string) (string, error) {
	dir := filepath.Join(os.TempDir(), "prumo-wt-"+branch)
	_ = os.MkdirAll(filepath.Dir(dir), 0o755)
	cmd := exec.Command("git", "worktree", "add", "--detach", dir)
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("worktree add failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return dir, nil
}

// ReviewInput is the least-context payload a reviewer receives.
type ReviewInput struct {
	Diff         string   `json:"diff"`
	Requirements []string `json:"requirements"`
	Evidence     []string `json:"evidence"`
}

// ReviewVerdict is the explicit review outcome.
type ReviewVerdict struct {
	Approve bool   `json:"approve"`
	Notes   string `json:"notes,omitempty"`
}
