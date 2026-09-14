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
	if len(covered) != 1 || covered[0] != "req-R-seed" || len(uncovered) != 0 {
		t.Fatalf("seeded run must be covered: %v %v", covered, uncovered)
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
