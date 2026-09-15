package knowledge

import "testing"

func TestNewIDRejectsInvalidSlugs(t *testing.T) {
	for _, slug := range []string{"", "with space", "-leading", "accent-ção", "ku:already"} {
		if _, err := NewID(slug); err == nil {
			t.Errorf("NewID(%q) should fail", slug)
		}
	}
	if id, err := NewID("docs.control-plane"); err != nil || id.String() != "ku:docs.control-plane" {
		t.Fatalf("unexpected id: %v %v", id, err)
	}
}

// Slugs are normalized to lowercase so a case difference never forks identity.
func TestNewIDNormalizesCase(t *testing.T) {
	id, err := NewID("Docs.Control-Plane")
	if err != nil {
		t.Fatal(err)
	}
	if id.String() != "ku:docs.control-plane" {
		t.Fatalf("expected lowercase normalization, got %q", id)
	}
}

func TestIdentitySurvivesRename(t *testing.T) {
	id, err := NewID("cli.agent-run")
	if err != nil {
		t.Fatal(err)
	}
	table := NewAliasTable(map[ID][]string{id: {"docs/reference/cli.md#agent-run"}})
	table.Add(id, "docs/reference/commands.md#agent-run")

	got, ok := table.Resolve("docs/reference/commands.md#agent-run")
	if !ok || got != id {
		t.Fatalf("renamed locator must resolve to the same id, got %q ok=%v", got, ok)
	}
	if locators := table.Locators(id); len(locators) != 2 {
		t.Fatalf("expected both locators retained, got %v", locators)
	}
}

func TestAliasTableRejectsInvalidID(t *testing.T) {
	table := NewAliasTable(map[ID][]string{ID("not-an-id"): {"docs/x.md"}})
	if _, ok := table.Resolve("docs/x.md"); ok {
		t.Fatal("invalid ids must never be registered")
	}
}

func TestVisibilityIsFailClosed(t *testing.T) {
	if !VisibleTo(VisibilityPublic, VisibilityInternal) {
		t.Fatal("public content is visible on an internal surface")
	}
	if VisibleTo(VisibilityConfidential, VisibilityPublic) {
		t.Fatal("confidential content must never reach a public surface")
	}
	if VisibleTo(Visibility("nonsense"), VisibilityPublic) {
		t.Fatal("unknown visibility must fail closed")
	}
}

func TestKindsAndRelationshipsAreComplete(t *testing.T) {
	if len(Kinds()) != 12 {
		t.Fatalf("expected 12 kinds, got %d", len(Kinds()))
	}
	if len(Relationships()) != 13 {
		t.Fatalf("expected 13 relationships, got %d", len(Relationships()))
	}
	for i := 1; i < len(Kinds()); i++ {
		if Kinds()[i-1] >= Kinds()[i] {
			t.Fatalf("kinds must be sorted: %v", Kinds())
		}
	}
	for i := 1; i < len(Relationships()); i++ {
		if Relationships()[i-1] >= Relationships()[i] {
			t.Fatalf("relationships must be sorted: %v", Relationships())
		}
	}
}
