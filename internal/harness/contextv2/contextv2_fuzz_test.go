package contextv2

import "testing"

func FuzzCompilePacking(f *testing.F) {
	f.Add("R-seed", "a", 1.0, 100, 250)
	f.Fuzz(func(t *testing.T, runID, ref string, score float64, cost, budget int) {
		if ref == "" || budget <= 0 || cost < 0 {
			t.Skip()
		}
		if score < 0 {
			score = -score
		}
		m := Compile(runID, []Item{{Ref: ref, Authority: "canonical", Score: score, TokenCost: cost}}, budget, "L1")
		if m.EstimatedTokens > budget {
			t.Fatalf("budget exceeded: %d > %d", m.EstimatedTokens, budget)
		}
		if m.Version != 2 {
			t.Fatalf("manifest version must be 2, got %d", m.Version)
		}
	})
}
