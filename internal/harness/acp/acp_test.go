package acp

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/daemon"
)

type stubTools struct{}

func (stubTools) Execute(_ context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	return agent.ToolResult{ToolCallID: call.ID, Output: "ok"}, nil
}
func (stubTools) KindOf(string) string { return "read-only" }

func serveBridge(t *testing.T, dir string) (Bridge, context.CancelFunc) {
	t.Helper()
	sock := filepath.Join(dir, "agentd.sock")
	srv := daemon.New(sock, filepath.Join(dir, "store"), daemon.Deps{Tools: stubTools{}})
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = srv.Serve(ctx) }()
	b := Bridge{Daemon: daemon.Client{SocketPath: sock}}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := b.Daemon.Call(map[string]any{"op": "protocol"}); err == nil {
			return b, cancel
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("daemon did not come up")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestBridgeNewPromptCancel(t *testing.T) {
	dir := t.TempDir()
	b, cancel := serveBridge(t, dir)
	defer cancel()
	ctx := context.Background()

	s, err := b.NewSession(ctx, "editor goal")
	if err != nil || s.ID == "" || s.Provider != "prumo-native" {
		t.Fatalf("new failed: %+v %v", s, err)
	}
	// Prompt while starting/running steers the same session...
	s2, err := b.Prompt(ctx, s.ID, "follow-up")
	if err != nil {
		t.Fatalf("prompt failed: %v", err)
	}
	if s2.ID != s.ID {
		// ...unless the run already finished, in which case input is
		// preserved as a fresh run instead of lost.
		t.Logf("run finished fast; prompt spawned %s", s2.ID)
	}
	// Prompt to a dead session preserves input as a new run.
	s3, err := b.Prompt(ctx, "R-nope", "rescued input")
	if err != nil || s3.ID == "" || s3.ID == "R-nope" {
		t.Fatalf("dead-session prompt must spawn fresh run: %+v %v", s3, err)
	}
	wctx, stop := context.WithTimeout(ctx, 10*time.Second)
	defer stop()
	if _, err := b.Wait(wctx, s3.ID); err != nil {
		t.Fatalf("wait failed: %v", err)
	}
	evs, err := b.Events(ctx, s3.ID)
	if err != nil || len(evs) == 0 {
		t.Fatalf("events failed: %v", err)
	}
	if err := b.Cancel(ctx, s3.ID); err == nil {
		t.Fatal("cancel of finished run must error")
	}
}
