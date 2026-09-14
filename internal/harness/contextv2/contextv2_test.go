package contextv2

import "testing"

func TestGatesAndPacking(t *testing.T) {
	if !Eligible(Item{Authority: "canonical", Trust: "high", Privacy: "internal"}, "reference", false) {
		t.Fatal("canonical should be eligible")
	}
	if Eligible(Item{Authority: "canonical", Trust: "high", Privacy: "restricted"}, "reference", false) {
		t.Fatal("restricted must be gated")
	}
	fused := RRF([][]Item{{{Ref: "a", Score: 1}, {Ref: "b"}}, {{Ref: "b"}, {Ref: "a"}}}, 60)
	if len(fused) != 2 || fused[0].Ref == "" {
		t.Fatalf("rrf failed: %+v", fused)
	}
	m := Compile("R1", []Item{
		{Ref: "a", Authority: "canonical", Score: 1.0, TokenCost: 100},
		{Ref: "b", Authority: "canonical", Score: 0.9, TokenCost: 100, DependsOn: []string{"a"}},
		{Ref: "c", Authority: "canonical", Score: 0.1, TokenCost: 10000},
	}, 250, "L1")
	if m.EstimatedTokens > 250 {
		t.Fatal("budget exceeded")
	}
	if len(m.Included) < 2 {
		t.Fatalf("expected a+b packed, got %+v", m)
	}
}
