package runlayer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/perm"
)

// The write side of the permission trail has existed all along. The read side
// did not: a daemon that stopped for approval, restarted, and resumed asked the
// same human the same question again, because the decision only ever lived in
// the engine's map (GAP-106).

func TestPermissionTrailRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "permissions-R1.jsonl")
	req := agent.PermissionRequest{ID: "perm-c1", Action: "edit.delete", Resource: "docs/keep.md"}
	fingerprint := perm.Fingerprint(req)

	first := perm.New(perm.Policy{DefaultAction: agent.PermissionDeny})
	if got := first.Evaluate(req, "destructive", "policy"); got.Decision != agent.PermissionDeny {
		t.Fatalf("policy must deny first, got %s", got.Decision)
	}
	if _, err := first.Approve(req.ID, fingerprint, "operator"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if err := SavePermissions(path, first); err != nil {
		t.Fatalf("save: %v", err)
	}

	// A fresh engine, as after a daemon restart.
	second := perm.New(perm.Policy{DefaultAction: agent.PermissionDeny})
	if err := LoadPermissions(path, second); err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := second.Evaluate(req, "destructive", "policy"); got.Decision != agent.PermissionAllow {
		t.Fatalf("the approval must survive the process, got %s", got.Decision)
	}
}

func TestLoadPermissionsReusesADecisionOnlyForTheSameContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "permissions-R1.jsonl")
	approved := agent.PermissionRequest{ID: "perm-c1", Action: "edit.delete", Resource: "docs/keep.md"}

	engine := perm.New(perm.Policy{DefaultAction: agent.PermissionDeny})
	if _, err := engine.Approve(approved.ID, perm.Fingerprint(approved), "operator"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if err := SavePermissions(path, engine); err != nil {
		t.Fatalf("save: %v", err)
	}

	restored := perm.New(perm.Policy{DefaultAction: agent.PermissionDeny})
	if err := LoadPermissions(path, restored); err != nil {
		t.Fatalf("load: %v", err)
	}
	// Same id, different content: the restored approval must not answer it.
	other := agent.PermissionRequest{ID: "perm-c1", Action: "edit.delete", Resource: ".env"}
	if got := restored.Evaluate(other, "destructive", "policy"); got.Decision != agent.PermissionDeny {
		t.Fatalf("a reloaded approval must not transfer to other content, got %s", got.Decision)
	}
}

func TestLoadPermissionsIgnoresEntriesWithNoFingerprint(t *testing.T) {
	// A trail written before fingerprints existed identifies a request but not
	// its content. Reusing it would re-open the hole the fingerprint closed.
	path := filepath.Join(t.TempDir(), "permissions-R1.jsonl")
	legacy := `{"request_id":"perm-c1","decision":"allow","actor":"old-operator","decided_at":"2026-01-01T00:00:00Z"}` + "\n"
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	engine := perm.New(perm.Policy{DefaultAction: agent.PermissionDeny})
	if err := LoadPermissions(path, engine); err != nil {
		t.Fatalf("a legacy line must not fail the load: %v", err)
	}
	req := agent.PermissionRequest{ID: "perm-c1", Action: "edit.delete", Resource: "anything"}
	if got := engine.Evaluate(req, "destructive", "policy"); got.Decision != agent.PermissionDeny {
		t.Fatalf("a fingerprintless approval must not be reused, got %s", got.Decision)
	}
}

func TestLoadPermissionsToleratesCorruptLines(t *testing.T) {
	// One unreadable line must not make every other approval unusable.
	path := filepath.Join(t.TempDir(), "permissions-R1.jsonl")
	req := agent.PermissionRequest{ID: "perm-c1", Action: "edit.delete", Resource: "a.txt"}

	engine := perm.New(perm.Policy{})
	if _, err := engine.Approve(req.ID, perm.Fingerprint(req), "operator"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	good, err := marshalLines(engine.Log)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	content := "{not json at all\n" + good + "\n\n{\"no\":\"id\"}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	restored := perm.New(perm.Policy{DefaultAction: agent.PermissionDeny})
	if err := LoadPermissions(path, restored); err != nil {
		t.Fatalf("load must tolerate a corrupt line: %v", err)
	}
	if got := restored.Evaluate(req, "destructive", "policy"); got.Decision != agent.PermissionAllow {
		t.Fatalf("the readable line must still load, got %s", got.Decision)
	}
}

func TestLoadPermissionsOnAMissingFileIsNotAnError(t *testing.T) {
	// A run that never hit a gate has no trail. That is normal, not a failure.
	engine := perm.New(perm.Policy{})
	if err := LoadPermissions(filepath.Join(t.TempDir(), "absent.jsonl"), engine); err != nil {
		t.Fatalf("a missing trail must not fail: %v", err)
	}
}

func TestSavePermissionsStaysIdempotent(t *testing.T) {
	// A run that stops for approval stops twice; appending would duplicate the
	// whole trail.
	path := filepath.Join(t.TempDir(), "permissions-R1.jsonl")
	req := agent.PermissionRequest{ID: "perm-c1", Action: "edit.delete", Resource: "a.txt"}
	engine := perm.New(perm.Policy{DefaultAction: agent.PermissionAsk})
	_ = engine.Evaluate(req, "destructive", "policy")
	if _, err := engine.Approve(req.ID, perm.Fingerprint(req), "operator"); err != nil {
		t.Fatalf("approve: %v", err)
	}

	if err := SavePermissions(path, engine); err != nil {
		t.Fatalf("first save: %v", err)
	}
	first, _ := os.ReadFile(path)
	if err := SavePermissions(path, engine); err != nil {
		t.Fatalf("second save: %v", err)
	}
	second, _ := os.ReadFile(path)
	if string(first) != string(second) {
		t.Fatalf("a second save must not duplicate the trail:\n%s\n---\n%s", first, second)
	}
}

func marshalLines(log []agent.PermissionResolution) (string, error) {
	out := ""
	for _, res := range log {
		data, err := json.Marshal(res)
		if err != nil {
			return "", err
		}
		out += string(data) + "\n"
	}
	return out, nil
}
