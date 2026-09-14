package acpserver

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

func TestDaemonBackendPromptFlow(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "agentd.sock")
	srv := daemon.New(sock, filepath.Join(dir, "store"), daemon.Deps{Tools: stubTools{}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Serve(ctx) }()
	deadline := time.Now().Add(5 * time.Second)
	probe := daemon.Client{SocketPath: sock}
	for {
		if _, err := probe.Call(map[string]any{"op": "protocol"}); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("daemon did not come up")
		}
		time.Sleep(10 * time.Millisecond)
	}
	b := DaemonBackend{Daemon: daemon.Client{SocketPath: sock}}
	sid, err := b.NewSession(ctx, dir)
	if err != nil || sid == "" {
		t.Fatal("new session failed")
	}
	var chunks int
	stop, err := b.Prompt(ctx, sid, "hello", func(u Update) { chunks++ })
	if err != nil || stop != "end_turn" || chunks == 0 {
		t.Fatalf("prompt flow failed: %q %v chunks=%d", stop, err, chunks)
	}
	if err := b.Resume(ctx, sid); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if _, err := b.Load(ctx, sid); err != nil {
		t.Fatalf("load failed: %v", err)
	}
}
