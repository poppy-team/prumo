package chat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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

func TestSidebarGoalsAccordionAndNavigation(t *testing.T) {
	ws := t.TempDir()
	goalsDir := filepath.Join(ws, ".ai", "goals", "p01")
	if err := os.MkdirAll(goalsDir, 0755); err != nil {
		t.Fatal(err)
	}

	g1 := `{
		"id": "P01-G01",
		"title": "Goal One",
		"phase": "P01",
		"state": "PLANNED",
		"acceptance": ["Crit A"],
		"constraints": ["Const A"],
		"gates": {"gateA": "required"}
	}`
	g2 := `{
		"id": "P01-G02",
		"title": "Goal Two Active",
		"phase": "P01",
		"state": "EXECUTING",
		"acceptance": ["Crit B"],
		"constraints": ["Const B"],
		"gates": {"gateB": "optional"}
	}`

	if err := os.WriteFile(filepath.Join(goalsDir, "P01-G01.goal.json"), []byte(g1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goalsDir, "P01-G02.goal.json"), []byte(g2), 0644); err != nil {
		t.Fatal(err)
	}

	appInst := app.New(app.Options{
		Provider:  "fake",
		Workspace: ws,
	})

	sidebar := NewSidebarCmp(appInst)
	_ = sidebar.SetSize(40, 50)

	view := sidebar.View().Content
	if !strings.Contains(view, "GOALS") {
		t.Fatalf("expected GOALS in view:\n%s", view)
	}

	idxG2 := strings.Index(view, "P01-G02 [EXECUTING]")
	idxG1 := strings.Index(view, "P01-G01 [PLANNED]")
	if idxG2 == -1 || idxG1 == -1 || idxG2 > idxG1 {
		t.Fatalf("expected active goal P01-G02 before P01-G01, got:\n%s", view)
	}

	if !strings.Contains(view, "Crit B") || strings.Contains(view, "[x]") || strings.Contains(view, "[ ]") {
		t.Fatalf("expected active goal expanded with clean criteria (no checkmarks), got:\n%s", view)
	}
	if strings.Contains(view, "Crit A") {
		t.Fatalf("expected inactive goal P01-G01 collapsed by default, got:\n%s", view)
	}

	sidebar.Focus()
	if !sidebar.Focused() {
		t.Fatalf("expected sidebar to be focused")
	}

	// Move down and expand P01-G01
	for i := 0; i < 4; i++ {
		_, _ = sidebar.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	_, _ = sidebar.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	viewExpanded := sidebar.View().Content
	if !strings.Contains(viewExpanded, "Crit A") {
		t.Fatalf("expected P01-G01 to expand independently on Enter, got:\n%s", viewExpanded)
	}
	if !strings.Contains(viewExpanded, "Crit B") {
		t.Fatalf("expected P01-G02 to remain expanded, got:\n%s", viewExpanded)
	}

	sidebar.Blur()
	if sidebar.Focused() {
		t.Fatalf("expected sidebar to be blurred")
	}
}
