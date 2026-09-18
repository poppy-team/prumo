package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPortableContinuationAcrossProcesses(t *testing.T) {
	root := t.TempDir()
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	create := exec.Command("go", "run", "./cmd/prumo", "run", "--run", "R-PROCESS", "--path", root)
	create.Dir = projectRoot
	if output, err := create.CombinedOutput(); err != nil {
		t.Fatalf("create: %v %s", err, output)
	}
	continueCmd := exec.Command("go", "run", "./cmd/prumo", "continue", "--run", "R-PROCESS", "--path", root, "--prompt")
	continueCmd.Dir = projectRoot
	continueCmd.Env = append(os.Environ(), "PRUMO_REPO_ROOT="+projectRoot)
	output, err := continueCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("continue: %v %s", err, output)
	}
	if !strings.Contains(string(output), "Run: R-PROCESS") || !strings.Contains(string(output), "inspect Goal") {
		t.Fatalf("continuation missing state: %s", output)
	}
}
