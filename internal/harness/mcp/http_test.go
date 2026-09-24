package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPTransportSession(t *testing.T) {
	var gotSession string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var req Request
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotSession = r.Header.Get("Mcp-Session-Id")
		w.Header().Set("Mcp-Session-Id", "sess-1")
		w.Header().Set("Content-Type", "application/json")
		// A notification is answered with no body. Replying to one would leave
		// an unread response in the transport's pending queue, and the next real
		// request would read it instead of its own.
		if req.ID == 0 {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		var result map[string]any
		switch req.Method {
		case "initialize":
			result = map[string]any{
				"protocolVersion": protocolVersion,
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "fake", "version": "1"},
			}
		default:
			result = map[string]any{"tools": []any{map[string]any{"name": "t"}}}
		}
		resp, _ := json.Marshal(Response{JSONRPC: "2.0", ID: req.ID, Result: result})
		_, _ = w.Write(resp)
	}))
	defer srv.Close()
	tr := &HTTPTransport{URL: srv.URL}
	c := &Client{Transport: tr, Timeout: 5 * time.Second}
	tools, err := c.List(context.Background())
	if err != nil || len(tools) != 1 {
		t.Fatalf("http list failed: %+v %v", tools, err)
	}
	// Second call carries the session id back.
	if _, err := c.List(context.Background()); err != nil {
		t.Fatal(err)
	}
	if gotSession != "sess-1" {
		t.Fatalf("session id not echoed: %q", gotSession)
	}
	bad := &HTTPTransport{URL: srv.URL + "/missing"}
	if _, err := (&Client{Transport: bad, Timeout: time.Second}).List(context.Background()); err == nil {
		t.Fatal("http error must surface")
	}
}
