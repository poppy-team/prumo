package extagent

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/handoff"
)

// Live interop against a real `opencode serve` (PRUMO_LIVE_OPENCODE_URL).
// No model invocation: lifecycle only (create/list/messages/abort/
// permissions-shape/delete). Message send stays stub-tested until the user
// approves model spend.
func liveURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("PRUMO_LIVE_OPENCODE_URL")
	if url == "" {
		t.Skip("set PRUMO_LIVE_OPENCODE_URL to run live opencode interop")
	}
	return url
}

func TestOpenCodeLiveLifecycle(t *testing.T) {
	o := NewOpenCodeServer(liveURL(t))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s, err := o.CreateSession(ctx, "R-live-lifecycle")
	if err != nil || !strings.HasPrefix(s.ID, "ses_") {
		t.Fatalf("live create failed: %+v %v", s, err)
	}
	t.Cleanup(func() { _ = o.Close(context.Background(), s.ID) })

	list, err := o.ListSessions(ctx)
	if err != nil {
		t.Fatalf("live list failed: %v", err)
	}
	found := false
	for _, item := range list {
		if item.ID == s.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("created session missing from live list")
	}
	if err := o.Cancel(ctx, s.ID); err != nil {
		t.Fatalf("live abort failed: %v", err)
	}
	if err := o.Approve(ctx, s.ID, "per-missing", false); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("expected typed 404 on unknown permission, got %v", err)
	}
	if err := o.Close(ctx, s.ID); err != nil {
		t.Fatalf("live delete failed: %v", err)
	}
}

func TestOpenCodeLiveResumeAndUsage(t *testing.T) {
	o := NewOpenCodeServer(liveURL(t))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s, err := o.CreateSession(ctx, "R-live-resume")
	if err != nil {
		t.Fatalf("live create failed: %v", err)
	}
	t.Cleanup(func() { _ = o.Close(context.Background(), s.ID) })

	var lc SessionLifecycle = o
	rs, err := lc.ResumeSession(ctx, s.ID)
	if err != nil || rs.Status != "resumed" {
		t.Fatalf("live resume failed: %+v %v", rs, err)
	}
	u, err := lc.Usage(ctx, s.ID)
	if err != nil {
		t.Fatalf("live usage failed: %v", err)
	}
	t.Logf("fresh session usage: %+v", u)
	if err := o.Close(ctx, s.ID); err != nil {
		t.Fatalf("live delete failed: %v", err)
	}
	if _, err := lc.ResumeSession(ctx, s.ID); err == nil {
		t.Fatal("resume after delete must error")
	}
}

// TestHandoffLiveAttach dogfoods HA8 against the live server: a typed
// Prumo Handoff bundle becomes an external session titled with the handoff
// id, then the session is aborted and deleted. No transcript crosses.
func TestHandoffLiveAttach(t *testing.T) {
	o := NewOpenCodeServer(liveURL(t))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	b, err := handoff.Build("native", "opencode",
		agent.NativeAgentState{RunID: "R-live-handoff", ContextManifestID: "ctx-live"}, "rev-live", "live attach", nil)
	if err != nil || b.Validate() != nil {
		t.Fatal("handoff must validate")
	}
	s, err := o.CreateSession(ctx, b.Handoff.ID)
	if err != nil {
		t.Fatalf("attach failed: %v", err)
	}
	t.Cleanup(func() { _ = o.Close(context.Background(), s.ID) })
	if err := o.Cancel(ctx, s.ID); err != nil {
		t.Fatalf("abort after attach failed: %v", err)
	}
	if err := o.Close(ctx, s.ID); err != nil {
		t.Fatalf("delete after attach failed: %v", err)
	}
}
