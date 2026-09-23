package daemon

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/model"
	"github.com/raillen/prumo/internal/harness/perm"
	"github.com/raillen/prumo/internal/harness/runtime"
)

// Run contexts derived from context.Background(), so nothing the daemon did could
// reach them. A SIGTERM closed the listener and the process exited with its runs
// still mid-flight, and their checkpoints and records were simply gone
// (GAP-105).

// A run that finishes during the drain must be able to: the daemon waits for
// work in flight rather than cancelling it the moment the signal arrives.
func TestShutdownLetsARunInFlightFinish(t *testing.T) {
	dir := t.TempDir()
	tools := &slowTools{delay: 400 * time.Millisecond}
	srv := New(filepath.Join(dir, "s.sock"), filepath.Join(dir, "store"), toolCallingDeps(tools, dir))
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan struct{})
	go func() {
		_ = srv.Serve(ctx)
		close(served)
	}()
	waitForSocket(t, filepath.Join(dir, "s.sock"))

	if res := srv.opStart(map[string]any{
		"op": "run.start", "goal": "do the thing", "provider": "fake", "run_id": "R-drain", "max_turns": 1,
	}); res["ok"] != true {
		t.Fatalf("start failed: %v", res)
	}
	// Let the run get into the tool call before the signal.
	time.Sleep(150 * time.Millisecond)

	cancel()
	select {
	case <-served:
	case <-time.After(drainGrace + 10*time.Second):
		t.Fatal("Serve did not return after the context was cancelled")
	}

	// The run reached a real stopping point instead of being dropped.
	record, err := os.ReadFile(filepath.Join(dir, "store", "daemon-run-R-drain.json"))
	if err != nil {
		t.Fatalf("a drained run must leave its record: %v", err)
	}
	if len(record) == 0 {
		t.Fatal("the record is empty")
	}
	if tools.calls() == 0 {
		t.Fatal("the run was cancelled before its tool ran; a drain exists to let it finish")
	}
}

func TestShutdownCancelsARunThatWillNotFinish(t *testing.T) {
	// A run stuck in a call longer than the grace period must be cancelled, or
	// the daemon never exits. Cancelling unwinds it through the same path a
	// client-side cancel takes, so its record is still written.
	dir := t.TempDir()
	tools := &slowTools{delay: 30 * time.Second}
	srv := New(filepath.Join(dir, "s.sock"), filepath.Join(dir, "store"), toolCallingDeps(tools, dir))
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan struct{})
	go func() {
		_ = srv.Serve(ctx)
		close(served)
	}()
	waitForSocket(t, filepath.Join(dir, "s.sock"))

	srv.opStart(map[string]any{
		"op": "run.start", "goal": "hang", "provider": "fake", "run_id": "R-hang", "max_turns": 1,
	})
	time.Sleep(150 * time.Millisecond)

	begin := time.Now()
	cancel()
	select {
	case <-served:
	case <-time.After(drainGrace + 15*time.Second):
		t.Fatal("a run that ignores cancellation must not keep the daemon alive forever")
	}
	elapsed := time.Since(begin)
	if elapsed < drainGrace {
		t.Fatalf("the drain gave up after %s, before its grace period", elapsed)
	}
	if elapsed > drainGrace+cancelGrace+10*time.Second {
		t.Fatalf("the drain took %s, far past its grace period", elapsed)
	}
	// Cancelling is not finishing: the run must have left the map, so its
	// checkpoint and record are on disk before the daemon returns.
	srv.mu.Lock()
	remaining := len(srv.runs)
	srv.mu.Unlock()
	if remaining != 0 {
		t.Fatalf("%d runs were still present after the drain returned", remaining)
	}
	if _, err := os.Stat(filepath.Join(dir, "store", "daemon-run-R-hang.json")); err != nil {
		t.Fatalf("a cancelled run must still leave its record: %v", err)
	}
}

func TestRunContextsAreCancelledWithTheDaemon(t *testing.T) {
	// The root cause: a run context that nothing can cancel. A run started
	// directly and then cancelled through the server must see its context die.
	srv := New("s.sock", t.TempDir(), Deps{Workspace: t.TempDir(), Tools: &slowTools{}})
	res := srv.opStart(map[string]any{
		"op": "run.start", "goal": "g", "provider": "fake", "run_id": "R-ctx", "max_turns": 1,
	})
	if res["ok"] != true {
		t.Fatalf("start: %v", res)
	}
	srv.mu.Lock()
	ar := srv.runs["R-ctx"]
	srv.mu.Unlock()
	if ar == nil {
		t.Fatal("run not registered")
	}
	captured := ar.ctx
	if captured == nil {
		t.Fatal("the run must carry a context")
	}
	if captured == context.Background() {
		t.Fatal("the run context is context.Background(), so nothing the daemon does reaches it")
	}
	// A run context must be a child of the server's, so cancelling the server
	// reaches it.
	srv.rootCancel()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if captured.Err() != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("cancelling the server did not reach a run context")
}

func TestAParkedRunDoesNotHoldTheDrainOpen(t *testing.T) {
	// A run waiting for a permission is not going to be answered during a
	// shutdown. Counting it as work in flight would make the drain wait for an
	// answer that never comes.
	dir := t.TempDir()
	srv := New(filepath.Join(dir, "s.sock"), filepath.Join(dir, "store"), Deps{Workspace: dir})
	ar := &activeRun{cancel: func() {}}
	ar.runner = runtime.NewRunner(runtime.Services{
		Models: model.NewFake(nil),
		Perms:  perm.New(perm.Policy{DefaultAction: agent.PermissionAsk}),
		Tools:  &slowTools{},
	}, "R-park", "S1")
	// Actually parked. A run with nothing pending is work in flight, not a run
	// waiting for an answer nobody will give during a shutdown — the drain is
	// right to wait for the first and not the second.
	ar.runner.State.PendingPerms = []string{"perm-c1"}
	srv.runs["R-park"] = ar

	begin := time.Now()
	srv.drain()
	elapsed := time.Since(begin)
	if elapsed > time.Second {
		t.Fatalf("a parked run held the drain open for %s", elapsed)
	}
}

func waitForSocket(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("daemon socket %s never appeared", path)
}

// toolCallingDeps builds a daemon whose provider actually asks for a tool. The
// plain fake never does, so a run using it has nothing to drain.
func toolCallingDeps(tools *slowTools, workspace string) Deps {
	return Deps{
		NewProvider: func(name, baseURL, apiKey, mdl string) (model.Provider, error) {
			return model.NewFake(map[string][]model.ScriptStep{
				"*": {{Kind: "tool_call", Tool: &agent.ToolCall{ID: "c1", Name: "fs.read", IdempotencyKey: "k1"}}, {Kind: "complete"}},
			}), nil
		},
		Tools:      tools,
		Workspace:  workspace,
		PermPolicy: perm.Policy{DefaultAction: agent.PermissionAllow},
	}
}
