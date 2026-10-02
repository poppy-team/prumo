package page

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/raillen/prumo-tui/internal/agent"
	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/config"
	"github.com/raillen/prumo-tui/internal/llm/models"
	"github.com/raillen/prumo-tui/internal/message"
	"github.com/raillen/prumo-tui/internal/permission"
	"github.com/raillen/prumo-tui/internal/pubsub"
	"github.com/raillen/prumo-tui/internal/session"
	"github.com/raillen/prumo-tui/internal/tui/components/chat"
	"github.com/raillen/prumo-tui/internal/tui/components/dialog"
)

func TestChatPageTabs(t *testing.T) {
	fakeApp := &app.App{
		Workspace: "/tmp/my-project",
		Sessions:  session.NewStore(),
	}

	model := NewChatPage(fakeApp)
	page, ok := model.(*chatPage)
	if !ok {
		t.Fatalf("expected *chatPage, got %T", model)
	}

	if len(page.tabs) != 1 {
		t.Fatalf("expected initial 1 tab, got %d", len(page.tabs))
	}

	// Create a new tab
	page.Update(chat.NewTabMsg{Title: "Refactor Auth"})
	if len(page.tabs) != 2 {
		t.Fatalf("expected 2 tabs after NewTabMsg, got %d", len(page.tabs))
	}
	if page.activeTabIndex != 1 {
		t.Fatalf("expected activeTabIndex 1, got %d", page.activeTabIndex)
	}
	if page.tabs[1].Title != "Refactor Auth" {
		t.Fatalf("expected tab title 'Refactor Auth', got %q", page.tabs[1].Title)
	}

	// Switch back to tab 0
	page.Update(chat.SwitchTabMsg{Index: 0})
	if page.activeTabIndex != 0 {
		t.Fatalf("expected activeTabIndex 0, got %d", page.activeTabIndex)
	}

	// Next tab
	page.Update(chat.NextTabMsg{})
	if page.activeTabIndex != 1 {
		t.Fatalf("expected activeTabIndex 1 after NextTab, got %d", page.activeTabIndex)
	}

	// Tab bar rendering
	page.SetSize(100, 30)
	view := page.View().Content
	if !strings.Contains(view, "1:") || !strings.Contains(view, "2: Refactor Auth") {
		t.Fatalf("expected tab bar to render tabs, got:\n%s", view)
	}

	// Close tab 1
	page.Update(chat.CloseTabMsg{Index: 1})
	if len(page.tabs) != 1 {
		t.Fatalf("expected 1 tab after CloseTabMsg, got %d", len(page.tabs))
	}
	if page.activeTabIndex != 0 {
		t.Fatalf("expected activeTabIndex 0, got %d", page.activeTabIndex)
	}
}

func TestChatPageSidebarPersistence(t *testing.T) {
	// Set config sidebar to true
	_ = config.UpdateSidebarVisibility(true)
	model := NewChatPage(&app.App{Workspace: "/tmp/test"})
	page := model.(*chatPage)
	if !page.showSidebar {
		t.Fatalf("expected showSidebar true from config, got false")
	}

	// Toggle sidebar off
	page.Update(chat.ToggleSidebarMsg{})
	if page.showSidebar {
		t.Fatalf("expected showSidebar false after toggle, got true")
	}
	if config.Get().ShowSidebar {
		t.Fatalf("expected config.Get().ShowSidebar to be false after toggle")
	}
}

func TestChatPageResponsiveSidebarDefault(t *testing.T) {
	_ = config.UpdateSidebarVisibility(false)
	model := NewChatPage(&app.App{Workspace: "/tmp/test"})
	page := model.(*chatPage)

	page.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	if page.showSidebar {
		t.Fatalf("expected sidebar hidden on narrow terminal, got visible")
	}

	page.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	if !page.showSidebar {
		t.Fatalf("expected sidebar visible on wide terminal by default, got hidden")
	}

	page.Update(chat.ToggleSidebarMsg{})
	if page.showSidebar {
		t.Fatalf("expected sidebar hidden after user toggle, got visible")
	}
	page.Update(tea.WindowSizeMsg{Width: 160, Height: 30})
	if page.showSidebar {
		t.Fatalf("expected user override to keep sidebar hidden on resize, got visible")
	}
}

