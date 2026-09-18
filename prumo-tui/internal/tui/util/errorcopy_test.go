package util

import (
	"errors"
	"strings"
	"testing"
)

// The shape of a failure is the policy: what failed, the error itself, what to
// do next. A message that only says what broke leaves the reader guessing.
func TestAFailureNamesTheOperationAndTheNextAction(t *testing.T) {
	cmd := ReportFailure("Starting the run", "press enter to try again", errors.New("no harness attached"))
	msg, ok := cmd().(InfoMsg)
	if !ok {
		t.Fatal("a failure did not reach the statusline as a message")
	}
	if msg.Type != InfoTypeError {
		t.Fatalf("a failure was reported as %v", msg.Type)
	}

	for _, want := range []string{"Starting the run", "no harness attached", "press enter"} {
		if !strings.Contains(msg.Msg, want) {
			t.Errorf("the failure lost %q: %q", want, msg.Msg)
		}
	}
	// The three parts stay in one order, so a reader learns where to look.
	if strings.Index(msg.Msg, "Starting the run") > strings.Index(msg.Msg, "no harness attached") {
		t.Errorf("the operation is not named first: %q", msg.Msg)
	}
	if strings.Index(msg.Msg, "no harness attached") > strings.Index(msg.Msg, "press enter") {
		t.Errorf("the next action does not come last: %q", msg.Msg)
	}
}

// The error text is carried unchanged: the client surrounds the record, it does
// not rewrite it.
func TestTheErrorTextIsCarriedUnchanged(t *testing.T) {
	original := errors.New("dial unix /tmp/agentd.sock: connect: no such file or directory")
	cmd := ReportFailure("Re-attaching to the run", "pick another session", original)
	msg := cmd().(InfoMsg)

	if !strings.Contains(msg.Msg, original.Error()) {
		t.Fatalf("the error was paraphrased: %q", msg.Msg)
	}
}
