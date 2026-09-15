package knowledge

import "testing"

func TestBuildManifestIsDeterministicAndDerived(t *testing.T) {
	zeta, _ := NewID("requirement.zeta")
	alpha, _ := NewID("claim.alpha")
	manifest, problems := BuildManifest("2026-09-15T00:00:00Z", "rev-1", []UnitInput{
		{ID: zeta, Kind: KindRequirement, Title: "Zeta", Authority: "canonical",
			Visibility: VisibilityProject, Revision: 3, Locators: []string{"docs/z.md", "docs/m.md"}},
		{ID: alpha, Kind: KindClaim, Title: "Alpha", Authority: "derived",
			Visibility: VisibilityInternal, Revision: 1, Locators: []string{"docs/a.md"},
			Relationships: []Relationship{RelVerifies, RelDocuments}},
	})
	if len(problems) != 0 {
		t.Fatalf("unexpected problems: %v", problems)
	}
	if !manifest.Derived {
		t.Fatal("a manifest is a derived index and must say so")
	}
	if manifest.Version != ManifestVersion {
		t.Fatalf("unexpected version %d", manifest.Version)
	}
	if manifest.SourceRevision != "rev-1" {
		t.Fatalf("source revision must be recorded, got %q", manifest.SourceRevision)
	}
	// Ordered by id, and locators/relationships normalized.
	if manifest.Units[0].ID != alpha || manifest.Units[1].ID != zeta {
		t.Fatalf("units must be ordered by id: %#v", manifest.Units)
	}
	if got := manifest.Units[1].Locators; len(got) != 2 || got[0] != "docs/m.md" || got[1] != "docs/z.md" {
		t.Fatalf("locators must be sorted and deduplicated: %#v", got)
	}
	if got := manifest.Units[0].Relationships; len(got) != 2 || got[0] != RelDocuments || got[1] != RelVerifies {
		t.Fatalf("relationships must use the typed vocabulary in order: %#v", got)
	}
	// Counts derive from the emitted units only.
	if manifest.Counts.Total != 2 || manifest.Counts.ByKind["requirement"] != 1 || manifest.Counts.ByVisibility["project"] != 1 {
		t.Fatalf("unexpected counts: %#v", manifest.Counts)
	}
}

func TestBuildManifestRejectsInvalidUnits(t *testing.T) {
	valid, _ := NewID("claim.valid")
	manifest, problems := BuildManifest("now", "", []UnitInput{
		{ID: "not-a-stable-id", Kind: KindClaim, Authority: "canonical", Visibility: VisibilityPublic, Revision: 1},
		{ID: valid, Kind: "", Authority: "canonical", Visibility: VisibilityPublic, Revision: 1},
		{ID: valid, Kind: KindClaim, Authority: "canonical", Visibility: VisibilityPublic, Revision: 1},
		{ID: valid, Kind: KindClaim, Authority: "canonical", Visibility: VisibilityPublic, Revision: 1},
	})
	// The invalid id, the missing kind and the duplicate are all reported, and
	// the one valid entry is still emitted.
	if len(problems) != 3 {
		t.Fatalf("expected three problems, got %v", problems)
	}
	if len(manifest.Units) != 1 || manifest.Units[0].ID != valid {
		t.Fatalf("valid units must survive validation: %#v", manifest.Units)
	}
}

func TestManifestLookupSurvivesRename(t *testing.T) {
	id, _ := NewID("claim.rename-safe")
	manifest, _ := BuildManifest("now", "", []UnitInput{{
		ID: id, Kind: KindClaim, Authority: "canonical", Visibility: VisibilityPublic, Revision: 1,
		// A rename appends the new locator; the old one stays resolvable.
		Locators: []string{"docs/old-name.md", "docs/new-name.md"},
	}})
	for _, locator := range []string{"docs/old-name.md", "docs/new-name.md"} {
		unit, ok := manifest.Lookup(locator)
		if !ok || unit.ID != id {
			t.Fatalf("locator %q must resolve to %s, got %#v", locator, id, unit)
		}
	}
}

func TestManifestRejectsLocatorCollisions(t *testing.T) {
	first, _ := NewID("claim.first")
	second, _ := NewID("claim.second")
	_, problems := BuildManifest("now", "", []UnitInput{
		{ID: first, Kind: KindClaim, Authority: "canonical", Visibility: VisibilityPublic, Revision: 1, Locators: []string{"docs/shared.md"}},
		{ID: second, Kind: KindClaim, Authority: "canonical", Visibility: VisibilityPublic, Revision: 1, Locators: []string{"docs/shared.md"}},
	})
	if len(problems) != 1 {
		t.Fatalf("a locator bound twice must be reported once, got %v", problems)
	}
}
