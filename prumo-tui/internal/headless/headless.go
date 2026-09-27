// Package headless runs a goal and prints what happened, without a terminal
// interface.
//
// It exists for the two callers that cannot read a screen: a person using a
// screen reader, for whom a repainting interface is noise, and a script, for
// whom it is nothing at all. Both want the same thing — the run's account, in
// order, as text — and they differ only in how it is encoded, which is the
// difference between the two modes here.
package headless

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/raillen/prumo-tui/internal/agent"
	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/message"
	"github.com/raillen/prumo-tui/internal/permission"
	"github.com/raillen/prumo-tui/internal/pubsub"
	"github.com/raillen/prumo-tui/internal/session"
	"github.com/raillen/prumo-tui/internal/stream"
)

// Mode is how the run's account is written.
type Mode string

const (
	// ModePlain writes what happened as prose, in order, one line per fact.
	ModePlain Mode = "plain"
	// ModeJSON writes one object per fact, for a program.
	ModeJSON Mode = "json"
)

// Options configures a headless run.
type Options struct {
	Goal string
	Mode Mode
	Out  io.Writer
}

// Run starts a goal, prints its account and returns when the run ends.
//
// It takes the app the caller already wired rather than building a second one:
// the client, the workspace and the provider are decisions of the invocation,
// and a headless run that made its own would be a second answer to them.
func Run(ctx context.Context, application *app.App, opts Options) error {
	sink := NewPrinter(opts.Out, opts.Mode)
	streamCtx, stop := context.WithCancel(ctx)
	defer stop()
	stream.Start(streamCtx, application, sink)

	sessionID, err := seed(ctx, application, opts.Goal)
	if err != nil {
		return err
	}

	out, err := application.CoderAgent.Run(ctx, sessionID, opts.Goal)
	if err != nil {
		return err
	}
	// The run's own reader is fed as well as the bridge, and this one is
	// synchronous: the bridge is a goroutine, so a process that exited the
	// moment the run ended could leave the answer it just produced unwritten.
	// Both paths carry the same message, and the printer prints each one once.
	for event := range out {
		sink.Send(event)
	}
	// What the run spent is read once more from the client's own record: the
	// broker's copy may still be in flight when the process exits, and a number
	// a script reads must not depend on winning a race. The printer drops a
	// repeat, so this is a safety net rather than a second report.
	if spent, err := application.Sessions.Get(ctx, sessionID); err == nil {
		sink.Spend(spent)
	}
	sink.Close()
	return nil
}

// seed creates the session the run belongs to, so the goal's answer has a place
// to be folded into.
func seed(ctx context.Context, application *app.App, goal string) (string, error) {
	created, err := application.Sessions.Create(ctx, firstLine(goal))
	if err != nil {
		return "", err
	}
	return created.ID, nil
}

func firstLine(goal string) string {
	if idx := strings.IndexAny(goal, "\r\n"); idx >= 0 {
		goal = goal[:idx]
	}
	if len(goal) > 60 {
		return goal[:60]
	}
	return goal
}

// Printer writes what the client publishes, one fact at a time.
//
// It is a `stream.Sender`: the same bridge that feeds a terminal interface feeds
// this, which is what keeps the two from describing a run differently.
type Printer struct {
	out  io.Writer
	mode Mode

	mu        sync.Mutex
	printed   map[string]int
	lastSpend string
	closed    bool
}

// NewPrinter returns a printer for a mode.
func NewPrinter(out io.Writer, mode Mode) *Printer {
	if mode != ModeJSON {
		mode = ModePlain
	}
	return &Printer{out: out, mode: mode, printed: map[string]int{}}
}

// Send takes one message from the client and writes what it means.
func (p *Printer) Send(msg tea.Msg) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch event := msg.(type) {
	case pubsub.Event[message.Message]:
		p.answer(event.Payload)
	case agent.AgentEvent:
		p.agentEvent(event)
	case pubsub.Event[agent.AgentEvent]:
		p.agentEvent(event.Payload)
	case pubsub.Event[session.Session]:
		if event.Type == pubsub.UpdatedEvent {
			p.spend(event.Payload)
		}
	case pubsub.Event[permission.PermissionRequest]:
		// There is nobody here to answer a gate, so it is refused rather than
		// left open: a run that waits forever is worse than a run that says why
		// it stopped.
		p.fact("permission.refused", map[string]any{"tool": event.Payload.ToolName, "reason": "no one to answer in a headless run"})
	}
}

