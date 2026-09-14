package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGeneratedFresh pins protocol.d.ts to its generator: hand edits fail.
func TestGeneratedFresh(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := dir
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("go.mod not found")
		}
		root = parent
	}
	got, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(root, "sdk", "typescript", "protocol.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatal("protocol.d.ts stale: run go run ./sdk/typescript/gen")
	}
	for _, op := range []string{`"start"`, `"steer"`, `"schedule"`, `"protocol"`} {
		if !strings.Contains(got, op) {
			t.Fatalf("generated types missing %s", op)
		}
	}
}
