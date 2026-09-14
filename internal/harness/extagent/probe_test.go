package extagent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeStub(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestProbeBinariesWithFakePATH(t *testing.T) {
	dir := t.TempDir()
	writeStub(t, dir, "opencode", "echo 1.18.30")
	// codex intentionally absent.
	p := Prober{Path: dir, Dial: func(_ context.Context, _ string) error { return errClosed{} }, OpenCodeURL: "http://127.0.0.1:9"}
	rows := map[string]ProbeResult{}
	for _, r := range p.Matrix(context.Background()) {
		rows[r.Name] = r
	}
	if !rows["opencode-cli"].Available || rows["opencode-cli"].Version != "1.18.30" {
		t.Fatalf("opencode-cli misprobed: %+v", rows["opencode-cli"])
	}
	if rows["codex-cli"].Available {
		t.Fatalf("absent binary must be unavailable: %+v", rows["codex-cli"])
	}
	if rows["fake"].Available != true {
		t.Fatal("fake must always be available")
	}
}

type errClosed struct{}

func (errClosed) Error() string { return "closed" }

func TestProbeServerReachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	// Empty PATH keeps the probe hermetic: no real binaries execute.
	p := Prober{Path: t.TempDir(), OpenCodeURL: srv.URL}
	rows := map[string]ProbeResult{}
	for _, r := range p.Matrix(context.Background()) {
		rows[r.Name] = r
	}
	if !rows["opencode-server"].Available {
		t.Fatalf("httptest server must probe reachable: %+v", rows["opencode-server"])
	}
	closed := Prober{Path: t.TempDir(), OpenCodeURL: "http://127.0.0.1:9"}
	for _, r := range closed.Matrix(context.Background()) {
		if r.Name == "opencode-server" && r.Available {
			t.Fatal("closed port must probe unavailable")
		}
	}
}

func TestProbeModelConfig(t *testing.T) {
	t.Setenv("PRUMO_MODEL_BASE_URL", "")
	t.Setenv("PRUMO_MODEL_API_KEY", "")
	p := Prober{Path: t.TempDir()}
	for _, r := range p.Matrix(context.Background()) {
		if r.Name == "openai-compat" && r.Available {
			t.Fatal("openai-compat without endpoint must be unconfigured")
		}
		if r.Name == "anthropic" && r.Available {
			t.Fatal("anthropic without key must be unconfigured")
		}
	}
	t.Setenv("PRUMO_MODEL_BASE_URL", "http://stub:8080")
	t.Setenv("PRUMO_MODEL_API_KEY", "k")
	for _, r := range p.Matrix(context.Background()) {
		if (r.Name == "openai-compat" || r.Name == "anthropic") && !r.Available {
			t.Fatalf("%s must be configured: %+v", r.Name, r)
		}
	}
	if !strings.Contains(p.Matrix(context.Background())[0].Name, "fake") {
		t.Fatal("matrix order must start with fake")
	}
}
