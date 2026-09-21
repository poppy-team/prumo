package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunAsk_BasicAndJSON(t *testing.T) {
	// 1. Missing prompt
	if code := runAsk(false, []string{}); code != exitUsage {
		t.Fatalf("expected exitUsage for missing prompt, got %d", code)
	}

	// 2. Normal ask with fake provider
	if code := runAsk(false, []string{"hello", "world"}); code != exitOK {
		t.Fatalf("expected exitOK, got %d", code)
	}

	// 3. Ask with --json flag
	if code := runAsk(true, []string{"explain", "architecture"}); code != exitOK {
		t.Fatalf("expected exitOK with --json, got %d", code)
	}

	// 4. Ask with --file flag
	tmpDir := t.TempDir()
	sampleFile := filepath.Join(tmpDir, "context.txt")
	if err := os.WriteFile(sampleFile, []byte("system content"), 0644); err != nil {
		t.Fatalf("failed to write sample file: %v", err)
	}

	if code := runAsk(false, []string{"--file", sampleFile, "summarize", "this"}); code != exitOK {
		t.Fatalf("expected exitOK with --file, got %d", code)
	}

	// 5. Ask with nonexistent file
	if code := runAsk(false, []string{"--file", filepath.Join(tmpDir, "missing.txt"), "summarize"}); code != exitUnavailable {
		t.Fatalf("expected exitUnavailable for missing file, got %d", code)
	}
}
