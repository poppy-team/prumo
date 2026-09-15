package docengine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type evalContract struct {
	ID                   string   `json:"id"`
	RequiredKnowledge    []string `json:"required_knowledge"`
	EvidenceRequirements []string `json:"evidence_requirements"`
}

type evalSource struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type evalDocument struct {
	Path            string `json:"path"`
	Role            string `json:"role"`
	CanonicalSource string `json:"canonical_source"`
}

type evalCase struct {
	ID              string            `json:"id"`
	Kind            string            `json:"kind"`
	Title           string            `json:"title"`
	Requirement     string            `json:"requirement"`
	Content         string            `json:"content"`
	ExpectSatisfied *bool             `json:"expect_satisfied"`
	Contract        *evalContract     `json:"contract"`
	Sources         []evalSource      `json:"sources"`
	Evidence        []string          `json:"evidence"`
	RequirementMap  map[string]string `json:"requirement_map"`
	Claims          []ClaimBinding    `json:"claims"`
	ExpectState     string            `json:"expect_state"`
	ExpectFinding   string            `json:"expect_finding"`
	Documents       []evalDocument    `json:"documents"`
	ExpectFindings  int               `json:"expect_findings"`
}

// TestDocumentationEvalCorpus runs the documentation failure-mode corpus. Each
// case pins the outcome the evaluator must produce (W13.2).
func TestDocumentationEvalCorpus(t *testing.T) {
	cases := loadEvalCases(t)
	if len(cases) < 10 {
		t.Fatalf("eval corpus is unexpectedly small: %d cases", len(cases))
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			switch tc.Kind {
			case "knowledge-match":
				runKnowledgeMatch(t, tc)
			case "evidence-binding":
				runEvidenceBinding(t, tc)
			case "projection-cycle":
				runProjectionCycle(t, tc)
			default:
				t.Fatalf("case %s: unknown kind %q", tc.ID, tc.Kind)
			}
		})
	}
}

func loadEvalCases(t *testing.T) []evalCase {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "evals", "documentation", "cases", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("eval corpus has no cases")
	}
	cases := make([]evalCase, 0, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var tc evalCase
		if err := json.Unmarshal(data, &tc); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if tc.ID == "" || tc.Kind == "" {
			t.Fatalf("%s: eval case requires id and kind", f)
		}
		cases = append(cases, tc)
	}
	return cases
}

func runKnowledgeMatch(t *testing.T, tc evalCase) {
	t.Helper()
	if tc.ExpectSatisfied == nil {
		t.Fatalf("case %s: knowledge-match requires expect_satisfied", tc.ID)
	}
	got := matchesKnowledgeRequirement(tc.Content, tc.Requirement)
	if got != *tc.ExpectSatisfied {
		t.Fatalf("case %s (%s): matchesKnowledgeRequirement = %v, want %v", tc.ID, tc.Title, got, *tc.ExpectSatisfied)
	}
}

func runEvidenceBinding(t *testing.T, tc evalCase) {
	t.Helper()
	if tc.Contract == nil || tc.ExpectState == "" {
		t.Fatalf("case %s: evidence-binding requires contract and expect_state", tc.ID)
	}
	root := t.TempDir()
	paths := make([]string, 0, len(tc.Sources))
	for _, src := range tc.Sources {
		full := filepath.Join(root, src.Path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(src.Content), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, src.Path)
	}
	got := evaluate(root, Contract{
		ID:                   tc.Contract.ID,
		RequiredKnowledge:    tc.Contract.RequiredKnowledge,
		EvidenceRequirements: tc.Contract.EvidenceRequirements,
	}, Binding{
		Sources:        paths,
		Evidence:       tc.Evidence,
		RequirementMap: tc.RequirementMap,
		Claims:         tc.Claims,
	})
	if string(got.State) != tc.ExpectState {
		t.Fatalf("case %s (%s): state = %s, want %s (%v)", tc.ID, tc.Title, got.State, tc.ExpectState, got.Findings)
	}
	if tc.ExpectFinding != "" && !hasFinding(got.Findings, tc.ExpectFinding) {
		t.Fatalf("case %s (%s): expected finding %q in %v", tc.ID, tc.Title, tc.ExpectFinding, got.Findings)
	}
}

func runProjectionCycle(t *testing.T, tc evalCase) {
	t.Helper()
	documents := make([]AuthorityEntry, 0, len(tc.Documents))
	for _, d := range tc.Documents {
		documents = append(documents, AuthorityEntry{
			Path:            d.Path,
			Role:            AuthorityRole(d.Role),
			CanonicalSource: d.CanonicalSource,
		})
	}
	got := projectionCycleFindings(AuthorityMap{Documents: documents})
	if len(got) != tc.ExpectFindings {
		t.Fatalf("case %s (%s): findings = %d, want %d (%v)", tc.ID, tc.Title, len(got), tc.ExpectFindings, got)
	}
}
