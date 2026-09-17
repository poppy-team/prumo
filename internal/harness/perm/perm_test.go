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

// TestEvaluateHonoursRecordedDecision is the property the approval surface
// stands on: a policy that would ask again must not, once a decision exists
// for that request. Recording without consulting would leave every approval
// recorded and ignored.
func TestEvaluateHonoursRecordedDecision(t *testing.T) {
	e := New(Policy{DefaultAction: agent.PermissionDeny})
	req := agent.PermissionRequest{ID: "perm-c1", Action: "edit.delete"}
	if got := e.Evaluate(req, "destructive", "policy"); got.Decision != agent.PermissionDeny {
		t.Fatalf("policy must deny first, got %s", got.Decision)
	}
	e.Approve("perm-c1", "operator")
	if got := e.Evaluate(req, "destructive", "policy"); got.Decision != agent.PermissionAllow {
		t.Fatalf("recorded approval must win over policy, got %s", got.Decision)
	}
	if res, ok := e.Resolution("perm-c1"); !ok || res.Actor != "operator" {
		t.Fatalf("resolution not recorded: %+v %v", res, ok)
	}
	// The audit log keeps one entry per decision, not one per re-evaluation.
	if len(e.Log) != 2 {
		t.Fatalf("log has %d entries, want 2 (policy + approval)", len(e.Log))
	}
}
