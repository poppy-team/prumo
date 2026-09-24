// Package runtime implements the NativeAgent reentrant state machine (HA2).
// No monolithic for{model()} loop: Step advances one Phase at a time and
// every safe point can persist/cancel/pause/resume/handoff/replay.
package runtime

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/checkpoint"
	"github.com/raillen/prumo/internal/harness/directive"
	"github.com/raillen/prumo/internal/harness/gateway"
	"github.com/raillen/prumo/internal/harness/model"
	"github.com/raillen/prumo/internal/harness/perm"
)

// EffectJournal is the before/after record around a side effect. It is the port
// the runtime depends on, not the concrete store, so the runtime does not own
// persistence.
type EffectJournal interface {
	// RecordIntent records that an effect is about to be applied and reports
	// whether it should go ahead.
	RecordIntent(agent.PendingEffect) (bool, error)
	// RecordOutcome records what became of it.
	RecordOutcome(id string, status agent.EffectStatus) error
}

// Services wires the ports the loop depends on (no vendor types).
type Services struct {
	Models      model.Provider
	Tools       ToolExecutor
	Perms       *perm.Engine
	Checkpoints *checkpoint.Store
	// EffectJournal is the before/after record around every tool call. Nil
	// disables replay protection, which is correct for a run with no checkpoint
	// and therefore no history to consult.
	EffectJournal EffectJournal
	Events        func(agent.AgentEvent)
	// ContextManifest builds the context pointer for a turn.
	ContextManifest func(ctx context.Context, state agent.NativeAgentState) (string, error)
	// Budgets enforcement hook; nil disables.
	// ReserveBudget holds an allowance before a model call and returns a
	// release that settles it against the real usage. It replaces a
	// ConsumeBudget-after-the-call hook: usage arrives once the call is paid
	// for, so a limit checked only there bounds nothing (GAP-098).
	ReserveBudget func(tokens float64) (release func(actualTokens, actualCost float64), err error)
	// BudgetExhausted is the preflight, consulted before a call is made.
	BudgetExhausted func() error
	// ToolSpecs advertises callable tools to the model (MCP servers, ACI
	// catalogs). Nil sends no specs; execution still policy-gated.
	ToolSpecs func() []agent.ToolSpec
	// RecordDiff records what a tool changed on disk, for on-demand diff reads (ADR 014).
	RecordDiff func(runID, path, kind, content string)
	// Workspace is the workspace root used to resolve relative file references.
	Workspace string
	// HasVision reports whether the selected model declares vision capability.
	HasVision bool
}

// ToolExecutor executes one normalized ToolCall.
type ToolExecutor interface {
	Execute(ctx context.Context, call agent.ToolCall) (agent.ToolResult, error)
	KindOf(toolName string) string
	// OperationOf reports the file change a tool makes (created, modified,
	// moved, deleted), or "" when it makes none it can name.
	OperationOf(toolName string) string
}

// Runner holds mutable conversation buffers; canonical state stays in
// agent.NativeAgentState and is checkpointed at safe points. Inbox and
// State are mutex-guarded: daemons may Inject steering input and snapshot
// state while the loop runs.
type Runner struct {
	Svc      Services
	State    agent.NativeAgentState
	Messages []agent.Message
	Turn     agent.Turn
	Events   []agent.ModelEvent
	ToolQ    []agent.ToolCall
	Obs      []agent.Observation
	// AfterSideEffects flips true once an observable effect applies; the
	// gateway must then refuse transparent fallback.
	AfterSideEffects bool
	MaxTurns         int
	TurnsDone        int
	// QualityGate, when set, vets completion inside EvaluateStop: a failing
	// gate turns completion into PhaseFailed instead of Checkpoint.
	QualityGate func() error
	// CompactKeep bounds conversation growth: when Messages exceed twice
	// CompactKeep at model-request time, oldest tool observations collapse
	// into one deterministic summary. Zero disables.
	CompactKeep int
	// CompactBudget auto-triggers compaction when estimated conversation
	// tokens exceed it (0 = off). Estimate uses the versioned table.
	CompactBudget int
	// Directive holds the compiled DirectiveIR governing this execution.
	Directive *directive.DirectiveIR
	// mu guards Messages and State for cross-goroutine Inject/StateCopy.
	// Step holds it for the whole phase advance, so no State/Messages write
	// happens without the lock.
	mu sync.Mutex
}

// NewRunner initializes a Run session.
func NewRunner(svc Services, runID, sessionID string) *Runner {
	return &Runner{
		Svc:      svc,
		State:    agent.NativeAgentState{RunID: runID, SessionID: sessionID, TurnID: "turn-1", Phase: agent.PhasePrepare, Revision: 1, UpdatedAt: agent.Now()},
		Turn:     agent.Turn{ID: "turn-1", RunID: runID, SessionID: sessionID, Index: 1, Status: "open", StartedAt: agent.Now()},
		MaxTurns: 10,
	}
}

// NewRunnerWithDirective initializes a Run session governed by a compiled
// DirectiveIR.
//
// The run id is a parameter because it is not the task id. A directive describes
// a task, and a run may execute several of them under one id the caller chose;
// taking the id from the directive meant a caller that passed its own run id
// still got a runner reporting a different one, and every event it emitted
// belonged to a run the caller was not watching (GAP-127).
func NewRunnerWithDirective(svc Services, dir *directive.DirectiveIR, runID, sessionID string) *Runner {
	r := NewRunner(svc, runID, sessionID)
	r.Directive = dir
	// Seed prompt projection as initial system directive
	r.Messages = []agent.Message{
		{
			ID:        "directive-init",
			Role:      agent.RoleSystem,
			Content:   dir.FormatAgentPrompt(),
			CreatedAt: agent.Now(),
		},
	}
	return r
}
func (r *Runner) emit(kind string, payload map[string]any) {
	if r.Svc.Events == nil {
		return
	}
	r.emitEvent(kind, payload)
}

