package headless

import (
	"bytes"
	"context"
	"strings"
	"testing"

	prumo "github.com/raillen/prumo/sdk/prumo"

	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/runtime"
)

// A headless run is the whole path a script takes: a daemon that answers, the
// bridge that carries its events, and the printer. Anything less would test the
// printer and not the mode.
func TestAHeadlessRunPrintsWhatHappened(t *testing.T) {
	client := &stubDaemon{events: []runtime.Event{
		{Kind: "text_delta", Payload: map[string]any{"text": "Two files were written."}},
		{Kind: "usage", Payload: map[string]any{"prompt_tokens": float64(140), "completion_tokens": float64(12), "cost_usd": 0.0007}},
	}}
	application := app.New(app.Options{Client: client, Provider: "fake", MaxTurns: 3})

	out := &bytes.Buffer{}
	err := Run(context.Background(), application, Options{Goal: "write two files", Mode: ModePlain, Out: out})
	if err != nil {
		t.Fatalf("headless run: %v", err)
	}

	got := out.String()
	for _, want := range []string{"Two files were written.", "140 tokens"} {
		if !strings.Contains(got, want) {
			t.Errorf("the run's account lost %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Errorf("the account carries terminal styling:\n%q", got)
	}
}

// The same run in JSON is the same facts, encoded for a program.
func TestAHeadlessRunCanSpeakJSON(t *testing.T) {
	client := &stubDaemon{events: []runtime.Event{
		{Kind: "text_delta", Payload: map[string]any{"text": "done"}},
	}}
	application := app.New(app.Options{Client: client, Provider: "fake", MaxTurns: 3})

	out := &bytes.Buffer{}
	if err := Run(context.Background(), application, Options{Goal: "do it", Mode: ModeJSON, Out: out}); err != nil {
		t.Fatalf("headless run: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if !strings.HasPrefix(line, "{") {
			t.Fatalf("a line is not a JSON object: %q", line)
		}
	}
}

// stubDaemon is a daemon that has already done the work.
type stubDaemon struct {
	events []runtime.Event
}

func (d *stubDaemon) Start(context.Context, prumo.StartRequest) (string, error) {
	return "S1", nil
}
func (d *stubDaemon) Status(context.Context, string) (prumo.RunStatus, error) {
	return prumo.RunStatus{Status: "complete"}, nil
}
func (d *stubDaemon) Events(context.Context, string) ([]prumo.Event, error) {
	out := make([]prumo.Event, 0, len(d.events))
	for _, event := range d.events {
		out = append(out, prumo.Event{Kind: event.Kind, Payload: event.Payload})
	}
	return out, nil
}
func (d *stubDaemon) Cancel(context.Context, string) error               { return nil }
func (d *stubDaemon) Steer(context.Context, string, string) error        { return nil }
func (d *stubDaemon) Approve(context.Context, string, string) error      { return nil }
func (d *stubDaemon) Deny(context.Context, string, string, string) error { return nil }
func (d *stubDaemon) List(context.Context) ([]prumo.RunStatus, error)    { return nil, nil }
func (d *stubDaemon) Jobs(context.Context) ([]prumo.Job, error)          { return nil, nil }
func (d *stubDaemon) ModelInfo(context.Context, prumo.ModelsRequest) ([]prumo.ModelInfo, error) {
	return nil, nil
}
func (d *stubDaemon) Unschedule(context.Context, string) error { return nil }
func (d *stubDaemon) Models(context.Context, prumo.ModelsRequest) ([]string, error) {
	return nil, nil
}
func (d *stubDaemon) Diff(context.Context, string, string) (prumo.DiffResponse, error) {
	return prumo.DiffResponse{}, nil
}
func (d *stubDaemon) Subscribe(context.Context, string, int) (<-chan prumo.Event, error) {
	return nil, nil
}

var _ runtime.Client = (*stubDaemon)(nil)
