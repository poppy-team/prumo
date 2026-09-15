package contextv2

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type contextEvalItem struct {
	Ref             string  `json:"ref"`
	Authority       string  `json:"authority"`
	Score           float64 `json:"score"`
	Content         string  `json:"content"`
	TokenCost       int     `json:"token_cost"`
	Projection      bool    `json:"projection"`
	CanonicalSource string  `json:"canonical_source"`
}

type contextEvalExclusion struct {
	Ref    string `json:"ref"`
	Reason string `json:"reason"`
}

type contextEvalCase struct {
	ID               string                 `json:"id"`
	Kind             string                 `json:"kind"`
	Title            string                 `json:"title"`
	Level            string                 `json:"level"`
	Content          string                 `json:"content"`
	ExpectRevealed   []string               `json:"expect_revealed"`
	ExpectAbsent     []string               `json:"expect_absent"`
	ExpectFull       *bool                  `json:"expect_full"`
	Items            []contextEvalItem      `json:"items"`
	ExpectKept       []string               `json:"expect_kept"`
	ExpectExcluded   []contextEvalExclusion `json:"expect_excluded"`
	Budget           int                    `json:"budget"`
	ExpectIncluded   []string               `json:"expect_included"`
	Included         []string               `json:"included"`
	Required         []string               `json:"required"`
	ExpectSufficient *bool                  `json:"expect_sufficient"`
	ExpectMissing    []string               `json:"expect_missing"`
}

// TestContextEvalCorpus runs the context-compilation failure-mode corpus. Each
// case pins the outcome the compiler must produce, and the corpus always
// contains a control that must still succeed (W13.1).
func TestContextEvalCorpus(t *testing.T) {
	cases := loadContextCases(t)
	if len(cases) < 8 {
		t.Fatalf("context eval corpus is unexpectedly small: %d cases", len(cases))
	}
	controls := 0
	for _, tc := range cases {
		tc := tc
		if strings.HasPrefix(tc.Title, "Control:") {
			controls++
		}
		t.Run(tc.ID, func(t *testing.T) {
			switch tc.Kind {
			case "disclosure":
				runDisclosureCase(t, tc)
			case "dedup":
				runDedupCase(t, tc)
			case "budget":
				runBudgetCase(t, tc)
			case "sufficiency":
				runSufficiencyCase(t, tc)
			default:
				t.Fatalf("case %s: unknown kind %q", tc.ID, tc.Kind)
			}
		})
	}
	if controls == 0 {
		t.Fatal("a corpus of only negative cases cannot show that the positive path still works")
	}
}

func loadContextCases(t *testing.T) []contextEvalCase {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "..", "evals", "context", "cases", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("context eval corpus has no cases")
	}
	cases := make([]contextEvalCase, 0, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var tc contextEvalCase
		if err := json.Unmarshal(data, &tc); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if tc.ID == "" || tc.Kind == "" {
			t.Fatalf("%s: a context eval case requires id and kind", f)
		}
		cases = append(cases, tc)
	}
	return cases
}

func runDisclosureCase(t *testing.T, tc contextEvalCase) {
	t.Helper()
	if tc.ExpectFull == nil {
		t.Fatalf("case %s: disclosure requires expect_full", tc.ID)
	}
	level := ParseLevel(tc.Level)
	content, full := level.Disclose(tc.Content)
	if full != *tc.ExpectFull {
		t.Fatalf("case %s (%s): full = %v, want %v", tc.ID, tc.Title, full, *tc.ExpectFull)
	}
	for _, want := range tc.ExpectRevealed {
		if !strings.Contains(content, want) {
			t.Errorf("case %s (%s): level %s did not reveal %q in %q", tc.ID, tc.Title, level, want, content)
		}
	}
	for _, unwanted := range tc.ExpectAbsent {
		if strings.Contains(content, unwanted) {
			t.Errorf("case %s (%s): level %s must not reveal %q", tc.ID, tc.Title, level, unwanted)
		}
	}
}