// nextEventID returns a per-run event id and advances the counter.
//
// The id used to be derived from len(payload), so every event of the same shape
// got the same number: every text_delta was ev-2, every usage ev-6. A
// subscriber that deduplicates on id then dropped the live events as repeats of
// the replayed ones, and a client missed them (GAP-118). The counter is part of
// the state, so a run resumed from a checkpoint continues the sequence instead
// of reissuing ids it has already used.
func (r *Runner) nextEventID() string {
	r.State.EventSeq++
	return fmt.Sprintf("ev-%d", r.State.EventSeq)
}

// emitEvent is the one place an event is built, so the id cannot be derived
// differently by the locked and unlocked paths.
func (r *Runner) emitEvent(kind string, payload map[string]any) {
	r.Svc.Events(agent.AgentEvent{
		ID: r.nextEventID(), RunID: r.State.RunID, TurnID: r.State.TurnID,
		Kind: kind, Payload: payload, CreatedAt: agent.Now(),
	})
}

// Inject enqueues steering input consumed at the next model request.
func (r *Runner) Inject(m agent.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch r.State.Phase {
	case agent.PhaseComplete, agent.PhaseFailed:
		return fmt.Errorf("run %s is %s: cannot steer", r.State.RunID, r.State.Phase)
	}
	if m.CreatedAt == "" {
		m.CreatedAt = agent.Now()
	}
	r.Messages = append(r.Messages, m)
	return nil
}

// conversationTokensLocked estimates current conversation size.
// Caller must hold mu.
func (r *Runner) conversationTokensLocked() int {
	total := 0
	for _, m := range r.Messages {
		total += model.EstimateTokens(m.Content, "")
	}
	return total
}

// StateCopy snapshots canonical state for observers (status/replay)
// without racing the loop.
func (r *Runner) StateCopy() agent.NativeAgentState {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.State
}

// TryStateCopy returns the current state without waiting, and reports whether it
// could be read.
//
// StateCopy blocks for as long as a step is running, because a step holds the
// lock across the work it does — including a model call and a tool call, each of
// which can take seconds. That is fine for a caller that has nothing better to
// do and wrong for a reader that must not stall behind the thing it is reading
// about: a daemon answering "is this run parked?" cannot afford to wait for the
// run to finish before finding out.
//
// A busy runner reports false rather than a stale answer. A reader that cannot
// get the state knows the run is mid-step, which is the more useful of the two
// facts.
func (r *Runner) TryStateCopy() (agent.NativeAgentState, bool) {
	if !r.mu.TryLock() {
		return agent.NativeAgentState{}, false
	}
	defer r.mu.Unlock()
	return r.State, true
}

// maybeCompactLocked collapses oldest tool observations into a summary.
// Caller must hold mu.
func (r *Runner) maybeCompactLocked() {
	over := r.CompactKeep > 0 && len(r.Messages) > r.CompactKeep*2
	if !over && r.CompactBudget > 0 {
		over = r.conversationTokensLocked() > r.CompactBudget
	}
	if !over {
		return
	}
	keepN := r.CompactKeep
	if keepN <= 0 {
		keepN = 10
	}
	if keepN >= len(r.Messages) {
		return
	}
	keep := r.Messages[len(r.Messages)-keepN:]
	dropped := len(r.Messages) - keepN
	obs, tools := 0, 0
	for _, m := range r.Messages[:len(r.Messages)-keepN] {
		if m.Role == agent.RoleTool {
			obs++
		} else {
			tools++
		}
	}
	summary := agent.Message{
		ID: "compact-1", Role: agent.RoleSystem,
		Content:   fmt.Sprintf("compacted %d messages (%d tool observations, %d other); see checkpoint for full history", dropped, obs, tools),
		CreatedAt: agent.Now(),
	}
	r.Messages = append([]agent.Message{summary}, keep...)
}

func (r *Runner) emitLocked(kind string, payload map[string]any) {
	if r.Svc.Events == nil {
		return
	}
	r.emitEvent(kind, payload)
}

// SeedMessages replaces the conversation buffer. Drivers seed the initial
// goal before the loop starts; the lock keeps setup safe against a Serve
// loop that is already accepting steer/status ops.
func (r *Runner) SeedMessages(msgs []agent.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Messages = append([]agent.Message{}, msgs...)
}

// persist writes the current state as a new checkpoint revision. Safe points
// and permission yields both use it, because a run that stops for approval is
// exactly the run that has to be recoverable. Caller holds mu.
//
// The snapshot carries the conversation, the ready tool queue, the observations
// already produced and the effect flag, not just the state machine position.
// A phase without a history cannot be continued, so persisting only the state
// made resume impossible and turned it into a report instead (GAP-123).
func (r *Runner) persist() error {
	if r.Svc.Checkpoints == nil {
		return nil
	}
	r.State.Revision++
	r.State.UpdatedAt = agent.Now()
	cp := agent.Checkpoint{
		ID:               fmt.Sprintf("%s-r%d", r.State.RunID, r.State.Revision),
		RunID:            r.State.RunID,
		State:            r.State,
		Messages:         append([]agent.Message(nil), r.Messages...),
		ToolQ:            append([]agent.ToolCall(nil), r.ToolQ...),
		Obs:              append([]agent.Observation(nil), r.Obs...),
		AfterSideEffects: r.AfterSideEffects,
		TurnsDone:        r.TurnsDone,
		CreatedAt:        agent.Now(),
	}
	return r.Svc.Checkpoints.Save(cp)
}

