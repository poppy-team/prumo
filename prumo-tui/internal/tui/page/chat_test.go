package page

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/config"
	"github.com/raillen/prumo-tui/internal/session"
	"github.com/raillen/prumo-tui/internal/tui/components/chat"
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
