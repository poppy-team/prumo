package daemon

import (
	"strings"
	"testing"

	harnessprotocol "github.com/raillen/prumo/internal/harness/protocol"
)

// Negotiation existed and nothing called it. The version a client sent was never
// read, so a client built against a different protocol was served until an op
// behaved differently than it expected. The reconnect contract says a version
// mismatch fails closed and does not guess, and that a reconnect begins with a
// handshake (GAP-165).

func TestARequestWithoutAVersionIsRefused(t *testing.T) {
	srv := New("s.sock", t.TempDir(), fakeDeps(false))
	res := srv.dispatch(map[string]any{"op": "list"})
	if res["ok"] != false {
		t.Fatalf("a request with no protocol version must be refused, got %v", res)
	}
	err, _ := res["error"].(string)
	if !strings.Contains(err, "protocol_version") {
		t.Fatalf("the refusal must say what is missing: %q", err)
	}
	// And it must be actionable: the client is told what to send.
	if res["version"] != harnessprotocol.Version || res["min_compatible"] == "" {
		t.Fatalf("a refusal must name the supported versions: %v", res)
	}
}

func TestAMismatchedVersionIsRefused(t *testing.T) {
	srv := New("s.sock", t.TempDir(), fakeDeps(false))
	for _, version := range []string{"99.0.0", "0.0.1", "1.0.0"} {
		res := srv.dispatch(map[string]any{"protocol_version": version, "op": "list"})
		if res["ok"] != false {
			t.Fatalf("client %s must be refused against engine %s, got %v", version, harnessprotocol.Version, res)
		}
	}
}

func TestAMalformedVersionIsRefusedRatherThanGuessed(t *testing.T) {
	srv := New("s.sock", t.TempDir(), fakeDeps(false))
	for _, version := range []string{"latest", "0", "0.x.0", "v0.4.0"} {
		res := srv.dispatch(map[string]any{"protocol_version": version, "op": "list"})
		if res["ok"] != false {
			t.Fatalf("a malformed version %q must be refused, got %v", version, res)
		}
	}
}

func TestACompatibleVersionIsServed(t *testing.T) {
	srv := New("s.sock", t.TempDir(), fakeDeps(false))
	for _, version := range []string{harnessprotocol.Version, harnessprotocol.MinCompatible} {
		res := srv.dispatch(map[string]any{"protocol_version": version, "op": "list"})
		if res["ok"] != true {
			t.Fatalf("version %s is within the supported range and must be served: %v", version, res)
		}
	}
}

func TestTheProtocolOpAnswersWithoutAVersionDeclared(t *testing.T) {
	// A client asks what is supported in order to send a version. Requiring one
	// first would make the question unanswerable.
	srv := New("s.sock", t.TempDir(), fakeDeps(false))
	res := srv.dispatch(map[string]any{"op": "protocol"})
	if res["ok"] != true {
		t.Fatalf("the protocol op must answer an unversioned client: %v", res)
	}
	if res["version"] != harnessprotocol.Version {
		t.Fatalf("the protocol op must report the engine version: %v", res)
	}
}

func TestEveryDeclaredOpIsReachableByAVersionedClient(t *testing.T) {
	// The refusal is on the front door, so it must not also close any op behind
	// it. A client that negotiates correctly has to reach everything.
	srv := New("s.sock", t.TempDir(), fakeDeps(false))
	for _, op := range harnessprotocol.Ops {
		res := srv.dispatch(map[string]any{"protocol_version": harnessprotocol.Version, "op": op})
		if res["ok"] == false {
			if msg, _ := res["error"].(string); strings.Contains(msg, "protocol") {
				t.Fatalf("op %q is unreachable to a correctly versioned client: %v", op, res)
			}
		}
	}
}
