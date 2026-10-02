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

func TestExtractRust(t *testing.T) {
	src := `
pub struct Engine {
    pub id: String,
}

pub enum State {
    Running,
    Stopped,
}

pub trait Runner {
    fn execute(&self);
}

pub type Result<T> = std::result::Result<T, ()>;

pub fn start_engine(e: &Engine) -> State {
    State::Running
}
`
	syms, refs := ExtractRust("lib.rs", src)
	names := map[string]string{}
	for _, s := range syms {
		names[s.Name] = s.Kind
	}
	if names["Engine"] != "struct" {
		t.Errorf("expected struct Engine, got %v", names)
	}
	if names["State"] != "enum" {
		t.Errorf("expected enum State, got %v", names)
	}
	if names["Runner"] != "trait" {
		t.Errorf("expected trait Runner, got %v", names)
	}
	if names["start_engine"] != "func" {
		t.Errorf("expected func start_engine, got %v", names)
	}
	if len(refs) == 0 {
		t.Error("expected non-empty references")
	}
}

func TestExtractTypeScript(t *testing.T) {
	src := `
export interface User {
    id: string;
}

export type Role = "admin" | "editor";

export class SessionManager {
    login() {}
}

export function authenticate(u: User): boolean {
    return true;
}

export const API_VERSION = "v1";
`
	syms, refs := ExtractTypeScript("auth.ts", src)
	names := map[string]string{}
	for _, s := range syms {
		names[s.Name] = s.Kind
	}
	if names["User"] != "interface" {
		t.Errorf("expected interface User, got %v", names)
	}
	if names["Role"] != "type" {
		t.Errorf("expected type Role, got %v", names)
	}
	if names["SessionManager"] != "class" {
		t.Errorf("expected class SessionManager, got %v", names)
	}
	if names["authenticate"] != "func" {
		t.Errorf("expected func authenticate, got %v", names)
	}
	if names["API_VERSION"] != "const" {
		t.Errorf("expected const API_VERSION, got %v", names)
	}
	if len(refs) == 0 {
		t.Error("expected non-empty references")
	}
}

func TestExtractPython(t *testing.T) {
	src := `
class Worker:
    def __init__(self):
        pass

def process_item(item):
    return item
`
	syms, refs := ExtractPython("worker.py", src)
	names := map[string]string{}
	for _, s := range syms {
		names[s.Name] = s.Kind
	}
	if names["Worker"] != "class" {
		t.Errorf("expected class Worker, got %v", names)
	}
	if names["process_item"] != "func" {
		t.Errorf("expected func process_item, got %v", names)
	}
	if len(refs) == 0 {
		t.Error("expected non-empty references")
	}
}

func TestPersonalizedPageRank(t *testing.T) {
	dir := t.TempDir()

	// File 1 defines CentralService
	f1 := "package core\n\ntype CentralService struct{}\nfunc (c *CentralService) Start() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "central.go"), []byte(f1), 0o644); err != nil {
		t.Fatal(err)
	}

	// File 2 references CentralService
	f2 := "package api\n\nimport \"core\"\n\ntype ApiServer struct{\n\tSvc core.CentralService\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "api.go"), []byte(f2), 0o644); err != nil {
		t.Fatal(err)
	}

	// File 3 also references CentralService
	f3 := "package worker\n\nimport \"core\"\n\ntype WorkerPool struct{\n\tSvc core.CentralService\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "worker.go"), []byte(f3), 0o644); err != nil {
		t.Fatal(err)
	}

	m := Build(dir, 0, 0, 0)
	if m.Graph == nil {
		t.Fatal("expected Graph to be constructed")
	}
	if len(m.FileRanks) != 3 {
		t.Fatalf("expected 3 ranked files, got %d", len(m.FileRanks))
	}

	// central.go has in-degree 2 (from api.go and worker.go), so its PageRank should be higher than leaf nodes
	centralRank := m.FileRanks["central.go"]
	apiRank := m.FileRanks["api.go"]
	workerRank := m.FileRanks["worker.go"]

	if centralRank <= apiRank || centralRank <= workerRank {
		t.Errorf("expected central.go (%f) to outrank api.go (%f) and worker.go (%f)", centralRank, apiRank, workerRank)
	}

	// Personalized PageRank with query "worker"
	workerPersonalized := m.Graph.ComputePageRank("worker", 0.85, 50)
	if workerPersonalized["worker.go"] <= workerPersonalized["api.go"] {
		t.Errorf("expected worker.go (%f) to outrank api.go (%f) under 'worker' query",
			workerPersonalized["worker.go"], workerPersonalized["api.go"])
	}

	// RenderWithBudget produces compact representation
	rendered := m.RenderWithBudget(100, "worker")
	if !strings.Contains(rendered, "WorkerPool") {
		t.Errorf("expected WorkerPool in rendered output under worker budget, got:\n%s", rendered)
	}
}
