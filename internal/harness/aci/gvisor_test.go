package aci

import (
	"context"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

func TestRuntimeArgsReachDaemon(t *testing.T) {
	args := containerArgs("docker", "/w", ContainerSpec{Image: "img", Workdir: "/w", Script: "x", RuntimeArgs: []string{"--runtime=runsc"}})
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--runtime=runsc") || !strings.Contains(joined, "--rm") {
		t.Fatalf("runtime args missing: %v", args)
	}
	f := &FakeRunner{}
	e := NewContainer(t.TempDir(), "img", f)
	e.RuntimeArgs = []string{"--runtime=runsc"}
	_, _ = e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "process.exec", Arguments: map[string]any{"command": "echo hi"}})
	if len(f.Specs) != 1 || len(f.Specs[0].RuntimeArgs) != 1 || f.Specs[0].RuntimeArgs[0] != "--runtime=runsc" {
		t.Fatalf("executor must forward runtime args: %+v", f.Specs)
	}
}
