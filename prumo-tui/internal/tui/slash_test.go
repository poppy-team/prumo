package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/config"
	"github.com/raillen/prumo-tui/internal/tui/components/chat"
	"github.com/raillen/prumo-tui/internal/tui/page"
	"github.com/raillen/prumo-tui/internal/tui/util"
)

func newTestApp(t *testing.T) (*app.App, *appModel) {
	t.Helper()
	config.Set(config.Config{WorkingDir: t.TempDir(), Provider: "fake", Onboarded: true})
	t.Setenv("PRUMO_TUI_CONFIG_DIR", t.TempDir())

	application := app.New(app.Options{
		Provider:  "fake",
		Workspace: t.TempDir(),
	})
	m := New(application).(*appModel)
	_ = m.Init()
	m.width = 100
	m.height = 30
	return application, m
}

func TestSlashCommandHelp(t *testing.T) {
	_, m := newTestApp(t)

	if m.showHelp {
		t.Fatal("expected help to be hidden initially")
	}

	updated, _ := m.Update(chat.SendMsg{Text: "/help"})
	model := updated.(appModel)
	if !model.showHelp {
		t.Fatal("expected /help to open help dialog")
	}
}

func TestSlashCommandProvider(t *testing.T) {
	appInst, m := newTestApp(t)

	// Direct switch via /provider opencode
	updated, _ := m.Update(chat.SendMsg{Text: "/provider opencode"})
	model := updated.(appModel)
	if appInst.CurrentProvider() != "opencode" {
		t.Fatalf("expected provider 'opencode', got %q", appInst.CurrentProvider())
	}

	// Open dialog via /provider
	updated, _ = model.Update(chat.SendMsg{Text: "/provider"})
	model = updated.(appModel)
	if !model.showProviderDialog {
		t.Fatal("expected /provider with no args to open provider dialog")
	}
}

func TestSlashCommandModel(t *testing.T) {
	_, m := newTestApp(t)

	// Direct switch via /model
	updated, _ := m.Update(chat.SendMsg{Text: "/model test-model"})
	model := updated.(appModel)
	if m.app.CoderAgent.Model().Name != "test-model" {
		t.Fatalf("expected model 'test-model', got %q", m.app.CoderAgent.Model().Name)
	}

	// Open dialog via /models
	updated, _ = model.Update(chat.SendMsg{Text: "/models"})
	model = updated.(appModel)
	if !model.showModelDialog {
		t.Fatal("expected /models to open models dialog")
	}
}

func TestSlashCommandSidebarToggle(t *testing.T) {
	_, m := newTestApp(t)

	// Toggle sidebar via /sidebar command
	updated, cmd := m.Update(chat.SendMsg{Text: "/sidebar"})
	if cmd == nil {
		t.Fatal("expected command from /sidebar")
	}

	// Route ToggleSidebarMsg to chat page
	chatP := m.pages[page.ChatPage]
	updatedChat, _ := chatP.Update(chat.ToggleSidebarMsg{})
	_ = updatedChat
	_ = updated
}

func TestSlashCommandUnknown(t *testing.T) {
	_, m := newTestApp(t)

	updated, cmd := m.Update(chat.SendMsg{Text: "/unknowncmd"})
	_ = updated
	if cmd == nil {
		t.Fatal("expected command for unknown slash command")
	}
	msg := cmd()
	infoMsg, ok := msg.(util.InfoMsg)
	if !ok || !strings.Contains(infoMsg.Msg, "Unknown command") {
		t.Fatalf("expected warning about unknown command, got: %v", msg)
	}
}

func TestSlashCommandNewSession(t *testing.T) {
	_, m := newTestApp(t)
	m.selectedSession.ID = "session-123"

	updated, _ := m.Update(chat.SendMsg{Text: "/new"})
	model := updated.(appModel)
	if model.selectedSession.ID != "" {
		t.Fatalf("expected empty session ID, got %q", model.selectedSession.ID)
	}
}