func TestChatPageSidebarFocusRouting(t *testing.T) {
	_ = config.UpdateSidebarVisibility(true)
	model := NewChatPage(&app.App{Workspace: "/tmp/test"})
	page := model.(*chatPage)
	page.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	if !page.showSidebar {
		t.Fatalf("expected sidebar visible for focus test")
	}

	page.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !page.sidebarCmp.Focused() {
		t.Fatalf("expected tab to focus the sidebar")
	}

	page.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if !page.sidebarCmp.Focused() {
		t.Fatalf("expected sidebar to stay focused after navigation key")
	}

	page.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if page.sidebarCmp.Focused() {
		t.Fatalf("expected esc to return focus to the composer")
	}
}

type stubBusyAgent struct {
	agent.Service
	busy map[string]bool
	ran  *string
}

func (s stubBusyAgent) IsSessionBusy(id string) bool { return s.busy[id] }
func (s stubBusyAgent) Model() models.Model          { return models.Model{} }
func (s stubBusyAgent) Run(_ context.Context, sessionID, content string) (<-chan agent.AgentEvent, error) {
	if s.ran != nil {
		*s.ran = sessionID + ":" + content
	}
	out := make(chan agent.AgentEvent)
	close(out)
	return out, nil
}

func TestChatPageTabBadges(t *testing.T) {
	fakeApp := app.New(app.Options{Provider: "fake", Workspace: "/tmp/badges"})
	model := NewChatPage(fakeApp)
	page := model.(*chatPage)

	page.Update(chat.NewTabMsg{Title: "foreground"})
	page.Update(chat.SessionSelectedMsg(session.Session{ID: "s-active", Title: "foreground"}))
	page.tabs[0].ID = "s-background"

	if got := page.tabBadge(page.tabs[0]); got != "" {
		t.Fatalf("expected no badge on quiet tab, got %q", got)
	}

	page.Update(pubsub.Event[agent.AgentEvent]{
		Type:    pubsub.CreatedEvent,
		Payload: agent.AgentEvent{SessionID: "s-background", Done: true},
	})
	if !page.tabs[0].Unread {
		t.Fatalf("expected background tab marked unread on run completion")
	}
	if got := page.tabBadge(page.tabs[0]); got != " *" {
		t.Fatalf("expected unread badge, got %q", got)
	}

	page.Update(pubsub.Event[agent.AgentEvent]{
		Type:    pubsub.CreatedEvent,
		Payload: agent.AgentEvent{SessionID: "s-active", Done: true},
	})
	if page.tabs[1].Unread {
		t.Fatalf("active tab must never badge itself unread")
	}

	page.Update(pubsub.Event[message.Message]{
		Type:    pubsub.CreatedEvent,
		Payload: message.Message{SessionID: "s-background"},
	})
	if !page.tabs[0].Unread {
		t.Fatalf("expected background tab kept unread on message arrival")
	}

	page.Update(pubsub.Event[permission.PermissionRequest]{
		Type:    pubsub.CreatedEvent,
		Payload: permission.PermissionRequest{ID: "p1", SessionID: "s-background"},
	})
	if !page.tabs[0].NeedsApproval {
		t.Fatalf("expected background tab flagged for approval")
	}
	if got := page.tabBadge(page.tabs[0]); got != " !" {
		t.Fatalf("expected approval badge to win, got %q", got)
	}

	page.Update(dialog.PermissionResponseMsg{
		Action:     dialog.PermissionDeny,
		Permission: permission.PermissionRequest{ID: "p1", SessionID: "s-background"},
	})
	if page.tabs[0].NeedsApproval {
		t.Fatalf("expected approval flag cleared on response")
	}

	page.Update(chat.SwitchTabMsg{Index: 0})
	if page.tabs[0].Unread {
		t.Fatalf("expected unread cleared when tab becomes active")
	}
	if bar := page.renderTabBar(); strings.Contains(bar, "*") || strings.Contains(bar, "!") {
		t.Fatalf("expected no badges after visiting tab, got:\n%s", bar)
	}
}

func TestChatPageTabBusyBadge(t *testing.T) {
	fakeApp := app.New(app.Options{Provider: "fake", Workspace: "/tmp/badges"})
	model := NewChatPage(fakeApp)
	page := model.(*chatPage)
	page.app.CoderAgent = stubBusyAgent{busy: map[string]bool{"s-running": true}}

	page.Update(chat.NewTabMsg{Title: "worker"})
	page.Update(chat.SessionSelectedMsg(session.Session{ID: "s-running", Title: "worker"}))
	page.tabs[0].ID = "s-idle"

	if got := page.tabBadge(page.tabs[1]); got != " ●" {
		t.Fatalf("expected busy badge on running tab, got %q", got)
	}
	if got := page.tabBadge(page.tabs[0]); got != "" {
		t.Fatalf("expected no badge on idle tab, got %q", got)
	}
	if bar := page.renderTabBar(); !strings.Contains(bar, "●") {
		t.Fatalf("expected busy badge in tab bar, got:\n%s", bar)
	}
}

