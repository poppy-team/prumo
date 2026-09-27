package perm

import (
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
)

// A request id is derived from the tool call id, which the model chooses. So a
// later call can arrive carrying the id of an earlier, already-approved one.
// Before the fingerprint existed, the recorded decision answered by id alone,
// and a human's approval for `rm important.txt` silently authorised
// `rm everything.txt` (GAP-107).

func TestRecordedApprovalDoesNotCarryToDifferentContent(t *testing.T) {
	e := New(Policy{DefaultAction: agent.PermissionDeny})
	first := agent.PermissionRequest{ID: "perm-c1", Action: "edit.delete", Resource: "docs/keep.md"}

	if got := e.Evaluate(first, "destructive", "policy"); got.Decision != agent.PermissionDeny {
		t.Fatalf("policy must deny first, got %s", got.Decision)
	}
	fingerprint := Fingerprint(first)
	if _, err := e.Approve(first.ID, fingerprint, "operator"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if got := e.Evaluate(first, "destructive", "policy"); got.Decision != agent.PermissionAllow {
		t.Fatalf("the same content must still be allowed, got %s", got.Decision)
	}

	// Same request id, different content: a different call, a different decision.
	second := agent.PermissionRequest{ID: "perm-c1", Action: "edit.delete", Resource: ".env"}
	got := e.Evaluate(second, "destructive", "policy")
	if got.Decision != agent.PermissionDeny {
		t.Fatalf("an approval for %q must not authorise %q; got %s",
			first.Resource, second.Resource, got.Decision)
	}
}

func TestRecordedApprovalDoesNotCarryToADifferentAction(t *testing.T) {
	e := New(Policy{DefaultAction: agent.PermissionDeny})
	read := agent.PermissionRequest{ID: "perm-c1", Action: "fs.read", Resource: "README.md"}
	fingerprint := Fingerprint(read)
	if _, err := e.Approve(read.ID, fingerprint, "operator"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	// The same id and the same path, but a destructive action.
	del := agent.PermissionRequest{ID: "perm-c1", Action: "edit.delete", Resource: "README.md"}
	if got := e.Evaluate(del, "destructive", "policy"); got.Decision != agent.PermissionDeny {
		t.Fatalf("a read approval must not authorise a delete; got %s", got.Decision)
	}
}

func TestRecordedDenialDoesNotCarryToDifferentContent(t *testing.T) {
	e := New(Policy{DefaultAction: agent.PermissionAllow})
	denied := agent.PermissionRequest{ID: "perm-c1", Action: "edit.delete", Resource: "secrets/key.pem"}
	if _, err := e.Deny(denied.ID, Fingerprint(denied), "operator", "not that one"); err != nil {
		t.Fatalf("deny: %v", err)
	}
	other := agent.PermissionRequest{ID: "perm-c1", Action: "edit.delete", Resource: "docs/readme.md"}
	got := e.Evaluate(other, "destructive", "policy")
	// The point is that the decision came from the policy, not from the recorded
	// denial of another path. A destructive action defaults to ask, so ask is
	// what an independent evaluation looks like here.
	if got.Decision != agent.PermissionAsk {
		t.Fatalf("a denial of one path must not decide another; got %s", got.Decision)
	}
}

func TestResolutionRequiresTheMatchingFingerprint(t *testing.T) {
	e := New(Policy{})
	req := agent.PermissionRequest{ID: "perm-c1", Action: "edit.delete", Resource: "a.txt"}
	fingerprint := Fingerprint(req)
	if _, err := e.Approve(req.ID, fingerprint, "operator"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if _, ok := e.Resolution(req.ID, "some-other-fingerprint"); ok {
		t.Fatal("a resolution must not be returned for content it was not made about")
	}
	if _, ok := e.Resolution(req.ID, fingerprint); !ok {
		t.Fatal("the matching resolution must be returned")
	}
	// An empty fingerprint is the permissive form: only for callers that have no
	// request in hand, such as an audit view.
	if _, ok := e.Resolution(req.ID, ""); !ok {
		t.Fatal("an audit lookup by id alone must still work")
	}
}

func TestFingerprintCoversEveryFieldThatChangesTheCall(t *testing.T) {
	base := agent.PermissionRequest{
		Action: "edit.patch", Resource: "a.txt", ArgumentsSummary: "x=1",
		FilesystemScope: []string{"src"}, NetworkDests: []string{"api.example.com"},
		CredentialScopes: []string{"github"}, DataClass: "internal",
		Reversibility: "reversible",
	}
	baseFP := Fingerprint(base)

	variants := map[string]agent.PermissionRequest{
		"action":        {Action: "edit.delete", Resource: "a.txt", ArgumentsSummary: "x=1", FilesystemScope: []string{"src"}, NetworkDests: []string{"api.example.com"}, CredentialScopes: []string{"github"}, DataClass: "internal", Reversibility: "reversible"},
		"resource":      {Action: "edit.patch", Resource: "b.txt", ArgumentsSummary: "x=1", FilesystemScope: []string{"src"}, NetworkDests: []string{"api.example.com"}, CredentialScopes: []string{"github"}, DataClass: "internal", Reversibility: "reversible"},
		"arguments":     {Action: "edit.patch", Resource: "a.txt", ArgumentsSummary: "x=2", FilesystemScope: []string{"src"}, NetworkDests: []string{"api.example.com"}, CredentialScopes: []string{"github"}, DataClass: "internal", Reversibility: "reversible"},
		"fs_scope":      {Action: "edit.patch", Resource: "a.txt", ArgumentsSummary: "x=1", FilesystemScope: []string{"etc"}, NetworkDests: []string{"api.example.com"}, CredentialScopes: []string{"github"}, DataClass: "internal", Reversibility: "reversible"},
		"network":       {Action: "edit.patch", Resource: "a.txt", ArgumentsSummary: "x=1", FilesystemScope: []string{"src"}, NetworkDests: []string{"evil.example.com"}, CredentialScopes: []string{"github"}, DataClass: "internal", Reversibility: "reversible"},
		"credentials":   {Action: "edit.patch", Resource: "a.txt", ArgumentsSummary: "x=1", FilesystemScope: []string{"src"}, NetworkDests: []string{"api.example.com"}, CredentialScopes: []string{"aws"}, DataClass: "internal", Reversibility: "reversible"},
		"data_class":    {Action: "edit.patch", Resource: "a.txt", ArgumentsSummary: "x=1", FilesystemScope: []string{"src"}, NetworkDests: []string{"api.example.com"}, CredentialScopes: []string{"github"}, DataClass: "restricted", Reversibility: "reversible"},
		"reversibility": {Action: "edit.patch", Resource: "a.txt", ArgumentsSummary: "x=1", FilesystemScope: []string{"src"}, NetworkDests: []string{"api.example.com"}, CredentialScopes: []string{"github"}, DataClass: "internal", Reversibility: "destructive"},
	}
	for name, variant := range variants {
		if Fingerprint(variant) == baseFP {
			t.Errorf("changing %s must change the fingerprint", name)
		}
	}
}

func TestFingerprintIsStableAcrossCalls(t *testing.T) {
	req := agent.PermissionRequest{ID: "perm-c1", Action: "edit.patch", Resource: "a.txt"}
	if Fingerprint(req) != Fingerprint(req) {
		t.Fatal("the same request must always fingerprint the same")
	}
	// The request id is deliberately not part of it: the id names the request,
	// the fingerprint names the content.
	other := req
	other.ID = "perm-c99"
	if Fingerprint(req) != Fingerprint(other) {
		t.Fatal("the fingerprint must describe content, not identity")
	}
}