// RestoreFrom rebuilds a runner's resumable state from a checkpoint: the state
// machine position plus the conversation, tool queue, observations and effect
// flag that position is meaningless without. It is the continuation counterpart
// of persist, and it is what makes a checkpoint a recoverable point rather than
// a status snapshot.
func (r *Runner) RestoreFrom(cp agent.Checkpoint) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.State = cp.State
	r.Messages = append([]agent.Message(nil), cp.Messages...)
	r.ToolQ = append([]agent.ToolCall(nil), cp.ToolQ...)
	r.Obs = append([]agent.Observation(nil), cp.Obs...)
	r.AfterSideEffects = cp.AfterSideEffects
	if cp.TurnsDone > 0 {
		r.TurnsDone = cp.TurnsDone
	}
}

// beginEffect records the intent to apply an effect and reports whether the
// caller should go ahead.
//
// A run may be replayed — resumed from a checkpoint, or restarted after a crash
// — and a tool call that already happened must not happen again. The journal is
// what knows. Without a store configured there is nothing to consult, so the
// answer is yes: the previous behaviour, and the only correct one when there is
// no history to consult.
func (r *Runner) beginEffect(effectID string, tc agent.ToolCall) (bool, error) {
	if r.Svc.EffectJournal == nil {
		return true, nil
	}
	target := ""
	if tc.Arguments != nil {
		target = fmt.Sprint(tc.Arguments["path"])
		if target == "<nil>" || target == "" {
			target = ""
		}
		if target == "" && tc.Name == "edit.move" {
			if to, ok := tc.Arguments["to"].(string); ok {
				target = to
			}
		}
	}
	effect := agent.PendingEffect{
		ID:               effectID,
		Kind:             r.Svc.Tools.KindOf(tc.Name),
		IdempotencyKey:   effectID,
		Target:           target,
		RecoveryPolicy:   r.recoveryPolicyFor(tc),
		ObservableEffect: r.Svc.Tools.OperationOf(tc.Name),
	}
	return r.Svc.EffectJournal.RecordIntent(effect)
}

// finishEffect records what became of an effect. A journal that cannot be
// written to is not turned into a run failure: the effect already happened, and
// reporting the run as failed would claim the opposite. The trace is emitted
// instead, so the gap is visible.
func (r *Runner) finishEffect(effectID string, status agent.EffectStatus) {
	if r.Svc.EffectJournal == nil {
		return
	}
	if err := r.Svc.EffectJournal.RecordOutcome(effectID, status); err != nil {
		r.emitLocked("side_effect_journal_failed", map[string]any{
			"effect_id": effectID, "status": string(status), "error": err.Error(),
		})
	}
}

// recoveryPolicyFor decides what a replay of this call should do. Reading and
// listing are safe to repeat. A change is not: repeating an edit whose outcome
// is unknown is how a file gets written twice, so the default is to stop and
// let a person decide.
func (r *Runner) recoveryPolicyFor(tc agent.ToolCall) string {
	switch r.Svc.Tools.KindOf(tc.Name) {
	case "read-only", "idempotent":
		return "retry"
	case "destructive", "side-effecting":
		return "skip"
	default:
		return "fail"
	}
}

// toolSpecs reports what the model is told it can call.
//
// The executor is the source: a runner that can run a tool can describe it, and
// asking every runner construction site to remember a separate wiring is how the
// daemon ended up sending no tools at all while the provider advertised
// tool-calling support (GAP-114). An explicit Services.ToolSpecs still wins, for
// a caller that wants a different view.
func (r *Runner) toolSpecs(ctx context.Context) []agent.ToolSpec {
	if r.Svc.ToolSpecs != nil {
		return r.Svc.ToolSpecs()
	}
	if r.Svc.Tools == nil {
		return nil
	}
	switch executor := r.Svc.Tools.(type) {
	case interface{ Specs() []agent.ToolSpec }:
		return executor.Specs()
	case interface {
		Specs(context.Context) ([]agent.ToolSpec, error)
	}:
		specs, err := executor.Specs(ctx)
		if err != nil {
			// A tool surface that cannot be listed is still executable; the model
			// simply does not hear about it. Failing the run here would take down
			// a turn over a listing.
			return nil
		}
		return specs
	default:
		return nil
	}
}

// TurnTokenAllowance is what one model call is assumed to cost when a budget is
// enforced. It is a reservation, not a prediction: the real usage settles it, and
// anything left over goes back. Its job is to make the ceiling bind the turn
// that would otherwise cross it.
const TurnTokenAllowance = 32_000

