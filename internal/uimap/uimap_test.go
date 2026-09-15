package uimap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resolverStub answers symbol existence from a fixed list, so validation can be
// tested without a repository on disk.
type resolverStub map[string]bool

func (r resolverStub) Resolves(symbol, _ string) bool { return r[symbol] }

// sampleMap is a minimal map that passes every rule, so each test can break
// exactly one thing and prove the rule is the reason.
func sampleMap() Map {
	return Map{
		ID:           "ui:test-shell",
		Version:      1,
		Surface:      "Test surface",
		Platforms:    []string{"tui"},
		StatesSource: "docs/ui-ux/state-matrix.json",
		TokensSource: "docs/ui-ux/design-tokens.json",
		Tree: []Node{{
			ID:   "node:shell",
			Kind: "shell",
			Slot: Slot{Region: "root", Order: 0, Align: "stretch"},
			Children: []Node{
				{
					ID: "node:palette", Kind: "pane", Label: "Palette",
					Slot:           Slot{Region: "body", Order: 0, Align: "stretch"},
					States:         []string{"default"},
					Tokens:         []string{"token:color.text"},
					Implementation: &Impl{Symbol: "tui.Palette", Status: "implemented"},
				},
				{
					ID: "node:goal", Kind: "input", Label: "goal",
					Slot:           Slot{Region: "body", Order: 0, Align: "stretch", Condition: "stage = goal"},
					Implementation: &Impl{Symbol: "tui.Goal", Status: "implemented"},
				},
			},
			Implementation: &Impl{Symbol: "tui.Model", Status: "implemented"},
		}},
		Edges: []Edge{{Kind: "focus", From: "node:palette", To: "node:goal", Label: "opens the prompt"}},
	}
}

func sampleVocabulary() Vocabulary {
	return Vocabulary{
		States:        map[string]bool{"default": true, "focus": true},
		Tokens:        map[string]bool{"token:color.text": true},
		MatrixSurface: "Test surface",
	}
}

func sampleResolver() Resolver {
	return resolverStub{"tui.Model": true, "tui.Palette": true, "tui.Goal": true}
}

func validate(t *testing.T, m Map) Report {
	t.Helper()
	return Validate(".", m, sampleVocabulary(), sampleResolver())
}

func codes(report Report) []string {
	out := make([]string, 0, len(report.Findings))
	for _, f := range report.Findings {
		out = append(out, f.Code)
	}
	return out
}

func wantCode(t *testing.T, report Report, code string) {
	t.Helper()
	for _, f := range report.Findings {
		if f.Code == code {
			if f.Severity != SeverityError {
				t.Fatalf("finding %s is %s, want an error", code, f.Severity)
			}
			if f.Message == "" {
				t.Fatalf("finding %s carries no message", code)
			}
			return
		}
	}
	t.Fatalf("expected finding %s, got %v", code, codes(report))
}

func TestValidMapPasses(t *testing.T) {
	report := validate(t, sampleMap())
	if !report.OK() {
		t.Fatalf("sample map should pass, got %v", codes(report))
	}
	if report.Elements != 3 || report.Edges != 1 {
		t.Fatalf("counted %d elements and %d edges, want 3 and 1", report.Elements, report.Edges)
	}
}

func TestPositionIsMandatory(t *testing.T) {
	m := sampleMap()
	m.Tree[0].Children[0].Slot = Slot{}
	wantCode(t, validate(t, m), "missing-position")
}

func TestConditionalSiblingsMayShareAPosition(t *testing.T) {
	// The sample map already has two body/0 children, one of them conditional;
	// removing the condition must make the sharing ambiguous.
	m := sampleMap()
	m.Tree[0].Children[1].Slot.Condition = ""
	wantCode(t, validate(t, m), "ambiguous-order")
}

func TestUserVisibleKindsNeedALabel(t *testing.T) {
	m := sampleMap()
	m.Tree[0].Children[0].Label = ""
	wantCode(t, validate(t, m), "missing-label")
	// A region is structure and may be nameless.
	m = sampleMap()
	m.Tree[0].Children[1].Kind = "region"
	m.Tree[0].Children[1].Label = ""
	if report := validate(t, m); !report.OK() {
		t.Fatalf("a nameless region should be allowed, got %v", codes(report))
	}
}

func TestStatesComeFromTheMatrix(t *testing.T) {
	m := sampleMap()
	m.Tree[0].Children[0].States = []string{"hover"}
	wantCode(t, validate(t, m), "unknown-state")
}

