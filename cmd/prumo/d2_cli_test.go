package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestD2ToolCLI(t *testing.T) {
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	// Tool list
	cmdList := exec.Command("go", "run", "./cmd/prumo", "tool", "list")
	cmdList.Dir = projectRoot
	outList, err := cmdList.CombinedOutput()
	if err != nil {
		t.Fatalf("tool list failed: %v (%s)", err, outList)
	}
	if !strings.Contains(string(outList), "read_file") || !strings.Contains(string(outList), "git_commit") {
		t.Fatalf("expected tool list to show read_file and git_commit: %s", outList)
	}

	// Tool inspect
	cmdInspect := exec.Command("go", "run", "./cmd/prumo", "--json", "tool", "inspect", "read_file")
	cmdInspect.Dir = projectRoot
	outInspect, err := cmdInspect.CombinedOutput()
	if err != nil {
		t.Fatalf("tool inspect failed: %v (%s)", err, outInspect)
	}
	if !strings.Contains(string(outInspect), `"read_file"`) || !strings.Contains(string(outInspect), `"read-only"`) {
		t.Fatalf("unexpected tool inspect output: %s", outInspect)
	}

	// Tool evaluate allowed
	cmdEvalOK := exec.Command("go", "run", "./cmd/prumo", "tool", "evaluate", "read_file", projectRoot)
	cmdEvalOK.Dir = projectRoot
	outEvalOK, err := cmdEvalOK.CombinedOutput()
	if err != nil {
		t.Fatalf("tool evaluate ok failed: %v (%s)", err, outEvalOK)
	}
	if !strings.Contains(string(outEvalOK), "Allowed") {
		t.Fatalf("expected tool evaluate allowed: %s", outEvalOK)
	}
}

func TestD2ModelCLI(t *testing.T) {
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	// Model list
	cmdList := exec.Command("go", "run", "./cmd/prumo", "model", "list")
	cmdList.Dir = projectRoot
	outList, err := cmdList.CombinedOutput()
	if err != nil {
		t.Fatalf("model list failed: %v (%s)", err, outList)
	}
	if !strings.Contains(string(outList), "local-default") || !strings.Contains(string(outList), "cloud-deep") {
		t.Fatalf("expected model list to show models: %s", outList)
	}

	// Model route with restricted data class
	cmdRoute := exec.Command("go", "run", "./cmd/prumo", "--json", "model", "route", "--data-class", "restricted", "--tools")
	cmdRoute.Dir = projectRoot
	outRoute, err := cmdRoute.CombinedOutput()
	if err != nil {
		t.Fatalf("model route failed: %v (%s)", err, outRoute)
	}
	if !strings.Contains(string(outRoute), `"local-default"`) {
		t.Fatalf("expected local-default to be selected for restricted data class: %s", outRoute)
	}
}

func TestD2EnvCLI(t *testing.T) {
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	cmdEnv := exec.Command("go", "run", "./cmd/prumo", "env", "list")
	cmdEnv.Dir = projectRoot
	outEnv, err := cmdEnv.CombinedOutput()
	if err != nil {
		t.Fatalf("env list failed: %v (%s)", err, outEnv)
	}
	if !strings.Contains(string(outEnv), "local") || !strings.Contains(string(outEnv), "worktree") {
		t.Fatalf("expected env list to show local and worktree: %s", outEnv)
	}
}

func TestBuiltinToolCLI(t *testing.T) {
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	// Tool list includes scan-secrets and analyze-complexity
	cmdList := exec.Command("go", "run", "./cmd/prumo", "tool", "list")
	cmdList.Dir = projectRoot
	outList, err := cmdList.CombinedOutput()
	if err != nil {
		t.Fatalf("tool list failed: %v (%s)", err, outList)
	}
	for _, toolName := range []string{"scan-secrets", "analyze-complexity", "check-permissions", "check-escape-hatches"} {
		if !strings.Contains(string(outList), toolName) {
			t.Fatalf("expected tool list to contain %s: %s", toolName, outList)
		}
	}

	// Tool stride
	cmdStride := exec.Command("go", "run", "./cmd/prumo", "--json", "tool", "stride", "OrderService")
	cmdStride.Dir = projectRoot
	outStride, err := cmdStride.CombinedOutput()
	if err != nil {
		t.Fatalf("tool stride failed: %v (%s)", err, outStride)
	}
	if !strings.Contains(string(outStride), "OrderService") || !strings.Contains(string(outStride), "Spoofing") {
		t.Fatalf("unexpected stride output: %s", outStride)
	}

	// Tool security-checklist
	cmdChecklist := exec.Command("go", "run", "./cmd/prumo", "tool", "security-checklist", "42")
	cmdChecklist.Dir = projectRoot
	outChecklist, err := cmdChecklist.CombinedOutput()
	if err != nil {
		t.Fatalf("tool security-checklist failed: %v (%s)", err, outChecklist)
	}
	if !strings.Contains(string(outChecklist), "Security Review Checklist for PR #42") {
		t.Fatalf("unexpected security checklist output: %s", outChecklist)
	}

	// Tool check-permissions (on temporary clean dir)
	tempDir := t.TempDir()
	cmdPerm := exec.Command("go", "run", "./cmd/prumo", "tool", "check-permissions", tempDir)
	cmdPerm.Dir = projectRoot
	outPerm, err := cmdPerm.CombinedOutput()
	if err != nil {
		t.Fatalf("tool check-permissions failed: %v (%s)", err, outPerm)
	}
	if !strings.Contains(string(outPerm), "Permissions clean") {
		t.Fatalf("expected permissions clean: %s", outPerm)
	}

	// Tool check-escape-hatches (on temporary clean dir)
	cmdHatch := exec.Command("go", "run", "./cmd/prumo", "tool", "check-escape-hatches", tempDir)
	cmdHatch.Dir = projectRoot
	outHatch, err := cmdHatch.CombinedOutput()
	if err != nil {
		t.Fatalf("tool check-escape-hatches failed: %v (%s)", err, outHatch)
	}
	if !strings.Contains(string(outHatch), "Clean:") {
		t.Fatalf("expected clean escape hatches output: %s", outHatch)
	}
}
