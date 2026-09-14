package prumo_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/daemon"
	sdk "github.com/raillen/prumo/sdk/prumo"
)

// TestBoundaryNoInternalImports is the split-gate conformance kernel: the
// public SDK must not depend on prumo/internal. Test-only imports of
// internal packages live in this _test file, never in client.go.
func TestBoundaryNoInternalImports(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go tool unavailable")
	}
	out, err := exec.Command("go", "list", "-deps", "github.com/raillen/prumo/sdk/prumo").Output()
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}
	for _, dep := range strings.Fields(string(out)) {
		if strings.Contains(dep, "prumo/internal") {
			t.Fatalf("SDK depends on internal package %q", dep)
		}
	}
}

type daemonTestTools struct{}

func (daemonTestTools) Execute(_ context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	return agent.ToolResult{ToolCallID: call.ID, Output: "ok"}, nil
}
func (daemonTestTools) KindOf(string) string { return "read-only" }

func serveSDKTestDaemon(t *testing.T, dir string) (sdk.Client, context.CancelFunc) {
	t.Helper()
	sock := filepath.Join(dir, "agentd.sock")
	srv := daemon.New(sock, filepath.Join(dir, "store"), daemon.Deps{Tools: daemonTestTools{}})
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = srv.Serve(ctx) }()
	c := sdk.Client{SocketPath: sock}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := c.Protocol(context.Background()); err == nil {
			return c, cancel
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("daemon did not come up")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestSDKRoundtrip(t *testing.T) {
	dir := t.TempDir()
	c, cancel := serveSDKTestDaemon(t, dir)
	defer cancel()
	ctx := context.Background()

	info, err := c.Protocol(ctx)
	if err != nil || info.Version != sdk.ProtocolVersion {
		t.Fatalf("protocol failed: %+v %v", info, err)
	}
	if len(info.Ops) == 0 {
		t.Fatal("daemon must advertise ops")
	}

	id, err := c.Start(ctx, sdk.StartRequest{Goal: "sdk roundtrip", Provider: "fake", RunID: "R-sdk", MaxTurns: 1})
	if err != nil || id != "R-sdk" {
		t.Fatalf("start failed: %q %v", id, err)
	}
	stCtx, stop := context.WithTimeout(ctx, 10*time.Second)
	defer stop()
	st, err := c.Wait(stCtx, "R-sdk", 20*time.Millisecond)
	if err != nil || st.Status != "complete" {
		t.Fatalf("wait failed: %+v %v", st, err)
	}
	evs, err := c.Events(ctx, "R-sdk")
	if err != nil || len(evs) == 0 {
		t.Fatalf("events failed: %v", err)
	}
	runs, err := c.List(ctx)
	if err != nil || len(runs) != 1 || runs[0].RunID != "R-sdk" {
		t.Fatalf("list failed: %+v %v", runs, err)
	}
	if err := c.Cancel(ctx, "R-sdk"); err == nil {
		t.Fatal("cancel of a finished run must error")
	}
	if err := c.Steer(ctx, "R-sdk", "too late"); err == nil {
		t.Fatal("steer of a finished run must error")
	}
	job, err := c.Schedule(ctx, "nightly", "fake", 3600, 1)
	if err != nil || job == "" {
		t.Fatalf("schedule failed: %q %v", job, err)
	}
	jobs, err := c.Jobs(ctx)
	if err != nil || len(jobs) != 1 || jobs[0].ID != job {
		t.Fatalf("jobs failed: %+v %v", jobs, err)
	}
	if err := c.Unschedule(ctx, job); err != nil {
		t.Fatalf("unschedule failed: %v", err)
	}
	var _ sdk.RunStatus
	var _ sdk.Event
}
