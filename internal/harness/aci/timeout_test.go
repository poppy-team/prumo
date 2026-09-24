package aci

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
)

// The execution path had no timeout. `process.exec` inherited the run's context,
// and where the run had no deadline, a command that hung held the run for as
// long as the process lived. The registry carried a TimeoutMS that was displayed
// and enforced nowhere, so the surface read as bounded while nothing bounded it
// (GAP-169).

func TestASlowToolIsCutOffAtItsOwnDeadline(t *testing.T) {
	root := t.TempDir()
	e := New(root)
	// A command that cannot finish in the time given. A side-effecting tool gets
	// 60s, so this would take a minute to notice a timeout that is not there.
	real := toolTimeout
	toolTimeout = func(name string) (time.Duration, bool) {
		if name == "process.exec" {
			return 120 * time.Millisecond, true
		}
		return real(name)
	}
	defer func() { toolTimeout = real }()

	start := time.Now()
	res, err := e.Execute(context.Background(), agent.ToolCall{
		ID: "c1", TurnID: "t1", Name: "process.exec",
		Arguments: map[string]any{"cmd": "sleep 30"},
	})
	elapsed := time.Since(start)
	if err != nil && !strings.Contains(strings.ToLower(res.Error), "timeout") {
		t.Fatalf("unexpected error: %v (result %+v)", err, res)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("the call took %s; a 120ms tool deadline did not stop it", elapsed)
	}
	if res.OK() {
		t.Fatalf("a killed tool reported success: %+v", res)
	}
}

func TestTheEnforcedBoundIsTheAdvertisedBound(t *testing.T) {
	// The number in the descriptor and the number that stops the process must be
	// the same number. They used to be two functions in two packages.
	for _, tool := range Catalog() {
		bound, ok := toolTimeout(tool.Name)
		if !ok {
			t.Fatalf("%s has no execution bound at all", tool.Name)
		}
		want := time.Duration(KindTimeoutMS(tool.Kind)) * time.Millisecond
		if bound != want {
			t.Fatalf("%s advertises one bound and enforces another: %s vs %s", tool.Name, want, bound)
		}
	}
}

func TestAnUnknownToolGetsNoInheritedDeadline(t *testing.T) {
	// A tool the catalogue does not know is not a tool the executor can run, and
	// guessing a bound for it would hide the fact that it is unknown.
	if _, ok := toolTimeout("not.a.tool"); ok {
		t.Fatal("an unknown tool was given a timeout as though it were known")
	}
}
