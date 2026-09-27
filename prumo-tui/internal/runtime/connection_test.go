package runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/raillen/prumo-tui/internal/agent"
	"github.com/raillen/prumo-tui/internal/llm/models"
)

// A daemon that stops answering is retried before it is given up on, and the
// attempts are reported where a user can count them: a client that retried in
// silence is indistinguishable from one that hung, and one that gave up on the
// first failure reports a hiccup as a death.
func TestALostDaemonIsRetriedThenCalledOffline(t *testing.T) {
	client := &timelineClient{status: "running", err: errors.New("the daemon did not answer")}
	r := newTestRunner(client, models.Model{})
	sub := r.Subscribe(context.Background())

	out, err := r.Run(context.Background(), "S1", "say hello")
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var (
		attempts []int
		offline  bool
	)
	deadline := time.After(20 * time.Second)
	// The loop runs until the run ends, not until the link is called lost: the
	// failure that follows is what a user is left reading.
	for {
		select {
		case event, ok := <-sub:
			if !ok {
				t.Fatal("the subscription closed before the run ended")
			}
			switch event.Payload.Type {
			case agent.AgentEventTypeConnection:
				switch event.Payload.Connection.State {
				case agent.ConnectionReconnecting:
					attempts = append(attempts, event.Payload.Connection.Attempt)
				case agent.ConnectionOffline:
					offline = true
				}
			case agent.AgentEventTypeError:
				goto ended
			}
		case <-deadline:
			t.Fatalf("the client never called the link lost: attempts=%v offline=%v", attempts, offline)
		}
	}
ended:

	if len(attempts) != MaxPollFailures {
		t.Fatalf("attempts reported = %v, want %d of them", attempts, MaxPollFailures)
	}
	for i, attempt := range attempts {
		if attempt != i+1 {
			t.Fatalf("attempts were reported out of order: %v", attempts)
		}
	}
	if !offline {
		t.Fatal("the client gave up without saying the link was lost")
	}
	drain(t, out)
}

// A link that comes back is reported as back, or the statusline would keep
// saying the client is offline while it is answering.
func TestARecoveredLinkIsReportedLive(t *testing.T) {
	client := &flakyClient{timelineClient: &timelineClient{status: "complete"}, failures: 2}
	r := newTestRunner(client, models.Model{})
	sub := r.Subscribe(context.Background())

	out, err := r.Run(context.Background(), "S1", "say hello")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	drain(t, out)

	live := false
	for {
		select {
		case event := <-sub:
			if event.Payload.Type == agent.AgentEventTypeConnection && event.Payload.Connection.State == agent.ConnectionLive {
				live = true
			}
		case <-time.After(500 * time.Millisecond):
			if !live {
				t.Fatal("the link recovered and the client never said so")
			}
			return
		}
	}
}

// flakyClient answers nothing for a while and then answers normally. Every other
// call is the run log it wraps.
type flakyClient struct {
	*timelineClient
	failures int
	calls    int
}

func (c *flakyClient) Status(ctx context.Context, runID string) (RunStatus, error) {
	c.calls++
	if c.calls <= c.failures {
		return RunStatus{}, errors.New("the daemon did not answer")
	}
	return c.timelineClient.Status(ctx, runID)
}

func drain(t *testing.T, out <-chan agent.AgentEvent) {
	t.Helper()
	deadline := time.After(20 * time.Second)
	for {
		select {
		case _, ok := <-out:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("the run never finished")
		}
	}
}
