package docengine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/protocol"
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
		"docs/AUTHORITY_MAP.json":      fmt.Sprintf(`{"version":1,"current_version":%q,"documents":[{"path":"docs/**","role":"canonical"}]}`, protocol.CLIVersion),
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
		"docs/AUTHORITY_MAP.json":      fmt.Sprintf(`{"version":1,"current_version":%q,"documents":[{"path":"docs/**","role":"canonical"}]}`, protocol.CLIVersion),
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
		"docs/AUTHORITY_MAP.json":      fmt.Sprintf(`{"version":1,"current_version":%q,"documents":[{"path":"docs/**","role":"canonical"}]}`, protocol.CLIVersion),
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
		"docs/AUTHORITY_MAP.json":      fmt.Sprintf(`{"version":1,"current_version":%q,"documents":[{"path":"docs/**","role":"canonical"}]}`, protocol.CLIVersion),
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
		"docs/AUTHORITY_MAP.json":      fmt.Sprintf(`{"version":1,"current_version":%q,"documents":[{"path":"docs/**","role":"canonical"},{"path":"*.md","role":"canonical"}]}`, protocol.CLIVersion),
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

// TestVerifyDetectsBrokenBindingSource is the regression test for the finding
// that deleting a document named by a binding produced an identical verify
// report: the control plane reported healthy coverage for knowledge that was no
// longer in the repository.
func TestVerifyDetectsBrokenBindingSource(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"docs/contracts/builtin.json":  `[{"id":"product.vision","version":1,"role":"product-vision"}]`,
		"docs/profiles/builtin.json":   `[{"id":"core-software","version":1,"capabilities":["core"],"contracts":["product.vision"]}]`,
		"docs/contracts/bindings.json": `[{"contract_id":"product.vision","sources":["docs/product/vision.md"],"ownership":"human","authority":"canonical"}]`,
		"docs/AUTHORITY_MAP.json":      `{"version":1,"current_version":"0.6.0","documents":[{"path":"docs/**","role":"canonical"}]}`,
		"docs/product/vision.md":       "# Vision\n\nTarget users and primary outcome.\n",
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	clean, err := VerifyDocs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if hasBindingFinding(clean.Findings) {
		t.Fatalf("intact bindings must not produce a binding finding: %+v", clean.Findings)
	}

	// Remove the bound document: the check has to notice.
	if err := os.Remove(filepath.Join(dir, "docs/product/vision.md")); err != nil {
		t.Fatal(err)
	}
	broken, err := VerifyDocs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !hasBindingFinding(broken.Findings) {
		t.Fatalf("removing a bound document must produce a binding finding, got: %+v", broken.Findings)
	}
}

func hasBindingFinding(findings []VerifyFinding) bool {
	for _, f := range findings {
		if strings.HasPrefix(f.Kind, "binding:") {
			return true
		}
	}
	return false
}
