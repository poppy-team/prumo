package knowledge

import "testing"

func TestDeltaValidateCommit(t *testing.T) {
	s := New()
	if err := s.Put(Record{ID: "req-1", Kind: KindRequirement, Title: "harness", Authority: "canonical", Trust: "high", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	d := Delta{ID: "d1", Author: "test", Upserts: []Record{{ID: "ev-1", Kind: KindEvidence, Title: "tests pass", Authority: "reference", Trust: "high"}}, Links: []Relation{{From: "ev-1", Type: "evidences", To: "req-1"}}}
	if err := s.Validate(d); err != nil {
		t.Fatal(err)
	}
	if err := s.Commit(d); err != nil {
		t.Fatal(err)
	}
	covered, uncovered := s.Coverage()
	if len(covered) != 1 || len(uncovered) != 0 {
		t.Fatalf("expected covered req-1, got %v %v", covered, uncovered)
	}
	if ready, blockers := s.Readiness(); !ready || len(blockers) != 0 {
		t.Fatalf("expected ready, got %v", blockers)
	}
}

func TestContradictionBlocksReadiness(t *testing.T) {
	s := New()
	_ = s.Put(Record{ID: "a", Kind: KindClaim, Title: "x", Authority: "reference", Trust: "high", Status: "active"})
	_ = s.Put(Record{ID: "b", Kind: KindClaim, Title: "not x", Authority: "reference", Trust: "high", Status: "active"})
	s.Link(Relation{From: "a", Type: "contradicts", To: "b"})
	if got := s.DetectContradictions(); len(got) != 1 {
		t.Fatalf("expected 1 contradiction, got %v", got)
	}
	if ready, _ := s.Readiness(); ready {
		t.Fatal("contradiction must block readiness")
	}
}
