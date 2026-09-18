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
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	prumo "github.com/raillen/prumo/sdk/prumo"

	"github.com/raillen/prumo-tui/internal/agent"
	"github.com/raillen/prumo-tui/internal/llm/models"
	"github.com/raillen/prumo-tui/internal/logging"
	"github.com/raillen/prumo-tui/internal/message"
	"github.com/raillen/prumo-tui/internal/permission"
	"github.com/raillen/prumo-tui/internal/pubsub"
	"github.com/raillen/prumo-tui/internal/session"
)

// PollInterval is how often the client asks the daemon for new events when
// falling back to polling.
//
// When a daemon does not offer a subscription (or when the stream disconnects),
// the client learns about progress by asking (ADR 014). It is a named constant
// because the cost of that fallback should be visible in one place rather than
// buried in a loop.
const PollInterval = 250 * time.Millisecond

// MaxPollFailures is how many polls in a row may fail before the client calls
// its link to the harness lost. It is a count rather than a duration because
// what a user needs to tell a retry from a hang is the progress, and two seconds
// of quiet is a hiccup while eight is a daemon that is gone.
const MaxPollFailures = 8

// The transport's shapes are re-exported under this package's names so the view
// layer never imports the SDK directly. The transport can change without
// touching a component.
type (
	StartRequest = prumo.StartRequest
	RunStatus    = prumo.RunStatus
	Event        = prumo.Event
	DiffResponse = prumo.DiffResponse
)

// Client is the slice of the SDK the runner needs. *prumo.Client satisfies it.
type Client interface {
	Start(ctx context.Context, req prumo.StartRequest) (string, error)
	Status(ctx context.Context, runID string) (prumo.RunStatus, error)
	Events(ctx context.Context, runID string) ([]prumo.Event, error)
	Cancel(ctx context.Context, runID string) error
	Steer(ctx context.Context, runID, message string) error
	Jobs(ctx context.Context) ([]prumo.Job, error)
	Unschedule(ctx context.Context, jobID string) error
	Approve(ctx context.Context, runID, requestID string) error
	Deny(ctx context.Context, runID, requestID, reason string) error
	List(ctx context.Context) ([]prumo.RunStatus, error)
	Models(ctx context.Context, r prumo.ModelsRequest) ([]string, error)
	ModelInfo(ctx context.Context, r prumo.ModelsRequest) ([]prumo.ModelInfo, error)
	Diff(ctx context.Context, runID, path string) (prumo.DiffResponse, error)
	Subscribe(ctx context.Context, runID string, from int) (<-chan prumo.Event, error)
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
	// cursors is how far into each run's timeline the client has folded. A
	// second run in a session appends to the same log, so folding from zero
	// would re-read what was already drawn and counted.
	cursors map[string]int
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
		cursors:     map[string]int{},
	}
}

// cursor reports how far into a run's timeline the client has already folded.
func (r *Runner) cursor(sessionID string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cursors[sessionID]
}

