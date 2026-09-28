package page

import (
	"strings"
	"testing"

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
