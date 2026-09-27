package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	prumo "github.com/raillen/prumo/sdk/prumo"

	"github.com/raillen/prumo-tui/internal/agent"
	"github.com/raillen/prumo-tui/internal/llm/models"
	"github.com/raillen/prumo-tui/internal/message"
)

// timelineClient serves a run's log the way the daemon does: one file per run,
// appended to, read whole.
type timelineClient struct {
	log    []Event
	status string
	err    error
}

func (c *timelineClient) append(events ...Event) { c.log = append(c.log, events...) }

func (c *timelineClient) Start(context.Context, StartRequest) (string, error) { return "S1", nil }
func (c *timelineClient) Status(context.Context, string) (RunStatus, error) {
	if c.err != nil {
		return RunStatus{}, c.err
	}
	return RunStatus{Status: c.status}, nil
}
func (c *timelineClient) Events(context.Context, string) ([]Event, error) { return c.log, nil }
func (c *timelineClient) Cancel(context.Context, string) error            { return nil }
func (c *timelineClient) Steer(context.Context, string, string) error     { return nil }
func (c *timelineClient) Jobs(context.Context) ([]prumo.Job, error)       { return nil, nil }
func (c *timelineClient) ModelInfo(context.Context, prumo.ModelsRequest) ([]prumo.ModelInfo, error) {
	return nil, nil
}
func (c *timelineClient) Unschedule(context.Context, string) error      { return nil }
func (c *timelineClient) Approve(context.Context, string, string) error { return nil }
func (c *timelineClient) Deny(context.Context, string, string, string) error {
	return nil
}
func (c *timelineClient) List(context.Context) ([]RunStatus, error) { return nil, nil }
func (c *timelineClient) Models(context.Context, prumo.ModelsRequest) ([]string, error) {
	return nil, nil
}
func (c *timelineClient) Diff(context.Context, string, string) (prumo.DiffResponse, error) {
	return prumo.DiffResponse{}, nil
}
func (c *timelineClient) Subscribe(context.Context, string, int) (<-chan prumo.Event, error) {
	if c.err != nil {
		return nil, c.err
	}
	return nil, errors.ErrUnsupported
}

