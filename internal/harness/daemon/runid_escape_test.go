package daemon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

// A run id is accepted straight from a request and then becomes a filename.
// Before the fix, a value carrying a separator reached filepath.Join untouched
// and wrote outside the store (GAP-112).

func TestRunIDCannotEscapeTheStore(t *testing.T) {
	store := t.TempDir()
	outside := t.TempDir()
	srv := New("test.sock", store, Deps{Workspace: t.TempDir()})

	for _, runID := range []string{
		"../../../etc/cron.d/evil",
		"..",
		"a/b",
		"/absolute",
		".",
	} {
		srv.saveRecord(RunRecord{RunID: runID, Status: "running"})
		srv.appendEvent(runID, agent.AgentEvent{RunID: runID, Kind: "test"})

		if _, err := os.Stat(filepath.Join(outside, "cron.d")); err == nil {
			t.Fatal("a write escaped the store")
		}
		entries, err := os.ReadDir(outside)
		if err != nil {
			t.Fatalf("read outside dir: %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("run id %q created %v outside the store", runID, entries)
		}
	}
}

func TestRunIDIsRejectedAtTheRequestBoundary(t *testing.T) {
	// A caller that supplies its own id must learn that it is unusable, rather
	// than getting a run that starts and then records nothing.
	srv := New("test.sock", t.TempDir(), Deps{Workspace: t.TempDir()})
	res := srv.opStart(map[string]any{
		"op":     "run.start",
		"goal":   "do something",
		"run_id": "../escape",
	})
	okFlag, _ := res["ok"].(bool)
	if okFlag {
		t.Fatalf("a run id that escapes the store must be refused at the boundary: %v", res)
	}
}

func TestOrdinaryRunIDStillRoundTrips(t *testing.T) {
	store := t.TempDir()
	srv := New("test.sock", store, Deps{Workspace: t.TempDir()})
	srv.saveRecord(RunRecord{RunID: "R-daemon-1", Status: "running"})
	srv.appendEvent("R-daemon-1", agent.AgentEvent{RunID: "R-daemon-1", Kind: "run.started"})

	if _, err := os.Stat(filepath.Join(store, "daemon-run-R-daemon-1.json")); err != nil {
		t.Fatalf("an ordinary run id must still be recorded: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store, "events-R-daemon-1.jsonl")); err != nil {
		t.Fatalf("an ordinary run id must still get its event log: %v", err)
	}
	rec := srv.opStatus("R-daemon-1")
	if rec["status"] != "running" {
		t.Fatalf("unexpected record: %v", rec)
	}
}

// The base URL reaches the provider factory straight from a request. With the
// default factory the destination check now runs, so a client cannot aim the
// daemon at the cloud metadata endpoint (GAP-111).

func TestStartRunRefusesAMetadataBaseURL(t *testing.T) {
	srv := New("test.sock", t.TempDir(), Deps{Workspace: t.TempDir()})
	for _, baseURL := range []string{
		"https://169.254.169.254/latest/meta-data/",
		"https://10.0.0.5/v1",
		"http://127.0.0.1:11434/v1",
	} {
		res := srv.opStart(map[string]any{
			"op":       "run.start",
			"goal":     "read the instance credentials",
			"provider": "openai-compat",
			"base_url": baseURL,
			"api_key":  "k",
			"model":    "m",
		})
		okFlag, _ := res["ok"].(bool)
		if okFlag {
			t.Fatalf("base URL %s must be refused by the daemon", baseURL)
		}
	}
}

// The fingerprint is what makes an approval mean something. An approval quoted
// for different content must be refused, not applied to whatever happens to be
// pending (GAP-107).

func TestApproveRefusesAFingerprintForDifferentContent(t *testing.T) {
	dir := t.TempDir()
	tools := &stubTools{kinds: map[string]string{"edit.delete": "destructive"}}
	_, c, cancel := serveDepsForTest(t, dir, approvalDeps(tools))
	defer cancel()

	if _, err := c.Start("delete something", "fake", "R-fp", 1); err != nil {
		t.Fatal(err)
	}
	st := waitStatus(t, c, "R-fp", "awaiting_approval")
	requestID := pendingPermission(t, st)

	// A fingerprint for content the approver was never shown.
	res, err := c.Approve("R-fp", requestID, "0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatal(err)
	}
	if res["ok"] != false {
		t.Fatalf("a mismatched fingerprint must be refused, got %v", res)
	}
	if got := tools.callCount(); got != 0 {
		t.Fatalf("the tool must not run after a refused approval, got %d calls", got)
	}
	// The real fingerprint still works: the refusal was about the content, not
	// about breaking the flow.
	fingerprint := pendingFingerprint(t, st, requestID)
	if res, err := c.Approve("R-fp", requestID, fingerprint); err != nil || res["ok"] != true {
		t.Fatalf("the matching fingerprint must be accepted: %v %v", err, res)
	}
	waitStatus(t, c, "R-fp", "complete")
	if got := tools.callCount(); got != 1 {
		t.Fatalf("the approved tool must run exactly once, got %d", got)
	}
}
