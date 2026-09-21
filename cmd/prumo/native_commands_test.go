package main

import (
	"strings"
	"testing"
)

func TestNativeBinaryResolution(t *testing.T) {
	t.Setenv("PRUMO_NATIVE_BIN", "/opt/prumo/prumo-native")
	if got, _, err := resolveNativeBinary(); err != nil || got != "/opt/prumo/prumo-native" {
		t.Fatalf("PRUMO_NATIVE_BIN was ignored: %q %v", got, err)
	}

	t.Setenv("PRUMO_NATIVE_BIN", "")
	t.Setenv("PRUMO_VIEWER_BIN", "/opt/prumo/prumo-viewer")
	if got, _, err := resolveNativeBinary(); err != nil || got != "/opt/prumo/prumo-viewer" {
		t.Fatalf("PRUMO_VIEWER_BIN was ignored: %q %v", got, err)
	}

	// When neither env var is set and PATH is empty and no local build target
	t.Setenv("PRUMO_NATIVE_BIN", "")
	t.Setenv("PRUMO_VIEWER_BIN", "")
	t.Setenv("PATH", "")
	t.Setenv("PRUMO_REPO_ROOT", "/nonexistent")
	_, _, err := resolveNativeBinary()
	if err == nil {
		t.Fatal("a missing native client binary must be reported, not silently ignored")
	}
	if !strings.Contains(err.Error(), "PRUMO_NATIVE_BIN") || !strings.Contains(err.Error(), "prumo-viewer") {
		t.Fatalf("the error does not say how to fix it: %v", err)
	}
}

func TestNativeRefusesJSON(t *testing.T) {
	if code, _ := captureOutput(func() int { return run([]string{"--json", "native"}) }); code == 0 {
		t.Fatal("an interactive surface must refuse --json")
	}
	if code, _ := captureOutput(func() int { return run([]string{"--json", "viewer"}) }); code == 0 {
		t.Fatal("an interactive surface must refuse --json")
	}
}
