package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
	harnessprotocol "github.com/raillen/prumo/internal/harness/protocol"
)

// The SDK exposed Subscribe. The local connection handler implemented it. The
// remote one dispatched every request and replied, so a client that subscribed
// over TCP got one result and then the end of the stream — the method existed,
// was documented, and could not work against a remote daemon (GAP-166).

func TestSubscribeWorksOverARemoteConnection(t *testing.T) {
	dir := t.TempDir()
	srv := New(filepath.Join(dir, "remote.sock"), filepath.Join(dir, "store"), fakeDeps(false))
	runID, stopReason := srv.runOnceForTest(t.Context(), dir, "say something")
	if stopReason == "the run did not stop" {
		t.Fatal("the run did not finish, so there are no events to stream")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		conn, acceptErr := ln.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		srv.handleRemote(conn, "secret")
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))

	reader := bufio.NewReader(conn)
	send := func(v map[string]any) {
		t.Helper()
		data, _ := json.Marshal(v)
		if _, err := conn.Write(append(data, '\n')); err != nil {
			t.Fatal(err)
		}
	}
	readOne := func() map[string]any {
		t.Helper()
		for {
			line, readErr := reader.ReadBytes('\n')
			if readErr != nil {
				t.Fatalf("the stream ended: %v", readErr)
			}
			var event map[string]any
			if json.Unmarshal(line, &event) == nil {
				return event
			}
		}
	}

	send(map[string]any{"auth": "wrong-token"})
	if denied := readOne(); denied["ok"] != false {
		t.Fatalf("a wrong token was accepted: %v", denied)
	}
	// A wrong token ends the connection; reconnect for the real exchange.
	conn.Close()
	conn, err = net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
	reader = bufio.NewReader(conn)
	go func() {
		accepted, acceptErr := ln.Accept()
		if acceptErr != nil {
			return
		}
		srv.handleRemote(accepted, "secret")
	}()

	send(map[string]any{"auth": "secret"})
	if authed := readOne(); authed["ok"] != true {
		t.Fatalf("the right token was refused: %v", authed)
	}

	// The subscription itself. Before the change this got a single dispatched
	// result and the connection closed.
	send(map[string]any{
		"protocol_version": harnessprotocol.Version,
		"op":               "subscribe",
		"run_id":           runID,
		"from":             0,
	})

	// The run has already finished, so the stream is the backlog and then a
	// clean close. Before the change there was no stream: one dispatched
	// result, then the socket closed.
	seen := 0
	deadline := time.Now().Add(15 * time.Second)
	for seen < 2 && time.Now().Before(deadline) {
		if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Fatal(err)
		}
		message := readOne()
		if op, _ := message["op"].(string); op == "event" {
			seen++
		}
	}
	if seen < 2 {
		t.Fatalf("a remote subscription delivered %d events; the stream is not a stream", seen)
	}
}

func TestASubscriptionDoesNotOutliveItsConnection(t *testing.T) {
	// Closing the socket must take the stream with it. The local handler gained
	// this when slow subscribers were disconnected; the remote one never had it
	// because it never had a stream (GAP-166).
	s := newScheduleServer(t)
	path := mustEventPath(t, s, "R-stream")
	body := ""
	for i := range 3 {
		event, _ := json.Marshal(agent.AgentEvent{
			ID: "ev-" + string(rune('a'+i)), RunID: "R-stream", Kind: "text_delta",
		})
		body += string(event) + "\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	write := make(chan map[string]any, 64)
	done := make(chan struct{})
	var stop replacedSubscription
	stop.replace(s, ctx, map[string]any{"op": "subscribe", "run_id": "R-stream", "from": 0},
		func(v map[string]any) error {
			select {
			case write <- v:
			default:
			}
			return nil
		})
	go func() { defer close(done); <-ctx.Done() }()

	select {
	case <-write:
	case <-time.After(5 * time.Second):
		t.Fatal("the subscription delivered nothing at all")
	}
	cancel()
	stop()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("cancelling the connection did not stop the subscription")
	}
}
