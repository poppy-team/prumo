package automation

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
)

// The executor followed step.Next[0] and stopped, and replaced a step with no
// registered handler with a function returning nil. A recipe declaring a fan-out
// ran its first branch, ignored the rest, and returned success; a step whose
// capability had no handler did nothing and reported that it worked (GAP-134).

func okHandler(record func(string)) StepHandler {
	return func(_ context.Context, step RecipeStep) error {
		record(step.ID)
		return nil
	}
}

func TestEveryBranchOfAFanOutRuns(t *testing.T) {
	// Three branches off one step. The old executor ran the first and returned
	// success, so a recipe that skipped two thirds of its work looked finished.
	var mu sync.Mutex
	var ran []string
	record := func(id string) { mu.Lock(); ran = append(ran, id); mu.Unlock() }

	dag := RecipeDAG{
		ID: "r", Name: "fanout", Start: "root",
		Steps: map[string]RecipeStep{
			"root": {ID: "root", Actor: "a", Capability: "cap", Next: []string{"lint", "test", "docs"}},
			"lint": {ID: "lint", Actor: "a", Capability: "cap"},
			"test": {ID: "test", Actor: "a", Capability: "cap"},
			"docs": {ID: "docs", Actor: "a", Capability: "cap"},
		},
	}
	e := NewExecutor()
	e.Handlers["cap"] = okHandler(record)
	if err := e.Execute(context.Background(), dag); err != nil {
		t.Fatal(err)
	}
	sort.Strings(ran)
	if strings.Join(ran, ",") != "docs,lint,root,test" {
		t.Fatalf("ran %v; every step must run", ran)
	}
}

func TestAStepWithNoHandlerIsAFailureNotASuccess(t *testing.T) {
	// The one that matters most. A missing handler used to be a no-op returning
	// nil, so a typo in a capability name produced a step that does nothing and
	// reports that it worked.
	var ran []string
	record := func(id string) { ran = append(ran, id) }

	dag := RecipeDAG{
		ID: "r", Name: "missing", Start: "first",
		Steps: map[string]RecipeStep{
			"first":  {ID: "first", Actor: "a", Capability: "registered", Next: []string{"second"}},
			"second": {ID: "second", Actor: "a", Capability: "typoed"},
		},
	}
	e := NewExecutor()
	e.Handlers["registered"] = okHandler(record)
	err := e.Execute(context.Background(), dag)
	if err == nil {
		t.Fatal("a step with no handler must fail; it did nothing")
	}
	if !errors.Is(err, ErrStepFailed) {
		t.Fatalf("error = %v, want a step failure", err)
	}
	if !strings.Contains(err.Error(), "typoed") {
		t.Errorf("the error must name the capability nobody can run: %v", err)
	}
}

func TestAJoinWaitsForEveryPredecessor(t *testing.T) {
	// Two branches converging on one step. The old executor never reached it at
	// all from the second branch; here it must run exactly once, after both.
	var mu sync.Mutex
	var ran []string
	record := func(id string) { mu.Lock(); ran = append(ran, id); mu.Unlock() }

	dag := RecipeDAG{
		ID: "r", Name: "join", Start: "root",
		Steps: map[string]RecipeStep{
			"root":  {ID: "root", Actor: "a", Capability: "cap", Next: []string{"left", "right"}},
			"left":  {ID: "left", Actor: "a", Capability: "cap", Next: []string{"join"}},
			"right": {ID: "right", Actor: "a", Capability: "cap", Next: []string{"join"}},
			"join":  {ID: "join", Actor: "a", Capability: "cap"},
		},
	}
	e := NewExecutor()
	e.Handlers["cap"] = okHandler(record)
	if err := e.Execute(context.Background(), dag); err != nil {
		t.Fatal(err)
	}
	joinAt := -1
	joins := 0
	for i, id := range ran {
		if id == "join" {
			joins++
			joinAt = i
		}
	}
	if joins != 1 {
		t.Fatalf("the join step ran %d times, want once", joins)
	}
	if joinAt != len(ran)-1 {
		t.Fatalf("the join ran at position %d of %v; it must be last", joinAt, ran)
	}
}

func TestARecipeIsReproducibleRatherThanMapOrder(t *testing.T) {
	// Go randomises map iteration, so a ready set taken straight from a map would
	// give a different order every run. That is fine for correctness and useless
	// for a recipe whose evidence has to line up between two runs.
	var orders []string
	for i := 0; i < 6; i++ {
		var ran []string
		record := func(id string) { ran = append(ran, id) }
		dag := RecipeDAG{
			ID: "r", Name: "order", Start: "root",
			Steps: map[string]RecipeStep{
				"root": {ID: "root", Actor: "a", Capability: "cap", Next: []string{"z", "m", "a"}},
				"z":    {ID: "z", Actor: "a", Capability: "cap"},
				"m":    {ID: "m", Actor: "a", Capability: "cap"},
				"a":    {ID: "a", Actor: "a", Capability: "cap"},
			},
		}
		e := NewExecutor()
		e.Handlers["cap"] = okHandler(record)
		if err := e.Execute(context.Background(), dag); err != nil {
			t.Fatal(err)
		}
		orders = append(orders, strings.Join(ran, ","))
	}
	for _, order := range orders {
		if order != orders[0] {
			t.Fatalf("orders differ between runs: %v", orders)
		}
	}
}

