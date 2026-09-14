package knowledge

import (
	"strings"
	"testing"
)

func seedMemory(t *testing.T, s *Store, id, sens string) {
	t.Helper()
	r := Record{ID: id, Kind: KindMemory, Title: id, Authority: "reference",
		Trust: "medium", Status: "active", Provenance: "run:R", Sensitivity: sens}
	if err := s.Commit(Delta{ID: "seed-" + id, Author: "t", Upserts: []Record{r}}); err != nil {
		t.Fatal(err)
	}
}

func TestPromoteGates(t *testing.T) {
	src, dst := New(), New()
	seedMemory(t, src, "pub", "")
	seedMemory(t, src, "sec", "restricted")
	seedMemory(t, src, "con", "confidential")

	policy := PromotionPolicy{AllowProjects: []string{"proj-b"}}
	if _, err := Promote(src, []string{"pub"}, "proj-evil", policy, dst); err == nil {
		t.Fatal("unlisted target must be denied")
	}
	res, err := Promote(src, []string{"pub", "sec", "con", "missing"}, "proj-b", policy, dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Promoted) != 1 || res.Promoted[0] != "pub" {
		t.Fatalf("only public/internal cross: %+v", res)
	}
	if len(res.Denied) != 3 {
		t.Fatalf("expected 3 denials: %+v", res)
	}
	got, ok := dst.Get("pub")
	if !ok || !strings.Contains(got.Provenance, "proj-b") {
		t.Fatalf("provenance must stamp target: %+v", got)
	}
	if _, ok := dst.Get("sec"); ok {
		t.Fatal("restricted must never cross")
	}
	// Source untouched (copy, not move).
	if _, ok := src.Get("sec"); !ok {
		t.Fatal("source must keep its records")
	}
}

func TestPromoteCap(t *testing.T) {
	src, dst := New(), New()
	for _, id := range []string{"a", "b", "c"} {
		seedMemory(t, src, id, "")
	}
	res, err := Promote(src, []string{"a", "b", "c"}, "p2", PromotionPolicy{AllowProjects: []string{"p2"}, MaxRecords: 2}, dst)
	if err != nil || len(res.Promoted) != 2 {
		t.Fatalf("cap must bind: %+v %v", res, err)
	}
}
