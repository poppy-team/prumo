// Package worktree provides Git worktree discovery, creation, and management
// for isolated multi-tab agent workflows in prumo-tui.
package worktree

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Worktree describes a single Git worktree linked to the repository.
type Worktree struct {
	Path   string
	Branch string
	Head   string
	Bare   bool
}

// List queries the git repository containing baseDir and returns all worktrees.
func List(ctx context.Context, baseDir string) ([]Worktree, error) {
	cmd := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain")
	cmd.Dir = baseDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git worktree list failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return ParseList(string(out)), nil
}

// ParseList parses the machine-readable output of `git worktree list --porcelain`.
func ParseList(raw string) []Worktree {
	var list []Worktree
	var current Worktree
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Path != "" {
				list = append(list, current)
				current = Worktree{}
			}
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		key := parts[0]
		val := ""
		if len(parts) > 1 {
			val = strings.TrimSpace(parts[1])
		}
		switch key {
		case "worktree":
			current.Path = val
		case "HEAD":
			current.Head = val
		case "branch":
			current.Branch = strings.TrimPrefix(val, "refs/heads/")
		case "bare":
			current.Bare = true
		}
	}
	if current.Path != "" {
		list = append(list, current)
	}
	return list
}

// Create creates a new worktree at targetPath checked out to branch.
// If the branch does not exist, it creates a new branch (-b).
func Create(ctx context.Context, baseDir, targetPath, branch string) error {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Check if branch already exists
	checkCmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", branch)
	checkCmd.Dir = baseDir
	branchExists := checkCmd.Run() == nil

	var args []string
	if branchExists {
		args = []string{"worktree", "add", targetPath, branch}
	} else {
		args = []string{"worktree", "add", "-b", branch, targetPath}
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = baseDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree add failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Remove deletes a linked worktree.
func Remove(ctx context.Context, baseDir, targetPath string, force bool) error {
	args := []string{"worktree", "remove", targetPath}
	if force {
		args = append(args, "--force")
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = baseDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// DefaultWorktreePath generates a standard path for a worktree within .prumo/worktrees/.
func DefaultWorktreePath(baseDir, branch string) string {
	safeName := strings.ReplaceAll(branch, "/", "-")
	return filepath.Join(baseDir, ".prumo", "worktrees", safeName)
}
