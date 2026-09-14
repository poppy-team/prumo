package runlayer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEvaluateGates(t *testing.T) {
	pols := []GatePolicy{
		{Name: "tests", Kind: "require-test-pass"},
		{Name: "clean", Kind: "forbid-tool-failure"},
		{Name: "few", Kind: "max-tool-calls", Limit: 5},
		{Name: "cheap", Kind: "max-tokens", Limit: 100},
		{Name: "mystery", Kind: "bogus"},
	}
	reps := []ToolReport{{Name: "test.run", ExitCode: 0}, {Name: "fs.read", ExitCode: 0}}
	res := EvaluateGates(pols, reps, map[string]float64{"tokens": 10})
	byID := map[string]string{}
	for _, r := range res {
		byID[r.ID] = r.Status
	}
	for _, id := range []string{"tests", "clean", "few", "cheap"} {
		if byID[id] != "pass" {
			t.Fatalf("%s must pass: %+v", id, res)
		}
	}
	if byID["mystery"] != "fail" {
		t.Fatal("unknown kind must fail closed")
	}
	bad := EvaluateGates(pols, []ToolReport{{Name: "test.run", ExitCode: 1}}, map[string]float64{"tokens": 999})
	fails := 0
	for _, r := range bad {
		if r.Status == "fail" {
			fails++
		}
	}
	if fails != 4 { // tests, clean, cheap fail; few passes; mystery fails = 4
		t.Fatalf("expected 4 fails, got %d: %+v", fails, bad)
	}
	path := filepath.Join(t.TempDir(), "gates.json")
	if err := os.WriteFile(path, []byte(`[{"name":"t","kind":"require-test-pass"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadGatePolicies(path)
	if err != nil || len(loaded) != 1 {
		t.Fatalf("load failed: %+v %v", loaded, err)
	}
	if _, err := LoadGatePolicies(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("missing file must error")
	}
}
