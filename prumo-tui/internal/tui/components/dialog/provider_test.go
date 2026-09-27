package dialog

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/raillen/prumo-tui/internal/config"
)

func TestProviderDialogRendersAvailableProviders(t *testing.T) {
	d := NewProviderDialogCmp("fake")
	_ = d.Init()

	frame := plainFrame(d.View().Content)
	if !strings.Contains(frame, "Configure LLM Provider") {
		t.Fatalf("dialog header missing: %s", frame)
	}
	if !strings.Contains(frame, "OpenCode") {
		t.Fatalf("OpenCode provider missing: %s", frame)
	}
	if !strings.Contains(frame, "Anthropic Claude") {
		t.Fatalf("Anthropic provider missing: %s", frame)
	}
	if !strings.Contains(frame, "OpenAI Compatible") {
		t.Fatalf("OpenAI compatible provider missing: %s", frame)
	}
	if !strings.Contains(frame, "Fake") {
		t.Fatalf("Fake provider missing: %s", frame)
	}
	if !strings.Contains(frame, "(active)") {
		t.Fatalf("active indicator missing: %s", frame)
	}
}

func TestProviderDialogSelectionAndSwitching(t *testing.T) {
	config.Set(config.Config{WorkingDir: t.TempDir(), Provider: "fake"})
	t.Setenv("PRUMO_TUI_CONFIG_DIR", t.TempDir())

	d := NewProviderDialogCmp("fake")
	_ = d.Init()

	// Initial selection is on "fake"
	cmp := d.(*providerDialogCmp)
	if cmp.currentProvider != "fake" {
		t.Fatalf("expected initial provider 'fake', got %q", cmp.currentProvider)
	}

	// Move to first item (opencode)
	cmp.selectedIdx = 0
	_, cmd := d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected batch command on enter")
	}

	if cmp.currentProvider != "opencode" {
		t.Fatalf("expected current provider 'opencode', got %q", cmp.currentProvider)
	}
}