// reserveForCall takes the preflight and the reservation for one model call.
//
// Both halves matter and neither is sufficient alone. Exhausted stops a run
// that has already spent its budget. The reservation is what stops the turn
// that would take it past: without it, every turn is individually within the
// limit and the last one is unlimited.
func (r *Runner) reserveForCall(req agent.ModelRequest) (func(float64, float64), error) {
	if r.Svc.BudgetExhausted != nil {
		if err := r.Svc.BudgetExhausted(); err != nil {
			return nil, err
		}
	}
	if r.Svc.ReserveBudget == nil {
		return func(float64, float64) {}, nil
	}
	// A request with a large conversation needs a larger allowance, so the
	// reservation tracks the prompt rather than assuming one size.
	allowance := float64(TurnTokenAllowance)
	for _, msg := range req.Messages {
		// Four characters per token is the usual English ratio; it only has to
		// be the right order of magnitude for the reservation to bind.
		allowance += float64(len(msg.Content)) / 4
	}
	release, err := r.Svc.ReserveBudget(allowance)
	if err != nil {
		return nil, err
	}
	if release == nil {
		return func(float64, float64) {}, nil
	}
	return release, nil
}

// permissionRequestFor builds the request for one tool call. Both the
// evaluation and the fingerprint check go through it, so the two cannot drift:
// a fingerprint computed from a different shape of request would approve one
// call and answer another.
func permissionRequestFor(state agent.NativeAgentState, tc agent.ToolCall) agent.PermissionRequest {
	resource := ""
	if tc.Arguments != nil {
		resource = fmt.Sprint(tc.Arguments["path"])
	}
	return agent.PermissionRequest{
		ID: "perm-" + tc.ID, RunID: state.RunID, TurnID: state.TurnID,
		Action: tc.Name, Resource: resource,
		ArgumentsSummary: summarizeArguments(tc.Arguments),
	}
}

// summarizeArguments renders arguments deterministically so the fingerprint does
// not depend on map iteration order.
func summarizeArguments(args map[string]any) string {
	if len(args) == 0 {
		return ""
	}
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+fmt.Sprint(args[k]))
	}
	return strings.Join(parts, ";")
}

// pendingFingerprint recomputes the fingerprint of the tool call behind a
// pending request. It is derived from state, never from what the caller sent,
// which is the point: the caller's value is the claim being checked.
func (r *Runner) pendingFingerprint(requestID string) string {
	for _, tc := range r.State.PendingTools {
		if "perm-"+tc.ID == requestID {
			return perm.Fingerprint(permissionRequestFor(r.State, tc))
		}
	}
	return ""
}

// ResolvePermission answers a pending permission request: it records the
// decision in the engine and rewinds the run to re-evaluate it, so the turn
// continues from where it stopped. Denying needs no special case — the
// re-evaluation returns deny and the run fails the way a policy denial does.
//
// fingerprint must be the one the approver was shown. It is recomputed here from
// the tool call actually pending and compared, so an approval cannot be aimed at
// a different call that happens to carry the same request id (GAP-107).
func (r *Runner) ResolvePermission(requestID, fingerprint string, allow bool, actor, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.State.Phase != agent.PhaseYield || len(r.State.PendingPerms) == 0 {
		return fmt.Errorf("no pending permission for run %s", r.State.RunID)
	}
	if r.Svc.Perms == nil {
		return fmt.Errorf("no permission engine configured for run %s", r.State.RunID)
	}
	pending := false
	for _, id := range r.State.PendingPerms {
		if id == requestID {
			pending = true
		}
	}
	if !pending {
		return fmt.Errorf("no pending permission %q for run %s", requestID, r.State.RunID)
	}
	if fingerprint == "" {
		return fmt.Errorf("resolving %q requires the fingerprint of the pending request", requestID)
	}
	if actual := r.pendingFingerprint(requestID); actual == "" {
		return fmt.Errorf("cannot identify the tool call behind %q", requestID)
	} else if actual != fingerprint {
		return fmt.Errorf("fingerprint mismatch for %q: the pending request is %s", requestID, actual)
	}
	if allow {
		if _, err := r.Svc.Perms.Approve(requestID, fingerprint, actor); err != nil {
			return err
		}
	} else {
		if _, err := r.Svc.Perms.Deny(requestID, fingerprint, actor, reason); err != nil {
			return err
		}
	}
	r.State.PendingPerms = nil
	r.State.PendingTools = nil
	r.State.StopReason = ""
	r.State.Phase = agent.PhasePermissionCheck
	kind := "permission_approved"
	if !allow {
		kind = "permission_rejected"
	}
	r.emit(kind, map[string]any{"request_id": requestID, "actor": actor, "decision": kind})
	if err := r.persist(); err != nil {
		return err
	}
	return nil
}

