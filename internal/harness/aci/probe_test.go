package aci

import (
	"os"
	"path/filepath"
	"testing"
)

// A runtime is asked in its own language: `podman info` has no ServerVersion,
// so the Docker question fails on a podman that works.
func TestEachRuntimeIsAskedItsOwnQuestion(t *testing.T) {
	if got := probeArgs("podman"); got[len(got)-1] != "{{.Version.Version}}" {
		t.Fatalf("podman is asked the docker question: %v", got)
	}
	if got := probeArgs("docker"); got[len(got)-1] != "{{.ServerVersion}}" {
		t.Fatalf("docker is asked the wrong question: %v", got)
	}
}

// fakeRuntime puts a script named `name` on PATH that exits with `code`.
func fakeRuntime(t *testing.T, name string, code int) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\nexit " + string(rune('0'+code)) + "\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatalf("cannot write the fake runtime: %v", err)
	}
	t.Setenv("PATH", dir)
}

// Presence is not capability: a podman that cannot run a container must not take
// the sandbox away from a docker that can.
func TestABrokenRuntimeDoesNotWin(t *testing.T) {
	dir := t.TempDir()
	for name, code := range map[string]string{"podman": "exit 1", "docker": "exit 0"} {
		script := "#!/bin/sh\n" + code + "\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatalf("cannot write %s: %v", name, err)
		}
	}
	t.Setenv("PATH", dir)

	if got := DetectContainerRuntime(); got != "docker" {
		t.Fatalf("a podman that cannot run containers was preferred over a docker that can: %q", got)
	}
}

// With nothing usable there is no runtime, and saying otherwise would promise a
// sandbox that does not exist.
func TestNoUsableRuntimeIsNoRuntime(t *testing.T) {
	fakeRuntime(t, "podman", 1)

	if got := DetectContainerRuntime(); got != "" {
		t.Fatalf("an unusable host reported a runtime: %q", got)
	}
}
