package headless

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/raillen/prumo-tui/internal/agent"
	"github.com/raillen/prumo-tui/internal/message"
	"github.com/raillen/prumo-tui/internal/permission"
	"github.com/raillen/prumo-tui/internal/pubsub"
	"github.com/raillen/prumo-tui/internal/session"
)

// answer is one assistant message as the runner republishes it: whole, growing.
func answer(id string, finished bool, text string) message.Message {
	msg := message.Message{ID: id, SessionID: "S1", Role: message.Assistant,
		Parts: []message.ContentPart{message.TextContent{Text: text}}}
	if finished {
		msg.Parts = append(msg.Parts, message.Finish{Reason: message.FinishReasonEndTurn})
	}
	return msg
}

// A run republishes the whole answer as each delta arrives, so what is new is
// the tail: printing the message would repeat the answer once per delta, which
// is the defect a reader notices first.
func TestTheAnswerIsPrintedOnceAsItGrows(t *testing.T) {
	out := &bytes.Buffer{}
	p := NewPrinter(out, ModePlain)

	p.Send(pubsub.Event[message.Message]{Type: pubsub.CreatedEvent, Payload: answer("m1", false, "The run ")})
	p.Send(pubsub.Event[message.Message]{Type: pubsub.UpdatedEvent, Payload: answer("m1", false, "The run wrote ")})
	p.Send(pubsub.Event[message.Message]{Type: pubsub.UpdatedEvent, Payload: answer("m1", true, "The run wrote two files.")})
	p.Close()

	if got := out.String(); !strings.Contains(got, "The run wrote two files.") {
		t.Fatalf("the answer is missing or was cut: %q", got)
	}
	if strings.Count(out.String(), "The run") != 1 {
		t.Fatalf("the answer was repeated as it grew: %q", out.String())
	}
}

// The JSON mode is the same account, one object per fact, so a program can read
// what a person reads.
func TestJSONModeWritesOneObjectPerFact(t *testing.T) {
	out := &bytes.Buffer{}
	p := NewPrinter(out, ModeJSON)

	p.Send(pubsub.Event[message.Message]{Type: pubsub.CreatedEvent, Payload: answer("m1", false, "hello")})
	p.Send(pubsub.Event[session.Session]{Type: pubsub.UpdatedEvent, Payload: session.Session{ID: "S1", PromptTokens: 120, CompletionTokens: 30, Cost: 0.002}})
	p.Send(pubsub.Event[agent.AgentEvent]{Type: pubsub.UpdatedEvent, Payload: agent.AgentEvent{
		Type:       agent.AgentEventTypeConnection,
		Connection: agent.Connection{State: agent.ConnectionReconnecting, Attempt: 2, Of: 8, Reason: "no answer"},
	}})
	p.Close()

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("wrote %d line(s), want one per fact:\n%s", len(lines), out.String())
	}
	kinds := make([]string, 0, len(lines))
	for _, line := range lines {
		var body map[string]any
		if err := json.Unmarshal([]byte(line), &body); err != nil {
			t.Fatalf("a line is not JSON: %q (%v)", line, err)
		}
		kinds = append(kinds, body["kind"].(string))
	}
	want := []string{"answer.delta", "run.spend", "run.connection"}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("facts = %v, want %v", kinds, want)
		}
	}
}

// A gate cannot be answered here, and a run that waits forever is worse than a
// run that says why it stopped.
func TestAGateIsRefusedOutLoud(t *testing.T) {
	out := &bytes.Buffer{}
	p := NewPrinter(out, ModePlain)

	p.Send(pubsub.Event[permission.PermissionRequest]{
		Type:    pubsub.CreatedEvent,
		Payload: permission.PermissionRequest{ID: "r1", ToolName: "process.exec"},
	})

	got := out.String()
	if !strings.Contains(got, "process.exec") || !strings.Contains(got, "refused") {
		t.Fatalf("the refusal does not say what was refused: %q", got)
	}
}

// Nothing in the output is styled: it is read where there is no terminal to
// interpret escapes.
func TestTheOutputCarriesNoTerminalStyling(t *testing.T) {
	out := &bytes.Buffer{}
	p := NewPrinter(out, ModePlain)
	p.Send(pubsub.Event[message.Message]{Type: pubsub.CreatedEvent, Payload: answer("m1", true, "done")})
	p.fact("run.failed", map[string]any{"error": "the daemon went away"})

	if strings.Contains(out.String(), "\x1b[") {
		t.Fatalf("the output carries escapes: %q", out.String())
	}
}
