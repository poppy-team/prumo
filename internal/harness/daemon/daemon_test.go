package daemon

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/model"
	"github.com/raillen/prumo/internal/harness/perm"
	harnessprotocol "github.com/raillen/prumo/internal/harness/protocol"
	harnessruntime "github.com/raillen/prumo/internal/harness/runtime"
)

type stubTools struct {
	mu    sync.Mutex
	block bool
	calls int
	kinds map[string]string
}

func (s *stubTools) Execute(ctx context.Context, call agent.ToolCall) (agent.ToolResult, error) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	if s.block {
		<-ctx.Done()
		return agent.ToolResult{}, ctx.Err()
	}
	return agent.ToolResult{ToolCallID: call.ID, Output: "ok"}, nil
}

func (s *stubTools) OperationOf(string) string { return "" }
func (s *stubTools) KindOf(name string) string {
	if kind, ok := s.kinds[name]; ok {
		return kind
	}
	return "read-only"
}

// callCount reads the execution count under the lock: the run goroutine
// increments it while the test observes the daemon over a socket, which is not
// a Go happens-before edge.
func (s *stubTools) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func fakeDeps(block bool) Deps {
	return Deps{
		NewProvider: func(name, baseURL, apiKey, mdl string) (model.Provider, error) {
			if block {
				return model.NewFake(map[string][]model.ScriptStep{
					"*": {{Kind: "tool_call", Tool: &agent.ToolCall{ID: "c1", Name: "sleep", IdempotencyKey: "k1"}}, {Kind: "complete"}},
				}), nil
			}
			return model.NewFake(map[string][]model.ScriptStep{"*": {{Kind: "text", Text: "hi"}, {Kind: "complete"}}}), nil
		},
		Tools: &stubTools{block: block},
	}
}

func serveForTest(t *testing.T, dir string, block bool) (*Server, Client, context.CancelFunc) {
	t.Helper()
	return serveDepsForTest(t, dir, fakeDeps(block))
}

func serveDepsForTest(t *testing.T, dir string, deps Deps) (*Server, Client, context.CancelFunc) {
	t.Helper()
	sock := filepath.Join(dir, "agentd.sock")
	srv := New(sock, filepath.Join(dir, "store"), deps)
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan struct{})
	go func() {
		_ = srv.Serve(ctx)
		close(served)
	}()
	deadline := time.Now().Add(5 * time.Second)
	probe := Client{SocketPath: sock}
	for {
		if _, err := probe.Protocol(); err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			<-served
			t.Fatal("daemon did not come up")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cleanup := func() {
		cancel()
		select {
		case <-served:
		case <-time.After(2 * time.Second):
		}
	}
	return srv, Client{SocketPath: sock}, cleanup
}

