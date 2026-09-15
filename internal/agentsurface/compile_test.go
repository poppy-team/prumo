package agentsurface

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTempIR scaffolds a minimal repository with an IR and a curated AGENTS.md
// carrying the managed region.
func writeTempIR(t *testing.T, ir IR) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, IRPath, mustJSON(t, ir))
	writeFile(t, root, "AGENTS.md", "# AGENTS.md\n\n<!-- prumo:begin agents-core -->\n<!-- prumo:end agents-core -->\n")
	return root
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func smallIR() IR {
	return IR{Version: 1, TokenBudget: Budget{Root: 500, Scope: 500}, Rules: []Rule{
		{ID: "first", Precedence: 10, Text: "Preserve provider-neutral core policy."},
		{ID: "second", Precedence: 20, Text: "Keep entrypoints short."},
	}}
}

// TestInvariantChangeRequiresRegeneration pins the W16 exit criterion: editing
// the canonical IR makes the compiled surface stale until it is rebuilt.
func TestInvariantChangeRequiresRegeneration(t *testing.T) {
	ir := smallIR()
	root := writeTempIR(t, ir)

	if _, err := Build(root, ""); err != nil {
		t.Fatal(err)
	}
	report, err := Verify(root)
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK {
		t.Fatalf("expected a clean verify after build, got %#v", report.Findings)
	}

	ir.Rules = append(ir.Rules, Rule{ID: "third", Precedence: 30, Text: "Add tests for core behavior."})
	writeFile(t, root, IRPath, mustJSON(t, ir))

	report, err = Verify(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.OK || !hasFindingKind(report.Findings, "surface-stale") {
		t.Fatalf("expected surface-stale after an IR change, got %#v", report.Findings)
	}

	if _, err := Build(root, ""); err != nil {
		t.Fatal(err)
	}
	report, err = Verify(root)
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK {
		t.Fatalf("expected a clean verify after rebuild, got %#v", report.Findings)
	}
	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Add tests for core behavior.") {
		t.Fatalf("rebuilt region must contain the new rule:\n%s", data)
	}
}

func TestTokenBudgetIsEnforced(t *testing.T) {
	ir := IR{Version: 1, TokenBudget: Budget{Root: 2}, Rules: []Rule{
		{ID: "long", Text: strings.Repeat("verbose instruction text ", 40)},
	}}
	root := writeTempIR(t, ir)
	report, err := Verify(root)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFindingKind(report.Findings, "token-budget-exceeded") {
		t.Fatalf("expected a budget finding, got %#v", report.Findings)
	}
}

func TestStandaloneSurfacesNeverBecomeCanonical(t *testing.T) {
	root := writeTempIR(t, smallIR())
	result, err := Build(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.OutDir != DefaultOutDir {
		t.Fatalf("unexpected out dir: %s", result.OutDir)
	}
	for _, rel := range []string{".github/copilot-instructions.md", "CLAUDE.md", ".cursor/rules/prumo-root.mdc"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
			t.Fatalf("vendor surface %s must not be written into the repository", rel)
		}
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(DefaultOutDir), "CLAUDE.md")); err != nil {
		t.Fatalf("expected the claude projection under runtime: %v", err)
	}
}

func TestEverySurfaceCarriesVerifiableProvenance(t *testing.T) {
	root := writeTempIR(t, smallIR())
	ir, err := LoadIR(root)
	if err != nil {
		t.Fatal(err)
	}
	surfaces, err := RenderAll(ir)
	if err != nil {
		t.Fatal(err)
	}
	if len(surfaces) != len(Adapters()) {
		t.Fatalf("expected one root surface per adapter, got %d", len(surfaces))
	}
	for _, s := range surfaces {
		body, recorded, ok := StripProvenance(s.Content)
		if !ok {
			t.Fatalf("%s surface has no generated marker", s.Adapter)
		}
		if recorded != s.Fingerprint || Fingerprint(body) != s.Fingerprint {
			t.Fatalf("%s surface fingerprint mismatch", s.Adapter)
		}
		if s.Adapter == "claude" && s.Scope == "" {
			// The Claude root surface is an import, never a second copy of the
			// canonical instructions.
			if !strings.Contains(s.Content, "@AGENTS.md") {
				t.Fatal("claude root surface must import AGENTS.md")
			}
			if strings.Contains(s.Content, "Preserve provider-neutral core policy.") {
				t.Fatal("claude root surface must not duplicate the rules")
			}
			continue
		}
		for _, want := range []string{"Preserve provider-neutral core policy.", "Keep entrypoints short."} {
			if !strings.Contains(s.Content, want) {
				t.Fatalf("%s surface is missing rule text %q", s.Adapter, want)
			}
		}
	}
}