func TestTokensMustBeDeclared(t *testing.T) {
	m := sampleMap()
	m.Tree[0].Children[0].Tokens = []string{"token:color.invented"}
	wantCode(t, validate(t, m), "unknown-token")
}

func TestImplementationMustResolveOrDeclareItsAbsence(t *testing.T) {
	m := sampleMap()
	m.Tree[0].Children[0].Implementation = &Impl{Symbol: "tui.Gone", Status: "implemented"}
	wantCode(t, validate(t, m), "unresolved-symbol")

	// An element that is described but not built is a legitimate state of the
	// map, and its symbol is not checked — there is nothing to resolve. The
	// reason is what makes the absence reviewable.
	m = sampleMap()
	m.Tree[0].Children[0].Implementation = &Impl{Symbol: "tui.Gone", Status: "not-implemented",
		AbsentReason: "the underlying protocol has no op for it yet"}
	if report := validate(t, m); !report.OK() {
		t.Fatalf("not-implemented with a reason must pass, got %v", codes(report))
	}

	m = sampleMap()
	m.Tree[0].Children[0].Implementation = &Impl{Status: "not-implemented"}
	wantCode(t, validate(t, m), "absent-without-reason")

	m = sampleMap()
	m.Tree[0].Children[0].Implementation = nil
	wantCode(t, validate(t, m), "missing-implementation")

	m = sampleMap()
	m.Tree[0].Children[0].Implementation = &Impl{Symbol: "tui.Palette", Status: "shipped"}
	wantCode(t, validate(t, m), "implementation-status")
}

func TestDuplicateIdsAndBadEdgesAreCaught(t *testing.T) {
	m := sampleMap()
	m.Tree[0].Children[1].ID = m.Tree[0].Children[0].ID
	wantCode(t, validate(t, m), "duplicate-id")

	m = sampleMap()
	m.Edges = []Edge{{Kind: "focus", From: "node:palette", To: "node:absent"}}
	wantCode(t, validate(t, m), "edge-endpoint")

	m = sampleMap()
	m.Edges = []Edge{{Kind: "focus", From: "node:palette", To: "node:palette"}}
	wantCode(t, validate(t, m), "edge-self")

	m = sampleMap()
	m.Edges = []Edge{{Kind: "teleports", From: "node:palette", To: "node:goal"}}
	wantCode(t, validate(t, m), "edge-kind")

	m = sampleMap()
	m.Edges = []Edge{
		{Kind: "focus", From: "node:palette", To: "node:goal"},
		{Kind: "focus", From: "node:palette", To: "node:goal"},
	}
	wantCode(t, validate(t, m), "edge-duplicate")
}

func TestSurfaceMustMatchTheStateMatrix(t *testing.T) {
	m := sampleMap()
	m.Surface = "A different surface"
	wantCode(t, validate(t, m), "surface-mismatch")
}

func TestGeometryNeedsAPlatformAndAUnit(t *testing.T) {
	m := sampleMap()
	m.Tree[0].Children[0].Slot.Geometry = &Geometry{Platform: "desktop", Width: 10}
	wantCode(t, validate(t, m), "geometry-platform")
	wantCode(t, validate(t, m), "geometry-unit")
}

func TestMultipleRootsAreRefused(t *testing.T) {
	m := sampleMap()
	m.Tree = append(m.Tree, Node{ID: "node:second", Kind: "shell", Slot: Slot{Region: "root", Align: "stretch"}})
	wantCode(t, validate(t, m), "multiple-roots")
}

// --- derivation and merge ----------------------------------------------------

func TestGoDeriverFindsTypesMethodsFieldsAndSkipsTests(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "ui", "shell.go"), `package ui

type Model struct{ Palette *Palette }

type Palette struct{}

func NewPalette() *Palette { return &Palette{} }

func (m *Model) View() string { return "" }

const DefaultCapacity = 8
`)
	write(t, filepath.Join(root, "ui", "shell_test.go"), `package ui

func TestSomethingView(t *testing.T) {}
`)
	derived, err := GoSymbolDeriver{}.Derive(root, []string{"ui/**/*.go"})
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	set := derived.SymbolSet()
	for _, want := range []string{"ui.Model", "ui.Palette", "ui.NewPalette", "ui.Model.View", "ui.Model.Palette", "ui.DefaultCapacity"} {
		if !set[want] {
			t.Errorf("deriver missed %s; got %v", want, derived.Symbols)
		}
	}
	if set["ui.TestSomethingView"] {
		t.Error("the deriver read a test file; tests are not the interface")
	}
	if len(derived.Sources) != 1 {
		t.Errorf("read %d sources, want 1: %v", len(derived.Sources), derived.Sources)
	}
}

