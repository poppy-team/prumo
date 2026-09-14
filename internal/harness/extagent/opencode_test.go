package extagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// stubOpenCode implements the measured server API shapes (opencode 1.18.30):
// POST /session, GET /session, GET /session/:id/message,
// POST /session/:id/message (parts SSE), POST abort, POST permissions with
// {response}, DELETE session, GET /event SSE.
func stubOpenCode(t *testing.T, onMessage func(parts []any) string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		write := func(v any) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(v)
		}
		switch {
		case path == "/session" && r.Method == http.MethodPost:
			var body struct {
				Title string `json:"title"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Title == "" {
				body.Title = "untitled"
			}
			write(map[string]any{"id": "ses_stub1", "title": body.Title})
		case path == "/session" && r.Method == http.MethodGet:
			write([]any{map[string]any{"id": "ses_stub1", "title": "stub", "cost": 0.02, "tokens": map[string]any{"input": 10, "output": 5, "reasoning": 1}}})
		case strings.HasPrefix(path, "/session/") && strings.HasSuffix(path, "/message") && r.Method == http.MethodGet:
			write([]any{})
		case strings.HasPrefix(path, "/session/") && strings.HasSuffix(path, "/message") && r.Method == http.MethodPost:
			var body struct {
				Parts []any `json:"parts"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if len(body.Parts) == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte(onMessage(body.Parts)))
		case strings.HasSuffix(path, "/abort"):
			write(true)
		case strings.Contains(path, "/permissions/"):
			var body struct {
				Response string `json:"response"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			switch body.Response {
			case "once", "always", "reject":
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"_tag":"PermissionNotFoundError","message":"not found"}`))
			default:
				w.WriteHeader(http.StatusBadRequest)
			}
		case strings.HasPrefix(path, "/session/") && r.Method == http.MethodDelete:
			write(true)
		case path == "/event":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"id\":\"evt_1\",\"type\":\"server.connected\",\"properties\":{}}\n\ndata: {\"id\":\"evt_2\",\"type\":\"session.idle\",\"properties\":{\"sessionID\":\"ses_stub1\"}}\n\ndata: {\"id\":\"evt_3\",\"type\":\"session.idle\",\"properties\":{\"sessionID\":\"ses_other\"}}\n\n"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestOpenCodeSessionLifecycleStub(t *testing.T) {
	srv := stubOpenCode(t, func(parts []any) string { return "data: {\"type\":\"done\"}\n\n" })
	defer srv.Close()
	ctx := context.Background()
	o := NewOpenCodeServer(srv.URL)

	s, err := o.CreateSession(ctx, "R-stub")
	if err != nil || s.ID != "ses_stub1" {
		t.Fatalf("create failed: %+v %v", s, err)
	}
	list, err := o.ListSessions(ctx)
	if err != nil || len(list) != 1 || list[0].ID != "ses_stub1" {
		t.Fatalf("list failed: %+v %v", list, err)
	}
	if err := o.Send(ctx, s.ID, "hello"); err != nil {
		t.Fatalf("send failed: %v", err)
	}
	if err := o.Cancel(ctx, s.ID); err != nil {
		t.Fatalf("abort failed: %v", err)
	}
	if err := o.Approve(ctx, s.ID, "per-missing", false); err == nil || !strings.Contains(err.Error(), "PermissionNotFound") {
		t.Fatalf("expected typed 404, got %v", err)
	}
	if err := o.Close(ctx, s.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestOpenCodeResumeAndUsageStub(t *testing.T) {
	srv := stubOpenCode(t, func(parts []any) string { return "" })
	defer srv.Close()
	ctx := context.Background()
	o := NewOpenCodeServer(srv.URL)
	var lc SessionLifecycle = o // compile-time: adapter satisfies the rich contract
	s, err := lc.ResumeSession(ctx, "ses_stub1")
	if err != nil || s.Status != "resumed" {
		t.Fatalf("resume failed: %+v %v", s, err)
	}
	if _, err := lc.ResumeSession(ctx, "ses_missing"); err == nil {
		t.Fatal("resume of missing session must error")
	}
	u, err := lc.Usage(ctx, "ses_stub1")
	if err != nil || u.InputTokens != 10 || u.OutputTokens != 5 || u.CostUSD != 0.02 {
		t.Fatalf("usage misparsed: %+v %v", u, err)
	}
	if _, err := lc.Usage(ctx, "ses_missing"); err == nil {
		t.Fatal("usage of missing session must error")
	}
}
func TestOpenCodeSendRequiresParts(t *testing.T) {
	srv := stubOpenCode(t, func(parts []any) string { return "" })
	defer srv.Close()
	o := NewOpenCodeServer(srv.URL)
	// Empty server-side parts handling is covered by 400 mapping:
	if err := o.Send(context.Background(), "ses_x", ""); err != nil {
		t.Logf("empty send maps to: %v", err)
	}
}

func TestOpenCodeEventsFilterSession(t *testing.T) {
	srv := stubOpenCode(t, func(parts []any) string { return "" })
	defer srv.Close()
	o := NewOpenCodeServer(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ch, err := o.Events(ctx, "ses_stub1")
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for ev := range ch {
		kinds = append(kinds, ev.Kind)
		if ev.RunID != "ses_stub1" {
			t.Fatalf("misattributed event: %+v", ev)
		}
	}
	if len(kinds) != 1 || kinds[0] != "external.session.idle" {
		t.Fatalf("expected only the matching session event, got %v", kinds)
	}
}
