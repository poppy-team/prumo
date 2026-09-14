package aci

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

func TestContainerArgvDefaults(t *testing.T) {
	args := containerArgs("docker", "/work", ContainerSpec{Image: "img", Workdir: "/work", Script: "echo hi"})
	joined := strings.Join(args, " ")
	for _, want := range []string{"--rm", "--network none", "-v /work:/work", "-w /work", "sh -c"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("argv missing %q: %s", want, joined)
		}
	}
}

func TestContainerLimitsPropagated(t *testing.T) {
	f := &FakeRunner{}
	e := NewContainer(t.TempDir(), "img", f)
	e.Memory = "256m"
	e.CPUs = "0.5"
	e.PidsLimit = "64"
	_, _ = e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "process.exec", Arguments: map[string]any{"command": "echo hi"}})
	if len(f.Specs) != 1 {
		t.Fatalf("expected 1 container run, got %d", len(f.Specs))
	}
	sp := f.Specs[0]
	if sp.Memory != "256m" || sp.CPUs != "0.5" || sp.PidsLimit != "64" || sp.Network != "" {
		t.Fatalf("limits not propagated: %+v", sp)
	}
}

func TestContainerRoutesFileToolsToHost(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/a.txt", []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := &FakeRunner{}
	e := NewContainer(dir, "img", f)
	res, err := e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "fs.read", Arguments: map[string]any{"path": "a.txt"}})
	if err != nil || res.ExitCode != 0 || res.Output != "hello" {
		t.Fatalf("fs.read must run on host: %+v %v", res, err)
	}
	if len(f.Specs) != 0 {
		t.Fatal("file tools must not enter the container")
	}
}

func TestContainerUnavailableRefuses(t *testing.T) {
	e := NewContainer(t.TempDir(), "img", UnavailableRunner{Runtime: "docker"})
	res, _ := e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "process.exec", Arguments: map[string]any{"command": "echo hi"}})
	if res.ExitCode == 0 || !strings.Contains(res.Error, "unavailable") {
		t.Fatalf("must refuse without silent fallback: %+v", res)
	}
}

func TestContainerOutputBounded(t *testing.T) {
	f := &FakeRunner{Output: strings.Repeat("x", 100000)}
	e := NewContainer(t.TempDir(), "img", f)
	e.OutputMax = 100
	res, _ := e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "test.run"})
	if !res.Truncated || len(res.Output) > 200 {
		t.Fatalf("output must be bounded: len=%d trunc=%v", len(res.Output), res.Truncated)
	}
}

// TestContainerLive exercises a real runtime when PRUMO_LIVE_DOCKER=1.
// Skipped by default so CI stays hermetic.
func TestContainerLive(t *testing.T) {
	if os.Getenv("PRUMO_LIVE_DOCKER") == "" {
		t.Skip("set PRUMO_LIVE_DOCKER=1 with a reachable docker/podman daemon")
	}
	rt := DetectContainerRuntime()
	if rt == "" {
		t.Skip("no container runtime detected")
	}
	r := CLIRunner{Runtime: rt}
	if !r.Available() {
		t.Skip("runtime unreachable")
	}
	e := NewContainer(t.TempDir(), "alpine:latest", r)
	res, err := e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "process.exec", Arguments: map[string]any{"command": "echo live-ok"}})
	if err != nil || res.ExitCode != 0 || !strings.Contains(res.Output, "live-ok") {
		t.Fatalf("live run failed: %+v %v", res, err)
	}
}
