// Package runtime is where the client meets the harness.
//
// Everything above this package believes it is talking to an agent that streams
// messages. Everything below it is the Prumo protocol: start a run, read its
// events, answer its permission gates. This is the only package in the client
// that knows the protocol exists, and the only one that would change if the
// harness grew a push channel.
package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	prumo "github.com/raillen/prumo/sdk/prumo"

	"github.com/raillen/prumo-tui/internal/agent"
	"github.com/raillen/prumo-tui/internal/llm/models"
	"github.com/raillen/prumo-tui/internal/message"
	"github.com/raillen/prumo-tui/internal/permission"
	"github.com/raillen/prumo-tui/internal/pubsub"
	"github.com/raillen/prumo-tui/internal/session"
)

// PollInterval is how often the client asks the daemon for new events.
//
// The protocol exposes replay plus status, not a subscription, so a client
// learns about progress by asking. It is a named constant because the cost of
// that design should be visible in one place rather than buried in a loop.
const PollInterval = 250 * time.Millisecond

// The transport's shapes are re-exported under this package's names so the view
// layer never imports the SDK directly. The transport can change without
// touching a component.
type (
	StartRequest = prumo.StartRequest
	RunStatus    = prumo.RunStatus
	Event        = prumo.Event
)

// Client is the slice of the SDK the runner needs. *prumo.Client satisfies it.
type Client interface {
	Start(ctx context.Context, req prumo.StartRequest) (string, error)
	Status(ctx context.Context, runID string) (prumo.RunStatus, error)
	Events(ctx context.Context, runID string) ([]prumo.Event, error)
	Cancel(ctx context.Context, runID string) error
	Approve(ctx context.Context, runID, requestID string) error
	Deny(ctx context.Context, runID, requestID, reason string) error
	List(ctx context.Context) ([]prumo.RunStatus, error)
}

// Runner implements the agent surface the view layer consumes, over the
// harness protocol.
type Runner struct {
	*agentEvents

	client      Client
	sessions    *session.Store
	messages    *message.Store
	permissions *permission.Service

	mu        sync.Mutex
	model     models.Model
	provider  string
	maxTurns  int
	workspace string
	active    map[string]context.CancelFunc
	// changes is what the harness reported per session, kept so the view can
	// ask without re-reading the timeline on every frame.
	changes map[string][]Change
}

// Options configures a runner.
type Options struct {
	Client      Client
	Sessions    *session.Store
	Messages    *message.Store
	Permissions *permission.Service
	Provider    string
	Model       models.Model
	MaxTurns    int
	Workspace   string
}

// NewRunner builds a runner over a transport.
func NewRunner(opts Options) *Runner {
	if opts.Provider == "" {
		opts.Provider = "fake"
	}
	if opts.MaxTurns <= 0 {
		opts.MaxTurns = 5
	}
	return &Runner{
		agentEvents: newAgentEvents(),
		client:      opts.Client,
		sessions:    opts.Sessions,
		messages:    opts.Messages,
		permissions: opts.Permissions,
		model:       opts.Model,
		provider:    opts.Provider,
		maxTurns:    opts.MaxTurns,
		workspace:   opts.Workspace,
		active:      map[string]context.CancelFunc{},
		changes:     map[string][]Change{},
	}
}

// SetPermissions attaches the permission service. The app builds both, and the
// runner needs it to raise the requests the view presents.
func (r *Runner) SetPermissions(p *permission.Service) { r.permissions = p }

// Change is one file a run changed.
type Change struct {
	Path      string
	Operation string
	Tool      string
}

// Changes returns the files this session's run has changed, in the order the
// harness reported them.
//
// It is derived state: the daemon's timeline is the record, and this is what the
// view folds it into. A client that restarted re-reads the events and rebuilds
// the same list.
func (r *Runner) Changes(sessionID string) []Change {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Change{}, r.changes[sessionID]...)
}

func (r *Runner) recordChange(sessionID string, change Change) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.changes == nil {
		r.changes = map[string][]Change{}
	}
	r.changes[sessionID] = append(r.changes[sessionID], change)
}

// Model reports the model the client asks for.
func (r *Runner) Model() models.Model {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.model
}

// Update records a different model for subsequent runs. Which models exist is
// the harness's business, so this only changes what the client asks for.
func (r *Runner) Update(modelID models.ModelID) (models.Model, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.model = models.Model{ID: modelID, Name: string(modelID)}
	return r.model, nil
}

