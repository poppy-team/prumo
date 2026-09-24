package aci

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

// The cap bounded what the model saw and not what the process held. A command
// that printed ten gigabytes filled memory before `bound` ever ran, and a file
// read with os.ReadFile was loaded whole and truncated afterwards. The surface
// read as though output was bounded (GAP-170).

func TestALoudCommandIsCappedWhileItRuns(t *testing.T) {
	root := t.TempDir()
	e := New(root)
	e.OutputMax = 4096
	// Far more than the cap, from a process that keeps going: if the buffer only
	// truncated at the end, this is what filled memory.
	res, err := e.Execute(context.Background(), agent.ToolCall{
		ID: "c1", TurnID: "t1", Name: "process.exec",
		Arguments: map[string]any{"command": "seq 1 200000"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Output) > e.OutputMax+len("\n…[truncated]") {
		t.Fatalf("output was %d bytes, over the %d cap", len(res.Output), e.OutputMax)
	}
	if !res.Truncated {
		t.Fatal("dropped output was not reported as truncated")
	}
}

func TestReadingAHugeFileNeverLoadsItWhole(t *testing.T) {
	root := t.TempDir()
	// A file well past any sane OutputMax. Read whole, this is the allocation
	// that kills the process; read capped, it is four kilobytes.
	big := strings.Repeat("A", 8<<20)
	writeFile(t, filepath.Join(root, "big.txt"), big)
	e := New(root)
	e.OutputMax = 4096
	res, err := e.Execute(context.Background(), agent.ToolCall{
		ID: "c1", TurnID: "t1", Name: "fs.read", Arguments: map[string]any{"path": "big.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Output) > e.OutputMax+len("\n…[truncated]") {
		t.Fatalf("read returned %d bytes, over the %d cap", len(res.Output), e.OutputMax)
	}
	if !res.Truncated {
		t.Fatal("a partial read was not reported as truncated")
	}
}

func TestOutputUnderTheCapIsNotMarkedTruncated(t *testing.T) {
	// A cap that always claims truncation is a cap nobody can trust.
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "small.txt"), "hello")
	e := New(root)
	res, err := e.Execute(context.Background(), agent.ToolCall{
		ID: "c1", TurnID: "t1", Name: "fs.read", Arguments: map[string]any{"path": "small.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Truncated {
		t.Fatal("a small complete read was reported as truncated")
	}
	if res.Output != "hello" {
		t.Fatalf("output = %q, want %q", res.Output, "hello")
	}
}
