package tui

import (
	"context"

	prumo "github.com/raillen/prumo/sdk/prumo"

	"github.com/raillen/prumo-tui/internal/runtime"
)

// stubHarness is a daemon that answers without being one. The view layer never
// talks to a provider, so a test of the shell needs only the shape of the
// protocol, not a process behind it.
type stubHarness struct {
	status      string
	events      []prumo.Event
	started     []prumo.StartRequest
	stops       []string
	steered     []string
	jobs        []prumo.Job
	modelInfo   []prumo.ModelInfo
	unscheduled []string
}

func (h *stubHarness) Start(_ context.Context, r prumo.StartRequest) (string, error) {
	h.started = append(h.started, r)
	if r.RunID != "" {
		return r.RunID, nil
	}
	return "R-1", nil
}

func (h *stubHarness) Status(context.Context, string) (prumo.RunStatus, error) {
	status := h.status
	if status == "" {
		status = "complete"
	}
	return prumo.RunStatus{RunID: "R-1", Status: status}, nil
}

func (h *stubHarness) Events(context.Context, string) ([]prumo.Event, error) {
	return h.events, nil
}
func (h *stubHarness) Cancel(_ context.Context, runID string) error {
	h.stops = append(h.stops, runID)
	return nil
}
func (h *stubHarness) Jobs(context.Context) ([]prumo.Job, error) {
	return h.jobs, nil
}
func (h *stubHarness) ModelInfo(context.Context, prumo.ModelsRequest) ([]prumo.ModelInfo, error) {
	return h.modelInfo, nil
}
func (h *stubHarness) Unschedule(_ context.Context, jobID string) error {
	h.unscheduled = append(h.unscheduled, jobID)
	return nil
}
func (h *stubHarness) Steer(_ context.Context, runID, message string) error {
	h.steered = append(h.steered, message)
	return nil
}
func (h *stubHarness) Approve(context.Context, string, string) error { return nil }
func (h *stubHarness) Deny(context.Context, string, string, string) error {
	return nil
}
func (h *stubHarness) List(context.Context) ([]prumo.RunStatus, error) { return nil, nil }
func (h *stubHarness) Models(context.Context, prumo.ModelsRequest) ([]string, error) {
	return nil, nil
}
func (h *stubHarness) Diff(context.Context, string, string) (prumo.DiffResponse, error) {
	return prumo.DiffResponse{}, nil
}
func (h *stubHarness) Subscribe(context.Context, string, int) (<-chan prumo.Event, error) {
	return nil, nil
}

var _ runtime.Client = (*stubHarness)(nil)
