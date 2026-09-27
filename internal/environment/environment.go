package environment

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/raillen/prumo/internal/harness/safepath"
)

type Isolation string

const (
	IsolationNone        Isolation = "none"
	IsolationProjectRoot Isolation = "project-root"
	IsolationWorktree    Isolation = "worktree"
	IsolationSandbox     Isolation = "sandbox"
)

type Descriptor struct {
	ID                     string    `json:"id"`
	Version                int       `json:"version"`
	Kind                   string    `json:"kind"`      // "local", "worktree", "sandbox", "container"
	Isolation              Isolation `json:"isolation"` // "none", "project-root", "worktree", "sandbox"
	Network                string    `json:"network"`   // "deny", "restricted", "allow"
	FilesystemRoots        []string  `json:"filesystem_roots,omitempty"`
	EnvironmentFingerprint string    `json:"environment_fingerprint,omitempty"`
}

type ExecutionResult struct {
	Output     []byte `json:"output"`
	ExitCode   int    `json:"exit_code"`
	DurationMS int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

type Environment interface {
	Descriptor() Descriptor
	Prepare() error
	Execute(name string, args []string, env []string, dir string) (ExecutionResult, error)
	Cleanup() error
}

// LocalEnvironment runs directly on the local machine under project root.
type LocalEnvironment struct {
	ProjectRoot string
	SafeMode    bool
	desc        Descriptor
}

func NewLocalEnvironment(projectRoot string, safeMode bool) *LocalEnvironment {
	abs, _ := filepath.Abs(projectRoot)
	h := sha256.Sum256([]byte(abs))
	return &LocalEnvironment{
		ProjectRoot: abs,
		SafeMode:    safeMode,
		desc: Descriptor{
			ID:                     "env-local-" + hex.EncodeToString(h[:4]),
			Version:                1,
			Kind:                   "local",
			Isolation:              IsolationProjectRoot,
			Network:                "restricted",
			FilesystemRoots:        []string{abs},
			EnvironmentFingerprint: hex.EncodeToString(h[:8]),
		},
	}
}

func (e *LocalEnvironment) Descriptor() Descriptor {
	return e.desc
}

func (e *LocalEnvironment) Prepare() error {
	if e.ProjectRoot == "" {
		return errors.New("project root cannot be empty")
	}
	info, err := os.Stat(e.ProjectRoot)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("project root directory invalid: %s", e.ProjectRoot)
	}
	return nil
}

var blockedInSafeMode = []string{
	"rm -rf /",
	"mkfs",
	"dd if=",
	":(){ :|:& };:",
}

func (e *LocalEnvironment) Execute(name string, args []string, env []string, dir string) (ExecutionResult, error) {
	cmdStr := name + " " + strings.Join(args, " ")
	if e.SafeMode {
		for _, blocked := range blockedInSafeMode {
			if strings.Contains(cmdStr, blocked) {
				return ExecutionResult{ExitCode: 1, Error: "command blocked by safe mode"}, errors.New("safe mode denied execution")
			}
		}
	}

	workDir := dir
	if workDir == "" {
		workDir = e.ProjectRoot
	}

	// Verify workDir is inside ProjectRoot in safe mode. Path-component
	// containment, not string prefix: a sibling directory that shares the
	// project's name prefix passed the old check.
	if e.SafeMode {
		resolved, err := safepath.Resolve(e.ProjectRoot, workDir, false)
		if err != nil {
			return ExecutionResult{ExitCode: 1, Error: "workdir outside project root blocked in safe mode"}, errors.New("safe mode denied out-of-root directory")
		}
		workDir = resolved
	}

	cmd := exec.Command(name, args...)
	cmd.Dir = workDir
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	start := time.Now()
	output, err := cmd.CombinedOutput()
	dur := time.Since(start).Milliseconds()

	exitCode := 0
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return ExecutionResult{
		Output:     output,
		ExitCode:   exitCode,
		DurationMS: dur,
		Error:      errMsg,
	}, err
}

func (e *LocalEnvironment) Cleanup() error {
	return nil
}

// WorktreeEnvironment creates an isolated Git worktree for execution.
type WorktreeEnvironment struct {
	RepoRoot    string
	WorktreeDir string
	Branch      string
	desc        Descriptor
}

func NewWorktreeEnvironment(repoRoot, worktreeDir, branch string) *WorktreeEnvironment {
	absRepo, _ := filepath.Abs(repoRoot)
	absWorktree, _ := filepath.Abs(worktreeDir)
	h := sha256.Sum256([]byte(absWorktree))
	return &WorktreeEnvironment{
		RepoRoot:    absRepo,
		WorktreeDir: absWorktree,
		Branch:      branch,
		desc: Descriptor{
			ID:                     "env-worktree-" + hex.EncodeToString(h[:4]),
			Version:                1,
			Kind:                   "worktree",
			Isolation:              IsolationWorktree,
			Network:                "deny",
			FilesystemRoots:        []string{absWorktree},
			EnvironmentFingerprint: hex.EncodeToString(h[:8]),
		},
	}
}

func (w *WorktreeEnvironment) Descriptor() Descriptor {
	return w.desc
}

func (w *WorktreeEnvironment) Prepare() error {
	cmd := exec.Command("git", "worktree", "add", "-b", w.Branch, w.WorktreeDir, "HEAD")
	cmd.Dir = w.RepoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create git worktree: %v (%s)", err, out)
	}
	return nil
}

func (w *WorktreeEnvironment) Execute(name string, args []string, env []string, dir string) (ExecutionResult, error) {
	workDir := dir
	if workDir == "" {
		workDir = w.WorktreeDir
	}

	cmd := exec.Command(name, args...)
	cmd.Dir = workDir
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	start := time.Now()
	output, err := cmd.CombinedOutput()
	dur := time.Since(start).Milliseconds()

	exitCode := 0
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return ExecutionResult{
		Output:     output,
		ExitCode:   exitCode,
		DurationMS: dur,
		Error:      errMsg,
	}, err
}

func (w *WorktreeEnvironment) Cleanup() error {
	cmd := exec.Command("git", "worktree", "remove", "--force", w.WorktreeDir)
	cmd.Dir = w.RepoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove git worktree: %v (%s)", err, out)
	}
	// Delete branch
	delCmd := exec.Command("git", "branch", "-D", w.Branch)
	delCmd.Dir = w.RepoRoot
	_ = delCmd.Run()
	return nil
}
