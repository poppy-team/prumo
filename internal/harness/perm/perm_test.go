package perm

import (
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

func TestDenyPrefix(t *testing.T) {
	e := New(Policy{DefaultAction: agent.PermissionAllow, DenyPrefixes: []string{"/etc"}})
	res := e.Evaluate(agent.PermissionRequest{ID: "p1", Action: "fs.read", Resource: "/etc/passwd"}, "read-only", "test")
	if res.Decision != agent.PermissionDeny {
		t.Fatalf("expected deny, got %s", res.Decision)
	}
}

func TestAskKinds(t *testing.T) {
	e := New(Policy{DefaultAction: agent.PermissionAllow, AllowActions: []string{"edit.patch"}, AskKinds: []string{"side-effecting"}})
	res := e.Evaluate(agent.PermissionRequest{ID: "p1", Action: "edit.patch"}, "side-effecting", "test")
	if res.Decision != agent.PermissionAsk {
		t.Fatalf("expected ask, got %s", res.Decision)
	}
}

func TestApproveDenyCycle(t *testing.T) {
	e := New(Policy{})
	if got := e.Approve("p1", "user"); got.Decision != agent.PermissionAllow {
		t.Fatal("approve must allow")
	}
	if got := e.Deny("p2", "user", ""); got.Decision != agent.PermissionDeny {
		t.Fatal("deny must deny")
	}
}