func TestChatPageRunBackground(t *testing.T) {
	fakeApp := app.New(app.Options{Provider: "fake", Workspace: "/tmp/bg"})
	model := NewChatPage(fakeApp)
	page := model.(*chatPage)
	var ran string
	page.app.CoderAgent = stubBusyAgent{ran: &ran}

	before := len(page.tabs)
	active := page.activeTabIndex
	_, cmd := page.Update(chat.RunBackgroundMsg{Goal: "summarize the repo", Title: "summary"})
	if cmd == nil {
		t.Fatal("expected cmd starting the background run")
	}
	if len(page.tabs) != before+1 {
		t.Fatalf("expected dormant tab appended, got %d tabs", len(page.tabs))
	}
	if page.activeTabIndex != active {
		t.Fatalf("background run must not steal the active tab")
	}
	tab := page.tabs[len(page.tabs)-1]
	if tab.Title != "summary" || tab.ID == "" {
		t.Fatalf("dormant tab missing session binding: %+v", tab)
	}
	msg := cmd()
	started, ok := msg.(chat.BackgroundStartedMsg)
	if !ok {
		t.Fatalf("expected BackgroundStartedMsg, got %T", msg)
	}
	if started.SessionID != tab.ID {
		t.Fatalf("started session %q does not match tab %q", started.SessionID, tab.ID)
	}
	if ran != tab.ID+":summarize the repo" {
		t.Fatalf("run reached the wrong session or goal: %q", ran)
	}

	_, warnCmd := page.Update(chat.RunBackgroundMsg{Goal: "   "})
	if warnCmd == nil {
		t.Fatal("expected usage warning on empty goal")
	}
	if len(page.tabs) != before+1 {
		t.Fatalf("empty goal must not append a tab")
	}
}

func TestChatPageTabIndependence(t *testing.T) {
	fakeApp := app.New(app.Options{
		Provider:  "fake",
		Workspace: "/tmp/main-repo",
	})

	model := NewChatPage(fakeApp)
	page := model.(*chatPage)

	// Tab 0 starts with app defaults
	if page.tabs[0].Provider != "fake" || page.tabs[0].Workspace != "/tmp/main-repo" {
		t.Fatalf("tab 0 unexpected initial state: %+v", page.tabs[0])
	}

	// Create Tab 1 with independent provider, effort, worktree
	page.Update(chat.NewTabMsg{
		Title:           "Feature Auth",
		Provider:        "anthropic",
		Model:           "claude-3-7-sonnet",
		ReasoningEffort: "high",
		Workspace:       "/tmp/main-repo/.prumo/worktrees/feat-auth",
	})

	if len(page.tabs) != 2 || page.activeTabIndex != 1 {
		t.Fatalf("expected 2 tabs, activeTabIndex 1, got %d", page.activeTabIndex)
	}

	// App should now reflect Tab 1's isolated configuration
	if fakeApp.CurrentProvider() != "anthropic" {
		t.Fatalf("expected provider 'anthropic' on tab 1, got %q", fakeApp.CurrentProvider())
	}
	if fakeApp.CurrentReasoningEffort() != "high" {
		t.Fatalf("expected effort 'high' on tab 1, got %q", fakeApp.CurrentReasoningEffort())
	}
	if fakeApp.CurrentWorkspace() != "/tmp/main-repo/.prumo/worktrees/feat-auth" {
		t.Fatalf("expected workspace on tab 1 to be worktree, got %q", fakeApp.CurrentWorkspace())
	}

	// Switch back to Tab 0
	page.Update(chat.SwitchTabMsg{Index: 0})
	if fakeApp.CurrentProvider() != "fake" {
		t.Fatalf("expected provider 'fake' after switching to tab 0, got %q", fakeApp.CurrentProvider())
	}
	if fakeApp.CurrentWorkspace() != "/tmp/main-repo" {
		t.Fatalf("expected workspace '/tmp/main-repo' after switching to tab 0, got %q", fakeApp.CurrentWorkspace())
	}

	// Switch forward to Tab 1 again
	page.Update(chat.SwitchTabMsg{Index: 1})
	if fakeApp.CurrentProvider() != "anthropic" || fakeApp.CurrentReasoningEffort() != "high" {
		t.Fatalf("expected tab 1 settings preserved on switch, got prov=%q effort=%q",
			fakeApp.CurrentProvider(), fakeApp.CurrentReasoningEffort())
	}
}
