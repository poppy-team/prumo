package stream

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/raillen/prumo-tui/internal/agent"
	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/message"
	"github.com/raillen/prumo-tui/internal/permission"
	"github.com/raillen/prumo-tui/internal/pubsub"
	"github.com/raillen/prumo-tui/internal/session"
)

// recorder is a program that keeps what it was handed.
type recorder struct {
	msgs chan tea.Msg
}

func newRecorder() *recorder { return &recorder{msgs: make(chan tea.Msg, 64)} }

func (r *recorder) Send(msg tea.Msg) { r.msgs <- msg }

// waitFor returns the first message of the wanted type, or fails.
func waitFor[T any](t *testing.T, r *recorder) T {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case msg := <-r.msgs:
			if typed, ok := msg.(T); ok {
				return typed
			}
		case <-deadline:
			var zero T
			t.Fatalf("no %T reached the program", zero)
		}
	}
}

// startedApp builds a client with no harness attached: the bridge carries what
// the services publish, and nothing here needs a daemon to publish it.
func startedApp(t *testing.T) *app.App {
	t.Helper()
	return app.New(app.Options{})
}

// The conversation is the whole point of the client: a message the daemon
// produced has to reach the program, or the transcript never grows.
func TestStoreWritesReachTheProgram(t *testing.T) {
	application := startedApp(t)
	sink := newRecorder()
	Start(context.Background(), application, sink)

	if _, err := application.Messages.Create(context.Background(), "run-1", message.CreateMessageParams{
		Role:  message.Assistant,
		Parts: []message.ContentPart{message.TextContent{Text: "hello"}},
	}); err != nil {
		t.Fatalf("cannot create the message: %v", err)
	}

	event := waitFor[pubsub.Event[message.Message]](t, sink)
	if event.Type != pubsub.CreatedEvent {
		t.Fatalf("expected a created event, got %s", event.Type)
	}
	if text, ok := event.Payload.Parts[0].(message.TextContent); !ok || text.Text != "hello" {
		t.Fatalf("the message arrived without its content: %#v", event.Payload.Parts)
	}
}

// A permission gate that never opens is a run that hangs, so the request has to
// be carried like any other event.
func TestPermissionRequestsReachTheProgram(t *testing.T) {
	application := startedApp(t)
	sink := newRecorder()
	Start(context.Background(), application, sink)

	application.Permissions.Raise(permission.PermissionRequest{ID: "req-1", SessionID: "run-1", ToolName: "edit.patch"})

	event := waitFor[pubsub.Event[permission.PermissionRequest]](t, sink)
	if event.Payload.ID != "req-1" || event.Payload.ToolName != "edit.patch" {
		t.Fatalf("the request arrived incomplete: %+v", event.Payload)
	}
}

// A session update is what moves the statusline, including the usage the
// harness reported for the run.
func TestSessionUpdatesReachTheProgram(t *testing.T) {
	application := startedApp(t)
	sink := newRecorder()
	Start(context.Background(), application, sink)

	if _, err := application.Sessions.Save(context.Background(), session.Session{ID: "run-1", PromptTokens: 120}); err != nil {
		t.Fatalf("cannot save the session: %v", err)
	}

	event := waitFor[pubsub.Event[session.Session]](t, sink)
	if event.Payload.PromptTokens != 120 {
		t.Fatalf("the usage did not survive the trip: %+v", event.Payload)
	}
}

// The agent surface is a broker as well as a channel, and the broker is the
// half the program listens to.
func TestAgentEventsReachTheProgram(t *testing.T) {
	application := startedApp(t)
	sink := newRecorder()
	Start(context.Background(), application, sink)

	application.Runner.Publish(pubsub.CreatedEvent, agent.AgentEvent{Type: agent.AgentEventTypeResponse, SessionID: "run-1"})

	event := waitFor[pubsub.Event[agent.AgentEvent]](t, sink)
	if event.Payload.Type != agent.AgentEventTypeResponse {
		t.Fatalf("expected a response event, got %s", event.Payload.Type)
	}
}

// Closing the client must stop the forwarding, or a cancelled context leaves
// goroutines holding a program that has already quit.
func TestCancellingStopsDelivery(t *testing.T) {
	application := startedApp(t)
	ctx, stop := context.WithCancel(context.Background())
	sink := newRecorder()
	Start(ctx, application, sink)

	if application.Permissions.GetSubscriberCount() == 0 {
		t.Fatal("the bridge did not subscribe")
	}
	stop()

	// The broker drops a subscriber once its context ends, and it does so from
	// its own goroutine. Asking whether any subscription is left is what makes
	// the assertion below about the bridge rather than about a race.
	deadline := time.After(3 * time.Second)
	for application.Permissions.GetSubscriberCount() > 0 {
		select {
		case <-deadline:
			t.Fatal("the bridge kept its subscription after the context ended")
		case <-time.After(5 * time.Millisecond):
		}
	}

	application.Permissions.Raise(permission.PermissionRequest{ID: "req-2"})
	select {
	case msg := <-sink.msgs:
		t.Fatalf("a message crossed a cancelled bridge: %#v", msg)
	case <-time.After(200 * time.Millisecond):
	}
}
