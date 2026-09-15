package docengine

import (
	"os"
	"path/filepath"
	"testing"
)

func verifyFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestVerifyFindsBrokenLinks(t *testing.T) {
	root := verifyFixture(t, map[string]string{
		"AGENTS.md":                    "# AGENTS.md\n\nSee [present](docs/present.md) and [gone](docs/gone.md).\n",
		"docs/present.md":              "# Present\n\nBack to [agents](../AGENTS.md).\n",
		"docs/contracts/builtin.json":  `[]`,
		"docs/contracts/bindings.json": `[]`,
		"docs/AUTHORITY_MAP.json":      `{"version":1,"current_version":"0.6.0","documents":[{"path":"docs/**","role":"canonical"}]}`,
	})
	report, err := VerifyDocs(root)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]int{}
	for _, f := range report.Findings {
		kinds[f.Kind]++
	}
	if kinds["broken-link"] != 1 {
		t.Fatalf("expected exactly one broken link, got %#v", report.Findings)
	}
	if report.OK {
		t.Fatal("report must not be OK with a broken link")
	}
}

func TestVerifyFindsClaimDrift(t *testing.T) {
	root := verifyFixture(t, map[string]string{
		"docs/plan.md":                 "# Plan\n\nAST-aware editing remains future work; only `regions.go` exists today.\n",
		"internal/harness/x/x.go":      "package x\n\n// regions.go is implemented here.\nvar Regions = 1\n",
		"docs/contracts/builtin.json":  `[]`,
		"docs/contracts/bindings.json": `[]`,
		"docs/AUTHORITY_MAP.json":      `{"version":1,"current_version":"0.6.0","documents":[{"path":"docs/**","role":"canonical"}]}`,
	})
	report, err := VerifyDocs(root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasVerifyFinding(report.Findings, "claim-drift") {
		t.Fatalf("expected claim drift for `regions.go`, got %#v", report.Findings)
	}

	// Control: the same claim about something that does not exist is legitimate.
	control := verifyFixture(t, map[string]string{
		"docs/plan.md":                 "# Plan\n\n`NotImplementedThing` remains future work.\n",
		"docs/contracts/builtin.json":  `[]`,
		"docs/contracts/bindings.json": `[]`,
		"docs/AUTHORITY_MAP.json":      `{"version":1,"current_version":"0.6.0","documents":[{"path":"docs/**","role":"canonical"}]}`,
	})
	controlReport, err := VerifyDocs(control)
	if err != nil {
		t.Fatal(err)
	}
	if hasVerifyFinding(controlReport.Findings, "claim-drift") {
		t.Fatalf("a claim about genuinely missing work must not be drift: %#v", controlReport.Findings)
	}
}

func TestVerifyFindsManagedRegionProblems(t *testing.T) {
	root := verifyFixture(t, map[string]string{
		"AGENTS.md":                    "# AGENTS\n\n<!-- prumo:begin agents-core -->\n- rule\n",
		"docs/contracts/builtin.json":  `[]`,
		"docs/contracts/bindings.json": `[]`,
		"docs/AUTHORITY_MAP.json":      `{"version":1,"current_version":"0.6.0","documents":[{"path":"docs/**","role":"canonical"}]}`,
	})
	report, err := VerifyDocs(root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasVerifyFinding(report.Findings, "region-marker-unclosed") {
		t.Fatalf("expected an unclosed region finding, got %#v", report.Findings)
	}
}

func TestVerifyRepoIsClean(t *testing.T) {
	root, _ := filepath.Abs("../..")
	report, err := VerifyDocs(root)
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK {
		for _, f := range report.Findings {
			t.Errorf("%s: %s:%d %s", f.Kind, f.Path, f.Line, f.Detail)
		}
		t.Fatalf("repository documentation verification failed with %d findings", len(report.Findings))
	}
	if len(report.Checks) != len(VerifyChecks) {
		t.Fatalf("report must state which checks ran: %#v", report.Checks)
	}
}

// TestVerifyStrictRequiresSemanticReadiness is the W19.9 gate: a repository
// whose contracts are covered only lexically passes `verify` but must fail
// `verify --strict`.
func TestVerifyStrictRequiresSemanticReadiness(t *testing.T) {
	root := verifyFixture(t, map[string]string{
		"AGENTS.md":                    "# AGENTS.md\n",
		"docs/contracts/builtin.json":  `[{"id":"product.vision","version":1,"role":"product-vision","required_knowledge":["target users"]}]`,
		"docs/profiles/builtin.json":   `[{"id":"core-software","version":1,"capabilities":["core"],"contracts":["product.vision"]}]`,
		"docs/contracts/bindings.json": `[{"contract_id":"product.vision","sources":["docs/vision.md"],"answered_questions":["Why?"]}]`,
		"docs/vision.md":               "# Vision\n",
		"docs/AUTHORITY_MAP.json":      `{"version":1,"current_version":"0.6.0","documents":[{"path":"docs/**","role":"canonical"},{"path":"*.md","role":"canonical"}]}`,
	})
	lenient, err := VerifyDocs(root)
	if err != nil {
		t.Fatal(err)
	}
	if !lenient.OK {
		t.Fatalf("lexical coverage alone must pass the deterministic checks: %#v", lenient.Findings)
	}
	strict, err := VerifyDocsStrict(root)
	if err != nil {
		t.Fatal(err)
	}
	if strict.OK {
		t.Fatal("strict verification must fail when no contract is semantically verified")
	}
	if !hasVerifyFinding(strict.Findings, "semantic-unverified") {
		t.Fatalf("expected a semantic-unverified finding: %#v", strict.Findings)
	}
	if len(strict.Checks) != len(VerifyChecks)+1 {
		t.Fatalf("strict must report the extra check it ran: %#v", strict.Checks)
	}
}

func hasVerifyFinding(findings []VerifyFinding, kind string) bool {
	for _, f := range findings {
		if f.Kind == kind {
			return true
		}
	}
	return false
}
