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
