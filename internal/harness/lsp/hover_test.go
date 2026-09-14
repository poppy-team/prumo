package lsp

import (
	"context"
	"testing"
	"time"
)

func TestHoverAndDefinition(t *testing.T) {
	clientSide, serverSide := NewPipe()
	fakeServer(t, serverSide)
	c := &Client{Transport: clientSide, Root: "/work", Timeout: 5 * time.Second}
	hover, err := c.Hover(context.Background(), "/work/a.go", 10, 4)
	if err != nil || hover != "func NewRunner() *Runner" {
		t.Fatalf("hover failed: %q %v", hover, err)
	}
	def, err := c.Definition(context.Background(), "/work/a.go", 10, 4)
	if err != nil || def.File != "/work/runtime.go" || def.Line != 56 {
		t.Fatalf("definition failed: %+v %v", def, err)
	}
	fb := FallbackProvider{}
	if _, err := fb.Hover(context.Background(), "f", 1, 1); err == nil {
		t.Fatal("fallback hover must be explicit")
	}
	if _, err := fb.Definition(context.Background(), "f", 1, 1); err == nil {
		t.Fatal("fallback definition must be explicit")
	}
}
