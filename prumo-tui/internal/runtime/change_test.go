package runtime

import (
	"testing"
)

// TestChangesAreFoldedFromTheEventStream pins the client's half of the file
// vocabulary: the run's changes come from the timeline, so a client that
// restarted rebuilds the same list instead of keeping its own.
func TestChangesAreFoldedFromTheEventStream(t *testing.T) {
	r := NewRunner(Options{})
	payload := map[string]any{"path": "internal/x.go", "operation": "modified", "tool": "edit.patch"}
	var current turn
	if r.fold(Event{Kind: "file.changed", Payload: payload}, "S1", &current) {
		t.Fatal("a file change is not a turn: it must not open a message")
	}
	changes := r.Changes("S1")
	if len(changes) != 1 {
		t.Fatalf("changes = %v", changes)
	}
	if changes[0].Path != "internal/x.go" || changes[0].Operation != "modified" {
		t.Fatalf("the change lost what it reports: %+v", changes[0])
	}
	// An event that names no file reports nothing rather than an empty change.
	if r.fold(Event{Kind: "file.changed", Payload: map[string]any{}}, "S1", &current) {
		t.Fatal("a nameless change must not flush the view")
	}
	if len(r.Changes("S1")) != 1 {
		t.Fatalf("a nameless change was recorded: %v", r.Changes("S1"))
	}
	if len(r.Changes("S-other")) != 0 {
		t.Fatal("changes must not leak between sessions")
	}
}
