package daemon

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/checkpoint"
)

// A record saying "running" was written by a process that no longer exists. The
// daemon started with an empty run map and read those records as they were, so
// status reported a run as in flight when nothing was running it, and a client
// had no way to tell except by waiting for something that would never arrive
// (GAP-164).

// writeRecord leaves a record behind, the way a run that died mid-flight would.
func writeRecord(t *testing.T, storeDir string, rec RunRecord) {
	t.Helper()
	path, err := (&Server{StoreDir: storeDir}).recordPath(rec.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(rec)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readRecord(t *testing.T, storeDir, runID string) RunRecord {
	t.Helper()
	path, err := (&Server{StoreDir: storeDir}).recordPath(runID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("record for %s: %v", runID, err)
	}
	var rec RunRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		t.Fatal(err)
	}
	return rec
}

func TestARunningRecordIsNotLeftClaimingToRun(t *testing.T) {
	store := t.TempDir()
	writeRecord(t, store, RunRecord{RunID: "R-ghost", Status: "running", Phase: "execute_tool"})

	srv := New("s.sock", store, Deps{Workspace: t.TempDir()})
	rec := readRecord(t, store, "R-ghost")
	if rec.Status == "running" {
		t.Fatal("a record written by a process that is gone must not keep saying running")
	}
	if rec.Status != "interrupted" {
		t.Fatalf("status = %q, want interrupted", rec.Status)
	}
	if !rec.Interrupted {
		t.Error("the record must be marked as reconciled rather than written by its run")
	}
	if rec.StopReason == "" {
		t.Error("an interrupted run must say why, so an operator is not left guessing")
	}

	// And the status op must not report it as active.
	st := srv.opStatus("R-ghost")
	if st["active"] != false {
		t.Fatalf("a run nobody is serving must not be active: %v", st)
	}
	if st["status"] == "running" {
		t.Fatalf("status must not report a dead run as running: %v", st)
	}
}

func TestATerminalRecordIsNotReinterpreted(t *testing.T) {
	// The reconnect contract is about replay, and replay reads the event log. A
	// finished run's record said everything it had to say.
	store := t.TempDir()
	for _, status := range []string{"complete", "failed", "cancelled"} {
		runID := "R-" + status
		writeRecord(t, store, RunRecord{RunID: runID, Status: status, StopReason: "as it was"})
		New("s.sock", store, Deps{Workspace: t.TempDir()})
		rec := readRecord(t, store, runID)
		if rec.Status != status {
			t.Fatalf("a %s run became %q at startup", status, rec.Status)
		}
		if rec.Interrupted {
			t.Fatalf("a %s run must not be marked interrupted", status)
		}
		if rec.StopReason != "as it was" {
			t.Fatalf("a %s run's stop reason was rewritten: %q", status, rec.StopReason)
		}
	}
}

func TestARunWaitingForApprovalSurvivesAsAnswerable(t *testing.T) {
	// The contract's step 6: after a reconnect, status reports awaiting_approval
	// and lists the pending request ids, so the client still knows what to answer.
	dir := t.TempDir()
	store := filepath.Join(dir, "store")
	tools := &stubTools{kinds: map[string]string{"edit.delete": "destructive"}}
	_, c, cancel := serveDepsForTest(t, dir, approvalDeps(tools))

	if _, err := c.Start("delete something", "fake", "R-keep", 1); err != nil {
		t.Fatal(err)
	}
	st := waitStatus(t, c, "R-keep", "awaiting_approval")
	requestID := pendingPermission(t, st)
	fingerprint := pendingFingerprint(t, st, requestID)
	// Stop the daemon without answering, the way a crash or a kill would.
	cancel()
	waitForServe(t, dir)

	// A new daemon over the same store.
	srv := New(filepath.Join(dir, "agentd.sock"), store, approvalDeps(tools))
	rec := readRecord(t, store, "R-keep")
	if rec.Status != "awaiting_approval" {
		t.Fatalf("a run that was waiting must still be waiting, got %q", rec.Status)
	}
	if len(rec.PendingPermissions) != 1 || rec.PendingPermissions[0] != requestID {
		t.Fatalf("the pending request must survive the restart so a client can answer it: %v", rec.PendingPermissions)
	}
	if rec.StopReason == "" {
		t.Error("a recovered run must say it was recovered")
	}

	// And the status op must surface it, not hide it behind active:false alone.
	after := srv.opStatus("R-keep")
	if after["status"] != "awaiting_approval" {
		t.Fatalf("status after restart = %v, want awaiting_approval", after)
	}
	ids, _ := after["pending_permissions"].([]string)
	if len(ids) != 1 || ids[0] != requestID {
		t.Fatalf("status must name the pending request after a restart: %v", after["pending_permissions"])
	}
	_ = fingerprint
}

func TestARecoveredRunCanActuallyBeAnswered(t *testing.T) {
	// Answerable is not the same as answered. The recovered run must reach the
	// engine, or the pending id is decoration.
	dir := t.TempDir()
	store := filepath.Join(dir, "store")
	tools := &stubTools{kinds: map[string]string{"edit.delete": "destructive"}}
	_, c, cancel := serveDepsForTest(t, dir, approvalDeps(tools))
	if _, err := c.Start("delete something", "fake", "R-answer", 1); err != nil {
		t.Fatal(err)
	}
	st := waitStatus(t, c, "R-answer", "awaiting_approval")
	requestID := pendingPermission(t, st)
	fingerprint := pendingFingerprint(t, st, requestID)
	cancel()
	waitForServe(t, dir)

	srv := New(filepath.Join(dir, "agentd.sock"), store, approvalDeps(tools))
	// Rebuild the runner the way a resume would, from the checkpoint the
	// reconciliation found.
	resumed := srv.restoreRun("R-answer")
	if resumed == nil {
		t.Fatal("a run waiting for approval must be restorable from its checkpoint")
	}
	if err := resumed.ResolvePermission(requestID, fingerprint, true, "operator", ""); err != nil {
		t.Fatalf("answering a recovered run must work: %v", err)
	}
	// Answering rewinds the run to re-evaluate the request; the loop still has
	// to be driven, which is what the daemon's observe does.
	if err := resumed.RunUntilDone(context.Background()); err != nil {
		t.Fatalf("the recovered run must continue after approval: %v", err)
	}
	if got := tools.callCount(); got != 1 {
		t.Fatalf("the approved tool must run exactly once, got %d calls", got)
	}
}

func TestACrashedRunIsResumableButNotResumedOnItsOwn(t *testing.T) {
	// A graceful shutdown drains, so a record still saying "running" after one is
	// a process that did not get to drain: a crash, a kill -9, a power loss. That
	// is the case worth handling, and it is the one a record seeded by hand
	// reproduces exactly.
	dir := t.TempDir()
	store := filepath.Join(dir, "store")
	tools := &slowTools{}
	deps := toolCallingDeps(tools, dir)

	// Leave behind what a crash leaves: a run record mid-flight and the
	// checkpoint it had reached.
	cpStore := checkpoint.New(filepath.Join(store, "checkpoints"))
	cp := agent.Checkpoint{
		ID: "cp-R-crash", RunID: "R-crash",
		CreatedAt: agent.Now(),
		State: agent.NativeAgentState{
			RunID: "R-crash", SessionID: "S1", TurnID: "turn-1",
			Phase: agent.PhaseExecuteTool, Revision: 3,
		},
	}
	if err := cpStore.Save(cp); err != nil {
		t.Fatal(err)
	}
	writeRecord(t, store, RunRecord{RunID: "R-crash", Status: "running", Phase: "execute_tool"})

	srv := New(filepath.Join(dir, "agentd.sock"), store, deps)
	rec := readRecord(t, store, "R-crash")
	if rec.Status != "interrupted" {
		t.Fatalf("a crashed run must be interrupted, got %q", rec.Status)
	}
	if rec.ResumeCheckpoint == "" {
		t.Fatal("an interrupted run must name the checkpoint to resume from")
	}
	if rec.StopReason == "" {
		t.Error("an interrupted run must say why")
	}
	// The point of not resuming: continuing spends money nobody authorised.
	if tools.calls() != 0 {
		t.Fatalf("the run must not be re-executed on startup; %d calls", tools.calls())
	}
	// And the client can see all of that from status.
	st := srv.opStatus("R-crash")
	if st["status"] != "interrupted" {
		t.Fatalf("status must report the interruption: %v", st)
	}
	if st["resume_checkpoint"] == "" {
		t.Fatalf("status must carry the resume point: %v", st)
	}
	if st["interrupted"] != true {
		t.Fatalf("status must mark the record as reconciled: %v", st)
	}
}

func TestReconciliationIsIdempotent(t *testing.T) {
	// New runs the reconciliation; a second daemon over the same store must not
	// rewrite a record the first one already reconciled.
	store := t.TempDir()
	writeRecord(t, store, RunRecord{RunID: "R-twice", Status: "running", StopReason: "original"})
	New("s.sock", store, Deps{Workspace: t.TempDir()})
	first := readRecord(t, store, "R-twice")
	New("s.sock", store, Deps{Workspace: t.TempDir()})
	second := readRecord(t, store, "R-twice")
	if first.Status != second.Status || first.StopReason != second.StopReason {
		t.Fatalf("reconciliation is not idempotent:\n first:  %+v\n second: %+v", first, second)
	}
	if second.Status == "running" {
		t.Fatal("a second pass must not resurrect a dead run")
	}
}

// waitForTerminalRecord waits for the drain to finish writing a run's record.
func waitForTerminalRecord(t *testing.T, storeDir, runID string) {
	t.Helper()
	deadline := time.Now().Add(drainGrace + cancelGrace + 10*time.Second)
	for time.Now().Before(deadline) {
		if rec := readRecordOrZero(t, storeDir, runID); rec.Status == "cancelled" || rec.Status == "failed" || rec.Status == "complete" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("run %s never reached a terminal state", runID)
}

func readRecordOrZero(t *testing.T, storeDir, runID string) RunRecord {
	t.Helper()
	path, err := (&Server{StoreDir: storeDir}).recordPath(runID)
	if err != nil {
		return RunRecord{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return RunRecord{}
	}
	var rec RunRecord
	_ = json.Unmarshal(data, &rec)
	return rec
}

func waitForServe(t *testing.T, dir string) {
	t.Helper()
	// The socket is removed on the way out; waiting for it to go keeps a second
	// server from racing the first one's cleanup.
	sock := filepath.Join(dir, "agentd.sock")
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(sock); os.IsNotExist(err) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
