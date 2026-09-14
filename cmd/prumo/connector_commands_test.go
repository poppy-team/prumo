package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConnectorCommands(t *testing.T) {
	// Setup isolated test environment
	tmpHome := t.TempDir()
	tmpProject := t.TempDir()
	t.Setenv("PRUMO_HOME", tmpHome)

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpProject); err != nil {
		t.Fatalf("failed to chdir to tmpProject: %v", err)
	}

	// 1. List connectors
	if code := runConnector(false, []string{"list"}); code != exitOK {
		t.Fatalf("runConnector list expected %d, got %d", exitOK, code)
	}
	if code := runConnector(true, []string{"list"}); code != exitOK {
		t.Fatalf("runConnector --json list expected %d, got %d", exitOK, code)
	}

	// 2. Validate before install -> should fail
	if code := runConnector(false, []string{"validate", "opencode"}); code == exitOK {
		t.Fatalf("runConnector validate expected failure before install, got %d", code)
	}

	// 3. Install opencode
	if code := runConnector(false, []string{"install", "opencode"}); code != exitOK {
		t.Fatalf("runConnector install opencode expected %d, got %d", exitOK, code)
	}

	// Verify .opencode files exist
	opencodeConfig := filepath.Join(tmpProject, ".opencode", "opencode.json")
	if _, err := os.Stat(opencodeConfig); err != nil {
		t.Fatalf("expected opencode.json at %s", opencodeConfig)
	}

	// 4. Validate after install -> should succeed
	if code := runConnector(false, []string{"validate", "opencode"}); code != exitOK {
		t.Fatalf("runConnector validate after install expected %d, got %d", exitOK, code)
	}
	if code := runConnector(true, []string{"validate", "opencode"}); code != exitOK {
		t.Fatalf("runConnector --json validate after install expected %d, got %d", exitOK, code)
	}

	// 5. Capability negotiation
	if code := runConnector(false, []string{"negotiate", "opencode"}); code != exitOK {
		t.Fatalf("runConnector negotiate opencode expected %d, got %d", exitOK, code)
	}
	if code := runConnector(false, []string{"negotiate", "gemini"}); code != exitOK {
		t.Fatalf("runConnector negotiate gemini expected %d, got %d", exitOK, code)
	}
	// Strict negotiation for unsupported capability on gemini should return error
	if code := runConnector(false, []string{"negotiate", "gemini", "--strict", "--caps", "pre_tool_block"}); code == exitOK {
		t.Fatalf("expected strict negotiation failure on gemini for pre_tool_block")
	}

	// 6. Uninstall opencode
	if code := runConnector(false, []string{"uninstall", "opencode"}); code != exitOK {
		t.Fatalf("runConnector uninstall opencode expected %d, got %d", exitOK, code)
	}

	// 7. Test backward-compatible 'prumo install connector opencode'
	if code := runInstall(false, tmpHome, []string{"connector", "opencode"}); code != exitOK {
		t.Fatalf("runInstall connector opencode expected %d, got %d", exitOK, code)
	}
	if code := runConnector(false, []string{"validate", "opencode"}); code != exitOK {
		t.Fatalf("runConnector validate after install connector expected %d, got %d", exitOK, code)
	}
	if code := runConnector(false, []string{"uninstall", "opencode"}); code != exitOK {
		t.Fatalf("runConnector final uninstall expected %d, got %d", exitOK, code)
	}

	// 8. Test Gemini, Claude Code, Codex, and Antigravity install/validate/uninstall
	for _, harness := range []string{"gemini", "claude-code", "codex", "antigravity"} {
		if code := runConnector(false, []string{"install", harness}); code != exitOK {
			t.Fatalf("install %s expected %d, got %d", harness, exitOK, code)
		}
		if code := runConnector(false, []string{"validate", harness}); code != exitOK {
			t.Fatalf("validate %s expected %d, got %d", harness, exitOK, code)
		}
		if code := runConnector(false, []string{"uninstall", harness}); code != exitOK {
			t.Fatalf("uninstall %s expected %d, got %d", harness, exitOK, code)
		}
	}

	// 9. Test direct 'prumo install antigravity' and 'prumo uninstall antigravity'
	if code := runInstall(false, tmpHome, []string{"antigravity"}); code != exitOK {
		t.Fatalf("direct runInstall antigravity expected %d, got %d", exitOK, code)
	}
	geminiContent, err := os.ReadFile("GEMINI.md")
	if err != nil {
		t.Fatalf("failed to read generated GEMINI.md: %v", err)
	}
	if !strings.Contains(string(geminiContent), "Treat Prumo as an external CLI utility available in PATH ('prumo')") {
		t.Fatalf("GEMINI.md missing black-box CLI directive, got:\n%s", string(geminiContent))
	}
	if code := runConnector(false, []string{"validate", "antigravity"}); code != exitOK {
		t.Fatalf("validate after direct install expected %d, got %d", exitOK, code)
	}
	if code := runUninstall(false, tmpHome, []string{"antigravity"}); code != exitOK {
		t.Fatalf("direct runUninstall antigravity expected %d, got %d", exitOK, code)
	}
}
