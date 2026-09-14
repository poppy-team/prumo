package repomap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildGoSymbols(t *testing.T) {
	dir := t.TempDir()
	src := "package agent\n\n// Runner runs.\nfunc NewRunner() {}\n\ntype State struct{}\n\nfunc lowercase() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	m := Build(dir, 0, 0, 0)
	if len(m.Modules) != 1 || m.Modules[0].Package != "agent" {
		t.Fatalf("bad modules: %+v", m.Modules)
	}
	names := map[string]bool{}
	for _, s := range m.Modules[0].Symbols {
		names[s.Name] = true
	}
	if !names["NewRunner"] || !names["State"] || names["lowercase"] {
		t.Fatalf("exported-only index wrong: %+v", names)
	}
	got := m.Lookup("runner")
	if len(got) != 1 || got[0].Kind != "func" {
		t.Fatalf("lookup failed: %+v", got)
	}
	rendered := m.Render(100000)
	if !strings.Contains(rendered, "NewRunner") {
		t.Fatalf("render missing symbol:\n%s", rendered)
	}
	if short := m.Render(20); !strings.Contains(short, "truncated") {
		t.Fatalf("render must truncate:\n%s", short)
	}
}

func TestBuildSkips(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, ".git", "x.go"), []byte("package x\nfunc Y() {}\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "b.txt"), []byte("nope"), 0o644)
	m := Build(dir, 0, 0, 0)
	if len(m.Modules) != 0 {
		t.Fatalf("skip rules failed: %+v", m.Modules)
	}
}