func TestMergePrefersDeclarationAndOnlyOffersElements(t *testing.T) {
	derived := Derived{
		Sources:  []string{"ui/shell.go"},
		Symbols:  []string{"ui.NewThingView", "ui.DefaultPaletteCapacity"},
		Elements: []string{"ui.NewThingView"},
	}
	mode := func(mode string) MergeResult { return Merge(sampleMap(), derived, mode) }

	if got := mode("off").Undeclared; len(got) != 0 {
		t.Fatalf("derivation off still reported %v", got)
	}
	verify := mode("verify-only")
	if len(verify.Undeclared) != 1 || verify.Undeclared[0] != "ui.NewThingView" {
		t.Fatalf("verify-only reported %v", verify.Undeclared)
	}
	if len(verify.Map.Tree) != 1 {
		t.Fatal("verify-only must not add elements to the tree")
	}
	filled := mode("fill-gaps")
	// The gap region hangs under the declared root: a map describes one surface,
	// and a mode that produced a second root would produce an invalid map.
	if len(filled.Map.Tree) != 1 {
		t.Fatalf("fill-gaps produced %d roots, want the declared one", len(filled.Map.Tree))
	}
	children := filled.Map.Tree[0].Children
	added, ok := children[len(children)-1], len(children) > 0
	if !ok || added.ID != "node:derived-unmapped" {
		t.Fatalf("fill-gaps did not append the derived region under the root: %v", children)
	}
	if added.Origin != "derived" {
		t.Fatalf("derived region origin = %q", added.Origin)
	}
	if len(added.Children) != 1 || added.Children[0].Implementation.Symbol != "ui.NewThingView" {
		t.Fatalf("derived region did not carry the undeclared element: %+v", added.Children)
	}
	if strings.Contains(strings.ToLower(added.ID), "capacity") {
		t.Fatalf("a constant was offered as an interface element: %s", added.ID)
	}
}

// --- config ------------------------------------------------------------------

func TestConfigAutoEnablesOnADeclaredInterface(t *testing.T) {
	withUI := t.TempDir()
	write(t, filepath.Join(withUI, "prumo.json"), `{"version":3,"project":{"type":["cli","tui"]}}`)
	cfg, err := ResolveConfig(withUI)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !cfg.Enabled {
		t.Fatalf("a project declaring a tui type must get an interface map: %s", cfg.Reason)
	}
	if !cfg.Wants(TargetDeveloper) || !cfg.Wants(TargetSite) || !cfg.Wants(TargetAgent) {
		t.Fatalf("default targets = %v", cfg.Targets)
	}
	if cfg.Derivation != "verify-only" {
		t.Fatalf("default derivation = %q, want the conservative one", cfg.Derivation)
	}

	cliOnly := t.TempDir()
	write(t, filepath.Join(cliOnly, "prumo.json"), `{"version":3,"project":{"type":["cli","library"]}}`)
	cfg, err = ResolveConfig(cliOnly)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.Enabled {
		t.Fatal("a CLI-only project must not be given an interface map")
	}
	if !strings.Contains(cfg.Reason, "no interface") {
		t.Fatalf("the skip reason does not say why: %q", cfg.Reason)
	}
}

func TestConfigHonoursGranularTargets(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "prumo.json"), `{
	  "version": 3,
	  "project": {"type": ["web"]},
	  "ui": {"interface_map": {"targets": ["agent", "developer"]}}
	}`)
	cfg, err := ResolveConfig(root)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.Wants(TargetSite) {
		t.Fatal("the site target was requested but the project excluded it")
	}
	if !cfg.Wants(TargetDeveloper) || !cfg.Wants(TargetAgent) {
		t.Fatalf("targets = %v", cfg.Targets)
	}
}

func TestConfigRefusesSilentNoOps(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "prumo.json"), `{"version":3,"project":{"type":["tui"]},"ui":{"interface_map":{"targets":[]}}}`)
	if _, err := ResolveConfig(root); err == nil {
		t.Fatal("an empty target list must be refused: enabled with no output is a silent no-op")
	}

	root = t.TempDir()
	write(t, filepath.Join(root, "prumo.json"), `{"version":3,"project":{"type":["tui"]},"ui":{"interface_map":{"targets":["print"]}}}`)
	if _, err := ResolveConfig(root); err == nil {
		t.Fatal("an unknown target must be refused")
	}

	root = t.TempDir()
	write(t, filepath.Join(root, "prumo.json"), `{"version":3,"project":{"type":["tui"]},"ui":{"interface_map":{"enabled":false}}}`)
	cfg, err := ResolveConfig(root)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cfg.Enabled || cfg.ConfigSummary() == "" {
		t.Fatal("an explicit disable must be honoured and must still explain itself")
	}
}