func TestAFailedStepUndoesTheDestructiveOnesAlreadyDone(t *testing.T) {
	var mu sync.Mutex
	var order []string
	record := func(id string) { mu.Lock(); order = append(order, id); mu.Unlock() }

	dag := RecipeDAG{
		ID: "r", Name: "comp", Start: "prepare",
		Steps: map[string]RecipeStep{
			"prepare": {ID: "prepare", Actor: "a", Capability: "cap", Next: []string{"mutate"}},
			"mutate": {ID: "mutate", Actor: "a", Capability: "destructive", SideEffects: SideEffectDestructive,
				Compensation: "undo", Next: []string{"verify"}},
			"verify": {ID: "verify", Actor: "a", Capability: "boom"},
		},
	}
	e := NewExecutor()
	e.Handlers["cap"] = okHandler(record)
	// The destructive step has its own capability, and it has to be registered:
	// a step with no handler fails before it can leave anything to compensate for,
	// which is the right behaviour and not what this test is about.
	e.Handlers["destructive"] = okHandler(record)
	e.Handlers["boom"] = func(context.Context, RecipeStep) error { return errors.New("nope") }
	var compensated []string
	e.Compensations["undo"] = func(_ context.Context, step RecipeStep) error {
		compensated = append(compensated, step.ID)
		return nil
	}
	if err := e.Execute(context.Background(), dag); err == nil {
		t.Fatal("expected the failure to propagate")
	}
	if len(compensated) != 1 || compensated[0] != "mutate" {
		t.Fatalf("compensated = %v, want the destructive step undone", compensated)
	}
}

func TestContinuePolicySkipsTheFailedBranchAndFinishesTheRest(t *testing.T) {
	// `continue` is a decision someone makes on purpose, so it has to mean
	// something. It marks the failed step done — which unblocks only the branches
	// that do not depend on it — and keeps going.
	var mu sync.Mutex
	var ran []string
	record := func(id string) { mu.Lock(); ran = append(ran, id); mu.Unlock() }

	dag := RecipeDAG{
		ID: "r", Name: "continue", Start: "root",
		Steps: map[string]RecipeStep{
			"root":  {ID: "root", Actor: "a", Capability: "cap", Next: []string{"fails", "fine"}},
			"fails": {ID: "fails", Actor: "a", Capability: "boom", FailurePolicy: FailureContinue},
			"fine":  {ID: "fine", Actor: "a", Capability: "cap"},
		},
	}
	e := NewExecutor()
	e.Handlers["cap"] = okHandler(record)
	e.Handlers["boom"] = func(context.Context, RecipeStep) error { return errors.New("nope") }
	if err := e.Execute(context.Background(), dag); err != nil {
		t.Fatalf("a step that says continue must not end the recipe: %v", err)
	}
	if !contains(ran, "fine") {
		t.Fatalf("ran %v; the independent branch must still run", ran)
	}
}

func TestAStopIsTheDefaultWhenNoPolicyIsDeclared(t *testing.T) {
	// Carrying on past an unmentioned failure is a decision, so the absence of a
	// policy has to mean the conservative thing.
	dag := RecipeDAG{
		ID: "r", Name: "default", Start: "first",
		Steps: map[string]RecipeStep{
			"first": {ID: "first", Actor: "a", Capability: "boom", Next: []string{"never"}},
			"never": {ID: "never", Actor: "a", Capability: "cap"},
		},
	}
	var ran []string
	record := func(id string) { ran = append(ran, id) }
	e := NewExecutor()
	e.Handlers["cap"] = okHandler(record)
	e.Handlers["boom"] = func(context.Context, RecipeStep) error { return errors.New("nope") }
	if err := e.Execute(context.Background(), dag); err == nil {
		t.Fatal("an undeclared failure policy must stop the recipe")
	}
	if contains(ran, "never") {
		t.Fatalf("ran %v; a stopped recipe must not continue", ran)
	}
}

func TestAnInvalidDAGIsRefusedBeforeAnythingRuns(t *testing.T) {
	var ran []string
	record := func(id string) { ran = append(ran, id) }
	dag := RecipeDAG{
		ID: "r", Name: "bad", Start: "missing",
		Steps: map[string]RecipeStep{"only": {ID: "only", Actor: "a", Capability: "cap"}},
	}
	e := NewExecutor()
	e.Handlers["cap"] = okHandler(record)
	if err := e.Execute(context.Background(), dag); !errors.Is(err, ErrInvalidDAG) {
		t.Fatalf("error = %v, want an invalid DAG", err)
	}
	if len(ran) != 0 {
		t.Fatalf("ran %v before validating", ran)
	}
}

func TestACancelledContextStopsTheWalk(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dag := RecipeDAG{
		ID: "r", Name: "cancelled", Start: "only",
		Steps: map[string]RecipeStep{"only": {ID: "only", Actor: "a", Capability: "cap"}},
	}
	e := NewExecutor()
	e.Handlers["cap"] = okHandler(func(string) {})
	if err := e.Execute(ctx, dag); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want the cancellation", err)
	}
}

func contains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}
