package uimap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// mapCase is one behavioral expectation about the interface map.
//
// The corpus exists because a rule that only lives in Go is a rule an author has
// to read the source to trust. Each case states the expectation in data, so the
// contract and its control cases are reviewable side by side.
type mapCase struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Title      string `json:"title"`
	Map        Map    `json:"map"`
	Vocabulary struct {
		States []string `json:"states"`
		Tokens []string `json:"tokens"`
	} `json:"vocabulary"`
	Symbols []string `json:"symbols"`
	// Derivation is the scripted derivation, so a case can exercise the
	// declared-versus-derived rules without a repository of Go files.
	Derivation struct {
		Mode     string   `json:"mode"`
		Sources  []string `json:"sources"`
		Elements []string `json:"elements"`
	} `json:"derivation"`

	ExpectOK                 bool     `json:"expect_ok"`
	ExpectFindings           []string `json:"expect_findings"`
	ExpectUndeclared         []string `json:"expect_undeclared"`
	ExpectRoots              int      `json:"expect_roots"`
	ExpectProjectionContains []string `json:"expect_projection_contains"`
}

func loadMapCases(t *testing.T) []mapCase {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "evals", "interface-map", "cases")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".json") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	cases := make([]mapCase, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		var loaded mapCase
		if err := json.Unmarshal(data, &loaded); err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		if loaded.Kind != "interface-map" {
			t.Fatalf("%s: kind %q", name, loaded.Kind)
		}
		cases = append(cases, loaded)
	}
	return cases
}

// repoRoot walks up to the module root so the corpus can be located from any
// working directory the test binary is started in.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for range 8 {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("module root not found")
	return ""
}

func TestInterfaceMapEvalCorpus(t *testing.T) {
	cases := loadMapCases(t)
	if len(cases) < 10 {
		t.Fatalf("corpus has %d cases; the controls that prove a rule is load-bearing need to be present", len(cases))
	}
	// At least one case must pass, or the corpus would be satisfied by a
	// validator that refuses everything.
	passing := 0
	for _, tc := range cases {
		if tc.ExpectOK {
			passing++
		}
	}
	if passing == 0 {
		t.Fatal("every case expects a failure, so the corpus cannot distinguish strict from broken")
	}

	for _, tc := range cases {
		t.Run(tc.ID, func(t *testing.T) {
			vocab := Vocabulary{States: map[string]bool{}, Tokens: map[string]bool{}}
			for _, s := range tc.Vocabulary.States {
				vocab.States[s] = true
			}
			for _, s := range tc.Vocabulary.Tokens {
				vocab.Tokens[s] = true
			}
			declared := tc.Map
			derived := Derived{
				Sources:  tc.Derivation.Sources,
				Symbols:  append(append([]string{}, tc.Symbols...), tc.Derivation.Elements...),
				Elements: tc.Derivation.Elements,
			}
			// The resolver is built from the same derivation the merge saw, which
			// is what Compile does: a symbol the deriver found is a symbol that
			// exists, whether or not the declaration had heard of it.
			resolver := resolverStub{}
			for _, s := range derived.Symbols {
				resolver[s] = true
			}
			mode := "verify-only"
			if tc.Derivation.Mode != "" {
				mode = tc.Derivation.Mode
			}
			merged := Merge(declared, derived, mode)

			report := Validate(".", merged.Map, vocab, resolver)
			if got := report.OK(); got != tc.ExpectOK {
				t.Fatalf("ok = %v, want %v (findings: %v)", got, tc.ExpectOK, codes(report))
			}
			for _, want := range tc.ExpectFindings {
				wantCode(t, report, want)
			}
			if len(tc.ExpectFindings) == 0 && len(report.Findings) != 0 {
				t.Fatalf("expected a clean report, got %v", codes(report))
			}
			if tc.ExpectRoots != 0 && len(merged.Map.Tree) != tc.ExpectRoots {
				t.Fatalf("tree has %d roots, want %d", len(merged.Map.Tree), tc.ExpectRoots)
			}
			got := append([]string{}, merged.Undeclared...)
			sort.Strings(got)
			want := append([]string{}, tc.ExpectUndeclared...)
			sort.Strings(want)
			if !equalStrings(got, want) {
				t.Fatalf("undeclared = %v, want %v", got, want)
			}

			if len(tc.ExpectProjectionContains) > 0 {
				cfg := Config{Enabled: true, Targets: []Target{TargetDeveloper}, Derivation: mode}
				projections := Project(merged.Map, cfg, report, derived)
				if len(projections) == 0 {
					t.Fatal("no projection produced")
				}
				rendered := projections[0].Content
				for _, want := range tc.ExpectProjectionContains {
					if !strings.Contains(rendered, want) {
						t.Errorf("projection is missing %q:\n%s", want, rendered)
					}
				}
			}
		})
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