func runDedupCase(t *testing.T, tc contextEvalCase) {
	t.Helper()
	items := make([]Item, 0, len(tc.Items))
	for _, in := range tc.Items {
		items = append(items, Item{
			Ref: in.Ref, Authority: in.Authority, Score: in.Score, Content: in.Content,
			Projection: in.Projection, CanonicalSource: in.CanonicalSource,
		})
	}
	kept, excluded := DeduplicatePreferCanonical(items)
	gotKept := []string{}
	for _, it := range kept {
		gotKept = append(gotKept, it.Ref)
	}
	if strings.Join(gotKept, ",") != strings.Join(tc.ExpectKept, ",") {
		t.Fatalf("case %s (%s): kept = %v, want %v", tc.ID, tc.Title, gotKept, tc.ExpectKept)
	}
	if len(excluded) != len(tc.ExpectExcluded) {
		t.Fatalf("case %s (%s): excluded = %#v, want %#v", tc.ID, tc.Title, excluded, tc.ExpectExcluded)
	}
	for i, want := range tc.ExpectExcluded {
		if excluded[i].Ref != want.Ref || excluded[i].Reason != want.Reason {
			t.Errorf("case %s (%s): exclusion %d = %#v, want %#v", tc.ID, tc.Title, i, excluded[i], want)
		}
	}
}

func runBudgetCase(t *testing.T, tc contextEvalCase) {
	t.Helper()
	items := make([]Item, 0, len(tc.Items))
	for _, in := range tc.Items {
		cost := in.TokenCost
		if cost == 0 {
			cost = len(in.Content) / 4
		}
		items = append(items, Item{
			Ref: in.Ref, Authority: orDefault(in.Authority, "reference"), Trust: "medium",
			Privacy: "internal", Score: in.Score, Content: in.Content, TokenCost: cost,
			Method: "exact",
		})
	}
	manifest := CompileWithPolicy(tc.ID, items, CompilePolicy{Budget: tc.Budget, Level: ParseLevel(tc.Level)})
	if manifest.EstimatedTokens > tc.Budget {
		t.Fatalf("case %s (%s): budget exceeded: %d > %d", tc.ID, tc.Title, manifest.EstimatedTokens, tc.Budget)
	}
	gotIncluded := []string{}
	for _, it := range manifest.Included {
		gotIncluded = append(gotIncluded, it.Ref)
	}
	if strings.Join(gotIncluded, ",") != strings.Join(tc.ExpectIncluded, ",") {
		t.Fatalf("case %s (%s): included = %v, want %v", tc.ID, tc.Title, gotIncluded, tc.ExpectIncluded)
	}
	byRef := map[string]string{}
	for _, ex := range manifest.Excluded {
		if ex.Detail == "" {
			t.Errorf("case %s (%s): exclusion %s must state a reason detail", tc.ID, tc.Title, ex.Ref)
		}
		byRef[ex.Ref] = ex.Reason
	}
	for _, want := range tc.ExpectExcluded {
		if byRef[want.Ref] != want.Reason {
			t.Errorf("case %s (%s): exclusion reason for %s = %q, want %q", tc.ID, tc.Title, want.Ref, byRef[want.Ref], want.Reason)
		}
	}
}

func runSufficiencyCase(t *testing.T, tc contextEvalCase) {
	t.Helper()
	if tc.ExpectSufficient == nil {
		t.Fatalf("case %s: sufficiency requires expect_sufficient", tc.ID)
	}
	manifest := Manifest{}
	for _, ref := range tc.Included {
		manifest.Included = append(manifest.Included, Item{Ref: ref})
	}
	verdict := EvaluateSufficiency(manifest, tc.Required)
	if verdict.Sufficient != *tc.ExpectSufficient {
		t.Fatalf("case %s (%s): sufficient = %v, want %v", tc.ID, tc.Title, verdict.Sufficient, *tc.ExpectSufficient)
	}
	if strings.Join(verdict.Missing, ",") != strings.Join(tc.ExpectMissing, ",") {
		t.Fatalf("case %s (%s): missing = %v, want %v", tc.ID, tc.Title, verdict.Missing, tc.ExpectMissing)
	}
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
