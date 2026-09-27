package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/raillen/prumo-tui/internal/llm/models"
	"github.com/raillen/prumo-tui/internal/message"
)

// A run in flight is the one thing a client can add to and not restart: steering
// says something to the run that is already going, and the daemon owns the loop.
func TestSteeringSaysItToTheRunningRun(t *testing.T) {
	client := &steeringClient{timelineClient: &timelineClient{status: "running"}}
	r := newTestRunner(client, models.Model{})

	out, err := r.Run(context.Background(), "S1", "say hello")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer r.Cancel("S1")
	_ = out

	if err := r.Steer(context.Background(), "S1", "  also check the tests  "); err != nil {
		t.Fatalf("steer: %v", err)
	}
	if len(client.steered) != 1 {
		t.Fatalf("the daemon was told %d time(s), want once", len(client.steered))
	}
	if client.steered[0] != "also check the tests" {
		t.Fatalf("what the daemon received is %q, want it trimmed and unchanged", client.steered[0])
	}

	// What was said is part of the conversation: it is in the transcript even
	// though the daemon is the one that acted on it.
	stored := messagesOf(t, r)
	last := stored[len(stored)-1]
	if last.Role != message.User || !strings.Contains(textOf(last), "also check the tests") {
		t.Fatalf("the steer is missing from the transcript: %+v", last)
	}
}

// There is nothing to steer when nothing is going, and saying so is better than
// sending a message the daemon will refuse.
func TestSteeringWithoutARunIsRefused(t *testing.T) {
	client := &steeringClient{timelineClient: &timelineClient{status: "running"}}
	r := newTestRunner(client, models.Model{})

	if err := r.Steer(context.Background(), "S1", "hurry up"); err == nil {
		t.Fatal("steering a session with no run in flight was accepted")
	}
	if len(client.steered) != 0 {
		t.Fatal("a steer reached the daemon with no run to receive it")
	}
}

// steeringClient records what the client asked the daemon to pass on.
type steeringClient struct {
	*timelineClient
	steered []string
}

func (c *steeringClient) Steer(_ context.Context, _ string, message string) error {
	c.steered = append(c.steered, message)
	return nil
}