// IsBusy reports whether any run is in flight.
func (r *Runner) IsBusy() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.active) > 0
}

// IsSessionBusy reports whether this run is in flight.
func (r *Runner) IsSessionBusy(sessionID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, busy := r.active[sessionID]
	return busy
}

// Cancel stops a run. The daemon owns the loop; the client only asks.
func (r *Runner) Cancel(sessionID string) {
	r.mu.Lock()
	cancel := r.active[sessionID]
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if r.client == nil {
		return
	}
	ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	_ = r.client.Cancel(ctx, sessionID)
}

// Summarize is refused rather than faked: compaction belongs to the harness, and
// a client that invented its own summary would be writing a second history.
func (r *Runner) Summarize(context.Context, string) error {
	return errors.New("compaction belongs to the harness: the client does not summarize a run")
}

// Run starts a run for a goal and returns the events it produces.
func (r *Runner) Run(ctx context.Context, sessionID, content string, _ ...message.Attachment) (<-chan agent.AgentEvent, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("a goal is required")
	}
	if r.client == nil {
		return nil, errors.New("no harness attached")
	}
	if _, err := r.sessions.Get(ctx, sessionID); err != nil {
		if _, err := r.sessions.Save(ctx, session.Session{ID: sessionID, Title: firstLine(content), CreatedAt: time.Now().Unix()}); err != nil {
			return nil, err
		}
	}
	if _, err := r.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:  message.User,
		Parts: []message.ContentPart{message.TextContent{Text: content}},
	}); err != nil {
		return nil, err
	}

	model := r.Model()
	if _, err := r.client.Start(ctx, StartRequest{
		Goal:      content,
		Provider:  r.provider,
		Model:     model.APIModel,
		MaxTurns:  r.maxTurns,
		RunID:     sessionID,
		Workspace: r.workspace,
	}); err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithCancel(ctx)
	r.mu.Lock()
	r.active[sessionID] = cancel
	r.mu.Unlock()

	out := make(chan agent.AgentEvent, 64)
	go r.observe(runCtx, sessionID, out)
	return out, nil
}

// turn accumulates the assistant message for one run.
type turn struct {
	id    string
	parts []message.ContentPart
}

func (t *turn) text() *message.TextContent {
	for i := len(t.parts) - 1; i >= 0; i-- {
		if c, ok := t.parts[i].(message.TextContent); ok {
			return &c
		}
	}
	return nil
}

// observe polls the run and folds its events into the conversation the view is
// drawing. This loop is the whole of the integration.
func (r *Runner) observe(ctx context.Context, sessionID string, out chan<- agent.AgentEvent) {
	defer close(out)
	defer func() {
		r.mu.Lock()
		delete(r.active, sessionID)
		r.mu.Unlock()
	}()

	var (
		current  turn
		cursor   int
		finished bool
	)
	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	flush := func(done bool) {
		if current.id == "" {
			return
		}
		msg := message.Message{
			ID: current.id, SessionID: sessionID, Role: message.Assistant,
			Parts: append([]message.ContentPart{}, current.parts...), Model: r.Model().ID,
		}
		_ = r.messages.Update(ctx, msg)
		ev := agent.AgentEvent{Type: agent.AgentEventTypeResponse, Message: msg, SessionID: sessionID, Done: done}
		select {
		case out <- ev:
		default:
		}
		r.publish(ev)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		status, err := r.client.Status(ctx, sessionID)
		if err != nil {
			out <- agent.AgentEvent{Type: agent.AgentEventTypeError, Error: err, SessionID: sessionID}
			return
		}
		events, err := r.client.Events(ctx, sessionID)
		if err == nil {
			// A shorter log means the daemon rotated or restarted; rewind rather
			// than return nothing forever.
			if cursor > len(events) {
				cursor = 0
			}
			for _, ev := range events[cursor:] {
				if r.fold(ev, sessionID, &current) {
					flush(false)
				}
			}
			cursor = len(events)
		}
		if terminal(status.Status) {
			if !finished {
				reason := message.FinishReasonEndTurn
				switch status.Status {
				case "failed":
					reason = message.FinishReasonError
				case "cancelled":
					reason = message.FinishReasonCanceled
				}
				current.parts = append(current.parts, message.Finish{Reason: reason, Time: time.Now().Unix()})
				flush(true)
				finished = true
			}
			return
		}
	}
}

