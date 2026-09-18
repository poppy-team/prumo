package runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/raillen/prumo-tui/internal/llm/models"
	"github.com/raillen/prumo-tui/internal/message"
	"github.com/raillen/prumo-tui/internal/permission"
)

// waitUntilNotBusy waits for the follow loop to run out, which is what a
// finished run makes it do after one read.
func waitUntilNotBusy(t *testing.T, r *Runner, sessionID string) {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for r.IsSessionBusy(sessionID) {
		select {
		case <-deadline:
			t.Fatalf("the client never stopped following %s", sessionID)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

// A client that started after the run did has no conversation; reconnecting has
// to rebuild it from the run's own log rather than from memory it never had.
func TestAttachRebuildsTheTranscript(t *testing.T) {
	client := &timelineClient{status: "complete"}
	client.append(
		Event{Kind: "text_delta", Payload: map[string]any{"text": "already "}},
		Event{Kind: "text_delta", Payload: map[string]any{"text": "answered"}},
		Event{Kind: "usage", Payload: map[string]any{"prompt_tokens": float64(300), "completion_tokens": float64(20)}},
	)
	r := newTestRunner(client, models.Model{})

	if err := r.Attach(context.Background(), "S1"); err != nil {
		t.Fatalf("attach: %v", err)
	}
	waitUntilNotBusy(t, r, "S1")

	stored := messagesOf(t, r)
	if len(stored) != 1 {
		t.Fatalf("messages = %d, want the run's answer rebuilt", len(stored))
	}
	if got := textOf(stored[0]); got != "already answered" {
		t.Fatalf("the transcript reads %q", got)
	}
	session, err := r.sessions.Get(context.Background(), "S1")
	if err != nil {
		t.Fatalf("the run was not indexed: %v", err)
	}
	if session.PromptTokens != 300 {
		t.Fatalf("the spend was not rebuilt: %+v", session)
	}
}

// Reconnecting twice must leave the same conversation, not two copies of it.
func TestAttachIsIdempotent(t *testing.T) {
	client := &timelineClient{status: "complete"}
	client.append(Event{Kind: "text_delta", Payload: map[string]any{"text": "once"}})
	r := newTestRunner(client, models.Model{})

	for i := 0; i < 2; i++ {
		if err := r.Attach(context.Background(), "S1"); err != nil {
			t.Fatalf("attach %d: %v", i, err)
		}
		waitUntilNotBusy(t, r, "S1")
	}

	stored := messagesOf(t, r)
	if len(stored) != 1 || textOf(stored[0]) != "once" {
		t.Fatalf("the transcript was duplicated: %d messages", len(stored))
	}
}

// A run the daemon does not know is an error. An empty conversation would look
// like a session that had produced nothing, which is a claim, not a reading.
func TestAttachRefusesAnUnknownRun(t *testing.T) {
	client := &timelineClient{status: "complete", err: errors.New("no run S9")}
	r := newTestRunner(client, models.Model{})

	if err := r.Attach(context.Background(), "S9"); err == nil {
		t.Fatal("attaching to an unknown run must fail rather than report an empty session")
	}
}

// A run that stopped for a decision has to raise the gate again on reconnect,
// or the run waits forever behind a dialog nobody is shown.
func TestAttachRaisesThePendingGate(t *testing.T) {
	client := &timelineClient{status: "awaiting_approval"}
	client.append(Event{Kind: "permission_wait", Payload: map[string]any{"request_id": "req-9", "tool": "process.exec"}})
	r := newTestRunner(client, models.Model{})
	permissions := permission.NewService(r)
	r.SetPermissions(permissions)

	if err := r.Attach(context.Background(), "S1"); err != nil {
		t.Fatalf("attach: %v", err)
	}

	deadline := time.After(10 * time.Second)
	for len(permissions.Pending()) == 0 {
		select {
		case <-deadline:
			t.Fatal("the pending gate was never raised")
		case <-time.After(5 * time.Millisecond):
		}
	}
	pending := permissions.Pending()[0]
	if pending.ID != "req-9" || pending.ToolName != "process.exec" {
		t.Fatalf("the gate lost what it was waiting for: %+v", pending)
	}
}

// An attaching client must also keep following: a run still in flight streams to
// whoever reconnected to it.
func TestAttachKeepsFollowingALiveRun(t *testing.T) {
	client := &timelineClient{status: "running"}
	client.append(Event{Kind: "text_delta", Payload: map[string]any{"text": "started"}})
	r := newTestRunner(client, models.Model{})

	if err := r.Attach(context.Background(), "S1"); err != nil {
		t.Fatalf("attach: %v", err)
	}

	// The run answers while the client is attached.
	deadline := time.After(10 * time.Second)
	for {
		stored := messagesOf(t, r)
		if len(stored) == 1 && textOf(stored[0]) == "started" {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("the attach did not read the run: %v", stored)
		case <-time.After(5 * time.Millisecond):
		}
	}

	client.append(Event{Kind: "text_delta", Payload: map[string]any{"text": " and more"}})
	deadline = time.After(10 * time.Second)
	for {
		stored := messagesOf(t, r)
		if len(stored) == 1 && textOf(stored[0]) == "started and more" {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("the live run stopped streaming after a reconnect: %v", stored)
		case <-time.After(5 * time.Millisecond):
		}
	}

	// And the run finishing closes the follow out.
	client.status = "complete"
	waitUntilNotBusy(t, r, "S1")
	if last := messagesOf(t, r)[0]; last.Role != message.Assistant {
		t.Fatalf("the follow ended on something other than the answer: %+v", last)
	}
}

// A gate asks a person to approve what a tool is about to do, so the request
// carries the tool's arguments: a dialog that can only show the tool's name is
// one a person can only answer on faith.
func TestAGateCarriesWhatItIsAskingAbout(t *testing.T) {
	client := &timelineClient{status: "awaiting_approval"}
	client.append(Event{Kind: "permission_wait", Payload: map[string]any{
		"request_id": "req-3",
		"tool":       "process.exec",
		"arguments":  map[string]any{"command": "go test ./..."},
	}})
	r := newTestRunner(client, models.Model{})
	permissions := permission.NewService(r)
	r.SetPermissions(permissions)

	if err := r.Attach(context.Background(), "S1"); err != nil {
		t.Fatalf("attach: %v", err)
	}

	deadline := time.After(10 * time.Second)
	for len(permissions.Pending()) == 0 {
		select {
		case <-deadline:
			t.Fatal("the gate was never raised")
		case <-time.After(5 * time.Millisecond):
		}
	}
	params, ok := permissions.Pending()[0].Params.(map[string]any)
	if !ok {
		t.Fatalf("the request carries no arguments: %#v", permissions.Pending()[0].Params)
	}
	if params["command"] != "go test ./..." {
		t.Fatalf("the arguments were lost: %#v", params)
	}
}