// Step advances exactly one Phase. Callers loop until Complete/Failed,
// checkpointing between steps to survive process restarts. The whole
// advance runs under mu: Inject/StateCopy may enqueue steering and snapshot
// state while the loop runs, so the phase machine never mutates State or
// Messages lock-free. This cannot deadlock with the daemon: drivers acquire
// runner.mu only and never hold their own locks while blocking on it.
func (r *Runner) Step(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch r.State.Phase {
	case agent.PhasePrepare:
		r.State.Phase = agent.PhaseCompileContext
	case agent.PhaseCompileContext:
		id := "ctx-manifest"
		if r.Svc.ContextManifest != nil {
			var err error
			id, err = r.Svc.ContextManifest(ctx, r.State)
			if err != nil {
				return err
			}
		}
		r.State.ContextManifestID = id
		if r.Svc.ContextManifest != nil {
			// Only a configured hook produces content. The placeholder id is an
			// identifier — delivering it would put the literal text
			// "ctx-manifest" in front of the model as context, and would shift
			// every message in the conversation.
			r.deliverContextManifest(id)
		}
		r.State.Phase = agent.PhaseRequestModel
	case agent.PhaseRequestModel:
		r.maybeCompactLocked()
		resolvedMsgs, err := ResolveReferences(r.Messages, r.Svc.Workspace, r.Svc.HasVision)
		if err != nil {
			r.State.Phase = agent.PhaseFailed
			r.State.StopReason = err.Error()
			return err
		}
		r.Messages = resolvedMsgs

		specs := r.toolSpecs(ctx)
		req := agent.ModelRequest{
			RequestID: fmt.Sprintf("%s:%s:req", r.State.RunID, r.State.TurnID),
			RunID:     r.State.RunID, TurnID: r.State.TurnID,
			Messages: append([]agent.Message{}, r.Messages...),
			Tools:    specs,
			// Once the run has applied an observable effect, a provider that
			// fails mid-stream cannot be swapped for another: the next one has
			// not seen the tool results and would re-plan from a different
			// state. The run has to hand off explicitly. A routing layer that
			// could not see this would eventually retry the call on another
			// provider and duplicate the effect (GAP-102).
			NoTransparentFallback: r.AfterSideEffects,
		}
		// Budget preflight, before the call. Usage arrives afterwards, so
		// checking there bounds nothing: the last turn of a run is the one that
		// can cross the ceiling. The reservation covers what this call may cost
		// and is released when the real usage is known (GAP-098).
		release, budgetErr := r.reserveForCall(req)
		if budgetErr != nil {
			r.State.Phase = agent.PhaseFailed
			r.State.StopReason = "budget exhausted: " + budgetErr.Error()
			r.emitLocked("budget_exhausted", map[string]any{
				"reason": budgetErr.Error(), "turn": r.State.TurnID,
			})
			return budgetErr
		}
		ch, err := r.Svc.Models.Stream(ctx, req)
		if err != nil {
			release(0, 0)
			r.State.Phase = agent.PhaseFailed
			r.State.StopReason = err.Error()
			// A provider that runs its own tools failed after beginning, and the
			// router refused to move the turn. The run is not in the state where a
			// retry or a resume makes sense: the delegate already did work that
			// this run cannot see, attribute or undo. Marking it explicitly is what
			// stops a resume from quietly re-running a delegation whose effects
			// are still there (GAP-173).
			// The phase stays failed: the turn did fail, and there is no phase for
			// "failed with unaccounted side effects" — inventing one would put a
			// state the machine has no transition out of. What makes this different
			// from an ordinary model failure is carried by the typed error and said
			// plainly in the stop reason, so a resume does not read it as "the call
			// failed, try again" when the work already happened.
			if gateway.DelegateWorkInFlight(err) {
				r.State.StopReason = "delegated run failed after starting; work it did is not accounted for by this run and must be inspected before resuming: " + err.Error()
			}
			return err
		}
		// The stream reports the real cost once its usage event arrives; until
		// then the reservation stands.
		settled := false
		settle := func(tokens, cost float64) {
			if settled {
				return
			}
			settled = true
			release(tokens, cost)
		}
		r.Events = nil
		for ev := range ch {
			select {
			case <-ctx.Done():
				r.State.Phase = agent.PhaseYield
				r.State.StopReason = "cancelled"
				return ctx.Err()
			default:
			}
			r.Events = append(r.Events, ev)
			// The model's stream reaches clients as events, not only as state:
			// a client that could not see the deltas could render a run's
			// lifecycle but never its conversation. Payloads stay minimal
			// because the timeline is replayed whole.
			switch ev.Kind {
			case agent.EventTextDelta:
				r.emitLocked("text_delta", map[string]any{"text": ev.Text})
			case agent.EventReasoningDelta:
				r.emitLocked("reasoning_delta", map[string]any{"text": ev.Text})
			case agent.EventToolCallReady:
				if ev.ToolCall != nil {
					r.emitLocked("tool_call_ready", map[string]any{"id": ev.ToolCall.ID, "name": ev.ToolCall.Name})
				}
			}
			if ev.Kind == agent.EventUsageUpdated && ev.Usage != nil {
				// What a run spent belongs on its record, not only in the
				// budget: a client that cannot read usage can only render a
				// statusline that is wrong, and the timeline is replayed whole
				// so the total survives a reconnect.
				// A metered input token, a cached one and an output token cost
				// differently, so they travel separately: a client that only
				// received a total could report what a run spent but never why.
				r.emitLocked("usage", map[string]any{
					"prompt_tokens":      ev.Usage.InputTokens,
					"completion_tokens":  ev.Usage.OutputTokens,
					"cache_read_tokens":  ev.Usage.CacheReadTokens,
					"cache_write_tokens": ev.Usage.CacheWriteTokens,
					"cost_usd":           ev.Usage.CostUSD,
				})
				// The call is accounted for: the reservation becomes the real
				// cost, so the next turn's preflight sees the truth.
				// The real usage settles the reservation. It counts every token the provider
				// processed, so a cached run cannot settle below the capacity it used
				// while the reservation was held at full size (GAP-130).
				settle(float64(ev.Usage.TotalTokens()), ev.Usage.CostUSD)
			}
			if ev.Kind == agent.EventError {
				// Any error ends the turn, retryable or not.
				//
				// A retryable error is one the gateway would normally retry, but
				// the gateway is not in this path yet (GAP-102), so the runtime is
				// handed the raw provider. Falling through on a retryable error
				// therefore meant a 429, a 500 or a stream cut mid-answer was
				// swallowed: the loop advanced as though the model had finished,
				// leaving a partial reply in the transcript and a run recorded as
				// successful. The flag is kept on the error so the layer that does
				// retry can still tell the two apart (GAP-131).
				//
				// A failed call spent nothing that will be reported, so the
				// reservation is released rather than left held against a limit
				// that was never crossed.
				settle(0, 0)
				r.State.Phase = agent.PhaseFailed
				r.State.StopReason = ev.Error
				if ev.Retryable {
					return &retryableModelError{cause: fmt.Errorf("model error: %s", ev.Error)}
				}
				return fmt.Errorf("model error: %s", ev.Error)
			}
		}
		// A stream that ended without a usage event still released its hold:
		// the reservation exists to cover an unreported call, not to be spent.
		settle(0, 0)
		r.State.Phase = agent.PhaseConsumeModelEvent
	case agent.PhaseConsumeModelEvent:
		r.State.Phase = agent.PhasePlanToolCalls
	case agent.PhasePlanToolCalls:
		r.ToolQ = nil
		for _, ev := range r.Events {
			if ev.Kind == agent.EventToolCallReady && ev.ToolCall != nil {
				r.ToolQ = append(r.ToolQ, *ev.ToolCall)
			}
		}
		if len(r.ToolQ) == 0 {
			r.State.Phase = agent.PhaseEvaluateStop
		} else {
			r.State.Phase = agent.PhasePermissionCheck
		}
	case agent.PhasePermissionCheck:
		if r.Svc.Perms == nil {
			r.State.Phase = agent.PhaseExecuteTool
			break
		}
		for _, tc := range r.ToolQ {
			kind := ""
			if r.Svc.Tools != nil {
				kind = r.Svc.Tools.KindOf(tc.Name)
			}
			res := r.Svc.Perms.Evaluate(permissionRequestFor(r.State, tc), kind, "policy")
			if res.Decision == agent.PermissionDeny {
				r.State.Phase = agent.PhaseFailed
				r.State.StopReason = "permission denied: " + res.Reason
				r.emit("permission_denied", map[string]any{"tool": tc.Name, "reason": res.Reason})
				return fmt.Errorf("permission denied for %s: %s", tc.Name, res.Reason)
			}
			if res.Decision == agent.PermissionAsk {
				r.State.Phase = agent.PhaseYield
				r.State.StopReason = "permission wait: " + tc.ID
				r.State.PendingPerms = []string{res.RequestID}
				// The pending call travels with the state: a resumed run (or a
				// daemon answering the request) needs the tool it stopped on,
				// and the request id is what a client answers with.
				r.State.PendingTools = append([]agent.ToolCall{}, r.ToolQ...)
				// The arguments and the fingerprint travel with the request, which
				// is the one place this timeline is not minimal: a gate asks a
				// person to approve what a tool is about to do, and a request that
				// carries no evidence cannot be answered — only obeyed or refused
				// on faith. The fingerprint is what the answer must quote back, so
				// approval is bound to this call and not merely to its id.
				// The cost is bounded by the calls a policy gates, and the
				// timeline already carries them in the checkpoint.
				r.emit("permission_wait", map[string]any{
					"tool": tc.Name, "request_id": res.RequestID, "arguments": tc.Arguments,
					"fingerprint": res.Fingerprint,
				})
				if err := r.persist(); err != nil {
					// Refusing to wait is honest: a pending approval nobody can
					// find on disk is worse than a failed run.
					r.State.Phase = agent.PhaseFailed
					r.State.StopReason = "checkpoint failed at permission wait: " + err.Error()
					return err
				}
				return nil
			}
		}
		r.State.Phase = agent.PhaseExecuteTool
	case agent.PhaseExecuteTool:
		if r.Svc.Tools == nil {
			return fmt.Errorf("no tool executor")
		}
		for _, tc := range r.ToolQ {
			if r.Directive != nil {
				targetPath := fmt.Sprint(tc.Arguments["path"])
				if targetPath == "" && tc.Name == "edit.move" {
					targetPath = fmt.Sprint(tc.Arguments["to"])
				}
				if targetPath != "" && targetPath != "<nil>" {
					kind := ""
					if r.Svc.Tools != nil {
						kind = r.Svc.Tools.KindOf(tc.Name)
					}
					if kind == "side-effecting" || kind == "destructive" {
						if !r.Directive.CanMutatePath(targetPath) {
							r.State.Phase = agent.PhaseFailed
							r.State.StopReason = fmt.Sprintf("scope firewall violation: path '%s' is not allowed for mutation", targetPath)
							r.emitLocked("scope_violation", map[string]any{"path": targetPath, "tool": tc.Name})
							return fmt.Errorf("directive scope violation: path '%s' forbidden", targetPath)
						}
					}
				}
			}
			// The effect journal is written around the call, not only in tests.
			// The intent goes down before the tool runs and the outcome after it
			// returns, so a process that dies mid-call leaves a pending record
			// rather than no trace. Without this the journal described a mechanism
			// that nothing used (GAP-126).
			effectID := "fx-" + tc.ID
			proceed, journalErr := r.beginEffect(effectID, tc)
			if journalErr != nil {
				r.State.Phase = agent.PhaseFailed
				r.State.StopReason = journalErr.Error()
				r.emitLocked("side_effect_refused", map[string]any{"tool": tc.Name, "reason": journalErr.Error()})
				return journalErr
			}
			if !proceed {
				// Already applied in an earlier life of this run. The effect is
				// not repeated; the run continues as though it had happened.
				r.emitLocked("side_effect_skipped", map[string]any{
					"tool": tc.Name, "effect_id": effectID,
					"reason": "an earlier attempt of this effect is already recorded as applied",
				})
				continue
			}
			res, err := r.Svc.Tools.Execute(ctx, tc)
			if err != nil {
				r.finishEffect(effectID, agent.EffectFailed)
				return err
			}
			if res.ExitCode == 0 {
				r.finishEffect(effectID, agent.EffectApplied)
			} else {
				// The tool reported a failure rather than the transport failing:
				// the effect is known not to have taken, so replay is safe.
				r.finishEffect(effectID, agent.EffectFailed)
			}
			// The observation is a faithful record of what the tool did. A tool
			// that failed put the reason in Error and left Output empty, and only
			// Output was kept, so the model was handed an empty result for a call
			// that had failed (GAP-116).
			r.Obs = append(r.Obs, toolObservation(r.State.TurnID, tc, res))
			if !res.OK() {
				// A failed tool is an event a client needs to see, not only a line
				// in the conversation: a run that is quietly making no progress
				// looks exactly like one that is working.
				r.emitLocked("tool.failed", map[string]any{
					"tool": tc.Name, "tool_call_id": tc.ID,
					"exit_code": res.ExitCode, "error": res.Error,
				})
			}
			if res.ExitCode == 0 {
				r.AfterSideEffects = true
				if op := r.Svc.Tools.OperationOf(tc.Name); op != "" {
					path, _ := tc.Arguments["path"].(string)
					if path == "" && tc.Name == "edit.move" {
						path, _ = tc.Arguments["to"].(string)
					}
					if path != "" {
						r.emitLocked("file.changed", map[string]any{"path": path, "operation": op, "tool": tc.Name})
						if r.Svc.RecordDiff != nil {
							kind, content := diffOf(tc)
							r.Svc.RecordDiff(r.State.RunID, path, kind, content)
						}
					}
				}
			}
		}
		r.State.Phase = agent.PhaseRecordObservation
	case agent.PhaseRecordObservation:
		r.State.Phase = agent.PhaseEvaluateStop
	case agent.PhaseEvaluateStop:
		r.TurnsDone++
		// The turn is over, so its identity is spent. TurnID was set once when the
		// runner was built and never moved, which meant every model request in a
		// run carried the same turn and the same request id — and since a tool
		// call's idempotency key is derived from the request id, two different
		// turns calling the same tool collided on it. Observations were equally
		// untraceable, since they all carried one turn (GAP-117).
		r.advanceTurn()
		if r.TurnsDone >= r.MaxTurns {
			if r.QualityGate != nil {
				if err := r.QualityGate(); err != nil {
					r.State.Phase = agent.PhaseFailed
					r.State.StopReason = err.Error()
					return err
				}
			}
			r.State.Phase = agent.PhaseCheckpoint
			r.State.StopReason = "max turns reached"
			break
		}
		// Completion when last model events contained Completed and no tools.
		completed := false
		for _, ev := range r.Events {
			if ev.Kind == agent.EventCompleted {
				completed = true
			}
		}
		if completed && len(r.ToolQ) == 0 {
			if r.QualityGate != nil {
				if err := r.QualityGate(); err != nil {
					r.State.Phase = agent.PhaseFailed
					r.State.StopReason = err.Error()
					return err
				}
			}
			r.State.Phase = agent.PhaseCheckpoint
			r.State.StopReason = "completed"
		} else if len(r.ToolQ) > 0 {
			// Feed the round trip back: what the assistant asked for, then what
			// each call returned.
			//
			// Only the results used to be recorded, so the follow-up request
			// carried a tool message with nothing to attach it to. A conversation
			// that reports a result for a call it never records asking for is not
			// a conversation any provider accepts (GAP-115).
			if len(r.ToolQ) > 0 {
				r.Messages = append(r.Messages, agent.Message{
					ID:        "msg-toolreq-" + r.State.TurnID,
					TurnID:    r.State.TurnID,
					Role:      agent.RoleAgent,
					ToolCalls: append([]agent.ToolCall{}, r.ToolQ...),
					CreatedAt: agent.Now(),
				})
			}
			for _, o := range r.Obs {
				r.Messages = append(r.Messages, agent.Message{
					ID: o.ID, TurnID: o.TurnID, Role: agent.RoleTool,
					Content: o.Content, ToolCallID: o.ToolCallID,
					Metadata: observationMeta(o), CreatedAt: agent.Now(),
				})
			}
			r.Obs = nil
			r.State.Phase = agent.PhaseRequestModel
		} else {
			r.State.Phase = agent.PhaseCheckpoint
			r.State.StopReason = "no tool calls and no completion"
		}
	case agent.PhaseCheckpoint:
		if err := r.persist(); err != nil {
			return err
		}
		if r.State.StopReason == "completed" {
			r.State.Phase = agent.PhaseComplete
		} else if r.TurnsDone >= r.MaxTurns {
			r.State.Phase = agent.PhaseComplete
		} else {
			r.State.Phase = agent.PhaseYield
		}
	case agent.PhaseYield, agent.PhaseComplete, agent.PhaseFailed:
		return nil
	default:
		return fmt.Errorf("unknown phase %s", r.State.Phase)
	}
	r.State.UpdatedAt = agent.Now()
	return nil
}