func waitStatus(t *testing.T, c Client, runID, want string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		res, err := c.Status(runID)
		if err != nil {
			t.Fatalf("status failed: %v", err)
		}
		if res["ok"] == true && res["status"] == want {
			return res
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s, last: %v", want, res)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestDispatchCoversManifestOps pins every IDL op to a handler: adding an
// op to protocol.Ops without a dispatch branch fails here.
func TestDispatchCoversManifestOps(t *testing.T) {
	srv := New(t.TempDir()+"/s.sock", t.TempDir(), fakeDeps(false))
	for _, op := range harnessprotocol.Ops {
		res := srv.dispatch(map[string]any{"op": op})
		if res["ok"] == false && res["error"] == "unknown op" {
			t.Fatalf("op %q in manifest but unhandled", op)
		}
	}
	if res := srv.dispatch(map[string]any{"op": "nope"}); res["error"] != "unknown op" {
		t.Fatalf("unknown op must error: %v", res)
	}
	if res := srv.dispatch(map[string]any{"op": "protocol"}); res["ok"] != true || res["ops"] == nil {
		t.Fatalf("protocol op must serve ops: %v", res)
	}
}
func TestDaemonRunLifecycle(t *testing.T) {
	dir := t.TempDir()
	_, c, cancel := serveForTest(t, dir, false)
	defer cancel()
	start, err := c.Start("hello daemon", "fake", "R-d1", 2)
	if err != nil || start["ok"] != true {
		t.Fatalf("start failed: %v %v", err, start)
	}
	waitStatus(t, c, "R-d1", "complete")
	evs, err := c.Events("R-d1")
	if err != nil || evs["ok"] != true {
		t.Fatalf("events failed: %v %v", err, evs)
	}
	list, _ := evs["events"].([]any)
	if len(list) < 2 {
		t.Fatalf("expected lifecycle events, got %d", len(list))
	}
	foundCtx := false
	for _, e := range list {
		if m, ok := e.(map[string]any); ok && m["kind"] == "context.compiled" {
			foundCtx = true
		}
	}
	if !foundCtx {
		t.Fatal("expected context.compiled event from real v2 compilation")
	}
	lst, err := c.List()
	if err != nil || lst["ok"] != true {
		t.Fatalf("list failed: %v", err)
	}
	if _, err := c.Protocol(); err != nil {
		t.Fatalf("protocol failed: %v", err)
	}
	if _, err := c.Status("R-nope"); err == nil {
		// Status returns ok:false payload, not transport error.
		t.Log("unknown run handled at payload level")
	}
}

func TestDaemonCancel(t *testing.T) {
	dir := t.TempDir()
	_, c, cancel := serveForTest(t, dir, true)
	defer cancel()
	start, err := c.Start("blocking goal", "fake", "R-dcancel", 50)
	if err != nil || start["ok"] != true {
		t.Fatalf("start failed: %v %v", err, start)
	}
	time.Sleep(200 * time.Millisecond) // let the run reach the blocking tool
	res, err := c.Cancel("R-dcancel")
	if err != nil || res["ok"] != true {
		t.Fatalf("cancel failed: %v %v", err, res)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		st, err := c.Status("R-dcancel")
		if err != nil {
			t.Fatal(err)
		}
		if st["ok"] == true && st["status"] == "cancelled" {
			time.Sleep(50 * time.Millisecond)
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("cancel did not land, last: %v", st)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestDaemonReconnect(t *testing.T) {
	dir := t.TempDir()
	_, c, cancel := serveForTest(t, dir, false)
	if _, err := c.Start("persist me", "fake", "R-dre", 1); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, c, "R-dre", "complete")
	cancel()
	time.Sleep(100 * time.Millisecond)

	// New server over the same store: runs stay observable without memory.
	_, c2, cancel2 := serveForTest(t, dir, false)
	defer cancel2()
	st, err := c2.Status("R-dre")
	if err != nil || st["ok"] != true || st["status"] != "complete" {
		t.Fatalf("reconnect status failed: %v %v", err, st)
	}
	evs, err := c2.Events("R-dre")
	if err != nil || evs["ok"] != true {
		t.Fatalf("reconnect events failed: %v %v", err, evs)
	}
}

func TestDispatchSteer(t *testing.T) {
	srv := New(t.TempDir()+"/s.sock", t.TempDir(), fakeDeps(false))
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.runs["R-live"] = &activeRun{cancel: cancel, runner: harnessruntime.NewRunner(harnessruntime.Services{}, "R-live", "S")}
	if res := srv.dispatch(map[string]any{"op": "steer", "run_id": "R-live", "message": "pivot"}); res["ok"] != true {
		t.Fatalf("steer active run failed: %v", res)
	}
	if res := srv.dispatch(map[string]any{"op": "steer", "run_id": "R-gone", "message": "x"}); res["ok"] != false {
		t.Fatalf("steer inactive run must fail: %v", res)
	}
	if res := srv.dispatch(map[string]any{"op": "steer", "run_id": "R-live"}); res["ok"] != false {
		t.Fatalf("steer without message must fail: %v", res)
	}
}

// approvalDeps scripts one tool call whose kind requires approval, so the run
// stops at the permission gate exactly as a destructive edit would.
func approvalDeps(tools *stubTools) Deps {
	return Deps{
		NewProvider: func(name, baseURL, apiKey, mdl string) (model.Provider, error) {
			return model.NewFake(map[string][]model.ScriptStep{
				"*": {{Kind: "tool_call", Tool: &agent.ToolCall{ID: "c1", Name: "edit.delete", IdempotencyKey: "k1"}}, {Kind: "complete"}},
			}), nil
		},
		Tools:      tools,
		PermPolicy: perm.Policy{DefaultAction: agent.PermissionAllow, AskKinds: []string{"destructive"}},
	}
}

func pendingPermission(t *testing.T, st map[string]any) string {
	t.Helper()
	ids, _ := st["pending_permissions"].([]any)
	if len(ids) != 1 {
		t.Fatalf("status must name exactly one pending request: %v", st)
	}
	id, _ := ids[0].(string)
	return id
}

// TestDaemonApprovePermission is H10 criterion 2's mechanism: a run stops at a
// permission gate, a client answers over the protocol, and the run continues
// and finishes without the daemon being restarted.
func TestDaemonApprovePermission(t *testing.T) {
	dir := t.TempDir()
	tools := &stubTools{kinds: map[string]string{"edit.delete": "destructive"}}
	_, c, cancel := serveDepsForTest(t, dir, approvalDeps(tools))
	defer cancel()

	if _, err := c.Start("delete something", "fake", "R-approve", 1); err != nil {
		t.Fatal(err)
	}
	requestID := pendingPermission(t, waitStatus(t, c, "R-approve", "awaiting_approval"))

	res, err := c.Approve("R-approve", requestID)
	if err != nil || res["ok"] != true {
		t.Fatalf("approve failed: %v %v", err, res)
	}
	waitStatus(t, c, "R-approve", "complete")
	if got := tools.callCount(); got != 1 {
		t.Fatalf("approved tool executed %d times, want 1", got)
	}
	// The decision is on the audit trail, not only in memory.
	data, err := os.ReadFile(filepath.Join(dir, "store", "permissions-R-approve.jsonl"))
	if err != nil {
		t.Fatalf("permission trail missing: %v", err)
	}
	if !strings.Contains(string(data), "approved by client") {
		t.Fatalf("audit trail does not record the approval:\n%s", data)
	}
	if _, err := c.Events("R-approve"); err != nil {
		t.Fatal(err)
	}
	// A finished run is gone from the live map: answering it is not an option.
	if res, err := c.Approve("R-approve", requestID); err != nil || res["ok"] != false {
		t.Fatalf("approving an inactive run must fail: %v %v", err, res)
	}
}

func TestDaemonDenyPermission(t *testing.T) {
	dir := t.TempDir()
	tools := &stubTools{kinds: map[string]string{"edit.delete": "destructive"}}
	_, c, cancel := serveDepsForTest(t, dir, approvalDeps(tools))
	defer cancel()

	if _, err := c.Start("delete something", "fake", "R-deny", 1); err != nil {
		t.Fatal(err)
	}
	requestID := pendingPermission(t, waitStatus(t, c, "R-deny", "awaiting_approval"))

	res, err := c.Deny("R-deny", requestID, "outside the workspace")
	if err != nil || res["ok"] != true {
		t.Fatalf("deny failed: %v %v", err, res)
	}
	waitStatus(t, c, "R-deny", "failed")
	if got := tools.callCount(); got != 0 {
		t.Fatalf("a rejected tool must not execute, got %d calls", got)
	}
}

func TestPermissionOpValidatesItsArguments(t *testing.T) {
	srv := New(t.TempDir()+"/s.sock", t.TempDir(), fakeDeps(false))
	if res := srv.dispatch(map[string]any{"op": "approve"}); res["error"] != "run_id required" {
		t.Fatalf("approve without run_id must fail: %v", res)
	}
	if res := srv.dispatch(map[string]any{"op": "approve", "run_id": "R-x"}); res["error"] != "permission request required" {
		t.Fatalf("approve without a request id must fail: %v", res)
	}
	if res := srv.dispatch(map[string]any{"op": "approve", "run_id": "R-gone", "request_id": "perm-1"}); res["ok"] != false {
		t.Fatalf("approve on an inactive run must fail: %v", res)
	}
}
