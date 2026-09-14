package doccompile

import (
	"path/filepath"
	"testing"
)

func TestDAGOrderAndNoOp(t *testing.T) {
	c := New()
	c.Add(Node{ID: "a", Content: "# A"})
	c.Add(Node{ID: "b", Inputs: []string{"a"}, Content: "B"})
	rendered, order, err := c.Render()
	if err != nil || len(order) != 2 || order[0] != "a" {
		t.Fatalf("order failed: %v %v", order, err)
	}
	if rendered == "" {
		t.Fatal("empty render")
	}
	path := filepath.Join(t.TempDir(), "out.md")
	a1, err := Write(path, rendered)
	if err != nil || a1.NoOp {
		t.Fatalf("first write must not be noop: %v %+v", err, a1)
	}
	a2, err := Write(path, rendered)
	if err != nil || !a2.NoOp {
		t.Fatalf("second write must be noop: %v %+v", err, a2)
	}
}

func TestCycleDetected(t *testing.T) {
	c := New()
	c.Add(Node{ID: "a", Inputs: []string{"b"}, Content: "a"})
	c.Add(Node{ID: "b", Inputs: []string{"a"}, Content: "b"})
	if _, _, err := c.Render(); err == nil {
		t.Fatal("expected cycle error")
	}
}