// RunUntilDone steps until Complete/Failed/Yield or ctx cancel.
func (r *Runner) RunUntilDone(ctx context.Context) error {
	for i := 0; i < 1000; i++ {
		if r.State.Phase == agent.PhaseComplete || r.State.Phase == agent.PhaseFailed || r.State.Phase == agent.PhaseYield {
			return nil
		}
		if err := r.Step(ctx); err != nil {
			// Permission wait yields without error propagation beyond state.
			if r.State.Phase == agent.PhaseYield {
				return nil
			}
			return err
		}
	}
	return fmt.Errorf("runaway loop guard")
}

func diffOf(tc agent.ToolCall) (kind, content string) {
	switch tc.Name {
	case "edit.patch":
		patch, _ := tc.Arguments["patch"].(string)
		return "patch", patch
	case "edit.create":
		cnt, _ := tc.Arguments["content"].(string)
		return "created", cnt
	case "edit.delete":
		return "deleted", ""
	case "edit.move":
		from, _ := tc.Arguments["from"].(string)
		to, _ := tc.Arguments["to"].(string)
		return "moved", fmt.Sprintf("moved from %s to %s", from, to)
	default:
		return "modified", ""
	}
}

// PendingFingerprint reports the fingerprint of the tool call behind a pending
// request, so a client can be shown the content it is being asked to approve.
// It returns "" for an unknown request.
func (r *Runner) PendingFingerprint(requestID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.pendingFingerprint(requestID)
}

