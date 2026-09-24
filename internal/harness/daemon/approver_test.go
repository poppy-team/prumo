package daemon

import (
	"strings"
	"testing"
)

// Every approval was recorded with the actor "client": a transport, not a
// person. One operator answering one question and five hundred clients being
// offered the question produced identical records, and nothing in the
// permissions trail said who had allowed a shell command (GAP-160).

func TestAnApprovalSaysWhoMadeIt(t *testing.T) {
	if got := approverIdentity("alice"); got != "alice" {
		t.Fatalf("the actor was not recorded: %q", got)
	}
}

func TestAnUnattributedApprovalIsNotAttributedToTheTransport(t *testing.T) {
	// "client" was a fabrication: the daemon never asked who was on the other
	// end, it simply wrote the name of the protocol. "unknown" is a fact about
	// the record; "client" is a fact about nobody.
	if got := approverIdentity(""); got != "unknown" {
		t.Fatalf("an approval with no actor was recorded as %q", got)
	}
	if got := approverIdentity("   "); got != "unknown" {
		t.Fatalf("whitespace was recorded as an identity: %q", got)
	}
}

func TestAnAbsurdlyLongActorIsNotAnIdentity(t *testing.T) {
	// The identity is written into a JSONL trail a human reads. An unbounded
	// string is not an identity, it is a payload.
	long := strings.Repeat("a", 500)
	if got := approverIdentity(long); got != "unknown" {
		t.Fatalf("a 500-byte actor was recorded: %q", got)
	}
}