func (r *Runner) setCursor(sessionID string, at int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cursors == nil {
		r.cursors = map[string]int{}
	}
	r.cursors[sessionID] = at
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

// recordUsage adds one report of what a run spent to the session it belongs to.
//
// The harness meters the run and reports each model request; the client only
// sums what it was told. Nothing here is inferred, so a client that restarted
// and replayed the timeline arrives at the same total as one that never left.
func (r *Runner) recordUsage(sessionID string, payload map[string]any) {
	ctx := context.Background()
	current, err := r.sessions.Get(ctx, sessionID)
	if err != nil {
		current = session.Session{ID: sessionID}
	}
	current.PromptTokens += numberOf(payload, "prompt_tokens")
	current.CompletionTokens += numberOf(payload, "completion_tokens")
	current.CacheReadTokens += numberOf(payload, "cache_read_tokens")
	current.CacheWriteTokens += numberOf(payload, "cache_write_tokens")
	current.Cost += floatOf(payload, "cost_usd")
	current.UsageReports++
	if _, err := r.sessions.Save(ctx, current); err != nil {
		logging.ErrorPersist("cannot record what the run spent: " + err.Error())
	}
}

// resetUsage clears what a session is known to have spent, so a timeline that
// was rewritten is re-read rather than added to.
func (r *Runner) resetUsage(sessionID string) {
	ctx := context.Background()
	current, err := r.sessions.Get(ctx, sessionID)
	if err != nil {
		return
	}
	current.PromptTokens, current.CompletionTokens = 0, 0
	current.CacheReadTokens, current.CacheWriteTokens = 0, 0
	current.Cost, current.UsageReports = 0, 0
	if _, err := r.sessions.Save(ctx, current); err != nil {
		logging.ErrorPersist("cannot reset what the run spent: " + err.Error())
	}
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

// Run starts a run for a goal and returns the events it produces.
func (r *Runner) Run(ctx context.Context, sessionID, content string) (<-chan agent.AgentEvent, error) {
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
	// A model chosen from the harness's own list arrives as an id, with no
	// separate wire name: sending the empty APIModel would ask the daemon for
	// its default instead of for what the user picked.
	modelName := model.APIModel
	if modelName == "" {
		modelName = string(model.ID)
	}
	if _, err := r.client.Start(ctx, StartRequest{
		Goal:      content,
		Provider:  r.provider,
		Model:     modelName,
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

// Attach re-attaches the client to a run the daemon already has.
//
// This is what reconnecting is: the run is not restarted, it is re-read. The
// timeline is folded from wherever this client stopped — from the beginning,
// for a client that has just started — so the transcript and the spend are
// rebuilt from the record rather than remembered. A run that is still going
// keeps streaming from there; a finished one is read once and closed out.
//
// A run the daemon does not know is an error. Returning an empty conversation
// instead would be the client asserting something it was never told.
func (r *Runner) Attach(ctx context.Context, sessionID string) error {
	if r.client == nil {
		return errors.New("no harness attached")
	}
	if r.IsSessionBusy(sessionID) {
		// Already following it: a second reader would fold the same log twice.
		return nil
	}
	if _, err := r.client.Status(ctx, sessionID); err != nil {
		return err
	}
	if _, err := r.sessions.Get(ctx, sessionID); err != nil {
		if _, err := r.sessions.Save(ctx, session.Session{ID: sessionID, Title: sessionID, UpdatedAt: time.Now().Unix()}); err != nil {
			return err
		}
	}

	// The follow-outlives the call that asked for it, so it gets its own
	// context rather than the caller's, which is bounded.
	runCtx, cancel := context.WithCancel(context.Background())
	r.mu.Lock()
	r.active[sessionID] = cancel
	r.mu.Unlock()

	out := make(chan agent.AgentEvent, 64)
	go r.observe(runCtx, sessionID, out)
	return nil
}

// turn accumulates the assistant message for one run.
type turn struct {
	id string
	// at is when the message was opened. The store orders a conversation by
	// creation time, so a message that carried none would sort to the top of
	// the transcript — answers above the question that produced them.
	at    int64
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

// observe streams or polls the run and folds its events into the conversation the view is
// drawing.
func (r *Runner) observe(ctx context.Context, sessionID string, out chan<- agent.AgentEvent) {
	defer close(out)
	defer func() {
		r.mu.Lock()
		delete(r.active, sessionID)
		r.mu.Unlock()
	}()

	// The fold resumes where the last one stopped: the run log is appended to,
	// not restarted, so an earlier run's events are already on screen.
	var (
		current  turn
		cursor   = r.cursor(sessionID)
		finished bool
		// failures counts consecutive polls the daemon did not answer. One
		// failure is not a lost daemon, so the count is what decides.
		failures int
	)

	flush := func(done bool) {
		if current.id == "" {
			return
		}
		msg := message.Message{
			ID: current.id, SessionID: sessionID, Role: message.Assistant,
			Parts: append([]message.ContentPart{}, current.parts...), Model: r.Model().ID,
			CreatedAt: current.at,
		}
		_ = r.messages.Update(ctx, msg)
		r.emit(out, agent.AgentEvent{Type: agent.AgentEventTypeResponse, Message: msg, SessionID: sessionID, Done: done})
	}

	finishWithStatus := func(st string) {
		if !finished {
			reason := message.FinishReasonEndTurn
			switch st {
			case "failed":
				reason = message.FinishReasonError
			case "cancelled":
				reason = message.FinishReasonCanceled
			}
			current.parts = append(current.parts, message.Finish{Reason: reason, Time: time.Now().Unix()})
			flush(true)
			finished = true
		}
	}

	// Try push streaming subscription first (ADR 014).
	if subChan, err := r.client.Subscribe(ctx, sessionID, cursor); err == nil && subChan != nil {
		streamDone := false
		for !streamDone {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-subChan:
				if !ok {
					streamDone = true
					break
				}
				cursor++
				r.setCursor(sessionID, cursor)
				if r.fold(ev, sessionID, &current) {
					flush(false)
				}
				if ev.Kind == "run.finished" {
					st := stringOf(ev.Payload, "status")
					if st == "" {
						st = "complete"
					}
					if terminal(st) {
						finishWithStatus(st)
						return
					}
				}
			}
		}
		// If the stream closed, check if run has reached terminal status.
		if status, err := r.client.Status(ctx, sessionID); err == nil && terminal(status.Status) {
			finishWithStatus(status.Status)
			return
		}
	}

	// Fallback to polling when daemon does not offer subscription or when stream disconnects (ADR 014).
	cursor = r.cursor(sessionID)
	ticker := time.NewTicker(PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		status, err := r.client.Status(ctx, sessionID)
		if err != nil {
			// A daemon that stops answering is retried before it is given up on,
			// and the attempts are counted where a user can see them: a client
			// that retried in silence would be indistinguishable from one that
			// hung, and one that gave up on the first failure would report a
			// hiccup as a death.
			failures++
			if failures <= MaxPollFailures {
				r.publish(agent.AgentEvent{
					Type: agent.AgentEventTypeConnection, SessionID: sessionID,
					Connection: agent.Connection{
						State: agent.ConnectionReconnecting, Attempt: failures, Of: MaxPollFailures, Reason: err.Error(),
					},
				})
				continue
			}
			r.publish(agent.AgentEvent{
				Type: agent.AgentEventTypeConnection, SessionID: sessionID,
				Connection: agent.Connection{State: agent.ConnectionOffline, Attempt: MaxPollFailures, Of: MaxPollFailures, Reason: err.Error()},
			})
			r.emit(out, agent.AgentEvent{Type: agent.AgentEventTypeError, Error: err, SessionID: sessionID})
			return
		}
		if failures > 0 {
			failures = 0
			r.publish(agent.AgentEvent{
				Type: agent.AgentEventTypeConnection, SessionID: sessionID,
				Connection: agent.Connection{State: agent.ConnectionLive},
			})
		}
		events, err := r.client.Events(ctx, sessionID)
		if err == nil {
			// A shorter log means the daemon rotated or restarted; rewind rather
			// than return nothing forever. What was folded from the old log goes
			// with it: the totals are re-derived from the record that remains
			// instead of being added to what the vanished events already said.
			if cursor > len(events) {
				cursor = 0
				r.resetUsage(sessionID)
			}
			// The batch is folded whole and published once. Flushing per event
			// would take a snapshot per delta for frames nobody can see: the
			// view redraws at the poll's cadence, so the intermediate copies are
			// paid for and thrown away.
			changed := false
			for _, ev := range events[cursor:] {
				if r.fold(ev, sessionID, &current) {
					changed = true
				}
			}
			cursor = len(events)
			r.setCursor(sessionID, cursor)
			if changed {
				flush(false)
			}
		}
		if terminal(status.Status) {
			finishWithStatus(status.Status)
			return
		}
	}
}

// fold turns one protocol event into conversation state. It reports whether the
// view should redraw.
func (r *Runner) fold(ev Event, sessionID string, current *turn) bool {
	// A file change and a usage report are facts about the run, not turns in
	// the conversation, so both are recorded without opening an assistant
	// message for them.
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
	if ev.Kind == "usage" {
		r.recordUsage(sessionID, ev.Payload)
		return false
	}
	if current.id == "" {
		msg, err := r.messages.Create(context.Background(), sessionID, message.CreateMessageParams{Role: message.Assistant})
		if err != nil {
			return false
		}
		current.id, current.at = msg.ID, msg.CreatedAt
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
			// The arguments are what the gate is about, and they are what the
			// dialog draws its preview from: an approval asked for without them
			// is one a person can only answer on faith.
			r.permissions.Raise(permission.PermissionRequest{
				ID:        stringOf(ev.Payload, "request_id"),
				SessionID: sessionID,
				ToolName:  stringOf(ev.Payload, "tool"),
				Action:    stringOf(ev.Payload, "tool"),
				Params:    ev.Payload["arguments"],
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

// Steer adds a message to a run that is already going.
//
// It is the one way to talk to a run in flight, and it is deliberately not a
// second Run: the daemon owns the loop, and a client that started a second run
// to say "actually, also do this" would be running two.
func (r *Runner) Steer(ctx context.Context, sessionID, content string) error {
	if r.client == nil {
		return errors.New("no harness attached")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("there is nothing to say to the run")
	}
	if !r.IsSessionBusy(sessionID) {
		return errors.New("no run is in flight on this session")
	}
	if err := r.client.Steer(ctx, sessionID, content); err != nil {
		return err
	}
	// What was said belongs in the transcript: it is part of the conversation
	// even though the daemon is the one that acted on it.
	_, err := r.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:  message.User,
		Parts: []message.ContentPart{message.TextContent{Text: content}},
	})
	return err
}

// Job is a run the daemon repeats on a schedule.
//
// It is the transport's shape re-exported under this package's name, like the
// rest of the protocol surface: the view layer does not import the SDK.
type Job = prumo.Job

// Scheduled reports the runs the daemon repeats, in the order it gave them.
//
// The schedule belongs to the daemon: a client that kept its own list would be
// describing a queue it cannot start, stop or keep in step.
func (r *Runner) Scheduled(ctx context.Context) ([]Job, error) {
	if r.client == nil {
		return nil, errors.New("no harness attached")
	}
	return r.client.Jobs(ctx)
}

// Unschedule stops a recurring run. It is the one change a client may make to
// the daemon's queue, and it only ever removes.
func (r *Runner) Unschedule(ctx context.Context, jobID string) error {
	if r.client == nil {
		return errors.New("no harness attached")
	}
	if jobID == "" {
		return errors.New("there is no job to cancel")
	}
	return r.client.Unschedule(ctx, jobID)
}

// ExportDir is where a run's record is written when someone asks for it.
//
// It sits under the runtime directory because it is derived: the daemon's
// timeline is the record, and this is one rendering of it.
const ExportDir = ".prumo/runtime/exports"

// ExportTimeline writes a run's own record as plain text and returns the path.
//
// The export is a rendering of the daemon's timeline rather than of what the
// client folded from it: a file that exists to be read outside the client must
// not depend on the client's state. Keys are sorted so the same run exports the
// same bytes, and nothing is styled — the point of the file is to be readable
// where the terminal is not.
func (r *Runner) ExportTimeline(ctx context.Context, sessionID string) (string, error) {
	if r.client == nil {
		return "", errors.New("no harness attached")
	}
	if sessionID == "" {
		return "", errors.New("there is no session to export")
	}
	events, err := r.client.Events(ctx, sessionID)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "run %s\nevents %d\n\n", sessionID, len(events))
	for _, ev := range events {
		fmt.Fprintf(&b, "%s\t%s\n", ev.Kind, describePayload(ev.Payload))
	}

	path := filepath.Join(r.workspace, ExportDir, sessionID+".txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// describePayload renders a payload's keys in a fixed order, so two exports of
// the same run are the same file.
func describePayload(payload map[string]any) string {
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, payload[key]))
	}
	return strings.Join(parts, " ")
}

// ModelCapabilities is what one model declares it can do, as the view layer
// reads it: a list of words, and whether anyone declared them at all.
type ModelCapabilities struct {
	Declared bool
	Features []string
}

// ModelCatalogue asks the harness what the provider serves and what each model
// declares, in one call: the two questions are answered by the same operation.
func (r *Runner) ModelCatalogue(ctx context.Context, provider string) ([]string, map[string]ModelCapabilities, error) {
	if r.client == nil {
		return nil, nil, errors.New("no harness attached")
	}
	ids, err := r.client.Models(ctx, prumo.ModelsRequest{Provider: provider})
	if err != nil {
		return nil, nil, err
	}
	infos, err := r.client.ModelInfo(ctx, prumo.ModelsRequest{Provider: provider})
	if err != nil {
		return nil, nil, err
	}

	capabilities := make(map[string]ModelCapabilities, len(infos))
	for _, info := range infos {
		capabilities[info.ID] = ModelCapabilities{
			Declared: info.Declared,
			Features: featuresOf(info.Capabilities),
		}
	}
	return ids, capabilities, nil
}

// featuresOf names what a model declares, in a fixed order so the same model
// always reads the same way.
func featuresOf(c prumo.CapabilitySet) []string {
	features := make([]string, 0, 5)
	for _, feature := range []struct {
		name string
		set  bool
	}{
		{"text", c.Text}, {"reasoning", c.Reasoning}, {"vision", c.Vision},
		{"tools", c.Tools}, {"audio", c.Audio},
	} {
		if feature.set {
			features = append(features, feature.name)
		}
	}
	return features
}

// AvailableModels asks the harness what the provider can serve.
//
// It is a pass-through on purpose: the catalogue belongs to the provider, and
// the client's only job is to ask and show it.
func (r *Runner) AvailableModels(ctx context.Context, provider string) ([]string, error) {
	if r.client == nil {
		return nil, errors.New("no harness attached")
	}
	return r.client.Models(ctx, prumo.ModelsRequest{Provider: provider})
}

// Approve answers a permission request, satisfying permission.Responder.
func (r *Runner) Approve(ctx context.Context, runID, requestID string) error {
	return r.client.Approve(ctx, runID, requestID)
}

// Deny refuses a permission request, satisfying permission.Responder.
func (r *Runner) Deny(ctx context.Context, runID, requestID, reason string) error {
	return r.client.Deny(ctx, runID, requestID, reason)
}

// Diff asks the harness what a run changed in one file (ADR 014).
func (r *Runner) Diff(ctx context.Context, sessionID, path string) (DiffResponse, error) {
	if r.client == nil {
		return DiffResponse{}, errors.New("no harness attached")
	}
	return r.client.Diff(ctx, sessionID, path)
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

// numberOf reads a whole number from a decoded payload.
//
// The wire is JSON, so every number arrives as a float64 and the count fields
// have to be converted rather than asserted. A key that is missing or of
// another shape reads as zero: usage the client cannot parse is usage it does
// not claim.
func numberOf(payload map[string]any, key string) int64 {
	switch v := payload[key].(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0
		}
		return n
	}
	return 0
}

func floatOf(payload map[string]any, key string) float64 {
	switch v := payload[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0
		}
		return f
	}
	return 0
}

// emit hands an event to the run's reader and to the broker.
//
// Both paths are fed from one call on purpose: two delivery sites drift, and a
// run that reported an error only on the channel nobody reads would fail in
// silence. The channel is what the agent surface promises; the broker is what
// the running program actually consumes.
func (r *Runner) emit(out chan<- agent.AgentEvent, ev agent.AgentEvent) {
	select {
	case out <- ev:
	default:
	}
	r.publish(ev)
}

// agentEvents gives the runner the subscription half of the agent surface.
type agentEvents struct {
	*pubsub.Broker[agent.AgentEvent]
}

func newAgentEvents() *agentEvents {
	return &agentEvents{Broker: pubsub.NewBroker[agent.AgentEvent]()}
}

func (a *agentEvents) publish(ev agent.AgentEvent) { a.Publish(pubsub.UpdatedEvent, ev) }
