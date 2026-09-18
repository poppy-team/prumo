package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/protocol"
)

// contextFixture is a minimal project the compiler can walk.
func contextFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"AGENTS.md":   "# AGENTS.md\n\nAgent instructions.\n",
		"README.md":   "# Project\n\n## Usage\n\nrun it\n",
		"go.mod":      "module example.com/fixture\n\ngo 1.22\n",
		"docs/one.md": "# One\n\nbody\n",
	}
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestContextCompileEmitsReplayableManifest(t *testing.T) {
	root := contextFixture(t)
	code, out := captureOutput(func() int {
		return run([]string{"context", "compile", "--goal", "add a transition template",
			"--budget", "2000", "--level", "L2", "--id", "smoke", "--path", root, "--json"})
	})
	if code != exitOK {
		t.Fatalf("expected success, got %d: %s", code, out)
	}
	var env protocol.Envelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("output must be a JSON envelope: %v\n%s", err, out)
	}
	if !env.Ok {
		t.Fatalf("expected an ok envelope: %s", out)
	}
	data, _ := env.Data.(map[string]any)
	manifest, _ := data["manifest"].(map[string]any)
	if manifest["id"] != "CTX-smoke" {
		t.Fatalf("manifest must carry its own id: %#v", manifest["id"])
	}
	if manifest["level"] != "L2" {
		t.Fatalf("the requested level must be recorded: %#v", manifest["level"])
	}
	included, _ := manifest["included"].([]any)
	if len(included) == 0 {
		t.Fatal("expected the compilation to include something")
	}
	for _, raw := range included {
		item, _ := raw.(map[string]any)
		if reason, _ := item["reason"].(string); reason == "" {
			t.Fatalf("every included item must state why it was selected: %#v", item)
		}
	}
	// The manifest is written under runtime state, not into the repository.
	if _, err := os.Stat(filepath.Join(root, ".prumo", "runtime", "harness", "context-smoke.json")); err != nil {
		t.Fatalf("manifest must be persisted under runtime state: %v", err)
	}

	explainCode, explainOut := captureOutput(func() int {
		return run([]string{"context", "explain", "CTX-smoke", "--budget", "2000", "--path", root})
	})
	if explainCode != exitOK {
		t.Fatalf("expected explain to succeed, got %d: %s", explainCode, explainOut)
	}
	if !strings.Contains(explainOut, "included") || !strings.Contains(explainOut, "CTX-smoke") {
		t.Fatalf("explain must describe the compilation: %s", explainOut)
	}
}

func TestContextExplainFailsOnMissingObligation(t *testing.T) {
	root := contextFixture(t)
	if code, out := captureOutput(func() int {
		return run([]string{"context", "compile", "--goal", "goal", "--id", "req", "--path", root, "--json"})
	}); code != exitOK {
		t.Fatalf("compile failed: %d %s", code, out)
	}
	code, out := captureOutput(func() int {
		return run([]string{"context", "explain", "CTX-req", "--required", "file:absent.md", "--path", root})
	})
	if code != exitValidation {
		t.Fatalf("an unsatisfied obligation must fail validation, got %d: %s", code, out)
	}
	if !strings.Contains(out, "absent.md") {
		t.Fatalf("the missing obligation must be named: %s", out)
	}
}

func TestContextCompileRequiresAGoal(t *testing.T) {
	code, _ := captureOutput(func() int { return run([]string{"context", "compile"}) })
	if code != exitUsage {
		t.Fatalf("compile without --goal must be a usage error, got %d", code)
	}
}

func TestKnowledgeManifestIsDerivedAndValidated(t *testing.T) {
	root := t.TempDir()
	// A persisted run store, as the headless runtime leaves behind.
	dir := filepath.Join(root, ".prumo", "runtime", "harness")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	store := `{"records":[{"id":"ku:requirement.run-r1","kind":"requirement","title":"goal",
		"authority":"canonical","trust":"high","status":"active","updated_at":"2026-09-15T00:00:00Z"}],
		"rels":[],"aliases":{"req-r1":"ku:requirement.run-r1"}}`
	if err := os.WriteFile(filepath.Join(dir, "knowledge-r1.json"), []byte(store), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := captureOutput(func() int {
		return run([]string{"knowledge", "manifest", "--path", root, "--json"})
	})
	if code != exitOK {
		t.Fatalf("expected success, got %d: %s", code, out)
	}
	var env protocol.Envelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("output must be a JSON envelope: %v", err)
	}
	data, _ := env.Data.(map[string]any)
	manifest, _ := data["manifest"].(map[string]any)
	if manifest["derived"] != true {
		t.Fatalf("a manifest must declare itself derived: %#v", manifest["derived"])
	}
	units, _ := manifest["units"].([]any)
	if len(units) != 1 {
		t.Fatalf("expected one unit, got %#v", units)
	}
	unit, _ := units[0].(map[string]any)
	locators, _ := unit["locators"].([]any)
	// The pre-migration locator stays resolvable through the manifest.
	found := false
	for _, loc := range locators {
		if loc == "req-r1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("legacy locator must remain bound: %#v", locators)
	}
	// The default destination is runtime state.
	if _, err := os.Stat(filepath.Join(root, ".prumo", "runtime", "knowledge-manifest.json")); err != nil {
		t.Fatalf("manifest must be written under runtime state: %v", err)
	}

	idsCode, idsOut := captureOutput(func() int {
		return run([]string{"knowledge", "ids", "--path", root, "--json"})
	})
	if idsCode != exitOK {
		t.Fatalf("ids failed: %d %s", idsCode, idsOut)
	}
	if !strings.Contains(idsOut, "unstable_ids") {
		t.Fatalf("ids must report the migration surface: %s", idsOut)
	}
}

func TestKnowledgeRequiresASubcommand(t *testing.T) {
	code, _ := captureOutput(func() int { return run([]string{"knowledge"}) })
	if code != exitUsage {
		t.Fatalf("knowledge without a subcommand must be a usage error, got %d", code)
	}
}
