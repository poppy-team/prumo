package runtime

import (
	"sync"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/model"
	"github.com/raillen/prumo/internal/harness/perm"
)

// Event ids were derived from len(payload), so every event of the same shape got
// the same number: every text_delta was ev-2, every usage ev-6. A subscriber
// deduplicating on id dropped the live events as repeats of the replayed ones,
// and a client missed them (GAP-118).

func collectingRunner() (*Runner, *[]agent.AgentEvent, *sync.Mutex) {
	var mu sync.Mutex
	events := &[]agent.AgentEvent{}
	r := NewRunner(Services{
		Models: model.NewFake(nil),
		Perms:  perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Events: func(ev agent.AgentEvent) {
			mu.Lock()
			defer mu.Unlock()
			*events = append(*events, ev)
		},
	}, "R-ev", "S1")
	return r, events, &mu
}

func TestEventIDsAreUniqueWithinARun(t *testing.T) {
	r, events, mu := collectingRunner()
	// The same shape many times: this is the case the old derivation collapsed.
	for i := 0; i < 25; i++ {
		r.emitLocked("text_delta", map[string]any{"text": "same shape"})
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*events) != 25 {
		t.Fatalf("expected 25 events, got %d", len(*events))
	}
	seen := map[string]bool{}
	for _, ev := range *events {
		if ev.ID == "" {
			t.Fatal("every event must carry an id")
		}
		if seen[ev.ID] {
			t.Fatalf("event id %q was issued twice; ids must identify one event", ev.ID)
		}
		seen[ev.ID] = true
	}
}

func TestEventIDsAreUniqueAcrossDifferentShapes(t *testing.T) {
	// The old scheme gave ev-2 to anything with one key and ev-3 to anything
	// with two, so a text_delta and a tool_call_ready collided.
	r, events, mu := collectingRunner()
	shapes := []struct {
		kind    string
		payload map[string]any
	}{
		{"text_delta", map[string]any{"text": "a"}},
		{"tool_call_ready", map[string]any{"id": "c1", "name": "fs.read"}},
		{"text_delta", map[string]any{"text": "b"}},
		{"usage", map[string]any{"prompt_tokens": 1, "completion_tokens": 2, "cache_read_tokens": 3, "cache_write_tokens": 4, "cost_usd": 0.1}},
		{"tool_call_ready", map[string]any{"id": "c2", "name": "fs.read"}},
		{"text_delta", map[string]any{"text": "c"}},
	}
	for _, shape := range shapes {
		r.emitLocked(shape.kind, shape.payload)
	}
	mu.Lock()
	defer mu.Unlock()
	seen := map[string]bool{}
	for _, ev := range *events {
		if seen[ev.ID] {
			t.Fatalf("id %q issued twice across different event shapes", ev.ID)
		}
		seen[ev.ID] = true
	}
	if len(seen) != len(shapes) {
		t.Fatalf("expected %d distinct ids, got %d", len(shapes), len(seen))
	}
}

func TestEventIDsContinueAfterARestore(t *testing.T) {
	// The counter lives in the state so it travels in the checkpoint. A resumed
	// run that restarted at one would reissue ids it had already used, and a
	// subscriber deduplicating on id would drop the new events.
	r, events, mu := collectingRunner()
	r.emitLocked("text_delta", map[string]any{"text": "before"})
	beforeSeq := r.State.EventSeq
	if beforeSeq == 0 {
		t.Fatal("the state must carry the event counter")
	}

	restored := NewRunner(Services{
		Models: model.NewFake(nil),
		Perms:  perm.New(perm.Policy{DefaultAction: agent.PermissionAllow}),
		Events: func(ev agent.AgentEvent) {
			mu.Lock()
			defer mu.Unlock()
			*events = append(*events, ev)
		},
	}, "R-ev", "S1")
	restored.RestoreFrom(agent.Checkpoint{State: r.State})
	restored.emitLocked("text_delta", map[string]any{"text": "after"})

	mu.Lock()
	defer mu.Unlock()
	all := *events
	if len(all) < 2 {
		t.Fatalf("expected events before and after the restore, got %d", len(all))
	}
	if all[0].ID == all[len(all)-1].ID {
		t.Fatalf("a resumed run reissued id %q", all[len(all)-1].ID)
	}
}

func TestEventIDsAreScopedToTheirRun(t *testing.T) {
	// Two runs can both be ev-1. What matters is that an id identifies one
	// event, and the run is part of the event's identity.
	first, eventsA, muA := collectingRunner()
	first.State.RunID = "R-one"
	second, eventsB, muB := collectingRunner()
	second.State.RunID = "R-two"
	first.emitLocked("text_delta", map[string]any{"text": "x"})
	second.emitLocked("text_delta", map[string]any{"text": "x"})

	muA.Lock()
	defer muA.Unlock()
	muB.Lock()
	defer muB.Unlock()
	if (*eventsA)[0].ID != (*eventsB)[0].ID {
		t.Fatal("the counter is per runner, so both runs should start at one")
	}
	if (*eventsA)[0].RunID == (*eventsB)[0].RunID {
		t.Fatal("the run id is what tells two runs' events apart")
	}
}

func TestEventIDsLookLikeSequences(t *testing.T) {
	// A client cannot page by offset without knowing the order, and a numbered
	// id is what makes the log readable.
	r, events, mu := collectingRunner()
	for i := 0; i < 3; i++ {
		r.emitLocked("text_delta", map[string]any{"text": "x"})
	}
	mu.Lock()
	defer mu.Unlock()
	for i, ev := range *events {
		want := "ev-" + itoa(i+1)
		if ev.ID != want {
			t.Fatalf("event %d has id %q, want %q", i, ev.ID, want)
		}
	}
}

func TestNoEventIsEmittedWithoutACollector(t *testing.T) {
	// Services.Events is optional; a run without one must not panic.
	r := NewRunner(Services{Models: model.NewFake(nil), Perms: perm.New(perm.Policy{})}, "R-none", "S1")
	r.emitLocked("text_delta", map[string]any{"text": "x"})
	if r.State.EventSeq != 0 {
		t.Fatal("a run with no collector has no timeline, so the counter must not advance")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
