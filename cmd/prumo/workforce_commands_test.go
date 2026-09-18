package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkforceCLI(t *testing.T) {
	tempProject := t.TempDir()
	tempHome := t.TempDir()

	// 1. Run workforce sync
	exitCode := runWorkforce(false, tempHome, []string{"sync", "--path", tempProject, "--offline"})
	if exitCode != exitOK {
		t.Fatalf("expected workforce sync to exit OK, got %d", exitCode)
	}

	// 2. Check that core skill cognitive-clarity and clean-code were installed
	for _, skill := range []string{"cognitive-clarity", "clean-code"} {
		manifestPath := filepath.Join(tempProject, ".ai", "skills", skill, "manifest.json")
		if _, err := os.Stat(manifestPath); err != nil {
			t.Fatalf("expected skill manifest for %s: %v", skill, err)
		}
	}

	// 3. Check lockfile
	lockPath := filepath.Join(tempProject, "prumo.lock")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("expected prumo.lock: %v", err)
	}

	// 4. Run workforce list
	exitCodeList := runWorkforce(false, tempHome, []string{"list", "--path", tempProject})
	if exitCodeList != exitOK {
		t.Fatalf("expected workforce list to exit OK, got %d", exitCodeList)
	}

	// 5. Test package sync alias
	exitCodePkg := runPlatform(false, []string{"package", "sync", "--path", tempProject, "--offline"})
	if exitCodePkg != exitOK {
		t.Fatalf("expected package sync alias to exit OK, got %d", exitCodePkg)
	}
}