// observationMeta carries a tool result's verdict into the message, so a
// serializer that only knows role and content can still render the failure
// faithfully rather than passing an empty result through as prose.
func observationMeta(obs agent.Observation) map[string]any {
	if obs.OK && obs.Error == "" {
		return nil
	}
	meta := map[string]any{"ok": obs.OK, "exit_code": obs.ExitCode}
	if obs.Error != "" {
		meta["error"] = obs.Error
	}
	if obs.Truncated {
		meta["truncated"] = true
	}
	return meta
}

// toolObservation records one tool result as the conversation sees it.
//
// The verdict is the tool's own, not an inference from whether the output
// happens to be empty: a tool that legitimately returns nothing succeeded, and a
// tool that failed usually returns nothing at all. Only the tool knows which.
func toolObservation(turnID string, call agent.ToolCall, res agent.ToolResult) agent.Observation {
	ok := res.OK()
	obs := agent.Observation{
		ID:         "obs-" + call.ID,
		TurnID:     turnID,
		ToolCallID: call.ID,
		Content:    res.Output,
		OK:         ok,
		ExitCode:   res.ExitCode,
		Truncated:  res.Truncated,
		CreatedAt:  agent.Now(),
	}
	if !ok {
		// A failure with no reason would read as a success that returned nothing,
		// which is the exact confusion this closes. Saying so is better than
		// leaving it blank.
		reason := res.Error
		if reason == "" {
			reason = fmt.Sprintf("tool %s exited %d", call.Name, res.ExitCode)
		}
		obs.Error = reason
	}
	return obs
}

