package contextv2

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDedupPrefersCanonicalAndExplainsDrops(t *testing.T) {
	items := []Item{
		{Ref: "docs/reference/cli.md", Authority: "canonical"},
		{Ref: "docs/generated/cli.md", Authority: "reference", Projection: true, CanonicalSource: "docs/reference/cli.md"},
		{Ref: "docs/reference/cli.md", Authority: "canonical"},
		{Ref: "docs/other.md"},
		{Ref: "docs/standalone.md", Projection: true, CanonicalSource: "docs/absent.md"},
	}
	kept, excluded := DeduplicatePreferCanonical(items)
	if len(kept) != 3 {
		t.Fatalf("expected three survivors, got %#v", kept)
	}
	// The projection is dropped in favour of its canonical source; the repeated
	// ref is dropped once; a projection whose source is absent survives.
	reasons := map[string]string{}
	for _, ex := range excluded {
		reasons[ex.Ref] = ex.Reason
	}
	if reasons["docs/generated/cli.md"] != ReasonProjection {
		t.Fatalf("projection must be dropped in favour of canonical, got %#v", reasons)
	}
	if reasons["docs/reference/cli.md"] != ReasonDuplicate {
		t.Fatalf("a repeated ref must be dropped as a duplicate, got %#v", reasons)
	}
	if _, ok := reasons["docs/standalone.md"]; ok {
		t.Fatal("a projection whose canonical source is absent must survive")
	}
	if kept[0].Ref != "docs/reference/cli.md" {
		t.Fatalf("input order must be preserved, got %#v", kept)
	}
}

func TestCompileRecordsSelectionReasonsForEveryIncludedItem(t *testing.T) {
	m := CompileWithPolicy("R", []Item{
		{Ref: "goal", Authority: "canonical", Trust: "high", Privacy: "internal", Score: 1, TokenCost: 5, Content: "ship it"},
		{Ref: "file:a.md", Authority: "reference", Trust: "medium", Privacy: "internal", Score: 0.5, TokenCost: 5, Reason: "workspace file"},
	}, CompilePolicy{Budget: 1000, Level: LevelSummary})
	if len(m.Included) != 2 {
		t.Fatalf("expected both candidates to pack, got %#v", m.Included)
	}
	for _, it := range m.Included {
		if it.Reason == "" {
			t.Fatalf("every included item must state why it was selected: %#v", it)
		}
	}
	if m.ID != "CTX-R" {
		t.Fatalf("a manifest must carry its own id, got %q", m.ID)
	}
	if m.Level != LevelSummary {
		t.Fatalf("the level must be recorded, got %q", m.Level)
	}
}

func TestExplainSurfacesUnexplainedSelections(t *testing.T) {
	explanation := Explain(Manifest{
		RunID: "R", Included: []Item{{Ref: "mystery"}},
	}, 0, nil)
	if len(explanation.Notes) != 1 {
		t.Fatalf("an unexplained selection must be reported, not hidden: %#v", explanation.Notes)
	}
	if explanation.Included[0].Reason == "" {
		t.Fatal("the explanation must still state something for every included item")
	}
}

func TestSufficiencyReportsMissingObligations(t *testing.T) {
	m := Manifest{Included: []Item{{Ref: "file:a.md"}}}
	ok := EvaluateSufficiency(m, []string{"file:a.md"})
	if !ok.Sufficient || len(ok.Missing) != 0 {
		t.Fatalf("a satisfied requirement set must be sufficient: %#v", ok)
	}
	missing := EvaluateSufficiency(m, []string{"file:a.md", "file:b.md"})
	if missing.Sufficient {
		t.Fatalf("a missing obligation must make the compilation insufficient: %#v", missing)
	}
	if len(missing.Missing) != 1 || missing.Missing[0] != "file:b.md" {
		t.Fatalf("the absent obligation must be named: %#v", missing.Missing)
	}
}

func TestManifestRoundTripThroughRuntimeState(t *testing.T) {
	root := t.TempDir()
	saved, err := SaveManifest(root, Manifest{Version: 2, ID: "CTX-R-rt", RunID: "R-rt", Level: LevelOutline})
	if err != nil {
		t.Fatal(err)
	}
	// The manifest lives under runtime state: it is never a canonical artifact.
	want := filepath.Join(root, ".prumo", "runtime", "harness", "context-R-rt.json")
	if saved != want {
		t.Fatalf("unexpected manifest path %q", saved)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatal(err)
	}
	// Both spellings of the id resolve to the same compilation.
	for _, id := range []string{"CTX-R-rt", "ctx-R-rt", "R-rt"} {
		loaded, err := LoadManifest(root, id)
		if err != nil {
			t.Fatalf("LoadManifest(%q): %v", id, err)
		}
		if loaded.RunID != "R-rt" {
			t.Fatalf("LoadManifest(%q) returned run %q", id, loaded.RunID)
		}
	}
	if _, err := LoadManifest(root, "CTX-absent"); err == nil {
		t.Fatal("loading an unknown manifest must fail")
	}
}
