// Package acp is the editor-integration bridge (GAP-010 first slice).
//
// Scope honesty: this is a Prumo-shaped session bridge (new/prompt/cancel/
// events/approve-shape) over the daemon protocol — not a certified
// implementation of Zed's Agent Client Protocol. The full ACP handshake
// stays pending GAP-032. Editors talk here; the Harness stays canonical.
package acp

import (
	"context"
	"fmt"
	"time"

	"github.com/raillen/prumo/internal/harness/daemon"
)

// Session is the bridge's external session handle.
type Session struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
}

// Bridge maps editor session calls onto daemon runs.
type Bridge struct {
	Daemon daemon.Client
}

// NewSession starts a backing run titled for editors.
func (b Bridge) NewSession(ctx context.Context, goal string) (Session, error) {
	_ = ctx
	res, err := b.Daemon.Call(map[string]any{"op": "start", "goal": goal, "provider": "fake", "max_turns": 10})
	if err != nil {
		return Session{}, err
	}
	if ok, _ := res["ok"].(bool); !ok {
		return Session{}, fmt.Errorf("%v", res["error"])
	}
	id, _ := res["run_id"].(string)
	return Session{ID: id, Provider: "prumo-native"}, nil
}

// Prompt steers a live session; finished/unknown sessions start a fresh run
// carrying the prompt (editors never lose input to a dead session).
func (b Bridge) Prompt(ctx context.Context, sessionID, message string) (Session, error) {
	_ = ctx
	res, err := b.Daemon.Call(map[string]any{"op": "steer", "run_id": sessionID, "message": message})
	if err != nil {
		return Session{}, err
	}
	if ok, _ := res["ok"].(bool); ok {
		return Session{ID: sessionID, Provider: "prumo-native"}, nil
	}
	return b.NewSession(ctx, message)
}

// Cancel stops the backing run.
func (b Bridge) Cancel(_ context.Context, sessionID string) error {
	res, err := b.Daemon.Call(map[string]any{"op": "cancel", "run_id": sessionID})
	if err != nil {
		return err
	}
	if ok, _ := res["ok"].(bool); !ok {
		return fmt.Errorf("%v", res["error"])
	}
	return nil
}

// Events replays the session timeline (capped server-side).
func (b Bridge) Events(_ context.Context, sessionID string) ([]any, error) {
	res, err := b.Daemon.Call(map[string]any{"op": "events", "run_id": sessionID})
	if err != nil {
		return nil, err
	}
	if ok, _ := res["ok"].(bool); !ok {
		return nil, fmt.Errorf("%v", res["error"])
	}
	evs, _ := res["events"].([]any)
	return evs, nil
}

// Wait polls until the backing run leaves running.
func (b Bridge) Wait(ctx context.Context, sessionID string) (string, error) {
	t := time.NewTicker(50 * time.Millisecond)
	defer t.Stop()
	for {
		res, err := b.Daemon.Call(map[string]any{"op": "status", "run_id": sessionID})
		if err != nil {
			return "", err
		}
		if st, _ := res["status"].(string); st != "" && st != "running" {
			return st, nil
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-t.C:
		}
	}
}