func TestCtrlBTogglesSidebar(t *testing.T) {
	_, m := newTestApp(t)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("expected cmd on ctrl+b")
	}
	msg := cmd()
	if _, ok := msg.(chat.ToggleSidebarMsg); !ok {
		t.Fatalf("expected ToggleSidebarMsg on ctrl+b, got: %T", msg)
	}
	_ = updated
}

func TestShellPrefixCommand(t *testing.T) {
	_, m := newTestApp(t)

	updated, cmd := m.Update(chat.SendMsg{Text: "!echo 'prumo-tui-shell-test'"})
	_ = updated
	if cmd == nil {
		t.Fatal("expected cmd on !echo command")
	}
	msg := cmd()
	infoMsg, ok := msg.(util.InfoMsg)
	if !ok || !strings.Contains(infoMsg.Msg, "prumo-tui-shell-test") {
		t.Fatalf("expected info message with command output, got: %v", msg)
	}
}

func TestShellPrefixEmpty(t *testing.T) {
	_, m := newTestApp(t)

	updated, cmd := m.Update(chat.SendMsg{Text: "!"})
	_ = updated
	if cmd == nil {
		t.Fatal("expected cmd on empty ! command")
	}
	msg := cmd()
	infoMsg, ok := msg.(util.InfoMsg)
	if !ok || !strings.Contains(infoMsg.Msg, "No shell command provided") {
		t.Fatalf("expected warning about empty shell command, got: %v", msg)
	}
}

func TestSlashCommandUndo(t *testing.T) {
	_, m := newTestApp(t)

	updated, cmd := m.Update(chat.SendMsg{Text: "/undo"})
	_ = updated
	if cmd == nil {
		t.Fatal("expected cmd on /undo command")
	}
	// /undo will attempt git reset in test tempdir and return info or failure Msg
	msg := cmd()
	if msg == nil {
		t.Fatal("expected non-nil response for /undo")
	}
}

func TestSlashCommandThink(t *testing.T) {
	_, m := newTestApp(t)

	updated, cmd := m.Update(chat.SendMsg{Text: "/think"})
	_ = updated
	if cmd == nil {
		t.Fatal("expected cmd on /think command")
	}
}

func TestSlashCommandDirty(t *testing.T) {
	_, m := newTestApp(t)

	updated, cmd := m.Update(chat.SendMsg{Text: "/dirty"})
	_ = updated
	if cmd == nil {
		t.Fatal("expected cmd on /dirty command")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("expected non-nil response for /dirty")
	}
}

func TestSlashCommandCommit(t *testing.T) {
	_, m := newTestApp(t)

	// Without message: usage warning
	updated, cmd := m.Update(chat.SendMsg{Text: "/commit"})
	_ = updated
	if cmd == nil {
		t.Fatal("expected cmd on empty /commit")
	}
	msg := cmd()
	infoMsg, ok := msg.(util.InfoMsg)
	if !ok || infoMsg.Type != util.InfoTypeWarn {
		t.Fatalf("expected InfoTypeWarn for empty /commit, got: %v", msg)
	}

	// With message: attempts git commit in workspace
	_, cmd = m.Update(chat.SendMsg{Text: "/commit feat(test): sample commit"})
	if cmd == nil {
		t.Fatal("expected cmd on /commit with message")
	}
}

func TestSlashCommandWeb(t *testing.T) {
	_, m := newTestApp(t)

	// Without URL: usage warning
	updated, cmd := m.Update(chat.SendMsg{Text: "/web"})
	_ = updated
	if cmd == nil {
		t.Fatal("expected cmd on empty /web")
	}
	msg := cmd()
	infoMsg, ok := msg.(util.InfoMsg)
	if !ok || infoMsg.Type != util.InfoTypeWarn {
		t.Fatalf("expected InfoTypeWarn for empty /web, got: %v", msg)
	}

	// With URL: triggers async fetch
	_, cmd = m.Update(chat.SendMsg{Text: "/web https://example.com"})
	if cmd == nil {
		t.Fatal("expected cmd on /web with url")
	}
}
