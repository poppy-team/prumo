package localintel

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestSupervisorLifecycle(t *testing.T) {
	s := NewSupervisor()
	s.Backoff = 5 * time.Millisecond
	var runs int64
	if err := s.Start("w", func(ctx context.Context) error {
		atomic.AddInt64(&runs, 1)
		<-ctx.Done()
		return ctx.Err()
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Start("w", func(context.Context) error { return nil }); err == nil {
		t.Fatal("double start must fail")
	}
	time.Sleep(50 * time.Millisecond)
	if s.Health("w") != "running" {
		t.Fatalf("must be running: %s", s.Health("w"))
	}
	if atomic.LoadInt64(&runs) != 1 {
		t.Fatalf("clean worker must run once, got %d", runs)
	}
	if err := s.Stop("w"); err != nil {
		t.Fatal(err)
	}
	if s.Health("ghost") != "unknown" {
		t.Fatal("unknown worker must report unknown")
	}
	if err := s.Stop("w"); err == nil {
		t.Fatal("double stop must fail")
	}
}

func TestSupervisorRestartsFailures(t *testing.T) {
	s := NewSupervisor()
	s.Backoff = 5 * time.Millisecond
	var runs int64
	if err := s.Start("flaky", func(context.Context) error {
		atomic.AddInt64(&runs, 1)
		return errors.New("boom")
	}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt64(&runs) < 3 {
		if time.Now().After(deadline) {
			t.Fatalf("no restarts: %d", runs)
		}
		time.Sleep(5 * time.Millisecond)
	}
	_ = s.Stop("flaky")
}

func TestDetectLlamaServerHonest(t *testing.T) {
	// Must not crash; result depends on host (usually absent in CI).
	_ = DetectLlamaServer()
}