// answer prints a run's prose as it grows.
//
// A message is republished whole as deltas arrive, so what is new is the tail
// that was not printed before: printing the message would repeat the answer once
// per delta.
func (p *Printer) answer(msg message.Message) {
	if msg.Role != message.Assistant {
		return
	}
	text := textOf(msg)
	seen := p.printed[msg.ID]
	if seen > len(text) {
		seen = 0
	}
	delta := text[seen:]
	if delta == "" {
		return
	}
	p.printed[msg.ID] = len(text)

	if p.mode == ModeJSON {
		p.write(map[string]any{"kind": "answer.delta", "id": msg.ID, "text": delta})
		return
	}
	fmt.Fprint(p.out, delta)
	if msg.IsFinished() {
		fmt.Fprintln(p.out)
	}
}

func (p *Printer) agentEvent(payload agent.AgentEvent) {
	switch payload.Type {
	case agent.AgentEventTypeResponse:
		p.answer(payload.Message)
	case agent.AgentEventTypeError:
		p.fact("run.failed", map[string]any{"error": errorText(payload.Error)})
	case agent.AgentEventTypeConnection:
		p.fact("run.connection", map[string]any{
			"state": string(payload.Connection.State), "attempt": payload.Connection.Attempt,
			"of": payload.Connection.Of, "reason": payload.Connection.Reason,
		})
	}
}

// Spend reports the session's cost from outside the message stream.
//
// Send() already holds the printer's lock, so the two entry points cannot share
// one body: the direct call from a run's tail would otherwise read and write
// the same state as an event arriving on the bridge at that moment — which the
// race detector found, and which would also let a machine-readable stream
// interleave two half-written lines.
func (p *Printer) Spend(payload session.Session) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.spend(payload)
}

// spend prints what the session has cost, once per distinct figure.
//
// Callers hold the lock: Send() for one, Spend() for the other.
func (p *Printer) spend(payload session.Session) {
	if payload.PromptTokens == 0 && payload.CompletionTokens == 0 {
		return
	}
	// The same figure can arrive twice — from the stream and from the record
	// read at the end — and a reader wants it once.
	key := sessionKey(payload)
	if key == p.lastSpend {
		return
	}
	p.lastSpend = key
	fields := map[string]any{
		"prompt_tokens": payload.PromptTokens, "completion_tokens": payload.CompletionTokens,
		"cost_usd": payload.Cost,
	}
	// A cached token and an input token are priced differently, so a reader
	// analysing a run needs them apart — the same reason the statusline keeps
	// them apart.
	if payload.CacheReadTokens > 0 || payload.CacheWriteTokens > 0 {
		fields["cache_read_tokens"] = payload.CacheReadTokens
		fields["cache_write_tokens"] = payload.CacheWriteTokens
	}
	p.fact("run.spend", fields)
}

// fact writes one thing that happened.
func (p *Printer) fact(kind string, fields map[string]any) {
	if p.mode == ModeJSON {
		body := map[string]any{"kind": kind}
		for key, value := range fields {
			body[key] = value
		}
		p.write(body)
		return
	}

	// The plain rendering is a sentence rather than a dump: what happened, and
	// the values that make it readable without a parser.
	switch kind {
	case "run.spend":
		line := fmt.Sprintf("\n%d tokens, $%.4f", fields["prompt_tokens"], fields["cost_usd"])
		if cached, ok := fields["cache_read_tokens"]; ok {
			line += fmt.Sprintf(" (%v from cache)", cached)
		}
		fmt.Fprintln(p.out, line)
	case "run.failed":
		fmt.Fprintf(p.out, "\nrun failed: %s\n", fields["error"])
	case "run.connection":
		fmt.Fprintf(p.out, "\nconnection %s (%v/%v): %s\n", fields["state"], fields["attempt"], fields["of"], fields["reason"])
	case "permission.refused":
		fmt.Fprintf(p.out, "\npermission refused for %s: %v\n", fields["tool"], fields["reason"])
	default:
		fmt.Fprintf(p.out, "\n%s %v\n", kind, fields)
	}
}

func (p *Printer) write(body map[string]any) {
	data, err := json.Marshal(body)
	if err != nil {
		return
	}
	fmt.Fprintln(p.out, string(data))
}

// Close flushes what a reader needs even if the run produced nothing.
func (p *Printer) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	if p.mode == ModePlain {
		fmt.Fprintln(p.out)
	}
}

// sessionKey is what makes two reports of the same spend the same report.
func sessionKey(payload session.Session) string {
	return fmt.Sprintf("%d/%d/%d/%d/%.6f",
		payload.PromptTokens, payload.CompletionTokens,
		payload.CacheReadTokens, payload.CacheWriteTokens, payload.Cost)
}

func textOf(msg message.Message) string {
	var out strings.Builder
	for _, part := range msg.Parts {
		if text, ok := part.(message.TextContent); ok {
			out.WriteString(text.Text)
		}
	}
	return out.String()
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

var _ stream.Sender = (*Printer)(nil)
