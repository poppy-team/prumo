package acp

import (
	"context"
	"errors"
	"testing"
	"time"
)

// The bridge took a context and threw it away — `_ = ctx` — on both operations
// that start or steer a run. An editor closing a session could not stop the call
// it had already made, and the run it started outlived the request that asked
// for it (GAP-159).
//
// Bridge.Daemon is the concrete daemon.Client, so these tests use a real one
// against a socket that does not exist: a context that is not threaded through
// produces one specific wrong error, and a context that is produces another.

func TestNewSessionReportsCancellationRatherThanADialFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	b := Bridge{}
	// A cancelled context must be reported as a cancellation. Without the
	// context threaded through, the call fails on the socket instead, and a
	// caller cannot tell "the editor closed" from "the daemon is down".
	_, err := b.NewSession(ctx, "goal")
	if err == nil {
		t.Fatal("a cancelled context produced a session")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("a cancelled context produced %v, want context.Canceled", err)
	}
}

func TestPromptReportsCancellationRatherThanADialFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	b := Bridge{}
	if _, err := b.Prompt(ctx, "R-1", "hello"); !errors.Is(err, context.Canceled) {
		t.Fatalf("a cancelled context produced %v, want context.Canceled", err)
	}
}

func TestACancelledCallDoesNotWaitForTheSocket(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	b := Bridge{}
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	_, _ = b.NewSession(ctx, "goal")
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("a cancelled call took %s; the context never reached the dial", elapsed)
	}
}
