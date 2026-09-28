package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestErgonomicsInitZeroArgs(t *testing.T) {
	tempDir := t.TempDir()
	code := run([]string{"init", tempDir, "--non-interactive"})
	if code != 0 {
		t.Fatalf("expected exit code 0 for zero-arg init, got %d", code)
	}

	valCode := run([]string{"validate", tempDir})
	if valCode != 0 {
		t.Fatalf("expected exit code 0 for validate on newly initialized project, got %d", valCode)
	}
}

func TestErgonomicsInitWithPresetAndPrintProfile(t *testing.T) {
	tempDir := t.TempDir()
	printCode := run([]string{"init", tempDir, "--preset", "web", "--print-profile"})
	if printCode != 0 {
		t.Fatalf("expected exit code 0 for --print-profile, got %d", printCode)
	}

	initCode := run([]string{"init", tempDir, "--preset", "web", "--name", "my-web-tool", "--non-interactive"})
	if initCode != 0 {
		t.Fatalf("expected exit code 0 for init with preset, got %d", initCode)
	}

	valCode := run([]string{"validate", tempDir})
	if valCode != 0 {
		t.Fatalf("expected exit code 0 for validate on web preset project, got %d", valCode)
	}
}

func TestErgonomicsAdoptSubcommandsAndApply(t *testing.T) {
	tempDir := t.TempDir()

	// Create a brownfield Go project
	goMod := `module github.com/example/legacy-service

go 1.22

require github.com/gin-gonic/gin v1.9.1
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "api.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "service.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// 1. Test 'prumo adopt scan'
	if code := run([]string{"adopt", "scan", tempDir}); code != 0 {
		t.Fatalf("expected 0 for 'adopt scan', got %d", code)
	}

	// 2. Test 'prumo adopt facts'
	if code := run([]string{"adopt", "facts", tempDir}); code != 0 {
		t.Fatalf("expected 0 for 'adopt facts', got %d", code)
	}

	// 3. Test 'prumo adopt classify'
	if code := run([]string{"adopt", "classify", tempDir}); code != 0 {
		t.Fatalf("expected 0 for 'adopt classify', got %d", code)
	}

	// 4. Test 'prumo adopt scaffold' (dry run)
	if code := run([]string{"adopt", "scaffold", tempDir}); code != 0 {
		t.Fatalf("expected 0 for 'adopt scaffold', got %d", code)
	}

	// Verify prumo.json is not yet created
	if _, err := os.Stat(filepath.Join(tempDir, "prumo.json")); err == nil {
		t.Fatalf("scaffold must not create prumo.json on disk")
	}

	// 5. Test 'prumo adopt apply'
	if code := run([]string{"adopt", "apply", tempDir, "--non-interactive"}); code != 0 {
		t.Fatalf("expected 0 for 'adopt apply', got %d", code)
	}

	// Verify prumo.json now exists
	if _, err := os.Stat(filepath.Join(tempDir, "prumo.json")); err != nil {
		t.Fatalf("expected prumo.json to exist after adopt apply: %v", err)
	}

	// 6. Test 'prumo validate' passes 100% on the adopted project!
	if valCode := run([]string{"validate", tempDir}); valCode != 0 {
		t.Fatalf("expected adopted project to pass prumo validate, got %d", valCode)
	}
}

func TestErgonomicsConnectorStatus(t *testing.T) {
	if code := run([]string{"connector", "status", "opencode"}); code != 0 {
		t.Fatalf("expected 0 for 'connector status opencode', got %d", code)
	}

	if code := run([]string{"connector", "status", "claude"}); code != 0 {
		t.Fatalf("expected 0 for 'connector status claude', got %d", code)
	}

	// Test JSON output
	if code := run([]string{"--json", "connector", "status", "antigravity"}); code != 0 {
		t.Fatalf("expected 0 for '--json connector status antigravity', got %d", code)
	}
}

func TestErgonomicsDashboard(t *testing.T) {
	// Inside repo root (current directory is prumo project)
	if code := run([]string{}); code != 0 {
		t.Fatalf("expected 0 for bare 'prumo', got %d", code)
	}
	if code := run([]string{"--json"}); code != 0 {
		t.Fatalf("expected 0 for bare 'prumo --json', got %d", code)
	}
}
