package gauntlet

import "testing"

func TestDefaultPolicyIsOff(t *testing.T) {
	p := DefaultPolicy()
	if p.Mode != ModeOff {
		t.Fatalf("default mode must be off, got %q", p.Mode)
	}
	if p.Active() {
		t.Fatal("default policy must not activate critic rounds")
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("default policy must validate: %v", err)
	}
	if !p.ScoreInflationGuard {
		t.Fatal("score inflation guard must default on")
	}
}

func TestPolicyValidation(t *testing.T) {
	p := DefaultPolicy()
	p.Mode = "whenever"
	if err := p.Validate(); err == nil {
		t.Fatal("unknown mode must be rejected")
	}

	p = DefaultPolicy()
	p.CriticIsolation = false
	if err := p.Validate(); err == nil {
		t.Fatal("critic isolation is mandatory")
	}

	p = DefaultPolicy()
	p.MaxRounds = 99
	if err := p.Validate(); err == nil {
		t.Fatal("unbounded rounds must be rejected")
	}

	p = DefaultPolicy()
	p.Dimensions = []string{"vibes"}
	if err := p.Validate(); err == nil {
		t.Fatal("unknown dimension must be rejected")
	}

	p = DefaultPolicy()
	p.Mode = ModeAuto
	p.Dimensions = []string{"semantic_coverage", "freshness"}
	if err := p.Validate(); err != nil {
		t.Fatalf("valid policy rejected: %v", err)
	}
}

func TestStoppingConditions(t *testing.T) {
	p := DefaultPolicy()
	p.MaxRounds = 3

	if reason, stop := p.ShouldStop(1, true, 0, true); !stop || reason != StopGatesPassed {
		t.Fatalf("gates passed must stop immediately, got %q stop=%v", reason, stop)
	}
	if reason, stop := p.ShouldStop(3, false, 2, true); !stop || reason != StopMaxRounds {
		t.Fatalf("max rounds must stop, got %q stop=%v", reason, stop)
	}
	if _, stop := p.ShouldStop(1, false, 1, true); stop {
		t.Fatal("must not stop mid-run when gates fail and rounds remain")
	}

	p.Stopping.NoImprovementRounds = 2
	if reason, stop := p.ShouldStop(2, false, 1, false); !stop || reason != StopNoImprovement {
		t.Fatalf("no improvement must stop, got %q stop=%v", reason, stop)
	}
	if _, stop := p.ShouldStop(1, false, 1, false); stop {
		t.Fatal("must not stop before the no-improvement threshold")
	}
}

func TestDimensionsCoverTheDocumentationProfile(t *testing.T) {
	if len(Dimensions()) != 13 {
		t.Fatalf("expected 13 gauntlet dimensions, got %d", len(Dimensions()))
	}
	for i := 1; i < len(Dimensions()); i++ {
		if Dimensions()[i-1] >= Dimensions()[i] {
			t.Fatalf("dimensions must be sorted: %v", Dimensions())
		}
	}
}