// retryableModelError marks a model failure an upper layer may retry. The
// runtime ends the turn either way — it has no retry policy of its own and
// continuing past a failed call is what turns a partial answer into a successful
// run — but it says which failures were the provider's fault and which were not,
// so the caller does not have to parse the message.
type retryableModelError struct {
	cause error
}

func (e *retryableModelError) Error() string { return e.cause.Error() }
func (e *retryableModelError) Unwrap() error { return e.cause }

// Retryable reports whether the model failure may be retried.
func (e *retryableModelError) Retryable() bool { return true }

// Retryable reports whether err is a model failure worth retrying.
func Retryable(err error) bool {
	r, ok := err.(interface{ Retryable() bool })
	return ok && r.Retryable()
}

// advanceTurn moves the run to the next turn.
//
// The number comes from TurnsDone rather than from parsing the current id, so a
// resumed run continues the sequence instead of reusing an identifier a previous
// life already spent — which is the same collision the advance exists to
// prevent. A restored run with TurnsDone=5 goes on to turn-6.
func (r *Runner) advanceTurn() {
	r.State.TurnID = fmt.Sprintf("turn-%d", r.TurnsDone+1)
}

// deliverContextManifest puts the compiled context in front of the model.
//
// It was not delivered at all. The manifest was compiled, written to disk, and
// its id stored in state — and then nothing read either. The model request is
// built from the conversation, so the files the run decided were relevant, the
// pressure level and the token estimate never reached the model that was supposed
// to act on them. The hook returns the manifest's text; the id was never a thing
// anyone could consume (GAP-128).
func (r *Runner) deliverContextManifest(manifest string) {
	manifest = strings.TrimSpace(manifest)
	if manifest == "" {
		return
	}
	// Idempotent across turns: the manifest is context for the whole run, and
	// re-adding it each turn would grow the conversation with a copy of itself.
	for _, message := range r.Messages {
		if message.ID == contextManifestMessageID {
			return
		}
	}
	r.Messages = append([]agent.Message{{
		ID:        contextManifestMessageID,
		Role:      agent.RoleSystem,
		Content:   manifest,
		CreatedAt: agent.Now(),
	}}, r.Messages...)
}

// contextManifestMessageID is where the compiled context lives in the
// conversation.
const contextManifestMessageID = "context-manifest"
