// Package runtime implements the NativeAgent reentrant state machine (HA2).
// No monolithic for{model()} loop: Step advances one Phase at a time and
// every safe point can persist/cancel/pause/resume/handoff/replay.
package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/checkpoint"
	"github.com/raillen/prumo/internal/harness/directive"
	"github.com/raillen/prumo/internal/harness/model"
	"github.com/raillen/prumo/internal/harness/perm"
)

// Services wires the ports the loop depends on (no vendor types).
type Services struct {
	Models      model.Provider
	Tools       ToolExecutor
	Perms       *perm.Engine
	Checkpoints *checkpoint.Store
	Events      func(agent.AgentEvent)
	// ContextManifest builds the context pointer for a turn.
	ContextManifest func(ctx context.Context, state agent.NativeAgentState) (string, error)
	// Budgets enforcement hook; nil disables.
	ConsumeBudget func(usage agent.Usage) error
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

// NewRunnerWithDirective initializes a Run session governed by a compiled DirectiveIR.
func NewRunnerWithDirective(svc Services, dir *directive.DirectiveIR, sessionID string) *Runner {
	r := NewRunner(svc, dir.TaskID, sessionID)
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
	r.Svc.Events(agent.AgentEvent{ID: fmt.Sprintf("ev-%d", len(payload)+1), RunID: r.State.RunID, TurnID: r.State.TurnID, Kind: kind, Payload: payload, CreatedAt: agent.Now()})
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
	r.Svc.Events(agent.AgentEvent{ID: fmt.Sprintf("ev-%d", len(payload)+1), RunID: r.State.RunID, TurnID: r.State.TurnID, Kind: kind, Payload: payload, CreatedAt: agent.Now()})
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
func (r *Runner) persist() error {
	if r.Svc.Checkpoints == nil {
		return nil
	}
	r.State.Revision++
	r.State.UpdatedAt = agent.Now()
	cp := agent.Checkpoint{
		ID:        fmt.Sprintf("%s-r%d", r.State.RunID, r.State.Revision),
		RunID:     r.State.RunID,
		State:     r.State,
		CreatedAt: agent.Now(),
	}
	return r.Svc.Checkpoints.Save(cp)
}

// ResolvePermission answers a pending permission request: it records the
// decision in the engine and rewinds the run to re-evaluate it, so the turn
// continues from where it stopped. Denying needs no special case — the
// re-evaluation returns deny and the run fails the way a policy denial does.
func (r *Runner) ResolvePermission(requestID string, allow bool, actor, reason string) error {
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
	if allow {
		r.Svc.Perms.Approve(requestID, actor)
	} else {
		r.Svc.Perms.Deny(requestID, actor, reason)
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

		specs := []agent.ToolSpec{}
		if r.Svc.ToolSpecs != nil {
			specs = r.Svc.ToolSpecs()
		}
		req := agent.ModelRequest{
			RequestID: fmt.Sprintf("%s:%s:req", r.State.RunID, r.State.TurnID),
			RunID:     r.State.RunID, TurnID: r.State.TurnID,
			Messages: append([]agent.Message{}, r.Messages...),
			Tools:    specs,
		}
		ch, err := r.Svc.Models.Stream(ctx, req)
		if err != nil {
			r.State.Phase = agent.PhaseFailed
			r.State.StopReason = err.Error()
			return err
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
				if r.Svc.ConsumeBudget != nil {
					if err := r.Svc.ConsumeBudget(*ev.Usage); err != nil {
						r.State.Phase = agent.PhaseFailed
						r.State.StopReason = "budget exhausted: " + err.Error()
						return err
					}
				}
			}
			if ev.Kind == agent.EventError && !ev.Retryable {
				r.State.Phase = agent.PhaseFailed
				r.State.StopReason = ev.Error
				return fmt.Errorf("model error: %s", ev.Error)
			}
		}
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
			res := r.Svc.Perms.Evaluate(agent.PermissionRequest{
				ID: fmt.Sprintf("perm-%s", tc.ID), RunID: r.State.RunID, TurnID: r.State.TurnID,
				Action: tc.Name, Resource: fmt.Sprint(tc.Arguments["path"]),
			}, kind, "policy")
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
				// The arguments travel with the request, which is the one place
				// this timeline is not minimal: a gate asks a person to approve
				// what a tool is about to do, and a request that carries no
				// evidence cannot be answered — only obeyed or refused on faith.
				// The cost is bounded by the calls a policy gates, and the
				// timeline already carries them in the checkpoint.
				r.emit("permission_wait", map[string]any{
					"tool": tc.Name, "request_id": res.RequestID, "arguments": tc.Arguments,
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
			res, err := r.Svc.Tools.Execute(ctx, tc)
			if err != nil {
				return err
			}
			r.Obs = append(r.Obs, agent.Observation{ID: "obs-" + tc.ID, TurnID: r.State.TurnID, ToolCallID: tc.ID, Content: res.Output, CreatedAt: agent.Now()})
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
			// Feed observations back as messages and continue.
			for _, o := range r.Obs {
				r.Messages = append(r.Messages, agent.Message{ID: o.ID, TurnID: o.TurnID, Role: agent.RoleTool, Content: o.Content, CreatedAt: agent.Now()})
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
