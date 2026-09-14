package knowledge

import "testing"

func TestResearchLifecycle(t *testing.T) {
	s := New()
	if err := AddResearch(s, "r1", "which router", "compare", "run:R"); err != nil {
		t.Fatal(err)
	}
	if err := AddResearch(s, "r2", "other", "", "run:R"); err != nil {
		t.Fatal(err)
	}
	if got := OpenResearch(s); len(got) != 2 {
		t.Fatalf("expected 2 open, got %d", len(got))
	}
	if err := ResolveResearch(s, "r1", "use X"); err != nil {
		t.Fatal(err)
	}
	open := OpenResearch(s)
	if len(open) != 1 || open[0].ID != "r2" {
		t.Fatalf("expected only r2 open: %+v", open)
	}
	if _, ok := s.Get("r1-finding"); !ok {
		t.Fatal("finding must persist with lineage link")
	}
	if err := ResolveResearch(s, "nope", ""); err == nil {
		t.Fatal("unknown research must error")
	}
}