// --- impact ------------------------------------------------------------------

func TestImpactWalksBothDirections(t *testing.T) {
	impact, err := sampleMap().ImpactOf("node:palette")
	if err != nil {
		t.Fatalf("impact: %v", err)
	}
	reached := map[string]Transit{}
	for _, t2 := range impact.Transit {
		reached[t2.NodeID] = t2
	}
	for _, want := range []string{"node:shell", "node:goal"} {
		if _, ok := reached[want]; !ok {
			t.Fatalf("impact of node:palette missed %s: %v", want, impact.Transit)
		}
	}
	if !strings.Contains(reached["node:goal"].Via, "focus") {
		t.Fatalf("the reason for reaching node:goal is %q", reached["node:goal"].Via)
	}
	if _, err := sampleMap().ImpactOf("node:absent"); err == nil {
		t.Fatal("impact on an unknown element must fail rather than return nothing")
	}
}

// --- projections -------------------------------------------------------------

func TestProjectionsCarryARevisionAndDetectStaleness(t *testing.T) {
	root := t.TempDir()
	cfg := Config{Enabled: true, Targets: Targets(), Derivation: "verify-only", Path: "docs/ui-ux/interface-map.json"}
	m := sampleMap()
	m.SourceDigest = "sha256:test"
	m.SourcePath = "docs/ui-ux/interface-map.json"
	projections := Project(m, cfg, validate(t, m), Derived{})

	if len(projections) == 0 {
		t.Fatal("no projections produced")
	}
	for _, p := range projections {
		if !strings.HasPrefix(p.Content, "<!-- prumo:interface-map") {
			t.Errorf("projection %s does not carry its revision", p.Path)
		}
		if !strings.Contains(p.Content, "sha256:test") {
			t.Errorf("projection %s does not name the map revision it came from", p.Path)
		}
	}

	missing := CheckProjections(root, projections)
	if len(missing) != len(projections) {
		t.Fatalf("expected every projection to be reported missing, got %d", len(missing))
	}
	if err := WriteProjections(root, projections); err != nil {
		t.Fatalf("write: %v", err)
	}
	if findings := CheckProjections(root, projections); len(findings) != 0 {
		t.Fatalf("freshly written projections reported as %v", findings)
	}

	// A changed map must make them stale, which is the whole point of the
	// digest: a reference describing the previous interface misleads its reader.
	m.SourceDigest = "sha256:changed"
	stale := CheckProjections(root, Project(m, cfg, validate(t, m), Derived{}))
	if len(stale) != len(projections) {
		t.Fatalf("expected %d stale findings, got %d", len(projections), len(stale))
	}
}

func TestAgentSurfaceIsBounded(t *testing.T) {
	cfg := Config{Enabled: true, Targets: []Target{TargetAgent}, Derivation: "verify-only"}
	m := sampleMap()
	m.SourcePath = "docs/ui-ux/interface-map.json"
	// Grow well past the surface budget.
	for i := 0; i < agentSurfaceBudget+10; i++ {
		m.Tree[0].Children = append(m.Tree[0].Children, Node{
			ID: "node:pane" + strings.Repeat("x", i%3), Kind: "pane", Label: "Pane",
			Slot:           Slot{Region: "body", Order: 100 + i, Align: "stretch"},
			Implementation: &Impl{Symbol: "tui.Palette", Status: "implemented"},
		})
	}
	var surface Projection
	for _, p := range Project(m, cfg, validate(t, m), Derived{}) {
		if p.Path == "agent/interface-map.md" {
			surface = p
		}
	}
	if surface.Content == "" {
		t.Fatal("no agent surface")
	}
	if !strings.Contains(surface.Content, "further elements omitted") {
		t.Fatal("a bounded surface must say what it omitted rather than truncate silently")
	}
	if strings.Count(surface.Content, "\n- ") > agentSurfaceBudget+20 {
		t.Fatal("the agent surface exceeded its budget by more than the surrounding prose")
	}
}

// --- helpers -----------------------------------------------------------------

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
