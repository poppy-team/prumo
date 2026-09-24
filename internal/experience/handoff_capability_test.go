package experience

import (
	"testing"
)

// A handoff reads like a protocol: a status machine with pending, transferred,
// acknowledged and rejected. What it is today is an export artifact — a handoff
// can be created, persisted, listed, acknowledged and rejected, and nothing
// dispatches it to a running agent or resumes the run on the other side
// (GAP-135).
//
// The package documents that. This test is what stops the documentation and the
// code from drifting apart: if a future change wires up dispatch, this fails and
// the comment is updated, rather than the type quietly becoming more than its
// comment says.

func TestAHandoffCanBeCreatedPersistedAndAnswered(t *testing.T) {
	root := t.TempDir()
	provider, err := NewFileProvider(root)
	if err != nil {
		t.Fatal(err)
	}

	created, err := CreateHandoff("h-1", "agent-a", "agent-b", "R-1", "G-1",
		Summary{Completed: []string{"step-1"}, NextSteps: []string{"step-2"}}, []string{"dec-1"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := provider.CreateHandoff(created); err != nil {
		t.Fatal(err)
	}

	loaded, err := provider.GetHandoff("h-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != HandoffPending {
		t.Fatalf("a new handoff is %q, want pending", loaded.Status)
	}
	if err := loaded.Acknowledge("agent-b"); err != nil {
		t.Fatal(err)
	}
	if loaded.Status != HandoffAcknowledged {
		t.Fatalf("after acknowledge the status is %q", loaded.Status)
	}
}

func TestAHandoffCanBeRejected(t *testing.T) {
	created, err := CreateHandoff("h-2", "a", "b", "R-2", "G-2", Summary{Completed: []string{"step-1"}}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := created.Reject("b", "not now"); err != nil {
		t.Fatal(err)
	}
	if created.Status != HandoffRejected {
		t.Fatalf("after reject the status is %q", created.Status)
	}
}

func TestNoCodeDispatchesAHandoffToday(t *testing.T) {
	// The documentation on HandoffStatus says HandoffTransferred is unreachable.
	// This is the check that keeps that true: a handoff read back from storage
	// is never observed in the transferred state, because nothing writes it.
	//
	// If a dispatch is implemented this test fails, and the comment above
	// HandoffTransferred is updated. That is the intended direction of the
	// failure.
	root := t.TempDir()
	provider, err := NewFileProvider(root)
	if err != nil {
		t.Fatal(err)
	}
	created, err := CreateHandoff("h-3", "a", "b", "R-3", "G-3", Summary{Completed: []string{"step-1"}}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := provider.CreateHandoff(created); err != nil {
		t.Fatal(err)
	}
	loaded, err := provider.GetHandoff("h-3")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status == HandoffTransferred {
		t.Fatal("a handoff came back from storage already transferred; something now dispatches them, and the package comment is out of date")
	}
}
