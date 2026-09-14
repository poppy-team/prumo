// Sandbox providers: gradual isolation ladder.
//
// LocalTrusted and Worktree are available now (see internal/environment).
// Container is detected (docker/podman, rootless preferred) but execution
// behind it lands after the headless baseline; the provider reports
// availability honestly instead of pretending.
package aci

import (
	"fmt"
	"os/exec"
)

// SandboxKind is one rung of the isolation ladder.
type SandboxKind string

const (
	SandboxLocalTrusted SandboxKind = "local-trusted"
	SandboxWorktree     SandboxKind = "worktree"
	SandboxContainer    SandboxKind = "container"
	SandboxStrong       SandboxKind = "strong"
)

// SandboxProvider describes one executable isolation rung.
type SandboxProvider interface {
	Kind() SandboxKind
	Available() bool
	Describe() string
}

// LocalProvider always runs (explicit user trust).
type LocalProvider struct{ Root string }

func (l LocalProvider) Kind() SandboxKind { return SandboxLocalTrusted }
func (l LocalProvider) Available() bool   { return l.Root != "" }
func (l LocalProvider) Describe() string {
	return fmt.Sprintf("local-trusted root=%s network=restricted", l.Root)
}

// WorktreeProvider isolates git state, not processes (see docs).
type WorktreeProvider struct{ Path string }

func (w WorktreeProvider) Kind() SandboxKind { return SandboxWorktree }
func (w WorktreeProvider) Available() bool   { return w.Path != "" }
func (w WorktreeProvider) Describe() string {
	return fmt.Sprintf("worktree path=%s (git isolation only)", w.Path)
}

// ContainerProvider needs a container runtime; unavailability is explicit.
type ContainerProvider struct {
	Runtime string // docker | podman | ""
	Image   string
}

func (c ContainerProvider) Kind() SandboxKind { return SandboxContainer }
func (c ContainerProvider) Available() bool {
	if c.Runtime == "" {
		return false
	}
	_, err := exec.LookPath(c.Runtime)
	return err == nil
}
func (c ContainerProvider) Describe() string {
	return fmt.Sprintf("container runtime=%s image=%s available=%v", c.Runtime, c.Image, c.Available())
}

// DetectContainerRuntime prefers podman (rootless) over docker, else "".
func DetectContainerRuntime() string {
	if _, err := exec.LookPath("podman"); err == nil {
		return "podman"
	}
	if _, err := exec.LookPath("docker"); err == nil {
		return "docker"
	}
	return ""
}

// StrongProvider is reserved for strong isolation (gVisor runsc and
// successors). Today it reports availability honestly via runtime
// detection; execution behind it is future work (GAP-024).
type StrongProvider struct {
	Runtime string // runsc | ""
}

func (s StrongProvider) Kind() SandboxKind { return SandboxStrong }
func (s StrongProvider) Available() bool {
	if s.Runtime == "" {
		return false
	}
	_, err := exec.LookPath(s.Runtime)
	return err == nil
}
func (s StrongProvider) Describe() string {
	return "strong isolation runtime=" + s.Runtime + " (detection only)"
}

// DetectStrongRuntime finds a strong-isolation runtime, if any.
func DetectStrongRuntime() string {
	if _, err := exec.LookPath("runsc"); err == nil {
		return "runsc"
	}
	return ""
}

// DefaultChain returns the honest ladder for this host.
func DefaultChain(root string) []SandboxProvider {
	return []SandboxProvider{
		LocalProvider{Root: root},
		WorktreeProvider{Path: root},
		ContainerProvider{Runtime: DetectContainerRuntime(), Image: "prumo-harness:latest"},
		StrongProvider{Runtime: DetectStrongRuntime()},
	}
}
