package chat

import (
	"strings"
	"testing"

	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/runtime"
	"github.com/raillen/prumo-tui/internal/session"
)

func TestSidebarRendersSessionAndShortcuts(t *testing.T) {
	appInst := app.New(app.Options{
		Provider:  "fake",
		Workspace: t.TempDir(),
	})

	sidebar := NewSidebarCmp(appInst)
	_ = sidebar.SetSize(35, 40)
	sidebar.UpdateSession(session.Session{
		ID:               "test-sess-1",
		Title:            "My Test Session",
		PromptTokens:     1500,
		CompletionTokens: 500,
		ReasoningTokens:  250,
		Cost:             0.005,
	})

	// Record a subagent
	appInst.Runner.RecordSubagent("test-sess-1", runtime.SubagentInfo{
		ID:     "sub-1",
		Role:   "researcher",
		Status: "running",
	})

	view := sidebar.View().Content
	if !strings.Contains(view, "SESSION STATUS") {
		t.Fatalf("expected SESSION STATUS in view:\n%s", view)
	}
	if !strings.Contains(view, "Effort:") {
		t.Fatalf("expected Effort in view:\n%s", view)
	}
	if !strings.Contains(view, "reasoning: 250") {
		t.Fatalf("expected reasoning tokens in view:\n%s", view)
	}
	if !strings.Contains(view, "My Test Session") {
		t.Fatalf("expected session title in view:\n%s", view)
	}
	if !strings.Contains(view, "CONTEXT TOKENS") {
		t.Fatalf("expected CONTEXT TOKENS in view:\n%s", view)
	}
	if !strings.Contains(view, "WORKFORCE") {
		t.Fatalf("expected WORKFORCE in view:\n%s", view)
	}
	if !strings.Contains(view, "researcher") {
		t.Fatalf("expected subagent researcher in view:\n%s", view)
	}
	if !strings.Contains(view, "CHANGED FILES") {
		t.Fatalf("expected CHANGED FILES in view:\n%s", view)
	}
	if !strings.Contains(view, "None in this run") {
		t.Fatalf("expected 'None in this run' in view:\n%s", view)
	}
	if !strings.Contains(view, "ctrl+b") {
		t.Fatalf("expected ctrl+b in view:\n%s", view)
	}
}