// runToCompletion starts a run and waits for the client to stop folding it.
func runToCompletion(t *testing.T, r *Runner) {
	t.Helper()
	out, err := r.Run(context.Background(), "S1", "say hello")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	deadline := time.After(10 * time.Second)
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

// textOf is the whole of a message's prose, which is what a reader sees.
func textOf(m message.Message) string {
	var out strings.Builder
	for _, part := range m.Parts {
		if text, ok := part.(message.TextContent); ok {
			out.WriteString(text.Text)
		}
	}
	return out.String()
}

func messagesOf(t *testing.T, r *Runner) []message.Message {
	t.Helper()
	stored, err := r.messages.List(context.Background(), "S1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	return stored
}

// The conversation the daemon streamed has to end up as a message the view can
// draw. Before the timeline carried content this was the whole defect: the
// client could start runs and never render their output.
func TestRunOutputBecomesAMessage(t *testing.T) {
	client := &timelineClient{status: "complete"}
	client.append(
		Event{Kind: "text_delta", Payload: map[string]any{"text": "hello"}},
		Event{Kind: "text_delta", Payload: map[string]any{"text": " world"}},
	)
	r := newTestRunner(client, models.Model{})
	sub := r.Subscribe(context.Background())

	runToCompletion(t, r)

	stored := messagesOf(t, r)
	if len(stored) != 2 {
		t.Fatalf("messages = %d, want the user's and the assistant's", len(stored))
	}
	assistant := stored[len(stored)-1]
	if assistant.Role != message.Assistant {
		t.Fatalf("the run's output is not an assistant message: %+v", assistant)
	}
	if got := textOf(assistant); got != "hello world" {
		t.Fatalf("the deltas were not joined: %q", got)
	}

	// And the same content has to have gone out on the broker, which is the
	// path the running program actually consumes.
	select {
	case event := <-sub:
		if event.Payload.Type != agent.AgentEventTypeResponse {
			t.Fatalf("expected a response on the broker, got %s", event.Payload.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the assistant message never reached the broker")
	}
}

// Usage arrives as its own event and is not a turn: if it opened an assistant
// message, the transcript would grow a bubble per model request.
func TestUsageDoesNotOpenAMessage(t *testing.T) {
	client := &timelineClient{status: "complete"}
	client.append(Event{Kind: "usage", Payload: map[string]any{"prompt_tokens": float64(120)}})
	r := newTestRunner(client, models.Model{})

	runToCompletion(t, r)

	if stored := messagesOf(t, r); len(stored) != 1 {
		t.Fatalf("messages = %d, want only the user's", len(stored))
	}
}

// What the run spent is the session's, so the statusline reports the harness's
// numbers instead of zeros.
func TestUsageIsAddedToTheSession(t *testing.T) {
	client := &timelineClient{status: "complete"}
	client.append(
		Event{Kind: "usage", Payload: map[string]any{"prompt_tokens": float64(120), "completion_tokens": float64(30), "cost_usd": 0.0021}},
		Event{Kind: "usage", Payload: map[string]any{"prompt_tokens": float64(80), "completion_tokens": float64(10), "cost_usd": 0.0009}},
	)
	r := newTestRunner(client, models.Model{})

	runToCompletion(t, r)

	got, err := r.sessions.Get(context.Background(), "S1")
	if err != nil {
		t.Fatalf("the run's session is gone: %v", err)
	}
	if got.PromptTokens != 200 || got.CompletionTokens != 40 {
		t.Fatalf("tokens = %d/%d, want the reports summed", got.PromptTokens, got.CompletionTokens)
	}
	if got.Cost < 0.0029 || got.Cost > 0.0031 {
		t.Fatalf("cost = %f, want the reports summed", got.Cost)
	}
}

// A second message in a session appends to the same run log. Folding it from
// the start would draw the first answer twice and charge for it twice, which is
// the defect a cursor exists to prevent.
func TestASecondRunResumesTheFold(t *testing.T) {
	client := &timelineClient{status: "complete"}
	client.append(
		Event{Kind: "text_delta", Payload: map[string]any{"text": "one"}},
		Event{Kind: "usage", Payload: map[string]any{"prompt_tokens": float64(100)}},
	)
	r := newTestRunner(client, models.Model{})
	runToCompletion(t, r)

	client.append(
		Event{Kind: "text_delta", Payload: map[string]any{"text": "two"}},
		Event{Kind: "usage", Payload: map[string]any{"prompt_tokens": float64(50)}},
	)
	runToCompletion(t, r)

	stored := messagesOf(t, r)
	var answers []string
	for _, m := range stored {
		if m.Role == message.Assistant {
			answers = append(answers, textOf(m))
		}
	}
	if len(answers) != 2 || answers[0] != "one" || answers[1] != "two" {
		t.Fatalf("answers = %q, want each run drawn once", answers)
	}

	got, err := r.sessions.Get(context.Background(), "S1")
	if err != nil {
		t.Fatalf("the run's session is gone: %v", err)
	}
	if got.PromptTokens != 150 {
		t.Fatalf("tokens = %d, want each report counted once", got.PromptTokens)
	}
}

// A run that failed has to say so on the path the program reads. Reporting it
// only on the channel nobody reads is a run that dies in silence.
func TestRunFailureIsPublished(t *testing.T) {
	client := &timelineClient{status: "running", err: errors.New("daemon went away")}
	r := newTestRunner(client, models.Model{})
	sub := r.Subscribe(context.Background())

	out, err := r.Run(context.Background(), "S1", "say hello")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	deadline := time.After(10 * time.Second)
	for range out {
		select {
		case <-deadline:
			t.Fatal("the run never finished")
		default:
		}
	}

	// The client retries a poll that failed before it reports anything, so the
	// events before the failure are its own account of the retries.
	deadline2 := time.After(20 * time.Second)
	for {
		select {
		case event := <-sub:
			if event.Payload.Type == agent.AgentEventTypeConnection {
				continue
			}
			if event.Payload.Type != agent.AgentEventTypeError || event.Payload.Error == nil {
				t.Fatalf("expected an error event, got %+v", event.Payload)
			}
			return
		case <-deadline2:
			t.Fatal("the failure never reached the broker")
		}
	}
}

// A rotated log is re-read rather than added to, so a rewritten timeline cannot
// inflate what the session is said to have spent.
func TestRewoundLogIsReDerived(t *testing.T) {
	client := &timelineClient{status: "complete"}
	client.append(
		Event{Kind: "usage", Payload: map[string]any{"prompt_tokens": float64(100)}},
		Event{Kind: "usage", Payload: map[string]any{"prompt_tokens": float64(100)}},
	)
	r := newTestRunner(client, models.Model{})
	runToCompletion(t, r)

	// The daemon rotated the file: only the tail survives.
	client.log = []Event{{Kind: "usage", Payload: map[string]any{"prompt_tokens": float64(100)}}}
	runToCompletion(t, r)

	got, err := r.sessions.Get(context.Background(), "S1")
	if err != nil {
		t.Fatalf("the run's session is gone: %v", err)
	}
	if got.PromptTokens != 100 {
		t.Fatalf("tokens = %d, want the total re-derived from the surviving log", got.PromptTokens)
	}
}

// pushStreamingClient delivers events via push channel (ADR 014).
type pushStreamingClient struct {
	*timelineClient
	stream chan Event
	diff   prumo.DiffResponse
}

func (p *pushStreamingClient) Subscribe(ctx context.Context, runID string, from int) (<-chan Event, error) {
	return p.stream, nil
}

func (p *pushStreamingClient) Diff(ctx context.Context, runID, path string) (prumo.DiffResponse, error) {
	return p.diff, nil
}

func TestRunnerSubscribePush(t *testing.T) {
	ch := make(chan Event, 10)
	client := &pushStreamingClient{
		timelineClient: &timelineClient{status: "running"},
		stream:         ch,
	}
	r := newTestRunner(client, models.Model{})

	out, err := r.Run(context.Background(), "S-stream", "say hello")
	if err != nil {
		t.Fatal(err)
	}

	// Stream an event
	ch <- Event{Kind: "text_delta", Payload: map[string]any{"text": "hello push"}}
	ch <- Event{Kind: "run.finished", Payload: map[string]any{"status": "complete"}}
	close(ch)

	timeout := time.After(2 * time.Second)
	foundText := false
	for {
		select {
		case ev, ok := <-out:
			if !ok {
				if !foundText {
					t.Fatal("stream closed before receiving text")
				}
				return
			}
			if len(ev.Message.Parts) > 0 {
				if tc, ok := ev.Message.Parts[0].(message.TextContent); ok && strings.Contains(tc.Text, "hello push") {
					foundText = true
				}
			}
		case <-timeout:
			t.Fatal("timed out waiting for pushed events")
		}
	}
}

func TestRunnerDiff(t *testing.T) {
	client := &pushStreamingClient{
		timelineClient: &timelineClient{status: "complete"},
		diff:           prumo.DiffResponse{Path: "test.go", Kind: "patch", Content: "diff content"},
	}
	r := newTestRunner(client, models.Model{})

	resp, err := r.Diff(context.Background(), "S1", "test.go")
	if err != nil {
		t.Fatalf("diff failed: %v", err)
	}
	if resp.Path != "test.go" || resp.Kind != "patch" || resp.Content != "diff content" {
		t.Fatalf("unexpected diff: %+v", resp)
	}
}
