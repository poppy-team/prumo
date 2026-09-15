package extagent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
// TestOpenCodeLiveSend exercises a real model turn (user-approved spend).
// Gates: PRUMO_LIVE_OPENCODE_URL (live server) + PRUMO_LIVE_OPENCODE_SEND=1
// (explicit spend approval). Server-side credentials supply the model;
// the turn must complete and Usage must become non-zero.
func TestOpenCodeLiveSend(t *testing.T) {
	url := os.Getenv("PRUMO_LIVE_OPENCODE_URL")
	if url == "" {
		t.Skip("set PRUMO_LIVE_OPENCODE_URL to run live opencode interop")
	}
	if os.Getenv("PRUMO_LIVE_OPENCODE_SEND") == "" {
		t.Skip("set PRUMO_LIVE_OPENCODE_SEND=1 to approve real model spend")
	}
	o := NewOpenCodeServer(url)
	// Free OpenRouter models can take minutes; the 30s default client would
	// cut the SSE drain mid-turn and the server would abort the message.
	o.Client = &http.Client{Timeout: 180 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()

	s, err := o.CreateSession(ctx, "R-live-send")
	if err != nil {
		t.Fatalf("live create failed: %v", err)
	}
	t.Cleanup(func() { _ = o.Close(context.Background(), s.ID) })

	events, err := o.Events(ctx, s.ID)
	if err != nil {
		t.Fatalf("live events failed: %v", err)
	}
	if err := o.Send(ctx, s.ID, "Reply with exactly: live-send-ok"); err != nil {
		t.Fatalf("live send failed: %v", err)
	}

	// Informational: assistant text arrival via Events. Part shapes vary
	// across server versions, so this is logged, not asserted.
	deadline := time.After(10 * time.Second)
	var assistantEvents int
collect:
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				break collect
			}
			if strings.Contains(ev.Kind, "message") {
				assistantEvents++
			}
		case <-deadline:
			break collect
		}
	}

	// Hard assertion: a completed real turn must record usage. Retry loop
	// tolerates transient client timeouts while the server finishes the turn;
	// only total exhaustion fails the test.
	var inTok, outTok int
	var cost float64
	deadlineUsage := time.Now().Add(15 * time.Second)
	var lastErr error
	for {
		u, err := o.Usage(ctx, s.ID)
		if err == nil {
			lastErr = nil
			inTok, outTok, cost = u.InputTokens, u.OutputTokens, u.CostUSD
			if inTok > 0 || outTok > 0 || cost > 0 {
				break
			}
		} else {
			lastErr = err
		}
		if time.Now().After(deadlineUsage) {
			if lastErr != nil {
				t.Fatalf("live usage after send kept failing: %v", lastErr)
			}
			t.Fatalf("usage after real send must be non-zero: in=%d out=%d cost=%f", inTok, outTok, cost)
		}
		time.Sleep(1 * time.Second)
	}
	t.Logf("live send ok: message_events=%d usage={in=%d out=%d cost=%f}", assistantEvents, inTok, outTok, cost)

	// Structural proof: the turn must have an assistant message that
	// completed (no abort/error), not merely accounted tokens.
	msgs, err := o.messages(ctx, s.ID)
	if err != nil {
		t.Fatalf("live messages after send failed: %v", err)
	}
	completed := false
	for _, m := range msgs {
		if m.Role != "assistant" || m.Aborted {
			continue
		}
		completed = true
	}
	if !completed {
		t.Fatalf("expected a completed assistant message after real send (got %d messages)", len(msgs))
	}
}

// messages fetches raw session messages from the live server (test-only
// probe; shapes tolerated per measured 1.18.30 API).
type liveMessage struct {
	Role    string `json:"role"`
	Aborted bool
}

func (o *OpenCodeServer) messages(ctx context.Context, sessionID string) ([]liveMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.BaseURL+"/session/"+sessionID+"/message", nil)
	if err != nil {
		return nil, err
	}
	resp, err := o.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET message: %s", resp.Status)
	}
	var raw []struct {
		Info struct {
			Role  string         `json:"role"`
			Error map[string]any `json:"error"`
		} `json:"info"`
		Role  string         `json:"role"`
		Error map[string]any `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]liveMessage, 0, len(raw))
	for _, m := range raw {
		role := m.Info.Role
		errData := m.Info.Error
		if role == "" {
			role = m.Role
			errData = m.Error
		}
		_, aborted := errData["name"]
		out = append(out, liveMessage{Role: role, Aborted: aborted})
	}
	return out, nil
}

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