func TestNestedScopeRendersScopedSurfaces(t *testing.T) {
	ir := smallIR()
	ir.Rules = append(ir.Rules, Rule{
		ID: "scoped", Scope: "internal/harness", Precedence: 10,
		Text: "Harness changes update `docs/harness/gap-register.md`.", Sources: []string{"docs/harness/gap-register.md"},
	})
	root := writeTempIR(t, ir)
	if err := os.MkdirAll(filepath.Join(root, "docs", "harness"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "docs/harness/gap-register.md", "# Gap register\n")

	ctx, err := Explain(ir, "cursor", "internal/harness")
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Path != ".cursor/rules/prumo-internal-harness.mdc" {
		t.Fatalf("unexpected scoped path: %s", ctx.Path)
	}
	if !strings.Contains(ctx.Content, "globs: \"internal/harness/**\"") {
		t.Fatalf("scoped adapter must declare its glob:\n%s", ctx.Content)
	}
	if findings := ValidateReferences(root, []Referencer{{Target: ctx.Path, Text: ctx.Content}}); len(findings) != 0 {
		t.Fatalf("scoped surface references must resolve: %#v", findings)
	}
}

// TestAffectedSurfacesAreScoped pins impact precision: a change inside a nested
// scope invalidates only the surfaces that render that scope, while a change to
// a rule source invalidates the surfaces carrying the rule (W16.15).
func TestAffectedSurfacesAreScoped(t *testing.T) {
	ir := smallIR()
	ir.Rules[1].Sources = []string{"docs/development/testing-strategy.md"}
	ir.Rules = append(ir.Rules, Rule{
		ID: "scoped", Scope: "internal/harness", Precedence: 10,
		Text: "Harness changes update `docs/harness/gap-register.md`.", Sources: []string{"docs/harness/gap-register.md"},
	})

	// A root rule source is carried by every surface, nested scopes included.
	for _, s := range AffectedSurfaces(ir, []string{"docs/development/testing-strategy.md"}) {
		if s.Scope != "" && s.Scope != "internal/harness" {
			t.Fatalf("unexpected scope %s", s.Scope)
		}
	}

	// A scoped rule source reaches only the surfaces that render that scope.
	scoped := AffectedSurfaces(ir, []string{"docs/harness/gap-register.md"})
	if len(scoped) == 0 {
		t.Fatal("expected the harness surfaces to be affected")
	}
	for _, s := range scoped {
		if s.Scope != "internal/harness" {
			t.Fatalf("root surface %s must not be invalidated by a scoped rule source", s.Path)
		}
	}

	// A code change inside a scope invalidates that scope's surfaces only.
	for _, s := range AffectedSurfaces(ir, []string{"internal/harness/agent/turn.go"}) {
		if s.Scope != "internal/harness" {
			t.Fatalf("unscoped surface %s must not be invalidated by a scoped change", s.Path)
		}
	}

	// Editing the IR invalidates every compiled surface.
	if len(AffectedSurfaces(ir, []string{IRPath})) != len(ir.Scopes())*len(Adapters()) {
		t.Fatal("an IR change must invalidate every surface")
	}
}

func hasFindingKind(findings []Finding, kind string) bool {
	for _, f := range findings {
		if f.Kind == kind {
			return true
		}
	}
	return false
}
