package knowledge

import (
	"path/filepath"
	"testing"
)

func TestSeedLinksRequirementToEvidence(t *testing.T) {
	s := New()
	SeedRequirement(s, "R-seed", "ship the harness")
	SeedEvidence(s, "R-seed", "complete", "completed", "cp1")
	covered, uncovered := s.Coverage()
	// W3.8: the requirement carries a stable ku: id, not the pre-migration
	// run-scoped literal.
	want := RequirementID("R-seed")
	if len(covered) != 1 || covered[0] != want || len(uncovered) != 0 {
		t.Fatalf("seeded run must be covered by %s: %v %v", want, covered, uncovered)
	}
	if !IsStableID(want) {
		t.Fatalf("seeded requirement id must be stable, got %q", want)
	}
	// The pre-migration identifier still resolves to the same record, so a
	// store written before the migration keeps working.
	legacy, ok := s.Resolve(LegacyRequirementID("R-seed"))
	if !ok || legacy.ID != want {
		t.Fatalf("legacy run-scoped id must resolve to %s, got %#v", want, legacy)
	}
	if unstable := s.UnstableIDs(); len(unstable) != 0 {
		t.Fatalf("seeded store must contain only stable ids: %v", unstable)
	}
	if ready, blockers := s.Readiness(); !ready {
		t.Fatalf("seeded run must be ready: %v", blockers)
	}
	// Reseed is idempotent: same stable ids, no duplicates.
	SeedRequirement(s, "R-seed", "ship the harness")
	SeedEvidence(s, "R-seed", "complete", "completed", "cp1")
	records, _ := s.Snapshot()
	if len(records) != 2 {
		t.Fatalf("reseed must not duplicate: %d records", len(records))
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	s := New()
	SeedRequirement(s, "R-rt", "goal")
	SeedEvidence(s, "R-rt", "complete", "", "")
	path := filepath.Join(t.TempDir(), "knowledge-R-rt.json")
	if err := s.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	records, rels := loaded.Snapshot()
	if len(records) != 2 || len(rels) != 1 {
		t.Fatalf("roundtrip lost data: %d records %d rels", len(records), len(rels))
	}
	empty, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal("missing file must yield empty store, not error")
	}
	if records, _ := empty.Snapshot(); len(records) != 0 {
		t.Fatal("missing file must yield empty store")
	}
}
