package extagent

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// framedRW composes a reader and a writer into the ReadWriteCloser the ACP
// client expects over a bidirectional pipe.
type framedRW struct {
	r io.Reader
	w io.Writer
}

func (f framedRW) Read(p []byte) (int, error)  { return f.r.Read(p) }
func (f framedRW) Write(p []byte) (int, error) { return f.w.Write(p) }
func (f framedRW) Close() error                { return nil }

func writeFrame(w io.Writer, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		return
	}
	_, _ = w.Write(append([]byte("Content-Length: "+strconv.Itoa(len(body))+"\r\n\r\n"), body...))
}

// acpStubAgent answers client requests speaking the same framed JSON-RPC as
// internal/harness/acpserver, for hermetic client tests. It reads client
// frames from `in` and writes frames to `out`.
func acpStubAgent(t *testing.T, in io.Reader, out io.Writer) {
	t.Helper()
	br := bufio.NewReader(in)
	for {
		raw, err := readACPPacket(br)
		if err != nil {
			return
		}
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(raw, &req)
		switch req.Method {
		case "initialize":
			writeFrame(out, map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{"protocolVersion": 1}})
		case "session/new":
			writeFrame(out, map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{"sessionId": "ses_stub"}})
		case "session/prompt":
			// One streamed update, then the stop result.
			writeFrame(out, map[string]any{"jsonrpc": "2.0", "method": "session/update", "params": map[string]any{
				"sessionId": "ses_stub",
				"update": map[string]any{
					"sessionUpdate": "agent_message_chunk",
					"content":       map[string]any{"type": "text", "text": "stub says hi"},
				},
			}})
			writeFrame(out, map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{"stopReason": "end_turn"}})
		default:
			writeFrame(out, map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{}})
		}
	}
}

func TestACPClientSessionAndPrompt(t *testing.T) {
	c2a, c2w := io.Pipe() // client -> agent
	a2c, a2w := io.Pipe() // agent -> client
	go acpStubAgent(t, c2a, a2w)
	t.Cleanup(func() { _ = c2a.Close(); _ = a2w.Close() })

	c := NewACPClient("stub")
	c.Start = func(ctx context.Context) (io.ReadWriteCloser, func(), error) {
		return framedRW{r: a2c, w: c2w}, func() {}, nil
	}

	s, err := c.CreateSession(context.Background(), "R-acp")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if s.ID != "ses_stub" || s.Provider != "acp" {
		t.Fatalf("bad session: %+v", s)
	}
	if err := c.Send(context.Background(), s.ID, "hello"); err != nil {
		t.Fatalf("send: %v", err)
	}
	evs, err := c.Events(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	sawUpdate := false
	for ev := range evs {
		if strings.Contains(ev.Kind, "update") {
			if txt, _ := ev.Payload["text"].(string); strings.Contains(txt, "stub says hi") {
				sawUpdate = true
			}
		}
	}
	if !sawUpdate {
		t.Fatal("streamed update not collected")
	}
	if err := c.Close(context.Background(), s.ID); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestACPClientPromptError(t *testing.T) {
	c2a, c2w := io.Pipe()
	a2c, a2w := io.Pipe()
	go acpStubAgent(t, c2a, a2w)
	t.Cleanup(func() { _ = c2a.Close(); _ = a2w.Close() })

	c := NewACPClient("stub")
	c.Start = func(ctx context.Context) (io.ReadWriteCloser, func(), error) {
		return framedRW{r: a2c, w: c2w}, func() {}, nil
	}
	s, err := c.CreateSession(context.Background(), "R-acp-err")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	// Cancel the context mid-send: client must attempt session/cancel and
	// surface the context error, not a hang.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.Send(ctx, s.ID, "hello"); err == nil {
		t.Fatal("cancelled send must error")
	}
}

func TestACPImplementsProvider(t *testing.T) {
	var _ Provider = NewACPClient("x")
}

// TestACPLiveDogfood spawns the REAL prumo binary: a daemon backed by the
// fake model provider + `agent acp` server + this client. It proves the
// full no-API-key external loop (Prumo → ACP → Prumo → fake model).
func TestACPLiveDogfood(t *testing.T) {
	if os.Getenv("PRUMO_LIVE_ACP") == "" {
		t.Skip("set PRUMO_LIVE_ACP=1 to run the live ACP dogfood")
	}
	bin := filepath.Join(t.TempDir(), "prumo-acp-test")
	build := exec.Command("go", "build", "-o", bin, "./cmd/prumo")
	build.Dir = "../../.."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %s %v", out, err)
	}
	dir := t.TempDir()
	sock := filepath.Join(dir, "agentd.sock")
	daemon := exec.Command(bin, "agent", "serve", "--path", dir, "--socket", sock)
	if err := daemon.Start(); err != nil {
		t.Fatalf("daemon start: %v", err)
	}
	t.Cleanup(func() {
		_ = daemon.Process.Kill()
		_, _ = daemon.Process.Wait()
	})
	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(sock); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("daemon socket never appeared")
		}
		time.Sleep(100 * time.Millisecond)
	}

	c := NewACPClient(bin, "agent", "acp", "--socket", sock, "--path", dir)
	s, err := c.CreateSession(context.Background(), "R-acp-live")
	if err != nil {
		t.Fatalf("live acp session: %v", err)
	}
	t.Cleanup(func() { _ = c.Close(context.Background(), s.ID) })

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := c.Send(ctx, s.ID, "dogfood prompt"); err != nil {
		t.Fatalf("live acp send: %v", err)
	}
	evs, err := c.Events(ctx, s.ID)
	if err != nil {
		t.Fatalf("live acp events: %v", err)
	}
	// DaemonBackend.Prompt streams "started backing run <id>" when a fresh
	// session has no active run (our case) and a final "run <status>" chunk.
	sawStart := false
	for ev := range evs {
		if txt, _ := ev.Payload["text"].(string); strings.Contains(txt, "backing run") {
			sawStart = true
		}
	}
	if !sawStart {
		t.Fatal("expected backing-run chunk from live ACP prompt")
	}
}
