package aci

import (
	"context"
	"strings"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

func TestExtractEndpoints(t *testing.T) {
	got := ExtractEndpoints("curl https://api.example.com/x && wget http://cdn.example.com/y https://api.example.com/z")
	if len(got) != 2 || got[0].host != "api.example.com" || got[0].scheme != "https" {
		t.Fatalf("bad extraction: %+v", got)
	}
	if len(ExtractEndpoints("echo no-urls")) != 0 {
		t.Fatal("plain command must yield no endpoints")
	}
}

func TestEgressDeny(t *testing.T) {
	p := &EgressPolicy{DefaultDeny: true, AllowHosts: []string{"proxy.golang.org", ".internal.example"}}
	if err := CheckEgress(p, "curl https://evil.example/x"); err == nil {
		t.Fatal("unlisted host must be denied")
	}
	if err := CheckEgress(p, "GOPROXY=https://proxy.golang.org go mod download"); err != nil {
		t.Fatalf("listed host must pass: %v", err)
	}
	if err := CheckEgress(p, "curl https://svc.internal.example/x"); err != nil {
		t.Fatalf("suffix match must pass: %v", err)
	}
	if err := CheckEgress(nil, "curl https://anything.example/"); err != nil {
		t.Fatal("nil policy preserves legacy posture")
	}
	// Restricted data never leaves, even permissive.
	r := &EgressPolicy{DataClass: "restricted"}
	if err := CheckEgress(r, "curl https://api.example.com/x"); err == nil {
		t.Fatal("restricted data must not egress")
	}
}

func TestEgressEnforcedOnExec(t *testing.T) {
	e := New(t.TempDir())
	e.Egress = &EgressPolicy{DefaultDeny: true}
	res, _ := e.Execute(context.Background(), agent.ToolCall{ID: "c", Name: "process.exec",
		Arguments: map[string]any{"command": "curl https://evil.example/x"}})
	if res.ExitCode == 0 || !strings.Contains(res.Error, "egress denied") {
		t.Fatalf("exec must enforce egress: %+v", res)
	}
}
