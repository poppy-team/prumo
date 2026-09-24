package extagent

import (
	"context"
	"strings"
	"testing"
)

// A capability list is a contract a caller plans against. CodexCLI declared six
// — session, events, permissions, usage, cancel, resume — and implemented one.
// `usage` and `resume` had no method at all; `events` manufactured a single
// synthetic event and closed; `permissions` and `cancel` returned nil from a
// body that did nothing (GAP-142).

func TestACodexAdapterDeclaresOnlyWhatItDoes(t *testing.T) {
	ctx := context.Background()
	declared, err := NewCodexCLI("codex").Capabilities(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, capability := range declared {
		switch capability {
		case "session":
			// Real: CreateSession and Send drive the binary.
		default:
			t.Errorf("the codex adapter declares %q, which it does not implement", capability)
		}
	}
}

func TestAMethodThatDoesNothingSaysSo(t *testing.T) {
	// A nil here reads as "done". A caller cancels a session, gets no error, and
	// believes it — and the process it meant to stop is still running.
	adapter := NewCodexCLI("codex")
	ctx := context.Background()
	cases := []struct {
		name string
		call func() error
		want string
	}{
		{"approve", func() error { return adapter.Approve(ctx, "s", "r", true) }, "permissions"},
		{"cancel", func() error { return adapter.Cancel(ctx, "s") }, "cancel"},
		{"close", func() error { return adapter.Close(ctx, "s") }, "close"},
	}
	for _, c := range cases {
		err := c.call()
		if err == nil {
			t.Errorf("%s returned nil, which reads as success", c.name)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s refused without naming the capability: %v", c.name, err)
		}
	}
}

func TestAnEventStreamThatIsNotOneIsRefused(t *testing.T) {
	// A caller reading a channel that receives one event and closes sees a
	// stream that begins and ends, and a run waiting for a tool result waits
	// forever. Refusing is the honest answer.
	ch, err := NewCodexCLI("codex").Events(context.Background(), "s")
	if err == nil {
		t.Fatalf("the codex adapter handed out an event stream anyway: %v", ch)
	}
	if !strings.Contains(err.Error(), "events") {
		t.Fatalf("the refusal does not name the capability: %v", err)
	}
}

func TestAnAdapterThatImplementsEverythingStillSaysSo(t *testing.T) {
	// The fix must not have trimmed the one adapter that can back all of it:
	// OpenCodeServer talks to a real HTTP API with a real abort and a real
	// session list.
	declared, err := (&OpenCodeServer{BaseURL: "http://127.0.0.1:1"}).Capabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(declared) != 6 {
		t.Fatalf("the opencode adapter declares %v, want the six it implements", declared)
	}
}
