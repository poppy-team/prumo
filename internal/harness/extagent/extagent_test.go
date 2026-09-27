package extagent

import (
	"context"
	"testing"
)

func TestFakeAgentLifecycle(t *testing.T) {
	f := &FakeAgent{}
	ctx := context.Background()
	caps, err := f.Capabilities(ctx)
	if err != nil || len(caps) == 0 {
		t.Fatalf("caps failed: %v", caps)
	}
	s, err := f.CreateSession(ctx, "R1")
	if err != nil || s.ID == "" {
		t.Fatal("session failed")
	}
	if err := f.Send(ctx, s.ID, "hello"); err != nil {
		t.Fatal(err)
	}
	ch, err := f.Events(ctx, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for range ch {
		n++
	}
	if n != 2 {
		t.Fatalf("expected 2 normalized events, got %d", n)
	}
}

func TestCodexCLILifecycleAndCancel(t *testing.T) {
	c := NewCodexCLI("codex")
	ctx := context.Background()

	caps, err := c.Capabilities(ctx)
	if err != nil || len(caps) != 3 {
		t.Fatalf("unexpected caps: %v", caps)
	}

	started := make(chan struct{})
	c.Runner = func(ctx context.Context, bin string, args ...string) (string, error) {
		close(started)
		<-ctx.Done()
		return "", ctx.Err()
	}

	s, err := c.CreateSession(ctx, "run-1")
	if err != nil || s.ID == "" {
		t.Fatalf("CreateSession failed: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Send(ctx, s.ID, "do task")
	}()

	<-started
	if err := c.Cancel(ctx, s.ID); err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	sendErr := <-errCh
	if sendErr == nil || sendErr != context.Canceled {
		t.Fatalf("expected context.Canceled, got: %v", sendErr)
	}
}

func TestCursorCLILifecycleAndCancel(t *testing.T) {
	c := NewCursorCLI("cursor-agent")
	ctx := context.Background()

	caps, err := c.Capabilities(ctx)
	if err != nil || len(caps) != 3 {
		t.Fatalf("unexpected caps: %v", caps)
	}

	started := make(chan struct{})
	c.Runner = func(ctx context.Context, bin string, args ...string) (string, error) {
		close(started)
		<-ctx.Done()
		return "", ctx.Err()
	}

	s, err := c.CreateSession(ctx, "run-cursor")
	if err != nil || s.ID == "" {
		t.Fatalf("CreateSession failed: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Send(ctx, s.ID, "do task")
	}()

	<-started
	if err := c.Cancel(ctx, s.ID); err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	sendErr := <-errCh
	if sendErr == nil || sendErr != context.Canceled {
		t.Fatalf("expected context.Canceled, got: %v", sendErr)
	}
}
