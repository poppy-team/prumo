package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/install"
	"github.com/raillen/prumo/internal/protocol"
)

func captureOutput(f func() int) (int, string) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := f()
	_ = w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return code, buf.String()
}

func testRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	return root
}

func TestRunVersion(t *testing.T) {
	code, out := captureOutput(func() int { return run([]string{"version"}) })
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(out, protocol.CLIVersion) {
		t.Fatalf("expected version output, got %q", out)
	}
}

func TestRunVersionJSON(t *testing.T) {
	code, out := captureOutput(func() int { return run([]string{"--json", "version"}) })
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(out, `"ok": true`) || !strings.Contains(out, `"protocol_version": "1"`) {
		t.Fatalf("expected json envelope, got %q", out)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	if code := run([]string{"unknown"}); code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}

func TestRunEmptyArgs(t *testing.T) {
	if code := run([]string{}); code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}

func TestRunVersionRejectsArgs(t *testing.T) {
	if code := run([]string{"version", "extra"}); code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}

func TestRunStatus(t *testing.T) {
	if code := run([]string{"status", "--path", t.TempDir()}); code != 6 {
		t.Fatalf("expected exit code 6, got %d", code)
	}
	if code, out := captureOutput(func() int {
		return run([]string{"--json", "status", "--path", t.TempDir()})
	}); code != 1 || !strings.Contains(out, "PRUMO_PROJECT_NOT_FOUND") {
		t.Fatalf("expected project error envelope, got %d %q", code, out)
	}
}

func TestInstallLifecyclePreservesProject(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	projectFile := filepath.Join(project, "docs.txt")
	if err := os.WriteFile(projectFile, []byte("keep"), 0644); err != nil {
		t.Fatalf("write project: %v", err)
	}
	if code := run([]string{"--home", home, "setup"}); code != 0 {
		t.Fatalf("setup failed: %d", code)
	}
	if code := run([]string{"--home", home, "setup"}); code != 0 {
		t.Fatalf("second setup failed: %d", code)
	}
	if code := run([]string{"--home", home, "install", "connector", "opencode", "--path", project}); code != 0 {
		t.Fatalf("install failed: %d", code)
	}
	if code := run([]string{"--home", home, "uninstall", "--connectors", "--purge-cache"}); code != 0 {
		t.Fatalf("uninstall failed: %d", code)
	}
	if code := run([]string{"--home", home, "uninstall", "--purge-global-config"}); code != 0 {
		t.Fatalf("purge failed: %d", code)
	}
	if _, err := os.Stat(projectFile); err != nil {
		t.Fatalf("project data was removed: %v", err)
	}
}

func TestInstallStateRecordsRunningVersion(t *testing.T) {
	for _, command := range []string{"setup", "install"} {
		t.Run(command, func(t *testing.T) {
			home := t.TempDir()
			stale, err := install.LoadManifest(home)
			if err != nil {
				t.Fatalf("load empty manifest: %v", err)
			}
			stale.PrumoVersion = "0.5.0"
			if err := install.SaveManifest(home, stale); err != nil {
				t.Fatalf("seed manifest: %v", err)
			}
			if code := run([]string{"--home", home, command}); code != 0 {
				t.Fatalf("%s failed: %d", command, code)
			}
			got, err := install.LoadManifest(home)
			if err != nil {
				t.Fatalf("load manifest: %v", err)
			}
			if got.PrumoVersion != protocol.CLIVersion {
				t.Fatalf("%s recorded version %q, want %q", command, got.PrumoVersion, protocol.CLIVersion)
			}
		})
	}
}

func TestCommandsEndToEnd(t *testing.T) {
	t.Setenv("PRUMO_REPO_ROOT", testRepoRoot(t))
	dir := t.TempDir()
	profile := filepath.Join(testRepoRoot(t), "examples", "brasa", "project-profile.json")
	if code := run([]string{"init", dir, "--profile", profile, "--non-interactive"}); code != 0 {
		t.Fatalf("init failed: %d", code)
	}
	if code := run([]string{"validate", dir}); code != 0 {
		t.Fatalf("validate failed: %d", code)
	}
	if code, out := captureOutput(func() int { return run([]string{"--json", "resolve", profile}) }); code != 0 || !strings.Contains(out, `"skills"`) {
		t.Fatalf("resolve failed: %d %q", code, out)
	}
	if code := run([]string{"goal", "new", "P00-G01", "Foundation", "--phase", "P00", "--path", dir}); code != 0 {
		t.Fatalf("goal new failed: %d", code)
	}
	if code := run([]string{"goal", "state", "P00-G01", "PLANNED", "--path", dir}); code != 0 {
		t.Fatalf("goal state failed: %d", code)
	}
	if code := run([]string{"goal", "list", "--path", dir}); code != 0 {
		t.Fatalf("goal list failed: %d", code)
	}
	if code, out := captureOutput(func() int { return run([]string{"--json", "context", "plan", "fix typo", "--path", dir}) }); code != 0 || !strings.Contains(out, `"small"`) {
		t.Fatalf("context plan failed: %d %q", code, out)
	}
	if code := run([]string{"compile", "--target", "generic", "--path", dir}); code != 0 {
		t.Fatalf("compile failed: %d", code)
	}
	if code := run([]string{"compile", "--target", "codex", "--path", dir}); code != 0 {
		t.Fatalf("compile codex failed: %d", code)
	}
	if code := run([]string{"snapshot", dir, "--output", filepath.Join(dir, "snap.zip")}); code != 0 {
		t.Fatalf("snapshot failed: %d", code)
	}
	if code := run([]string{"--json", "doctor", dir}); code != 0 {
		t.Fatalf("doctor failed: %d", code)
	}
	if code, out := captureOutput(func() int { return run([]string{"--json", "explain", "workforce", profile}) }); code != 0 || !strings.Contains(out, `"skills"`) {
		t.Fatalf("explain failed: %d %q", code, out)
	}
	if code := run([]string{"framework-check"}); code != 0 {
		t.Fatalf("framework-check failed: %d", code)
	}
	if code, out := captureOutput(func() int { return run([]string{"--json", "migrate", dir, "--dry-run"}) }); code != 0 || !strings.Contains(out, `"to_version": 3`) {
		t.Fatalf("migrate failed: %d %q", code, out)
	}
}

func TestRunHelp(t *testing.T) {
	code, out := captureOutput(func() int { return run([]string{"--help"}) })
	if code != 0 {
		t.Fatalf("expected exit code 0 for --help, got %d", code)
	}
	if !strings.Contains(out, "Prumo CLI") || !strings.Contains(out, "Project Lifecycle:") {
		t.Fatalf("expected general help output, got %q", out)
	}

	// Also test -h and help
	code2, out2 := captureOutput(func() int { return run([]string{"-h"}) })
	if code2 != 0 || !strings.Contains(out2, "Usage:") {
		t.Fatalf("expected help for -h, got %d %q", code2, out2)
	}

	code3, out3 := captureOutput(func() int { return run([]string{"help"}) })
	if code3 != 0 || !strings.Contains(out3, "Usage:") {
		t.Fatalf("expected help for 'help', got %d %q", code3, out3)
	}
}

func TestRunCommandHelp(t *testing.T) {
	code, out := captureOutput(func() int { return run([]string{"help", "init"}) })
	if code != 0 {
		t.Fatalf("expected 0 for 'help init', got %d", code)
	}
	if !strings.Contains(out, "COMMAND: prumo init") || !strings.Contains(out, "--profile") {
		t.Fatalf("expected init command help, got %q", out)
	}

	// Test flag help: prumo tool --help
	code2, out2 := captureOutput(func() int { return run([]string{"tool", "--help"}) })
	if code2 != 0 || !strings.Contains(out2, "COMMAND: prumo tool") {
		t.Fatalf("expected tool help for 'tool --help', got %d %q", code2, out2)
	}
}

func TestRunHelpJSON(t *testing.T) {
	code, out := captureOutput(func() int { return run([]string{"--json", "--help"}) })
	if code != 0 {
		t.Fatalf("expected 0 for --json --help, got %d", code)
	}
	if !strings.Contains(out, `"categories"`) || !strings.Contains(out, `"ok": true`) {
		t.Fatalf("expected json help categories, got %q", out)
	}

	code2, out2 := captureOutput(func() int { return run([]string{"--json", "help", "compile"}) })
	if code2 != 0 || !strings.Contains(out2, `"command": "compile"`) {
		t.Fatalf("expected json command help, got %d %q", code2, out2)
	}
}
