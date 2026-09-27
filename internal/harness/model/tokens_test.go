package model

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	if got := EstimateTokens("", "x"); got != 1 {
		t.Fatalf("empty must be 1, got %d", got)
	}
	if got := EstimateTokens("abcdefgh", "gpt-4"); got != 2 {
		t.Fatalf("8 chars @4.0 must be 2, got %d", got)
	}
	if got := EstimateTokens("abcdefgh", "claude-3"); got != 2 {
		t.Fatalf("8 chars @3.5 must be 2, got %d", got)
	}
	if EstimateTokens("x", "unknown-model-zzz") != 1 {
		t.Fatal("unknown model must use default ratio")
	}
	if CappedEstimate("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", "gpt", 3) != 3 {
		t.Fatal("cap must bind")
	}
	if EstimatorVersion == "" {
		t.Fatal("estimator version pinned")
	}
}

func TestOpenAIDiscovery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			_, _ = w.Write([]byte(`{"data":[{"id":"m1"},{"id":"m2"}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	o := NewOpenAICompatWithPolicy(srv.URL, "", "fallback", LocalDevelopmentDestinationPolicy())
	models, err := o.Models(context.Background())
	if err != nil || len(models) != 2 || models[0] != "m1" {
		t.Fatalf("discovery failed: %v %v", models, err)
	}
	down := NewOpenAICompatWithPolicy("http://127.0.0.1:9", "", "fallback", LocalDevelopmentDestinationPolicy())
	models, err = down.Models(context.Background())
	if err != nil || len(models) != 1 || models[0] != "fallback" {
		t.Fatalf("unreachable must fall back: %v %v", models, err)
	}
}

func TestOpenAICompatSendsHeaders(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("x-session-id")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	o := NewOpenAICompatWithPolicy(srv.URL, "key", "m1", LocalDevelopmentDestinationPolicy()).WithHeaders(map[string]string{"x-session-id": "ses-1"})
	if _, err := o.Models(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got != "ses-1" {
		t.Fatalf("header not sent: %q", got)
	}
}

func TestModelHeaders(t *testing.T) {
	t.Setenv("PRUMO_MODEL_HEADERS", "")
	if h := ModelHeaders(); h != nil {
		t.Fatalf("empty env must yield nil, got %v", h)
	}
	t.Setenv("PRUMO_MODEL_HEADERS", `{"x-session-id":"ses-2","x-trace":"t1"}`)
	h := ModelHeaders()
	if len(h) != 2 || h["x-session-id"] != "ses-2" || h["x-trace"] != "t1" {
		t.Fatalf("bad headers: %v", h)
	}
	t.Setenv("PRUMO_MODEL_HEADERS", "{not-json")
	if h := ModelHeaders(); h != nil {
		t.Fatalf("malformed env must yield nil, got %v", h)
	}
}