// fold turns one protocol event into conversation state. It reports whether the
// view should redraw.
func (r *Runner) fold(ev Event, sessionID string, current *turn) bool {
	// A file change is a fact about the run, not a turn in the conversation, so
	// it is recorded without opening an assistant message for it.
	if ev.Kind == "file.changed" {
		path := stringOf(ev.Payload, "path")
		if path == "" {
			return false
		}
		r.recordChange(sessionID, Change{
			Path:      path,
			Operation: stringOf(ev.Payload, "operation"),
			Tool:      stringOf(ev.Payload, "tool"),
		})
		return false
	}
	if current.id == "" {
		msg, err := r.messages.Create(context.Background(), sessionID, message.CreateMessageParams{Role: message.Assistant})
		if err != nil {
			return false
		}
		current.id = msg.ID
	}
	switch ev.Kind {
	case "text_delta":
		text := stringOf(ev.Payload, "text")
		if text == "" {
			return false
		}
		if last, ok := lastText(current); ok {
			current.parts[last] = message.TextContent{Text: current.parts[last].(message.TextContent).Text + text}
		} else {
			current.parts = append(current.parts, message.TextContent{Text: text})
		}
		return true
	case "reasoning_delta":
		text := stringOf(ev.Payload, "text")
		if text == "" {
			return false
		}
		current.parts = append(current.parts, message.ReasoningContent{Thinking: text})
		return true
	case "tool_call_ready":
		name := stringOf(ev.Payload, "name")
		if name == "" {
			name = stringOf(ev.Payload, "tool")
		}
		if name == "" {
			return false
		}
		current.parts = append(current.parts, message.ToolCall{ID: stringOf(ev.Payload, "id"), Name: name, Type: "tool", Finished: true})
		return true
	case "permission_wait":
		if r.permissions != nil {
			r.permissions.Raise(permission.PermissionRequest{
				ID:        stringOf(ev.Payload, "request_id"),
				SessionID: sessionID,
				ToolName:  stringOf(ev.Payload, "tool"),
				Action:    stringOf(ev.Payload, "tool"),
			})
		}
		return true
	}
	return false
}

func lastText(t *turn) (int, bool) {
	for i := len(t.parts) - 1; i >= 0; i-- {
		if _, ok := t.parts[i].(message.TextContent); ok {
			return i, true
		}
	}
	return 0, false
}

// ListRuns folds the daemon's run list back into the client's session view, so a
// restarted client agrees with the harness instead of inventing history.
func (r *Runner) ListRuns(ctx context.Context) error {
	runs, err := r.client.List(ctx)
	if err != nil {
		return err
	}
	for _, run := range runs {
		if _, err := r.sessions.Save(ctx, session.Session{
			ID: run.RunID, Title: run.RunID, UpdatedAt: time.Now().Unix(),
		}); err != nil {
			return err
		}
	}
	return nil
}

// Approve answers a permission request, satisfying permission.Responder.
func (r *Runner) Approve(ctx context.Context, runID, requestID string) error {
	return r.client.Approve(ctx, runID, requestID)
}

// Deny refuses a permission request, satisfying permission.Responder.
func (r *Runner) Deny(ctx context.Context, runID, requestID, reason string) error {
	return r.client.Deny(ctx, runID, requestID, reason)
}

func terminal(status string) bool {
	switch status {
	case "running", "awaiting_approval":
		return false
	default:
		return true
	}
}

func firstLine(s string) string {
	if idx := strings.IndexAny(s, "\r\n"); idx >= 0 {
		s = s[:idx]
	}
	if len(s) > 60 {
		return s[:60]
	}
	return s
}

func stringOf(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if v, ok := payload[key].(string); ok {
		return v
	}
	if v, ok := payload[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// agentEvents gives the runner the subscription half of the agent surface.
type agentEvents struct {
	*pubsub.Broker[agent.AgentEvent]
}

func newAgentEvents() *agentEvents {
	return &agentEvents{Broker: pubsub.NewBroker[agent.AgentEvent]()}
}

func (a *agentEvents) publish(ev agent.AgentEvent) { a.Publish(pubsub.UpdatedEvent, ev) }
