package contextv2

import (
	"testing"
)

// A reference only entered inSet after its dependencies had been expanded, so
// two items that depend on each other sent the expansion into a loop: A marks
// nothing, asks for B, B asks for A, and A is still unmarked. A pointer in one
// document to the document that points back took the compiler down with a stack
// overflow instead of reporting a cycle (GAP-149).

// candidatesWithCycle returns two items that depend on each other.
func candidatesWithCycle() []Item {
	return []Item{
		{Ref: "a", Authority: "canonical", TokenCost: 10, DependsOn: []string{"b"}},
		{Ref: "b", Authority: "canonical", TokenCost: 10, DependsOn: []string{"a"}},
	}
}

func compileWith(t *testing.T, items []Item) Manifest {
	t.Helper()
	return Compile("R-cycle", items, 8000, "L1")
}

func TestADependencyCycleDoesNotRecurseForever(t *testing.T) {
	// The assertion that matters: this returns. Before the fix it did not.
	manifest := compileWith(t, candidatesWithCycle())
	if len(manifest.Included) == 0 {
		t.Fatal("the cycle compiled to nothing at all")
	}
}

func TestACycleIsReportedRatherThanSilentlyBroken(t *testing.T) {
	manifest := compileWith(t, candidatesWithCycle())
	if len(manifest.DependencyCycles) == 0 {
		t.Fatal("a dependency cycle compiled without saying so; a caller cannot tell what was left out")
	}
}

func TestALongerCycleAlsoTerminates(t *testing.T) {
	items := []Item{
		{Ref: "a", Authority: "canonical", TokenCost: 10, DependsOn: []string{"c"}},
		{Ref: "b", Authority: "canonical", TokenCost: 10, DependsOn: []string{"a"}},
		{Ref: "c", Authority: "canonical", TokenCost: 10, DependsOn: []string{"b"}},
	}
	manifest := compileWith(t, items)
	if len(manifest.DependencyCycles) == 0 {
		t.Fatal("a three-node cycle was not reported")
	}
	if len(manifest.Included) == 0 {
		t.Fatal("a three-node cycle compiled to nothing")
	}
}

func TestASelfReferenceTerminates(t *testing.T) {
	manifest := compileWith(t, []Item{
		{Ref: "selfish", Authority: "canonical", TokenCost: 10, DependsOn: []string{"selfish"}},
	})
	if len(manifest.Included) != 1 {
		t.Fatalf("a self-reference compiled to %d items, want 1", len(manifest.Included))
	}
}

func TestACycleDoesNotDropUnrelatedItems(t *testing.T) {
	// The cycle is contained. An item that depends on nothing must still be in
	// the context, or one broken pair of documents would blank the run.
	items := append(candidatesWithCycle(), Item{Ref: "unrelated", Authority: "canonical", TokenCost: 5})
	manifest := compileWith(t, items)
	found := false
	for _, item := range manifest.Included {
		if item.Ref == "unrelated" {
			found = true
		}
	}
	if !found {
		t.Fatal("a dependency cycle removed an unrelated document from the context")
	}
}

func TestAnAcyclicGraphStillOrdersDependenciesFirst(t *testing.T) {
	manifest := compileWith(t, []Item{
		{Ref: "dependent", Authority: "canonical", TokenCost: 10, DependsOn: []string{"base"}},
		{Ref: "base", Authority: "canonical", TokenCost: 10},
	})
	position := map[string]int{}
	for i, item := range manifest.Included {
		position[item.Ref] = i
	}
	if position["base"] > position["dependent"] {
		t.Fatalf("a dependency came after its dependent: %v", manifest.Included)
	}
}
