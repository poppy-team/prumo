package runtime

import (
	"context"
	"testing"

	prumo "github.com/raillen/prumo/sdk/prumo"

	"github.com/raillen/prumo-tui/internal/llm/models"
	"github.com/raillen/prumo-tui/internal/message"
	"github.com/raillen/prumo-tui/internal/session"
)

// stubClient records what the runner asked the harness, and answers with what a
// test tells it to. It stands in for the socket.
type stubClient struct {
	started   []StartRequest
	models    []string
	modelReqs []prumo.ModelsRequest
}

func (s *stubClient) Start(_ context.Context, r StartRequest) (string, error) {
	s.started = append(s.started, r)
	return r.RunID, nil
}
func (s *stubClient) Status(context.Context, string) (RunStatus, error) {
	return RunStatus{Status: "complete"}, nil
}
func (s *stubClient) Events(context.Context, string) ([]Event, error)    { return nil, nil }
func (s *stubClient) Cancel(context.Context, string) error               { return nil }
func (s *stubClient) Approve(context.Context, string, string) error      { return nil }
func (s *stubClient) Deny(context.Context, string, string, string) error { return nil }
func (s *stubClient) List(context.Context) ([]RunStatus, error)          { return nil, nil }
func (s *stubClient) Models(_ context.Context, r prumo.ModelsRequest) ([]string, error) {
	s.modelReqs = append(s.modelReqs, r)
	return s.models, nil
}

func newTestRunner(client Client, model models.Model) *Runner {
	return NewRunner(Options{
		Client:   client,
		Sessions: session.NewStore(),
		Messages: message.NewStore(),
		Model:    model,
	})
}

// TestRunUsesTheModelIDWhenThereIsNoWireName pins the path a picker takes: a
// model chosen from the harness's list arrives as an id, with no separate wire
// name. Sending the empty name would ask the daemon for its default instead of
// for what the user picked — a bug that only shows up with a real choice.
func TestRunUsesTheModelIDWhenThereIsNoWireName(t *testing.T) {
	client := &stubClient{}
	r := newTestRunner(client, models.Model{ID: "gpt-4o-mini", Name: "gpt-4o-mini"})

	if _, err := r.Run(context.Background(), "S1", "do the thing"); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(client.started) != 1 {
		t.Fatalf("the runner did not start a run: %v", client.started)
	}
	if got := client.started[0].Model; got != "gpt-4o-mini" {
		t.Fatalf("model sent = %q, want the id the user chose", got)
	}
}

// TestRunPrefersTheWireNameWhenThereIsOne: when the model carries the name the
// provider expects upstream, that is what travels — the id is the client's name
// for it, not necessarily the endpoint's.
func TestRunPrefersTheWireNameWhenThereIsOne(t *testing.T) {
	client := &stubClient{}
	r := newTestRunner(client, models.Model{ID: "local-default", APIModel: "granite-3.3-2b"})

	if _, err := r.Run(context.Background(), "S1", "go"); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got := client.started[0].Model; got != "granite-3.3-2b" {
		t.Fatalf("model sent = %q, want the provider's wire name", got)
	}
}

// TestAvailableModelsAsksTheHarness: the catalogue belongs to the provider, so
// the client's only job is to ask and hand the answer on.
func TestAvailableModelsAsksTheHarness(t *testing.T) {
	client := &stubClient{models: []string{"alpha", "beta"}}
	r := newTestRunner(client, models.Model{})

	got, err := r.AvailableModels(context.Background(), "openai-compat")
	if err != nil {
		t.Fatalf("models: %v", err)
	}
	if len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("models = %v", got)
	}
	if len(client.modelReqs) != 1 || client.modelReqs[0].Provider != "openai-compat" {
		t.Fatalf("the provider was not passed through: %v", client.modelReqs)
	}
}

// TestAvailableModelsWithoutAHarness is the honest failure: a client that is
// not attached says so rather than reporting an empty catalogue, which would
// read as "the provider serves nothing".
func TestAvailableModelsWithoutAHarness(t *testing.T) {
	r := NewRunner(Options{})
	if _, err := r.AvailableModels(context.Background(), ""); err == nil {
		t.Fatal("asking with no harness attached must be refused, not answered empty")
	}
}
