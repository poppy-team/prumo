package docengine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type uiEvalCase struct {
	ID             string         `json:"id"`
	Kind           string         `json:"kind"`
	Title          string         `json:"title"`
	Spec           UISpec         `json:"spec"`
	ExpectFindings []string       `json:"expect_findings"`
	ExpectCounts   map[string]int `json:"expect_counts"`
}

// TestUISpecEvalCorpus runs the UI specification failure-mode corpus. Each case
// pins the findings a defective specification must produce, and the corpus
// carries a control that must produce none (W13.3).
func TestUISpecEvalCorpus(t *testing.T) {
	cases := loadUISpecCases(t)
	if len(cases) < 5 {
		t.Fatalf("UI eval corpus is unexpectedly small: %d cases", len(cases))
	}
	controls := 0
	for _, tc := range cases {
		tc := tc
		if strings.HasPrefix(tc.Title, "Control:") {
			controls++
		}
		t.Run(tc.ID, func(t *testing.T) {
			if tc.Kind != "ui-specification" {
				t.Fatalf("case %s: unknown kind %q", tc.ID, tc.Kind)
			}
			findings := ValidateUISpec(tc.Spec)
			kinds := map[string]int{}
			for _, f := range findings {
				kinds[f.Kind]++
				if f.Subject == "" || f.Detail == "" {
					t.Errorf("case %s: finding %#v must name its subject and detail", tc.ID, f)
				}
			}
			if len(tc.ExpectFindings) == 0 && len(findings) != 0 {
				t.Fatalf("case %s (%s): expected no findings, got %#v", tc.ID, tc.Title, findings)
			}
			for _, want := range tc.ExpectFindings {
				if kinds[want] == 0 {
					t.Errorf("case %s (%s): expected a %q finding, got %#v", tc.ID, tc.Title, want, findings)
				}
			}
			for kind, want := range tc.ExpectCounts {
				if kinds[kind] != want {
					t.Errorf("case %s (%s): %s findings = %d, want %d", tc.ID, tc.Title, kind, kinds[kind], want)
				}
			}
		})
	}
	if controls == 0 {
		t.Fatal("a corpus of only negative cases cannot show that a valid specification still passes")
	}
}

func loadUISpecCases(t *testing.T) []uiEvalCase {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "evals", "ui-specification", "cases", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("UI eval corpus has no cases")
	}
	cases := make([]uiEvalCase, 0, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var tc uiEvalCase
		if err := json.Unmarshal(data, &tc); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if tc.ID == "" || tc.Kind == "" {
			t.Fatalf("%s: a UI eval case requires id and kind", f)
		}
		cases = append(cases, tc)
	}
	return cases
}

// TestUISpecRepositoryIsClean is the dogfood half: the repository's own UI
// specification must pass every check. A failure here is a real gap in the
// shipped UI contracts, not a test fixture problem.
func TestUISpecRepositoryIsClean(t *testing.T) {
	root := filepath.Join("..", "..")
	spec, err := LoadUISpec(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Contracts) == 0 || len(spec.Tokens) == 0 || len(spec.Themes) == 0 {
		t.Fatalf("the repository UI specification looks empty: %d contracts, %d tokens, %d themes",
			len(spec.Contracts), len(spec.Tokens), len(spec.Themes))
	}
	findings := ValidateUISpec(spec)
	for _, f := range findings {
		t.Errorf("%s: %s — %s", f.Kind, f.Subject, f.Detail)
	}
}

// LoadUISpec reads the repository's UI contracts, profiles and design tokens.
func LoadUISpec(root string) (UISpec, error) {
	spec := UISpec{
		Contracts: []UISpecContract{}, Profiles: []UISpecProfile{},
		Tokens: []UISpecToken{}, Themes: []UISpecTheme{},
	}
	if err := readSpecFile(filepath.Join(root, "docs", "contracts", "builtin.json"), &spec.Contracts); err != nil {
		return UISpec{}, err
	}
	if err := readSpecFile(filepath.Join(root, "docs", "profiles", "builtin.json"), &spec.Profiles); err != nil {
		return UISpec{}, err
	}
	var tokens struct {
		Tokens []UISpecToken `json:"tokens"`
		Themes []UISpecTheme `json:"themes"`
	}
	if err := readSpecFile(filepath.Join(root, "docs", "ui-ux", "design-tokens.json"), &tokens); err != nil {
		return UISpec{}, err
	}
	spec.Tokens = tokens.Tokens
	spec.Themes = tokens.Themes
	return spec, nil
}

func readSpecFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
