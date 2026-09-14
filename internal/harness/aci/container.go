// Container execution: command tools run inside an isolated container
// (docker/podman CLI), file tools stay on the host against the same mounted
// workspace. Network is denied by default; memory/CPU/PID limits apply.
// A missing/unreachable runtime is an explicit error, never silent fallback.
package aci

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/raillen/prumo/internal/harness/agent"
)

// ContainerSpec describes one isolated command invocation.
type ContainerSpec struct {
	Image     string
	Workdir   string // container workdir; host root is mounted there
	Network   string // default "none"
	Memory    string // e.g. "512m", "" disables
	CPUs      string // e.g. "1.0", "" disables
	PidsLimit string // e.g. "128", "" disables
	Env       []string
	Script    string // run via sh -c
	// RuntimeArgs injects daemon-side runtime flags (e.g. --runtime=runsc).
	RuntimeArgs []string
}

// Runner executes a ContainerSpec. Implementations: CLIRunner (real
// docker/podman), FakeRunner (deterministic tests).
type Runner interface {
	Run(ctx context.Context, root string, spec ContainerSpec) (output []byte, exit int, err error)
	Available() bool
	Describe() string
}

// CLIRunner shells out to a container runtime binary.
type CLIRunner struct {
	Runtime string // docker | podman
	// ExtraArgs inject runtime flags after "run" (e.g. --runtime=runsc
	// for gVisor-backed execution where the daemon offers it).
	ExtraArgs []string
}

func (r CLIRunner) Available() bool {
	if r.Runtime == "" {
		return false
	}
	if _, err := exec.LookPath(r.Runtime); err != nil {
		return false
	}
	// Ping the daemon; a present-but-unreachable runtime is unavailable.
	ctx := context.Background()
	if out, err := exec.CommandContext(ctx, r.Runtime, "info", "--format", "{{.ServerVersion}}").CombinedOutput(); err != nil {
		_ = out
		return false
	}
	return true
}

func (r CLIRunner) Describe() string { return "container runtime=" + r.Runtime }

// Args builds the argv for inspection/tests (Run adds ctx + capture).
func containerArgs(runtime, root string, spec ContainerSpec) []string {
	network := spec.Network
	if network == "" {
		network = "none"
	}
	args := []string{"run"}
	args = append(args, spec.RuntimeArgs...)
	args = append(args, "--rm", "-i", "--network", network)
	if spec.Memory != "" {
		args = append(args, "--memory", spec.Memory)
	}
	if spec.CPUs != "" {
		args = append(args, "--cpus", spec.CPUs)
	}
	if spec.PidsLimit != "" {
		args = append(args, "--pids-limit", spec.PidsLimit)
	}
	for _, e := range spec.Env {
		args = append(args, "-e", e)
	}
	args = append(args, "-v", root+":"+spec.Workdir, "-w", spec.Workdir, spec.Image, "sh", "-c", spec.Script)
	_ = runtime
	return args
}

func (r CLIRunner) Run(ctx context.Context, root string, spec ContainerSpec) ([]byte, int, error) {
	if !r.Available() {
		return nil, 1, fmt.Errorf("container runtime %q unavailable", r.Runtime)
	}
	spec.RuntimeArgs = append(append([]string{}, r.ExtraArgs...), spec.RuntimeArgs...)
	args := containerArgs(r.Runtime, root, spec)
	cmd := exec.CommandContext(ctx, r.Runtime, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	exit := 0
	if err != nil {
		exit = 1
		if ctx.Err() != nil {
			return buf.Bytes(), exit, ctx.Err()
		}
	}
	return buf.Bytes(), exit, nil
}

// FakeRunner records specs for deterministic tests.
type FakeRunner struct {
	Specs  []ContainerSpec
	Output string
	Exit   int
	Err    error
}

func (f *FakeRunner) Run(_ context.Context, _ string, spec ContainerSpec) ([]byte, int, error) {
	f.Specs = append(f.Specs, spec)
	return []byte(f.Output), f.Exit, f.Err
}
func (f *FakeRunner) Available() bool  { return true }
func (f *FakeRunner) Describe() string { return "fake container runner" }

// UnavailableRunner models a host without a container runtime.
type UnavailableRunner struct{ Runtime string }

func (u UnavailableRunner) Run(_ context.Context, _ string, _ ContainerSpec) ([]byte, int, error) {
	return nil, 1, fmt.Errorf("container runtime %q unavailable", u.Runtime)
}
func (u UnavailableRunner) Available() bool  { return false }
func (u UnavailableRunner) Describe() string { return "unavailable: " + u.Runtime }

// ContainerExecutor routes command tools into the container, file tools to
// the host executor (same workspace, mounted read-write).
type ContainerExecutor struct {
	Host      *Executor
	Runner    Runner
	Root      string
	Image     string
	Memory    string
	CPUs      string
	PidsLimit string
	OutputMax int
	// RuntimeArgs forwards daemon-side runtime flags (e.g. --runtime=runsc
	// for gVisor where the daemon offers it).
	RuntimeArgs []string
}

func NewContainer(root, image string, runner Runner) *ContainerExecutor {
	return &ContainerExecutor{
		Host: New(root), Runner: runner, Root: root, Image: image,
		Memory: "512m", CPUs: "1.0", PidsLimit: "128", OutputMax: 32 * 1024,
	}
}

func (e *ContainerExecutor) KindOf(name string) string { return e.Host.KindOf(name) }

// containerTools are executed inside the container; everything else runs on
// the host against the mounted workspace.
func containerTools() map[string]bool {
	return map[string]bool{"process.exec": true, "test.run": true, "code.diagnostics": true}
}

func (e *ContainerExecutor) scriptFor(call agent.ToolCall) (string, error) {
	arg := func(k string) string {
		if call.Arguments == nil {
			return ""
		}
		v, _ := call.Arguments[k].(string)
		return v
	}
	switch call.Name {
	case "process.exec":
		if arg("command") == "" {
			return "", fmt.Errorf("missing command")
		}
		return arg("command"), nil
	case "test.run":
		return "go test ./... 2>&1 | head -c 8000", nil
	case "code.diagnostics":
		return "go vet ./... 2>&1 | head -c 8000", nil
	default:
		return "", fmt.Errorf("not a container tool: %s", call.Name)
	}
}

func (e *ContainerExecutor) Execute(ctx context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	if !containerTools()[call.Name] {
		return e.Host.Execute(ctx, call)
	}
	script, err := e.scriptFor(call)
	if err != nil {
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: err.Error()}, nil
	}
	if !e.Runner.Available() {
		return agent.ToolResult{ToolCallID: call.ID, ExitCode: 1, Error: "container runtime unavailable: refusing silent host fallback"}, nil
	}
	out, exit, err := e.Runner.Run(ctx, e.Root, ContainerSpec{
		Image: e.Image, Workdir: e.Root, Memory: e.Memory,
		CPUs: e.CPUs, PidsLimit: e.PidsLimit, Script: script,
		RuntimeArgs: e.RuntimeArgs,
	})
	if err != nil && ctx.Err() != nil {
		return agent.ToolResult{}, err
	}
	text, trunc := bound(string(out), e.OutputMax)
	res := agent.ToolResult{ToolCallID: call.ID, ExitCode: exit, Output: text, Truncated: trunc}
	if err != nil {
		res.Error = strings.TrimSpace(err.Error())
	}
	return res, nil
}
