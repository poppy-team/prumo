package acpserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"
)

// `serveConn` checked ctx.Done() only between frames, with the frame read inline
// between the checks. So the check only ran once a frame had already arrived: a
// server whose peer went quiet sat in readFrame forever, and the context that
// was supposed to end it could not. A goroutine that blocked on ctx.Done() and
// did nothing else sat beside it (GAP-159).

func TestAServerWithAQuietPeerStopsWhenItsContextIsCancelled(t *testing.T) {
	// An io.Pipe with nothing written and never closed: the reader blocks, and
	// the test's assertion is that the server still returns.
	reader, writer := io.Pipe()
	defer writer.Close()

	server := &Server{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- server.serveConn(ctx, bufio.NewReader(reader), bufio.NewWriter(&bytes.Buffer{}))
	}()

	select {
	case err := <-done:
		t.Fatalf("the server returned before cancellation: %v", err)
	case <-time.After(200 * time.Millisecond):
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a cancelled context did not stop a server blocked reading from a quiet peer")
	}
}

func TestACancelledServerStopsBeforeStartingTheNextRequest(t *testing.T) {
	// A frame that arrives after cancellation must not spawn a handler. A server
	// that answers requests nobody is waiting for is a server with no end.
	reader, writer := io.Pipe()
	server := &Server{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// A complete frame, so the read succeeds if it is ever attempted.
	frame, _ := json.Marshal(rpcMsg{JSONRPC: "2.0", ID: ptr(int64(1)), Method: "initialize", Params: json.RawMessage(`{}`)})
	go func() {
		_, _ = writer.Write(append([]byte(fmt.Sprintf("Content-Length: %d\\r\\n\\r\\n", len(frame))), frame...))
		_ = writer.Close()
	}()
	done := make(chan error, 1)
	go func() {
		done <- server.serveConn(ctx, bufio.NewReader(reader), bufio.NewWriter(&bytes.Buffer{}))
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a cancelled server did not return")
	}
}

func ptr[T any](v T) *T { return &v }
