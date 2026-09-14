// DaemonBackend wires the ACP server to a live harness daemon.
package acpserver

import (
	"context"
	"fmt"
	"time"

	"github.com/raillen/prumo/internal/harness/daemon"
)

// DaemonBackend maps ACP sessions onto daemon runs. Sessions outlive runs:
// prompting a finished session starts a fresh backing run (noted in the
// stream) instead of losing input.
type DaemonBackend struct {
	Daemon daemon.Client
}

func (b DaemonBackend) NewSession(_ context.Context, cwd string) (string, error) {
	if cwd == "" {
		cwd = "."
	}
	return "acp-" + fmt.Sprintf("%d", time.Now().UTC().UnixNano()), nil
}

func (b DaemonBackend) runFor(session string) string { return "R-" + session }

func (b DaemonBackend) Load(_ context.Context, id string) ([]ReplayEvent, error) {
	res, err := b.Daemon.Call(map[string]any{"op": "events", "run_id": b.runFor(id)})
	if err != nil {
		return nil, err
	}
	if ok, _ := res["ok"].(bool); !ok {
		return nil, fmt.Errorf("%v", res["error"])
	}
	evs, _ := res["events"].([]any)
	out := []ReplayEvent{}
	for _, e := range evs {
		m, _ := e.(map[string]any)
		kind, _ := m["kind"].(string)
		out = append(out, ReplayEvent{Kind: "agent", Text: "event " + kind})
	}
	return out, nil
}

func (b DaemonBackend) Resume(_ context.Context, id string) error {
	res, err := b.Daemon.Call(map[string]any{"op": "status", "run_id": b.runFor(id)})
	if err != nil {
		return err
	}
	if ok, _ := res["ok"].(bool); !ok {
		return fmt.Errorf("unknown session %s", id)
	}
	return nil
}

func (b DaemonBackend) List(_ context.Context) ([]string, error) {
	res, err := b.Daemon.Call(map[string]any{"op": "list"})
	if err != nil {
		return nil, err
	}
	var out []string
	for _, r := range res["runs"].([]any) {
		if m, _ := r.(map[string]any); m != nil {
			if id, _ := m["run_id"].(string); id != "" {
				out = append(out, id)
			}
		}
	}
	return out, nil
}

func (b DaemonBackend) Delete(_ context.Context, id string) error {
	return b.Close(context.Background(), id)
}

func (b DaemonBackend) Close(_ context.Context, id string) error {
	res, err := b.Daemon.Call(map[string]any{"op": "cancel", "run_id": b.runFor(id)})
	if err != nil {
		return err
	}
	if ok, _ := res["ok"].(bool); !ok {
		return fmt.Errorf("%v", res["error"])
	}
	return nil
}

// Cancel aborts the backing run (session/cancel notification path).
func (b DaemonBackend) Cancel(_ context.Context, id string) error {
	return b.Close(context.Background(), id)
}

// Prompt steers a live backing run, or starts a fresh one when the session
// has no active run. Completion streams as one summary chunk; watchers
// wanting full detail use the daemon timeline.
func (b DaemonBackend) Prompt(ctx context.Context, id, text string, emit func(Update)) (string, error) {
	runID := b.runFor(id)
	steer, err := b.Daemon.Call(map[string]any{"op": "steer", "run_id": runID, "message": text})
	steered := err == nil
	if s, _ := steer["ok"].(bool); err != nil || !s {
		start, serr := b.Daemon.Call(map[string]any{
			"op": "start", "goal": text, "provider": "fake", "run_id": runID, "max_turns": 10,
		})
		if serr != nil {
			return "", serr
		}
		if ok, _ := start["ok"].(bool); !ok {
			return "", fmt.Errorf("%v", start["error"])
		}
		emit(Update{SessionUpdate: "agent_message_chunk",
			Content: map[string]any{"type": "text", "text": "started backing run " + runID}})
	} else {
		_ = steered
	}
	deadline := time.Now().Add(5 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			return "cancelled", nil
		default:
		}
		st, err := b.Daemon.Call(map[string]any{"op": "status", "run_id": runID})
		if err != nil {
			return "", err
		}
		status, _ := st["status"].(string)
		if status != "" && status != "running" {
			phase, _ := st["phase"].(string)
			stop, _ := st["stop_reason"].(string)
			emit(Update{SessionUpdate: "agent_message_chunk",
				Content: map[string]any{"type": "text", "text": "run " + status + " (" + phase + "): " + stop}})
			switch status {
			case "complete":
				return "end_turn", nil
			case "cancelled":
				return "cancelled", nil
			default:
				return "end_turn", nil
			}
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("prompt timed out waiting for %s", runID)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
