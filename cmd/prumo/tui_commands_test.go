package main

import (
	"strings"
	"testing"
)

// TestTuiBinaryResolution pins how `prumo tui` finds the client.
//
// The client is a separate binary, so the failure that matters is the one where
// a user has none: the message has to say what to install instead of pretending
// the command does not exist.
func TestTuiBinaryResolution(t *testing.T) {
	t.Setenv("PRUMO_TUI_BIN", "/opt/prumo/prumo-tui")
	if got, err := tuiBinary(); err != nil || got != "/opt/prumo/prumo-tui" {
		t.Fatalf("PRUMO_TUI_BIN was ignored: %q %v", got, err)
	}

	// No override, nothing beside the test binary, nothing on PATH.
	t.Setenv("PRUMO_TUI_BIN", "")
	t.Setenv("PATH", "")
	_, err := tuiBinary()
	if err == nil {
		t.Fatal("a missing client binary must be reported, not silently ignored")
	}
	if !strings.Contains(err.Error(), "PRUMO_TUI_BIN") || !strings.Contains(err.Error(), "prumo-tui") {
		t.Fatalf("the error does not say how to fix it: %v", err)
	}
}

func TestTuiRefusesJSON(t *testing.T) {
	if code, _ := captureOutput(func() int { return run([]string{"--json", "tui"}) }); code == 0 {
		t.Fatal("an interactive surface must refuse --json")
	}
}
