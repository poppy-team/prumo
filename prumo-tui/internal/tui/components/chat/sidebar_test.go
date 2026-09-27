package chat

import (
	"strings"
	"testing"

	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/session"
)

func TestSidebarRendersSessionAndShortcuts(t *testing.T) {
	appInst := app.New(app.Options{
		Provider:  "fake",
		Workspace: t.TempDir(),
	})

	sidebar := NewSidebarCmp(appInst)
	_ = sidebar.SetSize(35, 25)
	sidebar.UpdateSession(session.Session{
		ID:    "test-sess-1",
		Title: "My Test Session",
	})

	view := sidebar.View().Content
	if !strings.Contains(view, "SESSION STATUS") {
		t.Fatalf("expected SESSION STATUS in view:\n%s", view)
	}
	if !strings.Contains(view, "My Test Session") {
		t.Fatalf("expected session title in view:\n%s", view)
	}
	if !strings.Contains(view, "CHANGED FILES") {
		t.Fatalf("expected CHANGED FILES in view:\n%s", view)
	}
	if !strings.Contains(view, "None in this run") {
		t.Fatalf("expected 'None in this run' in view:\n%s", view)
	}
	if !strings.Contains(view, "QUICK SHORTCUTS") {
		t.Fatalf("expected QUICK SHORTCUTS in view:\n%s", view)
	}
	if !strings.Contains(view, "ctrl+b") {
		t.Fatalf("expected ctrl+b in view:\n%s", view)
	}
}
