package agentsurface

import (
	"os"
	"path/filepath"
	"testing"
)

func kindOf(refs []Reference, value string) string {
	for _, r := range refs {
		if r.Value == value {
			return string(r.Kind)
		}
	}
	return ""
}

func TestExtractReferencesClassification(t *testing.T) {
	text := "Run `prumo docs agents verify`, read `docs/PRUMO.md`, inspect " +
		"`internal/agentsurface/ir.go`, call `Readiness()`, and keep `TestGate` passing. " +
		"Ignore `--json`, `https://example.com/x.md`, `runtime/cache`, `{{placeholder}}` and `a == b`."
	refs := ExtractReferences(text)

	cases := map[string]string{
		"prumo docs agents verify":    "command",
		"docs/PRUMO.md":               "doc",
		"internal/agentsurface/ir.go": "path",
		"Readiness()":                 "symbol",
		"TestGate":                    "test",
		"--json":                      "",
		"https://example.com/x.md":    "",
		"runtime/cache":               "",
		"{{placeholder}}":             "",
		"a == b":                      "",
	}
	for value, want := range cases {
		if got := kindOf(refs, value); got != want {
			t.Fatalf("reference %q classified as %q, want %q", value, got, want)
		}
	}
}

func TestStaleReferencesAreFindings(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("docs/present.md", "# Present\n")
	write("internal/demo/demo.go", "package demo\n\nfunc Present() {}\n\nfunc TestDemo() {}\n")

	good := ValidateReferences(root, []Referencer{
		{Target: "AGENTS.md", Text: "See `docs/present.md`, `internal/demo/demo.go`, `Present()` and `TestDemo`."},
	})
	if len(good) != 0 {
		t.Fatalf("expected no findings, got %#v", good)
	}

	bad := ValidateReferences(root, []Referencer{
		{Target: "AGENTS.md", Text: "See `docs/removed.md`, `internal/demo/gone.go`, `Removed()` and `TestRemoved`."},
	})
	if len(bad) != 4 {
		t.Fatalf("expected four stale references, got %d (%#v)", len(bad), bad)
	}
	for _, f := range bad {
		if f.Kind != "stale-reference" || f.Target != "AGENTS.md" {
			t.Fatalf("unexpected finding: %#v", f)
		}
	}
}

func TestCommandReferenceIsCheckedAgainstCLISources(t *testing.T) {
	root, _ := filepath.Abs("../..")
	findings := ValidateReferences(root, []Referencer{
		{Target: "gate", Text: "Run `prumo docs authority` before review."},
	})
	if len(findings) != 0 {
		t.Fatalf("real CLI command must resolve: %#v", findings)
	}
	stale := ValidateReferences(root, []Referencer{
		{Target: "gate", Text: "Run `prumo docs teleport` before review."},
	})
	if len(stale) != 1 {
		t.Fatalf("invented CLI command must be a finding: %#v", stale)
	}
}
