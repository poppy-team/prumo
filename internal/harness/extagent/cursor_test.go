package extagent

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCursorSendSuccess(t *testing.T) {
	c := NewCursorCLI("cursor-agent")
	c.Runner = func(ctx context.Context, bin string, args ...string) (string, error) {
		if bin != "cursor-agent" {
			t.Fatalf("unexpected bin %q", bin)
		}
		joined := strings.Join(args, " ")
		for _, want := range []string{"-p", "--output-format json", "--trust"} {
			if !strings.Contains(joined, want) {
				t.Fatalf("argv missing %q: %s", want, joined)
			}
		}
		return `{"type":"system","subtype":"init"}
{"type":"result","result":"cursor-live-ok","is_error":false}`, nil
	}
	if err := c.Send(context.Background(), "cursor-R1", "hi"); err != nil {
		t.Fatalf("send must succeed: %v", err)
	}
}

func TestCursorSendProviderErrorEnvelope(t *testing.T) {
	c := NewCursorCLI("cursor-agent")
	c.Runner = func(ctx context.Context, bin string, args ...string) (string, error) {
		return `{"type":"result","result":"usage limit reached","is_error":true}`, nil
	}
	err := c.Send(context.Background(), "cursor-R1", "hi")
	if err == nil || !strings.Contains(err.Error(), "usage limit reached") {
		t.Fatalf("provider error envelope must surface: %v", err)
	}
}

func TestCursorSendCommandFailure(t *testing.T) {
	c := NewCursorCLI("cursor-agent")
	c.Runner = func(ctx context.Context, bin string, args ...string) (string, error) {
		return "trust prompt appeared", context.DeadlineExceeded
	}
	err := c.Send(context.Background(), "cursor-R1", "hi")
	if err == nil || !strings.Contains(err.Error(), "cursor exec failed") {
		t.Fatalf("command failure must surface: %v", err)
	}
}

func TestCursorProbeMatrixRow(t *testing.T) {
	dir := t.TempDir()
	writeStub(t, dir, "cursor-agent", "echo 2026.07.23-e383d2b")
	p := Prober{
		Path: dir,
		Runner: func(ctx context.Context, bin string, args ...string) (string, error) {
			return "2026.07.23-e383d2b", nil
		},
	}
	var row *ProbeResult
	for i := range p.Matrix(context.Background()) {
		if p.Matrix(context.Background())[i].Name == "cursor-cli" {
			row = &p.Matrix(context.Background())[i]
		}
	}
	if row == nil {
		t.Fatal("cursor-cli row missing from matrix")
	}
	if !row.Available || row.Version == "" {
		t.Fatalf("cursor-cli must probe available: %+v", row)
	}
}

func TestCursorImplementsProvider(t *testing.T) {
	var _ Provider = NewCursorCLI("")
}

// TestCursorLiveSend exercises a real model turn via cursor-agent
// (user-approved spend). Auth is the binary's own login; no API key crosses
// the boundary.
func TestCursorLiveSend(t *testing.T) {
	if os.Getenv("PRUMO_LIVE_CURSOR_SEND") == "" {
		t.Skip("set PRUMO_LIVE_CURSOR_SEND=1 to approve real model spend")
	}
	if _, err := exec.LookPath("cursor-agent"); err != nil {
		t.Skip("cursor-agent not on PATH")
	}
	c := NewCursorCLI("cursor-agent")
	if err := c.Send(context.Background(), "cursor-live", "Reply with exactly: cursor-live-ok"); err != nil {
		t.Fatalf("live cursor send failed: %v", err)
	}
}
