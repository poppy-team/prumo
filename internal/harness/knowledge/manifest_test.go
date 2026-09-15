package knowledge

import (
	"path/filepath"
	"testing"

	coreknowledge "github.com/raillen/prumo/internal/knowledge"
)

func TestStoreManifestMapsVocabularyAndFailsClosed(t *testing.T) {
	s := New()
	SeedRequirement(s, "R-man", "goal")
	SeedEvidence(s, "R-man", "complete", "", "")
	// A memory with no declared sensitivity and one explicitly restricted.
	if err := s.Put(Record{ID: StableIDFor(KindMemory, "note"), Kind: KindMemory, Title: "note",
		Authority: "reference", Trust: "medium", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	restricted := StableIDFor(KindMemory, "secret")
	if err := s.Put(Record{ID: restricted, Kind: KindMemory, Title: "secret",
		Authority: "reference", Trust: "medium", Status: "active", Sensitivity: "restricted"}); err != nil {
		t.Fatal(err)
	}
	unknown := StableIDFor(KindMemory, "odd")
	if err := s.Put(Record{ID: unknown, Kind: KindMemory, Title: "odd",
		Authority: "reference", Trust: "medium", Status: "active", Sensitivity: "top-secret"}); err != nil {
		t.Fatal(err)
	}

	manifest, problems := s.Manifest("2026-09-15T00:00:00Z", "rev")
	if len(problems) != 0 {
		t.Fatalf("unexpected problems: %v", problems)
	}
	// requirement + evidence + three memories.
	if manifest.Counts.Total != 5 {
		t.Fatalf("expected five units, got %d", manifest.Counts.Total)
	}
	// The harness `open_question`/`evidence` spellings must land on the schema
	// vocabulary, not leak through.
	for _, unit := range manifest.Units {
		if unit.Kind == "open_question" {
			t.Fatalf("harness spelling leaked into the manifest: %#v", unit)
		}
	}
	byID := map[coreknowledge.ID]coreknowledge.Unit{}
	for _, unit := range manifest.Units {
		byID[unit.ID] = unit
	}
	// An undeclared sensitivity is internal, never public.
	if got := byID[coreknowledge.ID(StableIDFor(KindMemory, "note"))].Visibility; got != coreknowledge.VisibilityInternal {
		t.Fatalf("undeclared sensitivity must be internal, got %q", got)
	}
	// An unrecognized sensitivity is the most restrictive known class.
	if got := byID[coreknowledge.ID(unknown)].Visibility; got != coreknowledge.VisibilityRestricted {
		t.Fatalf("unknown sensitivity must fail closed, got %q", got)
	}
	// `evidences` becomes the typed `verifies` relationship.
	evidence := byID[coreknowledge.ID(EvidenceID("R-man"))]
	if len(evidence.Relationships) != 1 || evidence.Relationships[0] != coreknowledge.RelVerifies {
		t.Fatalf("harness edge types must map to the typed vocabulary: %#v", evidence.Relationships)
	}
}

func TestMergeKeepsEarlierKnowledgeAndAliases(t *testing.T) {
	first := New()
	SeedRequirement(first, "R-1", "first goal")
	first.Alias(RequirementID("R-1"), "docs/old/goal.md")

	second := New()
	SeedRequirement(second, "R-2", "second goal")
	// A later run must not overwrite an id an earlier run established.
	SeedRequirement(second, "R-1", "rewritten")

	merged := Merge(first, second)
	if _, ok := merged.Resolve(RequirementID("R-1")); !ok {
		t.Fatal("merged store must keep the first run's requirement")
	}
	if _, ok := merged.Resolve("docs/old/goal.md"); !ok {
		t.Fatal("merged store must keep locator bindings")
	}
	record, _ := merged.Get(RequirementID("R-1"))
	if record.Title != "first goal" {
		t.Fatalf("a later run must not overwrite earlier knowledge, got %q", record.Title)
	}
	if _, ok := merged.Resolve(RequirementID("R-2")); !ok {
		t.Fatal("merged store must gain the second run's requirement")
	}
}

func TestSaveLoadKeepsAliases(t *testing.T) {
	s := New()
	SeedRequirement(s, "R-alias", "goal")
	s.Alias(RequirementID("R-alias"), "legacy-locator")
	path := filepath.Join(t.TempDir(), "knowledge-R-alias.json")
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := loaded.Resolve("legacy-locator"); !ok {
		t.Fatal("aliases must survive a save/load round trip")
	}
}

func TestSlugNormalizesArbitraryLabels(t *testing.T) {
	cases := map[string]string{
		"Ship the Harness!": "ship-the-harness",
		"R-seed":            "r-seed",
		"  spaced  ":        "spaced",
		"a/b\\c":            "a-b-c",
		"":                  "",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
	if id := StableIDFor(KindRequirement, "R-Ünicode"); !IsStableID(id) {
		t.Fatalf("a non-ASCII label must still produce a valid stable id, got %q", id)
	}
}
